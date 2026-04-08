package notification

// SMTPConfig holds mail server settings. Only one row exists (singleton).
type SMTPConfig struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"fromEmail"`
	Encryption string `json:"encryption"` // "none", "tls", "starttls"
	Enabled    bool   `json:"enabled"`
	UpdatedAt  int64  `json:"updatedAt"`
}

// NotifyRules controls which event categories trigger email notifications.
type NotifyRules struct {
	DeviceOnline  bool  `json:"deviceOnline"`
	DeviceOffline bool  `json:"deviceOffline"`
	RemoteAccess  bool  `json:"remoteAccess"` // SSH + Web + Control
	UpdatedAt     int64 `json:"updatedAt"`
}

// Recipient is a notification email address.
type Recipient struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	CreatedAt int64  `json:"createdAt"`
}
