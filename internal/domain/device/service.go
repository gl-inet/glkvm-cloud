package device

import (
    "context"
    "rttys/internal/domain/identity"

    "rttys/internal/domain/group"
)

type Service struct {
    repo  Repository
    grepo group.Repository
}

func NewService(repo Repository, grepo group.Repository) *Service {
    return &Service{repo: repo, grepo: grepo}
}

// ListVisible implements the scope rule:
// - admin: all devices, including ungrouped
// - user : only devices whose device_group_id is linked via (user -> user_groups -> device_groups)
func (s *Service) ListVisible(ctx context.Context, role identity.Role, userID int64) ([]Device, error) {
    if role == identity.RoleAdmin {
        return s.repo.ListAll(ctx)
    }

    dgIDs, err := s.grepo.ListDeviceGroupIDsByUser(ctx, userID)
    if err != nil || len(dgIDs) == 0 {
        return []Device{}, nil
    }
    return s.repo.ListByDeviceGroupIDs(ctx, dgIDs)
}
