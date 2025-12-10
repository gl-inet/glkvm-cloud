package db

// DeviceMeta represents a record in the gl_device table.
type DeviceMeta struct {
    DeviceID    string `gorm:"primaryKey;column:device_id"` // DeviceID is the globally unique and immutable ID of the device.
    Mac         string `gorm:"uniqueIndex;column:mac"`      // Mac is the unique and immutable MAC address of the device.
    IP          string `gorm:"column:ip"`                   // IP is the current IP address of the device.
    Description string `gorm:"column:description"`          // Description is a human-readable description of the device.
    CreateTime  int64  `gorm:"column:create_time"`          // CreateTime is the creation timestamp (Unix time).
    UpdateTime  int64  `gorm:"column:update_time"`          // UpdateTime is the last update timestamp (Unix time).
}

// TableName sets the name of the table in the database that this struct binds to.
func (DeviceMeta) TableName() string {
    return "devices"
}
