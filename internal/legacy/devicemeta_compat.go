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

func UpdateDeviceClient(deviceID, client string) error {
    repo := sqlite.MustContainer().DeviceMeta
    return repo.UpdateClient(context.Background(), deviceID, client)
}

func GetDeviceMetaByDeviceID(deviceID string) (*model.DeviceMeta, error) {
    repo := sqlite.MustContainer().DeviceMeta
    return repo.GetByDeviceID(context.Background(), deviceID)
}

func UpdateDeviceDescriptionIfEmpty(deviceID, description string) error {
    repo := sqlite.MustContainer().DeviceMeta
    return repo.UpdateDescriptionIfEmpty(context.Background(), deviceID, description)
}

func DeleteDeviceMetaByDeviceID(deviceID string) error {
    repo := sqlite.MustContainer().DeviceMeta
    return repo.DeleteByDeviceID(context.Background(), deviceID)
}

func MarkDeviceOffline(deviceID string) error {
    repo := sqlite.MustContainer().DeviceMeta
    return repo.MarkOffline(context.Background(), deviceID)
}
