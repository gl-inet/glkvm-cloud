package dto

type UserGroupRef struct {
	UserGroupID   int64  `json:"userGroupId"`
	UserGroupName string `json:"userGroupName"`
}

type User struct {
	ID            int64          `json:"id"`
	Username      string         `json:"username"`
	Description   string         `json:"description"`
	Role          string         `json:"role"`
	IsSystem      bool           `json:"isSystem"`
	UserGroupList []UserGroupRef `json:"userGroupList"`
}

type ListUsersResp struct {
	Items []User `json:"items"`
}

type CreateUserReq struct {
	Role         string  `json:"role"`
	Username     string  `json:"username"`
	Description  string  `json:"description"`
	Password     string  `json:"password"`
	Repassword   string  `json:"repassword"`
	UserGroupIDs []int64 `json:"userGroupIds"`
}

type CreateUserResp struct{}

type UpdateUserReq struct {
	Role         *string  `json:"role"`
	Username     *string  `json:"username"`
	Description  *string  `json:"description"`
	Password     *string  `json:"password"`
	Repassword   *string  `json:"repassword"`
	UserGroupIDs *[]int64 `json:"userGroupIds"`
}

type DeleteUserResp struct{}
