package sqlite

import (
    "sync"

    "gorm.io/gorm"

    "rttys/internal/domain/devicelog"
    "rttys/internal/domain/notification"
    "rttys/internal/domain/user"
)

type Container struct {
    Gorm            *gorm.DB
    DeviceMeta      *DeviceMetaRepo
    DeviceLogSvc    *devicelog.Service
    UserSvc         *user.Service
    NotificationSvc *notification.Service
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

// TryContainer returns the global container if it has been initialized,
// or nil otherwise. Use this from code paths (e.g. device runtime) that
// may execute before InitAppContainer has finished — calling MustContainer
// there would panic on early connections.
func TryContainer() *Container {
    mu.RLock()
    defer mu.RUnlock()
    return gContainer
}
