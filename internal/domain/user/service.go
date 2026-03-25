package user

import (
    "context"
    "errors"
    "strconv"

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

func (s *Service) GetSystemAdmin(ctx context.Context) (*User, error) {
	return s.repo.FindSystemAdmin(ctx)
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

// FindOrCreateExternalUser looks up a user by (provider, externalSub).
// If found, it updates email/description and returns the user.
// If not found, it creates a new user with the given role and status=active.
//
// role is determined by the caller based on admin-group/admin-users membership
// and is only applied at user creation time. Existing users keep their current role.
func (s *Service) FindOrCreateExternalUser(ctx context.Context, provider, externalSub, preferredUsername, email, displayName string, role identity.Role) (*User, error) {
    u, err := s.repo.FindByExternalID(ctx, provider, externalSub)
    if err != nil {
        return nil, err
    }
    if u != nil {
        // Update email and display name on each login (IdP may change them).
        changed := false
        if email != "" && u.Email != email {
            u.Email = email
            changed = true
        }
        if displayName != "" && u.Description != displayName {
            u.Description = displayName
            changed = true
        }
        if changed {
            _ = s.repo.Update(ctx, u)
        }
        return u, nil
    }

    // --- Create new user ---
    username := s.pickUniqueUsername(ctx, preferredUsername, email, provider)

    newUser := &User{
        Username:     username,
        Email:        email,
        Description:  displayName,
        PasswordHash: "", // external users never authenticate via password
        Role:         role,
        Status:       StatusActive,
        AuthProvider: provider,
        ExternalSub:  externalSub,
    }
    id, err := s.repo.Create(ctx, newUser)
    if err != nil {
        return nil, err
    }
    newUser.ID = id
    return newUser, nil
}

// pickUniqueUsername tries candidate usernames until one doesn't conflict.
func (s *Service) pickUniqueUsername(ctx context.Context, preferredUsername, email, provider string) string {
    candidates := make([]string, 0, 4)
    if preferredUsername != "" {
        candidates = append(candidates, preferredUsername)
    }
    if email != "" && email != preferredUsername {
        candidates = append(candidates, email)
    }
    // Fallback with provider suffix
    if preferredUsername != "" {
        candidates = append(candidates, preferredUsername+"_"+provider)
    }
    if email != "" {
        candidates = append(candidates, email+"_"+provider)
    }
    // Last resort
    if len(candidates) == 0 {
        candidates = append(candidates, provider+"_user")
    }

    for _, c := range candidates {
        existing, _ := s.repo.FindByUsername(ctx, c)
        if existing == nil {
            return c
        }
    }
    // All candidates taken — append a numeric suffix
    base := candidates[0] + "_" + provider
    for i := 2; ; i++ {
        name := base + "_" + strconv.Itoa(i)
        existing, _ := s.repo.FindByUsername(ctx, name)
        if existing == nil {
            return name
        }
    }
}
