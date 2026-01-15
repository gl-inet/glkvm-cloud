package group

import "context"

type Repository interface {
	ListUserGroupsVisibleToUser(ctx context.Context, userID int64, isAdmin bool) ([]UserGroup, error)
	ListDeviceGroupIDsByUser(ctx context.Context, userID int64) ([]int64, error)
}
