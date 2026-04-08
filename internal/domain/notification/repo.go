package notification

import "context"

// Repository persists notification configuration.
type Repository interface {
	GetSMTPConfig(ctx context.Context) (*SMTPConfig, error)
	SaveSMTPConfig(ctx context.Context, cfg *SMTPConfig) error

	GetNotifyRules(ctx context.Context) (*NotifyRules, error)
	SaveNotifyRules(ctx context.Context, rules *NotifyRules) error

	ListRecipients(ctx context.Context) ([]Recipient, error)
	AddRecipient(ctx context.Context, email string) (*Recipient, error)
	RemoveRecipient(ctx context.Context, id int64) error
}
