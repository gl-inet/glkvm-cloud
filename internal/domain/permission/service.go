package permission

import (
    "context"
    "rttys/internal/domain/identity"
)

type Repository interface {
    ListKeysByRole(ctx context.Context, role identity.Role) ([]Key, error)
}

type Service struct {
    repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) ListByRole(ctx context.Context, role identity.Role) ([]Key, error) {
    return s.repo.ListKeysByRole(ctx, role)
}
