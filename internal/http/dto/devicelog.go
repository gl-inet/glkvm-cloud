package dto

type DeviceEventLog struct {
	ID        int64  `json:"id"`
	DeviceMac string `json:"deviceMac"`
	EventType string `json:"eventType"`
	ActorName string `json:"actorName"`
	ClientIP  string `json:"clientIp"`
	Detail    string `json:"detail"`
	CreatedAt int64  `json:"createdAt"`
	EndedAt   int64  `json:"endedAt"`
}

type ListDeviceEventLogsResp struct {
	Items    []DeviceEventLog `json:"items"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
}
