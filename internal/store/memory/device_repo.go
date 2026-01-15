package memory

import (
	"context"
	"sync"

	"rttys/internal/domain/device"
)

type DeviceRepo struct {
	mu    sync.RWMutex
	items []device.Device
}

func NewDeviceRepo(seed []device.Device) *DeviceRepo {
	r := &DeviceRepo{items: append([]device.Device(nil), seed...)}
	return r
}

func (r *DeviceRepo) ListAll(ctx context.Context) ([]device.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := append([]device.Device(nil), r.items...)
	return out, nil
}

func (r *DeviceRepo) ListByDeviceGroupIDs(ctx context.Context, groupIDs []int64) ([]device.Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	allowed := map[int64]struct{}{}
	for _, id := range groupIDs {
		allowed[id] = struct{}{}
	}

	out := make([]device.Device, 0, len(r.items))
	for _, d := range r.items {
		if d.DeviceGroupID == nil {
			// ungrouped is not visible to normal users by design
			continue
		}
		if _, ok := allowed[*d.DeviceGroupID]; ok {
			out = append(out, d)
		}
	}
	return out, nil
}
