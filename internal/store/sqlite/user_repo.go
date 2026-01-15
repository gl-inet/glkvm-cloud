package sqlite

import (
    "context"
    "database/sql"
    "errors"
    "rttys/internal/domain/identity"
    "strings"

    "rttys/internal/domain/user"
)

type UserRepo struct{ db *sql.DB }

func NewUserRepo(db *sql.DB) *UserRepo { return &UserRepo{db: db} }

func (r *UserRepo) FindByID(ctx context.Context, id int64) (*user.User, error) {
    const q = `SELECT id, email, display_name, password_hash, role, status FROM users WHERE id=? LIMIT 1`
    var u user.User
    var role, status string
    err := r.db.QueryRowContext(ctx, q, id).Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &role, &status)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, errors.New("not found")
    }
    if err != nil {
        return nil, err
    }
    u.Role = identity.Role(role)
    u.Status = user.Status(status)
    return &u, nil
}

func (r *UserRepo) FindByUsername(ctx context.Context, email string) (*user.User, error) {
    const q = `SELECT id, email, display_name, password_hash, role, status FROM users WHERE email=? LIMIT 1`
    var u user.User
    var role, status string
    err := r.db.QueryRowContext(ctx, q, email).Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &role, &status)
    if errors.Is(err, sql.ErrNoRows) {
        return nil, errors.New("not found")
    }
    if err != nil {
        return nil, err
    }
    u.Role = identity.Role(role)
    u.Status = user.Status(status)
    return &u, nil
}

func (r *UserRepo) List(ctx context.Context) ([]user.User, error) {
    rows, err := r.db.QueryContext(ctx, `SELECT id,email,display_name,password_hash,role,status FROM users ORDER BY id`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var out []user.User
    for rows.Next() {
        var u user.User
        var role, status string
        if err := rows.Scan(&u.ID, &u.Email, &u.DisplayName, &u.PasswordHash, &role, &status); err != nil {
            return nil, err
        }
        u.Role = identity.Role(role)
        u.Status = user.Status(status)
        out = append(out, u)
    }
    return out, rows.Err()
}

func (r *UserRepo) Create(ctx context.Context, u *user.User) (int64, error) {
    res, err := r.db.ExecContext(ctx, `
INSERT INTO users(email, display_name, password_hash, role, status)
VALUES (?,?,?,?,?)`,
        u.Email, u.DisplayName, u.PasswordHash, string(u.Role), string(u.Status),
    )
    if err != nil {
        return 0, err
    }
    return res.LastInsertId()
}

func (r *UserRepo) Update(ctx context.Context, u *user.User) error {
    _, err := r.db.ExecContext(ctx, `
UPDATE users
SET email=?, display_name=?, password_hash=?, role=?, status=?
WHERE id=?`,
        u.Email, u.DisplayName, u.PasswordHash, string(u.Role), string(u.Status), u.ID,
    )
    return err
}

func (r *UserRepo) Delete(ctx context.Context, id int64) error {
    _, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id=?`, id)
    return err
}

func IsUniqueViolation(err error) bool {
    if err == nil {
        return false
    }
    // modernc/sqlite error is often a plain string containing "UNIQUE constraint failed"
    return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed")
}
