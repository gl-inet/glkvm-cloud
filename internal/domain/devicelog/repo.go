package devicelog

import "context"

type Repository interface {
	Create(ctx context.Context, l *Log) (int64, error)
	UpdateEndedAt(ctx context.Context, id int64, ts int64) error
	List(ctx context.Context, q Query) (items []Log, total int64, err error)
}
