package dto

type SetUserGroupsReq struct {
	GroupIDs []int64 `json:"groupIds"`
}
type SetUserGroupsResp struct {
	UserID   int64   `json:"userId"`
	GroupIDs []int64 `json:"groupIds"`
}

type SetUserGroupDeviceGroupsReq struct {
	DeviceGroupIDs []int64 `json:"deviceGroupIds"`
}
type SetUserGroupDeviceGroupsResp struct {
	UserGroupID    int64   `json:"userGroupId"`
	DeviceGroupIDs []int64 `json:"deviceGroupIds"`
}

type SetDeviceGroupDevicesReq struct {
	DeviceIDs []int64 `json:"deviceIds"`
}
type SetDeviceGroupDevicesResp struct {
	DeviceGroupID int64   `json:"deviceGroupId"`
	DeviceIDs     []int64 `json:"deviceIds"`
}
