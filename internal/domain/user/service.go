package user

import (
    "context"
    "errors"
    "rttys/internal/domain/identity"

    "rttys/internal/pkg/password"
)

var (
    ErrUserNotFound = errors.New("user not found")
    ErrUserDisabled = errors.New("user disabled")
    ErrBadPassword  = errors.New("bad password")
)

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) Authenticate(ctx context.Context, email, pw string) (*User, error) {
    u, err := s.repo.FindByUsername(ctx, email)
    if err != nil || u == nil {
        return nil, ErrUserNotFound
    }
    if u.Status == StatusDisabled {
        return nil, ErrUserDisabled
    }
    if !password.VerifyDemoSHA256(pw, u.PasswordHash) {
        return nil, ErrBadPassword
    }
    return u, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
    u, err := s.repo.FindByID(ctx, id)
    if err != nil || u == nil {
        return nil, ErrUserNotFound
    }
    if u.Status == StatusDisabled {
        return nil, ErrUserDisabled
    }
    return u, nil
}

func (s *Service) List(ctx context.Context) ([]User, error) {
    return s.repo.List(ctx)
}

// CreateUser creates a user; passwordPlain will be hashed.
func (s *Service) CreateUser(ctx context.Context, email, displayName, passwordPlain, role, status string) (int64, error) {
    u := &User{
        Email:        email,
        DisplayName:  displayName,
        PasswordHash: password.HashDemoSHA256(passwordPlain),
        Role:         identity.RoleFromString(role),
        Status:       Status(status),
    }
    return s.repo.Create(ctx, u)
}

// UpdateUser updates fields; if passwordPlain is empty, keep existing.
func (s *Service) UpdateUser(ctx context.Context, id int64, email, displayName, passwordPlain, role, status *string) error {
    exist, err := s.repo.FindByID(ctx, id)
    if err != nil || exist == nil {
        return ErrUserNotFound
    }

    if email != nil && *email != "" {
        exist.Email = *email
    }
    if displayName != nil {
        exist.DisplayName = *displayName
    }
    if role != nil && *role != "" {
        exist.Role = identity.RoleFromString(*role)
    }
    if status != nil && *status != "" {
        exist.Status = Status(*status)
    }
    if passwordPlain != nil && *passwordPlain != "" {
        exist.PasswordHash = password.HashDemoSHA256(*passwordPlain)
    }
    return s.repo.Update(ctx, exist)
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
    return s.repo.Delete(ctx, id)
}
