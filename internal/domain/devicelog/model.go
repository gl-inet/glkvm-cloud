package devicelog

// EventType identifies the kind of device event being logged.
type EventType string

const (
	EventDeviceOnline   EventType = "device_online"
	EventDeviceOffline  EventType = "device_offline"
	EventRemoteSSH      EventType = "remote_ssh"
	EventRemoteWeb      EventType = "remote_web"
	EventRemoteControl  EventType = "remote_control"
)

// IsSession reports whether the event represents a long-running session
// (SSH / Web / Control) for which we track both started_at and ended_at.
func (e EventType) IsSession() bool {
	return e == EventRemoteSSH || e == EventRemoteWeb || e == EventRemoteControl
}

// Log is a single device event row.
//
// For point events (online/offline) EndedAt is always 0.
// For session events (SSH/Web) CreatedAt is the session start and EndedAt
// is the session end (0 while still active).
type Log struct {
	ID          int64
	DeviceID    string
	DeviceMac   string
	EventType   EventType
	ActorUserID int64
	ActorName   string
	ClientIP    string
	Detail      string // JSON-encoded extra fields
	CreatedAt   int64
	EndedAt     int64
}

// Query holds the filter parameters for listing logs.
type Query struct {
	Mac        string      // substring match (LIKE %mac%)
	EventTypes []EventType // empty = no filter
	From       int64       // unix seconds, 0 = no lower bound
	To         int64       // unix seconds, 0 = no upper bound
	Page       int         // 1-based
	PageSize   int
}
