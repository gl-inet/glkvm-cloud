PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  username      TEXT    NOT NULL UNIQUE, 
  email         TEXT,                    
  description   TEXT    NOT NULL DEFAULT '',
  password_hash TEXT    NOT NULL,
  role          TEXT    NOT NULL CHECK (role IN ('admin','user')),
  status        TEXT    NOT NULL DEFAULT 'active' CHECK (status IN ('active','disabled')),
  is_system     INTEGER NOT NULL DEFAULT 0,
  created_at    INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at    INTEGER NOT NULL DEFAULT (unixepoch())
);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

CREATE TABLE IF NOT EXISTS user_groups (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  name        TEXT    NOT NULL UNIQUE,
  description TEXT    NOT NULL DEFAULT '',
  created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at  INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS user_group_members (
  user_id     INTEGER NOT NULL,
  group_id    INTEGER NOT NULL,
  created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
  PRIMARY KEY (user_id, group_id),
  FOREIGN KEY (user_id)  REFERENCES users(id)       ON DELETE CASCADE,
  FOREIGN KEY (group_id) REFERENCES user_groups(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ugm_group_id ON user_group_members(group_id);

CREATE TABLE IF NOT EXISTS device_groups (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  name        TEXT    NOT NULL UNIQUE,
  description TEXT    NOT NULL DEFAULT '',
  created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at  INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE IF NOT EXISTS user_group_device_group_links (
  user_group_id   INTEGER NOT NULL,
  device_group_id INTEGER NOT NULL,
  created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
  PRIMARY KEY (user_group_id, device_group_id),
  FOREIGN KEY (user_group_id)   REFERENCES user_groups(id)   ON DELETE CASCADE,
  FOREIGN KEY (device_group_id) REFERENCES device_groups(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_ug_dg_device_group_id
  ON user_group_device_group_links(device_group_id);

CREATE TABLE IF NOT EXISTS devices (
  id              INTEGER PRIMARY KEY AUTOINCREMENT,
  ddns            TEXT    NOT NULL UNIQUE,
  mac             TEXT    NOT NULL UNIQUE,
  name            TEXT    NOT NULL DEFAULT '',
  description     TEXT    NOT NULL DEFAULT '',
  ip              TEXT    NOT NULL DEFAULT '',
  client          TEXT    NOT NULL DEFAULT '',
  device_group_id INTEGER NULL,
  status          TEXT    NOT NULL DEFAULT 'online' CHECK (status IN ('online','offline','disabled')),
  last_seen_at    INTEGER NULL,
  created_at      INTEGER NOT NULL DEFAULT (unixepoch()),
  updated_at      INTEGER NOT NULL DEFAULT (unixepoch()),
  FOREIGN KEY (device_group_id) REFERENCES device_groups(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_devices_group_id ON devices(device_group_id);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen_at);

CREATE TRIGGER IF NOT EXISTS trg_users_updated_at
AFTER UPDATE ON users
FOR EACH ROW
BEGIN
  UPDATE users SET updated_at = unixepoch() WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS trg_user_groups_updated_at
AFTER UPDATE ON user_groups
FOR EACH ROW
BEGIN
  UPDATE user_groups SET updated_at = unixepoch() WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS trg_device_groups_updated_at
AFTER UPDATE ON device_groups
FOR EACH ROW
BEGIN
  UPDATE device_groups SET updated_at = unixepoch() WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS trg_devices_updated_at
AFTER UPDATE ON devices
FOR EACH ROW
BEGIN
  UPDATE devices SET updated_at = unixepoch() WHERE id = OLD.id;
END;

-- Devices allowed on the mTLS device port. A row is written when a device
-- claims a registration code, keyed by the MAC in its factory certificate; it
-- pins the device ID so another certificate cannot take it over.
CREATE TABLE IF NOT EXISTS device_whitelist (
  mac        TEXT    PRIMARY KEY,
  devid      TEXT    NOT NULL UNIQUE,
  code_id    INTEGER NULL,
  created_at INTEGER NOT NULL
);

-- One-time registration codes handed out inside access codes. Only a hash of
-- each code is stored.
CREATE TABLE IF NOT EXISTS registration_codes (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  code_hash  TEXT    NOT NULL UNIQUE,
  created_by INTEGER NULL,
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL,
  used_at    INTEGER NULL,
  used_mac   TEXT    NULL,
  revoked_at INTEGER NULL
);
CREATE INDEX IF NOT EXISTS idx_registration_codes_created_at ON registration_codes(created_at);
