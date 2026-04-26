package db

// Per-dialect migration schemas for WhatsGo tables.

func sqliteSchema() string {
	return `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    role TEXT DEFAULT 'user' CHECK(role IN ('user', 'admin')),
    active INTEGER DEFAULT 1,
    rate_limit INTEGER DEFAULT 100,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    phone_number TEXT,
    jid TEXT,
    push_name TEXT,
    status TEXT DEFAULT 'disconnected' CHECK(status IN ('connected', 'disconnected', 'connecting')),
    webhook_url TEXT,
    webhook_events TEXT DEFAULT 'message_received,message_sent,message_delivered,message_read,device_connected,device_disconnected',
    webhook_secret TEXT,
    store_path TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_devices_user ON devices(user_id);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    direction TEXT NOT NULL CHECK(direction IN ('inbound', 'outbound')),
    recipient TEXT,
    message_type TEXT DEFAULT 'text' CHECK(message_type IN ('text', 'image', 'video', 'audio', 'document', 'template')),
    content TEXT,
    media_url TEXT,
    media_mime_type TEXT,
    media_sha256 TEXT,
    status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'sent', 'delivered', 'read', 'failed')),
    whatsapp_message_id TEXT,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_messages_device ON messages(device_id);
CREATE INDEX IF NOT EXISTS idx_messages_user ON messages(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_direction ON messages(direction);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at);

CREATE TABLE IF NOT EXISTS api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    last_used_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    revoked_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_device ON api_keys(device_id);

CREATE TABLE IF NOT EXISTS webhook_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload TEXT NOT NULL,
    url TEXT NOT NULL,
    status_code INTEGER,
    response_body TEXT,
    response_time_ms INTEGER,
    success INTEGER NOT NULL,
    error TEXT,
    attempt_number INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_webhook_logs_device ON webhook_logs(device_id);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_created ON webhook_logs(created_at);

CREATE TABLE IF NOT EXISTS rate_limits (
    user_id INTEGER PRIMARY KEY,
    requests INTEGER DEFAULT 0,
    window_start DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
}

func postgresSchema() string {
	return `
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    role TEXT DEFAULT 'user' CHECK(role IN ('user', 'admin')),
    active BOOLEAN DEFAULT TRUE,
    rate_limit INTEGER DEFAULT 100,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS devices (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    phone_number TEXT,
    jid TEXT,
    push_name TEXT,
    status TEXT DEFAULT 'disconnected' CHECK(status IN ('connected', 'disconnected', 'connecting')),
    webhook_url TEXT,
    webhook_events TEXT DEFAULT 'message_received,message_sent,message_delivered,message_read,device_connected,device_disconnected',
    webhook_secret TEXT,
    store_path TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_devices_user ON devices(user_id);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);

CREATE TABLE IF NOT EXISTS messages (
    id SERIAL PRIMARY KEY,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    direction TEXT NOT NULL CHECK(direction IN ('inbound', 'outbound')),
    recipient TEXT,
    message_type TEXT DEFAULT 'text' CHECK(message_type IN ('text', 'image', 'video', 'audio', 'document', 'template')),
    content TEXT,
    media_url TEXT,
    media_mime_type TEXT,
    media_sha256 TEXT,
    status TEXT DEFAULT 'pending' CHECK(status IN ('pending', 'sent', 'delivered', 'read', 'failed')),
    whatsapp_message_id TEXT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_device ON messages(device_id);
CREATE INDEX IF NOT EXISTS idx_messages_user ON messages(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_direction ON messages(direction);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at);

CREATE TABLE IF NOT EXISTS api_keys (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    key_hash TEXT NOT NULL,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    revoked_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_device ON api_keys(device_id);

CREATE TABLE IF NOT EXISTS webhook_logs (
    id SERIAL PRIMARY KEY,
    device_id INTEGER NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    payload TEXT NOT NULL,
    url TEXT NOT NULL,
    status_code INTEGER,
    response_body TEXT,
    response_time_ms INTEGER,
    success BOOLEAN NOT NULL,
    error TEXT,
    attempt_number INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_webhook_logs_device ON webhook_logs(device_id);
CREATE INDEX IF NOT EXISTS idx_webhook_logs_created ON webhook_logs(created_at);

CREATE TABLE IF NOT EXISTS rate_limits (
    user_id INTEGER PRIMARY KEY,
    requests INTEGER DEFAULT 0,
    window_start TIMESTAMP DEFAULT NOW()
);
`
}

func mysqlSchema() string {
	return `
CREATE TABLE IF NOT EXISTS users (
    id INT PRIMARY KEY AUTO_INCREMENT,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(10) DEFAULT 'user' CHECK(role IN ('user', 'admin')),
    active TINYINT(1) DEFAULT 1,
    rate_limit INT DEFAULT 100,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS devices (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    phone_number VARCHAR(50),
    jid VARCHAR(255),
    push_name VARCHAR(255),
    status VARCHAR(20) DEFAULT 'disconnected' CHECK(status IN ('connected', 'disconnected', 'connecting')),
    webhook_url TEXT,
    webhook_events TEXT,
    webhook_secret VARCHAR(255),
    store_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_devices_user (user_id),
    INDEX idx_devices_status (status)
);

CREATE TABLE IF NOT EXISTS messages (
    id INT PRIMARY KEY AUTO_INCREMENT,
    device_id INT NOT NULL,
    user_id INT NOT NULL,
    direction VARCHAR(10) NOT NULL CHECK(direction IN ('inbound', 'outbound')),
    recipient VARCHAR(50),
    message_type VARCHAR(20) DEFAULT 'text' CHECK(message_type IN ('text', 'image', 'video', 'audio', 'document', 'template')),
    content TEXT,
    media_url TEXT,
    media_mime_type VARCHAR(100),
    media_sha256 VARCHAR(64),
    status VARCHAR(20) DEFAULT 'pending' CHECK(status IN ('pending', 'sent', 'delivered', 'read', 'failed')),
    whatsapp_message_id VARCHAR(255),
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_messages_device (device_id),
    INDEX idx_messages_user (user_id),
    INDEX idx_messages_direction (direction),
    INDEX idx_messages_status (status),
    INDEX idx_messages_created (created_at)
);

CREATE TABLE IF NOT EXISTS api_keys (
    id INT PRIMARY KEY AUTO_INCREMENT,
    user_id INT NOT NULL,
    device_id INT,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(20) NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    last_used_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    INDEX idx_api_keys_user (user_id),
    INDEX idx_api_keys_device (device_id)
);

CREATE TABLE IF NOT EXISTS webhook_logs (
    id INT PRIMARY KEY AUTO_INCREMENT,
    device_id INT NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    payload LONGTEXT NOT NULL,
    url TEXT NOT NULL,
    status_code INT,
    response_body TEXT,
    response_time_ms INT,
    success TINYINT(1) NOT NULL,
    error TEXT,
    attempt_number INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE,
    INDEX idx_webhook_logs_device (device_id),
    INDEX idx_webhook_logs_created (created_at)
);

CREATE TABLE IF NOT EXISTS rate_limits (
    user_id INT PRIMARY KEY,
    requests INT DEFAULT 0,
    window_start TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`
}
