package dto

type LoginReq struct {
    Username   string `json:"username"`
    Password   string `json:"password"`
    AuthMethod string `json:"authMethod,omitempty"`
}

type LoginResp struct {
    Token string `json:"token"`
}

type LogoutResp struct{}
