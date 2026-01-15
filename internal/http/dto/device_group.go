package dto

type DeviceGroup struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ListDeviceGroupsResp struct {
	Items []DeviceGroup `json:"items"`
}

type CreateDeviceGroupReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateDeviceGroupResp struct {
	ID int64 `json:"id"`
}

type UpdateDeviceGroupReq struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DeleteDeviceGroupResp struct{}
