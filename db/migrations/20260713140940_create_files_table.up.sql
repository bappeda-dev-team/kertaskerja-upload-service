CREATE TABLE IF NOT EXISTS files (
    id                   BIGINT PRIMARY KEY AUTO_INCREMENT,
    object_key           VARCHAR(500) NOT NULL UNIQUE,
    bucket               VARCHAR(100) NOT NULL,
    storage_provider     VARCHAR(20) NOT NULL,

    original_name        VARCHAR(255) NOT NULL,
    extension            VARCHAR(20),
    content_type         VARCHAR(100) NOT NULL,
    file_size            BIGINT NOT NULL,

    checksum_algorithm   VARCHAR(20),
    checksum             VARCHAR(128),

    category             VARCHAR(100),
    owner_type           VARCHAR(100),
    owner_id             VARCHAR(100),

    visibility           ENUM('private', 'public') DEFAULT 'private',

    created_by           VARCHAR(100),

    created_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at           TIMESTAMP NULL
);
