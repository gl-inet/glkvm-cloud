package device

type Status string

const (
	StatusOnline   Status = "online"
	StatusOffline  Status = "offline"
	StatusDisabled Status = "disabled"
)

type Device struct {
    ID            int64
    Ddns          string
    Mac           string
    Name          string
    Description   string
    IP            string
    Client        string
    DeviceGroupID *int64 // nil means ungrouped (admin-only visibility)
    Status        Status
    LastSeenAt    *int64
}
