package model

// DeviceMeta represents device metadata stored in the devices table.
type DeviceMeta struct {
    DeviceID    string `gorm:"column:ddns"`        // DeviceID maps to devices.ddns.
    Mac         string `gorm:"column:mac"`         // Mac is the unique and immutable MAC address of the device.
    IP          string `gorm:"column:ip"`          // IP is the current IP address of the device.
    Description string `gorm:"column:description"` // Description is a human-readable description of the device.
}

// TableName sets the name of the table in the database that this struct binds to.
func (DeviceMeta) TableName() string {
    return "devices"
}
