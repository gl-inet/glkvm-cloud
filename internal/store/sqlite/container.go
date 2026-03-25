package sqlite

import (
    "gorm.io/gorm"
    "sync"
)

type Container struct {
    Gorm       *gorm.DB
    DeviceMeta *DeviceMetaRepo
}

var (
    gContainer *Container
    mu         sync.RWMutex
)

func SetContainer(c *Container) {
    mu.Lock()
    defer mu.Unlock()
    gContainer = c
}

func MustContainer() *Container {
    mu.RLock()
    defer mu.RUnlock()
    if gContainer == nil {
        panic("sqlite container not initialized")
    }
    return gContainer
}
