package dto

type Device struct {
	ID            int64  `json:"id"`
	DeviceUID     string `json:"deviceUid"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	DeviceGroupID *int64 `json:"deviceGroupId"`
	Status        string `json:"status"`
	LastSeenAt    *int64 `json:"lastSeenAt"`
}

type ListDevicesResp struct {
	Items []Device `json:"items"`
}
