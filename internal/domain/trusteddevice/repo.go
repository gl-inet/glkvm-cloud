package trusteddevice

import "context"

type Repository interface {
	Create(ctx context.Context, d *Device) (int64, error)
	FindByToken(ctx context.Context, token string) (*Device, error)
	ListByUserID(ctx context.Context, userID int64) ([]Device, error)
	Delete(ctx context.Context, id, userID int64) error
	DeleteByUserID(ctx context.Context, userID int64) error
	TouchLastUsed(ctx context.Context, id int64, ts int64) error
	DeleteExpired(ctx context.Context, before int64) error
}
