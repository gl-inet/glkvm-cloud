package dto

// ---- profile ----

type PersonalProfileResp struct {
	ID               int64  `json:"id"`
	Username         string `json:"username"`
	DisplayName      string `json:"displayName"`
	Email            string `json:"email"`
	Role             string `json:"role"`
	AuthProvider     string `json:"authProvider"`
	RegistrationTime int64  `json:"registrationTime"`
	LastLoginTime    *int64 `json:"lastLoginTime"`
	TotpEnabled      bool   `json:"totpEnabled"`
}

type UpdatePersonalProfileReq struct {
	DisplayName *string `json:"displayName"`
}

// ---- 2fa ----

type Setup2faResp struct {
	Secret     string `json:"secret"`
	OtpauthURL string `json:"otpauthUrl"`
}

type Enable2faReq struct {
	Secret string `json:"secret"`
	Code   string `json:"code"`
}

type Disable2faReq struct {
	Code string `json:"code"`
}

// ---- trusted devices ----

type TrustedDevice struct {
	ID         int64  `json:"id"`
	DeviceName string `json:"deviceName"`
	IP         string `json:"ip"`
	CreatedAt  int64  `json:"createdAt"`
	LastUsedAt int64  `json:"lastUsedAt"`
	ExpiresAt  int64  `json:"expiresAt"`
}

type ListTrustedDevicesResp struct {
	Items []TrustedDevice `json:"items"`
}
