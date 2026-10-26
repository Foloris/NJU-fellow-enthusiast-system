CREATE TABLE IF NOT EXISTS users (
 id BIGINT PRIMARY KEY AUTO_INCREMENT,
 username VARCHAR(24) COLLATE utf8mb4_0900_as_ci NOT NULL UNIQUE,
 password_hash VARCHAR(255) NOT NULL,
 nickname VARCHAR(30) NOT NULL,
 avatar VARCHAR(500) NULL,
 bio VARCHAR(200) NOT NULL DEFAULT '',
 role ENUM('user','admin') NOT NULL DEFAULT 'user',
 status ENUM('active','banned') NOT NULL DEFAULT 'active',
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS exchange_posts (
 id BIGINT PRIMARY KEY AUTO_INCREMENT,
 author_id BIGINT NOT NULL,
 kind ENUM('share','news') NOT NULL,
 title VARCHAR(100) NOT NULL,
 content TEXT NOT NULL,
 image_url VARCHAR(500) NULL,
 status ENUM('published','removed') NOT NULL DEFAULT 'published',
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 INDEX idx_exchange_status_created(status, created_at DESC),
 FOREIGN KEY(author_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS mate_posts (
 id BIGINT PRIMARY KEY AUTO_INCREMENT,
 author_id BIGINT NOT NULL,
 title VARCHAR(100) NOT NULL,
 tags JSON NOT NULL,
 details TEXT NOT NULL,
 image_url VARCHAR(500) NULL,
 location VARCHAR(100) NULL,
 contact VARCHAR(100) NOT NULL,
 duration VARCHAR(100) NOT NULL,
 notes TEXT NULL,
 status ENUM('published','removed') NOT NULL DEFAULT 'published',
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 INDEX idx_mate_status_created(status, created_at DESC),
 FOREIGN KEY(author_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS comments (
 id BIGINT PRIMARY KEY AUTO_INCREMENT,
 post_id BIGINT NOT NULL,
 user_id BIGINT NOT NULL,
 content TEXT NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 FOREIGN KEY(post_id) REFERENCES exchange_posts(id) ON DELETE CASCADE,
 FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS likes (
 post_id BIGINT NOT NULL,
 user_id BIGINT NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 PRIMARY KEY(post_id,user_id),
 FOREIGN KEY(post_id) REFERENCES exchange_posts(id) ON DELETE CASCADE,
 FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS favorites (
 post_id BIGINT NOT NULL,
 user_id BIGINT NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 PRIMARY KEY(post_id,user_id),
 FOREIGN KEY(post_id) REFERENCES exchange_posts(id) ON DELETE CASCADE,
 FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS applications (
 id BIGINT PRIMARY KEY AUTO_INCREMENT,
 mate_post_id BIGINT NOT NULL,
 applicant_id BIGINT NOT NULL,
 message TEXT NOT NULL,
 status ENUM('pending','accepted','rejected') NOT NULL DEFAULT 'pending',
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 responded_at DATETIME(3) NULL,
 UNIQUE KEY uq_application(mate_post_id,applicant_id),
 INDEX idx_applications_mate(mate_post_id,status),
 FOREIGN KEY(mate_post_id) REFERENCES mate_posts(id) ON DELETE CASCADE,
 FOREIGN KEY(applicant_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS conversations (
 id BIGINT PRIMARY KEY AUTO_INCREMENT,
 application_id BIGINT NOT NULL UNIQUE,
 user_a_id BIGINT NOT NULL,
 user_b_id BIGINT NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 FOREIGN KEY(application_id) REFERENCES applications(id) ON DELETE CASCADE,
 FOREIGN KEY(user_a_id) REFERENCES users(id) ON DELETE CASCADE,
 FOREIGN KEY(user_b_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS messages (
 id BIGINT PRIMARY KEY AUTO_INCREMENT,
 conversation_id BIGINT NOT NULL,
 sender_id BIGINT NOT NULL,
 content TEXT NOT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 read_at DATETIME(3) NULL,
 INDEX idx_messages_conversation(conversation_id,id DESC),
 FOREIGN KEY(conversation_id) REFERENCES conversations(id) ON DELETE CASCADE,
 FOREIGN KEY(sender_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS reports (
 id BIGINT PRIMARY KEY AUTO_INCREMENT,
 reporter_id BIGINT NOT NULL,
 target_type ENUM('post','mate','user','message') NOT NULL,
 target_id BIGINT NOT NULL,
 reason VARCHAR(100) NOT NULL,
 details TEXT NOT NULL,
 snapshot JSON NOT NULL,
 status ENUM('pending','resolved','dismissed') NOT NULL DEFAULT 'pending',
 action VARCHAR(30) NULL,
 admin_note TEXT NULL,
 resolved_by BIGINT NULL,
 created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
 resolved_at DATETIME(3) NULL,
 INDEX idx_reports_status(status,created_at DESC),
 INDEX idx_reports_duplicate(reporter_id,target_type,target_id,status),
 FOREIGN KEY(reporter_id) REFERENCES users(id) ON DELETE CASCADE,
 FOREIGN KEY(resolved_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB;
