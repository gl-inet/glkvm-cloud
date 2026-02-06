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

    u := &user.User{
        ID:           row.ID,
        Username:     row.Username,
        Email:        row.Email,
        Description:  row.Description,
        PasswordHash: row.PasswordHash,
        Role:         identity.Role(row.Role),
        Status:       user.Status(row.Status),
        IsSystem:     row.IsSystem,
    }
    return u, nil
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

    return &user.User{
        ID:           row.ID,
        Username:     row.Username,
        Email:        row.Email,
        Description:  row.Description,
        PasswordHash: row.PasswordHash,
        Role:         identity.Role(row.Role),
        Status:       user.Status(row.Status),
        IsSystem:     row.IsSystem,
    }, nil
}

func (r *UserRepo) List(ctx context.Context) ([]user.User, error) {
    var rows []userRow
    if err := r.db.WithContext(ctx).Order("id").Find(&rows).Error; err != nil {
        return nil, err
    }

    out := make([]user.User, 0, len(rows))
    for _, row := range rows {
        out = append(out, user.User{
            ID:           row.ID,
            Username:     row.Username,
            Email:        row.Email,
            Description:  row.Description,
            PasswordHash: row.PasswordHash,
            Role:         identity.Role(row.Role),
            Status:       user.Status(row.Status),
            IsSystem:     row.IsSystem,
        })
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
        }).Error
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
    return r.db.WithContext(ctx).
        Exec("DELETE FROM users WHERE id = ?", id).Error
}

func IsUniqueViolation(err error) bool {
    if err == nil {
        return false
    }
    return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}
