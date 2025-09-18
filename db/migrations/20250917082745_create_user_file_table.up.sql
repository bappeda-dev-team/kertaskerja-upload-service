CREATE TABLE IF NOT EXISTS user_files (
       id BIGINT AUTO_INCREMENT PRIMARY KEY,
       user_id VARCHAR(150) NOT NULL,
       nama VARCHAR(255),
       kode_subkegiatan VARCHAR(255),
       kode_opd VARCHAR(255),
       file_name VARCHAR(255) NOT NULL,
       file_url TEXT NOT NULL,
       file_size BIGINT NOT NULL,
       bucket VARCHAR(255) NOT NULL,
       content_type VARCHAR(255) NOT NULL,
       tahun INT,
       kode_opd VARCHAR(255) NOT NULL,
       kode_subkegiatan VARCHAR(255) NOT NULL,
       kode_pemda VARCHAR(255),
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
