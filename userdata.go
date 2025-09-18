package main

import (
	"time"
)

type UserData struct {
	ID              int64     `json:"id"`
	UserID          string    `json:"user_id"`
	Nama            string    `json:"nama"`
	KodeSubkegiatan string    `json:"kode_subkegiatan"`
	KodeOpd         string    `json:"kode_opd"`
	FileName        string    `json:"file_name"`
	FileURL         string    `json:"file_url"`
	FileSize        int64     `json:"filesize"`
	ContentType     string    `json:"content_type"`
	Tahun           int       `json:"tahun"`
	CreatedAt       time.Time `json:"created_at"`
}

type UploadRequest struct {
	UserId   string `json:"user_id"`
	Tahun    int    `json:"tahun"`
	FileName string `json:"file_name"`
}
