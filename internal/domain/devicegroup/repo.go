package devicegroup

import "context"

type Repository interface {
	ListDeviceGroupsVisibleToUser(ctx context.Context, userID int64, isAdmin bool) ([]DeviceGroup, error)
}
