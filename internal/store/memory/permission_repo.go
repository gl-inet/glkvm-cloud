package memory

import (
    "context"
    "rttys/internal/domain/identity"

    "rttys/internal/domain/permission"
)

type PermissionRepo struct {
    roleToKeys map[identity.Role][]permission.Key
}

func NewPermissionRepo() *PermissionRepo {
    // Role defaults are aligned with the design docs:
    // - admin: all enabled permissions
    // - user : read (and basic auth/me)
    return &PermissionRepo{
        roleToKeys: map[identity.Role][]permission.Key{
            identity.RoleAdmin: {
                permission.MeRead, permission.AuthWrite,
                permission.DeviceRead, permission.DeviceWrite,
                permission.DeviceGroupRead, permission.DeviceGroupWrite,
                permission.UserGroupRead, permission.UserGroupWrite,
                permission.UserRead, permission.UserWrite,
                permission.RelationWrite,
                permission.DeviceLogRead,
            },
            identity.RoleUser: {
                permission.MeRead, permission.AuthWrite,
                permission.DeviceRead,
                permission.DeviceGroupRead, permission.UserGroupRead,
            },
        },
    }
}

func (r *PermissionRepo) ListKeysByRole(ctx context.Context, role identity.Role) ([]permission.Key, error) {
    keys := r.roleToKeys[role]
    out := make([]permission.Key, 0, len(keys))
    out = append(out, keys...)
    return out, nil
}
