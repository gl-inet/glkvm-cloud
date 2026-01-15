package dto

type UserGroup struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListUserGroupsResp struct {
	Items []UserGroup `json:"items"`
}

type CreateUserGroupReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateUserGroupResp struct {
	ID int64 `json:"id"`
}

type UpdateUserGroupReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DeleteUserGroupResp struct{}
