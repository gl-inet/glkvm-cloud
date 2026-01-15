package dto

type User struct {
	ID          int64  `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Role        string `json:"role"`
	Status      string `json:"status"`
}

type ListUsersResp struct {
	Items []User `json:"items"`
}

type CreateUserReq struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
	Role        string `json:"role"`   // admin/user (default: user)
	Status      string `json:"status"` // active/disabled (default: active)
}

type CreateUserResp struct {
	ID int64 `json:"id"`
}

type UpdateUserReq struct {
	Email       *string `json:"email"`
	DisplayName *string `json:"displayName"`
	Password    *string `json:"password"`
	Role        *string `json:"role"`
	Status      *string `json:"status"`
}

type DeleteUserResp struct{}
