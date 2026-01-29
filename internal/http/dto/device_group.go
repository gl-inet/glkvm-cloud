package dto

type DeviceGroupUserGroupRef struct {
	UserGroupID   int64  `json:"userGroupId"`
	UserGroupName string `json:"userGroupName"`
}

type DeviceGroup struct {
	ID            int64                     `json:"id"`
	Name          string                    `json:"name"`
	DeviceCount   int64                     `json:"deviceCount"`
	Description   string                    `json:"description"`
	UserGroupList []DeviceGroupUserGroupRef `json:"userGroupList"`
}

type ListDeviceGroupsResp struct {
	Items []DeviceGroup `json:"items"`
}

type DeviceGroupOption struct {
	GroupID int64  `json:"groupId"`
	Name    string `json:"name"`
}

type ListDeviceGroupOptionsResp struct {
	Items []DeviceGroupOption `json:"items"`
}

type CreateDeviceGroupReq struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	UserGroupIDs []int64 `json:"userGroupIds"`
	DeviceIDs    []int64 `json:"deviceIds"`
}

type CreateDeviceGroupResp struct {
	ID int64 `json:"id"`
}

type UpdateDeviceGroupReq struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	UserGroupIDs []int64 `json:"userGroupIds"`
}

type DeleteDeviceGroupResp struct{}

type ModifyDeviceGroupDevicesReq struct {
	DeviceIDs []int64 `json:"deviceIds"`
}
