package trusteddevice

type Device struct {
	ID         int64
	UserID     int64
	Token      string
	DeviceName string
	IP         string
	CreatedAt  int64
	LastUsedAt int64
	ExpiresAt  int64
}
