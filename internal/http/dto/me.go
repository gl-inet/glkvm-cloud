package dto

type MeUser struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	DisplayName  string `json:"displayName"`
	Role         string `json:"role"`
	AuthProvider string `json:"authProvider"`
}

type MeResp struct {
	User        MeUser   `json:"user"`
	Permissions []string `json:"permissions"`
}
