package dto

type LoginReq struct {
    Username       string `json:"username"`
    Password       string `json:"password"`
    AuthMethod     string `json:"authMethod,omitempty"`
    TotpCode       string `json:"totpCode,omitempty"`
    RememberDevice bool   `json:"rememberDevice,omitempty"`
}

type LoginResp struct {
    Token             string `json:"token,omitempty"`
    TwoFactorRequired bool   `json:"twoFactorRequired,omitempty"`
}

type LogoutResp struct{}
