package sqlite

import (
    "context"
    "errors"
    "strings"

    "gorm.io/gorm"

    "rttys/internal/domain/identity"
    "rttys/internal/domain/user"
)

type UserRepo struct{ db *gorm.DB }

func NewUserRepo(db *gorm.DB) *UserRepo { return &UserRepo{db: db} }

// 映射用的行结构
type userRow struct {
    ID           int64  `gorm:"column:id"`
    Username     string `gorm:"column:username"`
    Email        string `gorm:"column:email"`
    Description  string `gorm:"column:description"`
    PasswordHash string `gorm:"column:password_hash"`
    Role         string `gorm:"column:role"`
    Status       string `gorm:"column:status"`
    IsSystem     bool   `gorm:"column:is_system"`
    AuthProvider string `gorm:"column:auth_provider"`
    ExternalSub  string `gorm:"column:external_sub"`
    LastLoginAt  *int64 `gorm:"column:last_login_at"`
    TotpSecret   string `gorm:"column:totp_secret"`
    TotpEnabled  bool   `gorm:"column:totp_enabled"`
    CreatedAt    int64  `gorm:"column:created_at"`
}

func (r userRow) toDomain() *user.User {
    return &user.User{
        ID:           r.ID,
        Username:     r.Username,
        Email:        r.Email,
        Description:  r.Description,
        PasswordHash: r.PasswordHash,
        Role:         identity.Role(r.Role),
        Status:       user.Status(r.Status),
        IsSystem:     r.IsSystem,
        AuthProvider: r.AuthProvider,
        ExternalSub:  r.ExternalSub,
        LastLoginAt:  r.LastLoginAt,
        TotpSecret:   r.TotpSecret,
        TotpEnabled:  r.TotpEnabled,
        CreatedAt:    r.CreatedAt,
    }
}

func (userRow) TableName() string { return "users" }

func (r *UserRepo) FindByID(ctx context.Context, id int64) (*user.User, error) {
    var row userRow
    err := r.db.WithContext(ctx).
        Where("id = ?", id).
        Take(&row).Error

    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, errors.New("not found")
    }
    if err != nil {
        return nil, err
    }
    return row.toDomain(), nil
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*user.User, error) {
    var row userRow
    err := r.db.WithContext(ctx).
        Where("username = ?", username).
        Take(&row).Error

    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, errors.New("not found")
    }
    if err != nil {
        return nil, err
    }
    return row.toDomain(), nil
}

func (r *UserRepo) FindByExternalID(ctx context.Context, provider, externalSub string) (*user.User, error) {
    var row userRow
    err := r.db.WithContext(ctx).
        Where("auth_provider = ? AND external_sub = ?", provider, externalSub).
        Take(&row).Error

    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }
    return row.toDomain(), nil
}

func (r *UserRepo) FindSystemAdmin(ctx context.Context) (*user.User, error) {
	var row userRow
	err := r.db.WithContext(ctx).
		Where("is_system = ? AND status = ?", true, "active").
		Take(&row).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("system admin not found")
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

func (r *UserRepo) List(ctx context.Context) ([]user.User, error) {
    var rows []userRow
    if err := r.db.WithContext(ctx).Order("id").Find(&rows).Error; err != nil {
        return nil, err
    }

    out := make([]user.User, 0, len(rows))
    for _, row := range rows {
        out = append(out, *row.toDomain())
    }
    return out, nil
}

func (r *UserRepo) Create(ctx context.Context, u *user.User) (int64, error) {
    row := userRow{
        Username:     u.Username,
        Email:        u.Email,
        Description:  u.Description,
        PasswordHash: u.PasswordHash,
        Role:         string(u.Role),
        Status:       string(u.Status),
        IsSystem:     u.IsSystem,
        AuthProvider: u.AuthProvider,
        ExternalSub:  u.ExternalSub,
    }

    if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
        return 0, err
    }
    return row.ID, nil
}

func (r *UserRepo) Update(ctx context.Context, u *user.User) error {
    // 用 Updates 可以避免全量 Save 带来的误更新
    return r.db.WithContext(ctx).
        Model(&userRow{}).
        Where("id = ?", u.ID).
        Updates(map[string]any{
            "username":      u.Username,
            "email":         u.Email,
            "description":   u.Description,
            "password_hash": u.PasswordHash,
            "role":          string(u.Role),
            "status":        string(u.Status),
            "is_system":     u.IsSystem,
            "auth_provider": u.AuthProvider,
            "external_sub":  u.ExternalSub,
        }).Error
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
    return r.db.WithContext(ctx).
        Exec("DELETE FROM users WHERE id = ?", id).Error
}

func (r *UserRepo) UpdateLastLoginAt(ctx context.Context, id int64, ts int64) error {
    return r.db.WithContext(ctx).
        Exec("UPDATE users SET last_login_at = ? WHERE id = ?", ts, id).Error
}

func (r *UserRepo) UpdateDescription(ctx context.Context, id int64, description string) error {
    return r.db.WithContext(ctx).
        Exec("UPDATE users SET description = ? WHERE id = ?", description, id).Error
}

func (r *UserRepo) UpdateTotp(ctx context.Context, id int64, secret string, enabled bool) error {
    enabledInt := 0
    if enabled {
        enabledInt = 1
    }
    return r.db.WithContext(ctx).
        Exec("UPDATE users SET totp_secret = ?, totp_enabled = ? WHERE id = ?", secret, enabledInt, id).Error
}

func IsUniqueViolation(err error) bool {
    if err == nil {
        return false
    }
    return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}
