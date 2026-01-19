package legacy

import (
    "context"
    "rttys/internal/store/sqlite"
    "rttys/model"
)

func SaveOrUpdateDeviceMeta(deviceID, mac, description, ip string) error {
    repo := sqlite.MustContainer().DeviceMeta
    return repo.SaveOrUpdate(context.Background(), deviceID, mac, description, ip)
}

func GetDeviceMetaByDeviceID(deviceID string) (*model.DeviceMeta, error) {
    repo := sqlite.MustContainer().DeviceMeta
    return repo.GetByDeviceID(context.Background(), deviceID)
}

func GetAllDeviceMeta(keyword string) ([]model.DeviceMeta, error) {
    repo := sqlite.MustContainer().DeviceMeta
    return repo.List(context.Background(), keyword)
}

func DeleteDeviceMetaByDeviceID(deviceID string) error {
    repo := sqlite.MustContainer().DeviceMeta
    return repo.DeleteByDeviceID(context.Background(), deviceID)
}
