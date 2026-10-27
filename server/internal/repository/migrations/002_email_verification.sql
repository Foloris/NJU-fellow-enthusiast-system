CREATE TABLE IF NOT EXISTS user_emails (
 user_id BIGINT PRIMARY KEY,
 email VARCHAR(254) CHARACTER SET ascii COLLATE ascii_general_ci NOT NULL UNIQUE,
 verified_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS email_verifications (
 email VARCHAR(254) CHARACTER SET ascii COLLATE ascii_general_ci PRIMARY KEY,
 code_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 nonce CHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
 state ENUM('sending','ready','consumed') NOT NULL DEFAULT 'sending',
 expires_at DATETIME(3) NOT NULL,
 next_send_at DATETIME(3) NOT NULL,
 attempts TINYINT UNSIGNED NOT NULL DEFAULT 0,
 INDEX idx_verification_expiry(expires_at)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS email_verification_send_limits (
 source_hash CHAR(64) CHARACTER SET ascii COLLATE ascii_bin PRIMARY KEY,
 window_start DATETIME(3) NOT NULL,
 sent_count INT UNSIGNED NOT NULL DEFAULT 0
) ENGINE=InnoDB;
