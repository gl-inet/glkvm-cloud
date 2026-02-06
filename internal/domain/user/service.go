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

func (s *Service) Authenticate(ctx context.Context, username, pw string) (*User, error) {
    u, err := s.repo.FindByUsername(ctx, username)
    if err != nil || u == nil {
        return nil, ErrUserNotFound
    }
    if u.Status == StatusDisabled {
        return nil, ErrUserDisabled
    }
    if !password.VerifyPassword(pw, u.PasswordHash) {
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

// FindByID returns user even if disabled.
func (s *Service) FindByID(ctx context.Context, id int64) (*User, error) {
    u, err := s.repo.FindByID(ctx, id)
    if err != nil || u == nil {
        return nil, ErrUserNotFound
    }
    return u, nil
}

func (s *Service) List(ctx context.Context) ([]User, error) {
    return s.repo.List(ctx)
}

// CreateUser creates a user; passwordPlain will be hashed.
func (s *Service) CreateUser(ctx context.Context, username, description, passwordPlain, role, status string) (int64, error) {
    hash, err := password.HashPassword(passwordPlain)
    if err != nil {
        return 0, err
    }
    u := &User{
        Username:     username,
        Description:  description,
        PasswordHash: hash,
        Role:         identity.RoleFromString(role),
        Status:       Status(status),
    }
    return s.repo.Create(ctx, u)
}

// UpdateUser updates fields; if passwordPlain is empty, keep existing.
func (s *Service) UpdateUser(ctx context.Context, id int64, username, description, passwordPlain, role, status *string) error {
    exist, err := s.repo.FindByID(ctx, id)
    if err != nil || exist == nil {
        return ErrUserNotFound
    }

    if username != nil && *username != "" {
        exist.Username = *username
    }
    if description != nil {
        exist.Description = *description
    }
    if role != nil && *role != "" {
        exist.Role = identity.RoleFromString(*role)
    }
    if status != nil && *status != "" {
        exist.Status = Status(*status)
    }
    if passwordPlain != nil && *passwordPlain != "" {
        hash, err := password.HashPassword(*passwordPlain)
        if err != nil {
            return err
        }
        exist.PasswordHash = hash
    }
    return s.repo.Update(ctx, exist)
}

func (s *Service) DeleteUser(ctx context.Context, id int64) error {
    return s.repo.Delete(ctx, id)
}
