package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	_ "github.com/go-sql-driver/mysql"
)

// AI TOK
// ======== ENV VARS ========
// R2_ACCOUNT_ID           (wajib)
// R2_ACCESS_KEY_ID        (wajib)
// R2_SECRET_ACCESS_KEY    (wajib)
// R2_BUCKET               (wajib)
// MYSQL_DSN               (wajib) -> contoh: user:pass@tcp(localhost:3306)/dbname?parseTime=true
// R2_REGION               (opsional, default "auto")
// R2_ENDPOINT             (opsional)
// PUBLIC_BASE_URL         (opsional)
// PORT                    (opsional, default "8080")

type Uploader struct {
	s3     *s3.Client
	bucket string
	public string
	db     *sql.DB
}

func main() {
	// === koneksi ke MySQL ===
	dsn := mustGetenv("MYSQL_DSN")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("DB connect error: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("DB ping error: %v", err)
	}

	// === R2 config ===
	accountID := mustGetenv("R2_ACCOUNT_ID")
	accessKey := mustGetenv("R2_ACCESS_KEY_ID")
	secretKey := mustGetenv("R2_SECRET_ACCESS_KEY")
	bucket := mustGetenv("R2_BUCKET")
	region := getenv("R2_REGION", "auto")
	publicBase := getenv("PUBLIC_BASE_URL", "")

	endpoint := os.Getenv("R2_ENDPOINT")
	if endpoint == "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)
	}

	s3cli, err := newR2Client(endpoint, region, accessKey, secretKey)
	if err != nil {
		log.Fatalf("init R2 client error: %v", err)
	}

	u := &Uploader{s3: s3cli, bucket: bucket, public: strings.TrimRight(publicBase, "/"), db: db}

	// routes
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})
	// POST
	http.HandleFunc("/upload", u.handleUpload)
	http.HandleFunc("/files", u.handleListFiles)

	addr := ":" + getenv("PORT", "8080")
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, corsMiddleware(http.DefaultServeMux)))
}

func newR2Client(endpoint, region, accessKey, secretKey string) (*s3.Client, error) {
	resolver := aws.EndpointResolverWithOptionsFunc(func(service, r string, options ...interface{}) (aws.Endpoint, error) {
		if service == s3.ServiceID {
			return aws.Endpoint{
				URL:               endpoint,
				HostnameImmutable: true,
			}, nil
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}
	})

	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
		config.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		return nil, err
	}

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
	}), nil
}

func (u *Uploader) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use POST multipart/form-data"})
		return
	}

	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid multipart: " + err.Error()})
		return
	}

	// ambil user_id & tahun dari form
	userID := r.FormValue("user_id")
	nama := r.FormValue("nama")
	kode_subkegiatan := r.FormValue("kode_subkegiatan")
	kode_opd := r.FormValue("kode_opd")
	tahunStr := r.FormValue("tahun")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing user_id"})
		return
	}
	tahun, _ := strconv.Atoi(tahunStr)

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing form file 'file': " + err.Error()})
		return
	}
	defer file.Close()

	key := r.FormValue("key")
	if key == "" {
		now := time.Now()
		key = fmt.Sprintf("%04d/%02d/%02d/%d-%s",
			now.Year(), now.Month(), now.Day(), now.UnixNano(),
			safeName(header.Filename))
	}

	ctype := header.Header.Get("Content-Type")
	if ctype == "" {
		ctype = guessContentType(header, file)
		if seeker, ok := file.(io.Seeker); ok {
			seeker.Seek(0, io.SeekStart)
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	// upload ke R2
	_, err = u.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(ctype),
	})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upload failed: " + err.Error()})
		return
	}

	// simpan metadata ke DB
	fileURL := u.publicURL(key)
	size := header.Size

	res, err := u.db.Exec(`
        INSERT INTO user_files (user_id, nama, kode_subkegiatan, kode_opd, file_name, file_url, file_size, bucket, content_type, tahun)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, nama, kode_subkegiatan, kode_opd, header.Filename, fileURL, size, u.bucket, ctype, tahun,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "db insert failed: " + err.Error()})
		return
	}
	id, _ := res.LastInsertId()

	resp := UserData{
		ID:              id,
		UserID:          userID,
		Nama:            nama,
		KodeSubkegiatan: kode_subkegiatan,
		KodeOpd:         kode_opd,
		FileName:        header.Filename,
		FileURL:         fileURL,
		FileSize:        size,
		ContentType:     ctype,
		Tahun:           tahun,
		CreatedAt:       time.Now(),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (u *Uploader) handleListFiles(w http.ResponseWriter, r *http.Request) {
	kodeOpd := r.URL.Query().Get("kode_opd")
	if kodeOpd == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "kode_opd harus ada"})
		return
	}
	tahunStr := r.URL.Query().Get("tahun")
	if tahunStr == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "tahun harus ada"})
		return
	}
	tahun, _ := strconv.ParseInt(tahunStr, 10, 64)

	rows, err := u.db.Query(`SELECT id, user_id, nama, kode_subkegiatan, kode_opd, file_name, file_url, file_size, content_type, tahun, created_at
                             FROM user_files WHERE kode_opd = ? AND tahun = ? ORDER BY created_at DESC LIMIT 1`, kodeOpd, tahun)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer rows.Close()

	var files []UserData
	for rows.Next() {
		var f UserData
		if err := rows.Scan(&f.ID, &f.UserID, &f.Nama, &f.KodeSubkegiatan, &f.KodeOpd,
			&f.FileName, &f.FileURL, &f.FileSize, &f.ContentType, &f.Tahun, &f.CreatedAt); err != nil {
			continue
		}
		files = append(files, f)
	}

	writeJSON(w, http.StatusOK, files)
}

// ==== helpers ====

func (u *Uploader) publicURL(key string) string {
	if u.public == "" {
		return ""
	}
	return u.public + "/" + strings.TrimLeft(key, "/")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func safeName(name string) string {
	base := filepath.Base(name)
	base = strings.ReplaceAll(base, " ", "_")
	return base
}

func guessContentType(header *multipart.FileHeader, r io.Reader) string {
	switch strings.ToLower(filepath.Ext(header.Filename)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain; charset=utf-8"
	case ".json":
		return "application/json"
	}
	buf := make([]byte, 512)
	n, _ := io.ReadFull(r, buf)
	return http.DetectContentType(buf[:n])
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func mustGetenv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("ENV %s harus di-set", k)
	}
	return v
}
func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
