package device

import "context"

type Repository interface {
	ListAll(ctx context.Context) ([]Device, error)
	ListByDeviceGroupIDs(ctx context.Context, groupIDs []int64) ([]Device, error)
}
