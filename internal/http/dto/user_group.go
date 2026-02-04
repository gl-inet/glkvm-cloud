package dto

type UserGroupDeviceGroupRef struct {
	DeviceGroupID   int64  `json:"deviceGroupId"`
	DeviceGroupName string `json:"deviceGroupName"`
}

type UserGroup struct {
	ID              int64                   `json:"id"`
	UserGroup       string                  `json:"userGroup"`
	Description     string                  `json:"description"`
	UserCount       int64                   `json:"userCount"`
	DeviceGroupList []UserGroupDeviceGroupRef `json:"deviceGroupList"`
}

type ListUserGroupsResp struct {
	Items []UserGroup `json:"items"`
}

type UserGroupOption struct {
	UserGroupID int64  `json:"userGroupId"`
	Name        string `json:"name"`
}

type ListUserGroupOptionsResp struct {
	Items []UserGroupOption `json:"items"`
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
