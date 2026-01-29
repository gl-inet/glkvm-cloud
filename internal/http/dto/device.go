package dto

type Device struct {
	ID              int64  `json:"id"`
	Ddns            string `json:"ddns"`
	Status          string `json:"status"`
	ConnectedTime   int64  `json:"connectedTime"`
	UpTime          int64  `json:"upTime"`
	IP              string `json:"ip"`
	Mac             string `json:"mac"`
	Description     string `json:"description"`
	DeviceGroupID   *int64 `json:"deviceGroupId"`
	DeviceGroupName string `json:"deviceGroupName"`
}

type ListDevicesResp struct {
	Items    []Device `json:"items"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
	Total    int      `json:"total"`
}

type MoveDevicesToGroupReq struct {
	DeviceIDs []int64 `json:"deviceIds"`
	GroupID   int64   `json:"groupId"`
}

type MoveDevicesToGroupResp struct{}
