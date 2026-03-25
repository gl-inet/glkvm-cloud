package user

import "context"

type Repository interface {
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByExternalID(ctx context.Context, provider, externalSub string) (*User, error)
	FindSystemAdmin(ctx context.Context) (*User, error)

	Create(ctx context.Context, u *User) (int64, error)
	Update(ctx context.Context, u *User) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]User, error)
}
