package dto

// ─── SMTP Config ────────────────────────────────────────────────

type SMTPConfigReq struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"fromEmail"`
	Encryption string `json:"encryption"`
	Enabled    bool   `json:"enabled"`
}

type SMTPConfigResp struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"fromEmail"`
	Encryption string `json:"encryption"`
	Enabled    bool   `json:"enabled"`
	UpdatedAt  int64  `json:"updatedAt"`
}

// ─── SMTP Test ──────────────────────────────────────────────────

type SMTPTestReq struct {
	Email string `json:"email" binding:"required"`
}

// ─── Notify Rules ───────────────────────────────────────────────

type NotifyRulesReq struct {
	DeviceOnline  bool `json:"deviceOnline"`
	DeviceOffline bool `json:"deviceOffline"`
	RemoteAccess  bool `json:"remoteAccess"`
}

type NotifyRulesResp struct {
	DeviceOnline  bool  `json:"deviceOnline"`
	DeviceOffline bool  `json:"deviceOffline"`
	RemoteAccess  bool  `json:"remoteAccess"`
	UpdatedAt     int64 `json:"updatedAt"`
}

// ─── Recipients ─────────────────────────────────────────────────

type AddRecipientReq struct {
	Email string `json:"email" binding:"required"`
}

type RecipientResp struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	CreatedAt int64  `json:"createdAt"`
}

type ListRecipientsResp struct {
	Items []RecipientResp `json:"items"`
}
