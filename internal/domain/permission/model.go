package permission

import "rttys/internal/domain/identity"

type Key string

const (
    MeRead    Key = "me.read"
    AuthWrite Key = "auth.write"

    DeviceRead  Key = "device.read"
    DeviceWrite Key = "device.write"

    DeviceGroupRead  Key = "device_group.read"
    DeviceGroupWrite Key = "device_group.write"

    UserGroupRead  Key = "user_group.read"
    UserGroupWrite Key = "user_group.write"

    UserRead  Key = "user.read"
    UserWrite Key = "user.write"

    RelationWrite Key = "relation.write"

    DeviceLogRead Key = "device_log.read"

    NotificationRead  Key = "notification.read"
    NotificationWrite Key = "notification.write"
)

func DefaultKeysForRole(role identity.Role) []Key {
    switch role {
    case identity.RoleAdmin:
        return []Key{ /* ... */ }
    default:
        return []Key{ /* ... */ }
    }
}
