package sqlite

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"rttys/internal/domain/access"
	"rttys/utils"
)

type AccessRepo struct{ db *gorm.DB }

func NewAccessRepo(db *gorm.DB) *AccessRepo { return &AccessRepo{db: db} }

type whitelistRow struct {
	MAC       string `gorm:"column:mac;primaryKey"`
	Devid     string `gorm:"column:devid"`
	CodeID    *int64 `gorm:"column:code_id"`
	CreatedAt int64  `gorm:"column:created_at"`
}

func (whitelistRow) TableName() string { return "device_whitelist" }

func (r whitelistRow) toDomain() *access.WhitelistEntry {
	e := &access.WhitelistEntry{MAC: r.MAC, Devid: r.Devid, CreatedAt: r.CreatedAt}
	if r.CodeID != nil {
		e.CodeID = *r.CodeID
	}
	return e
}

// GetByMAC returns the whitelist entry for mac, or nil if it is not listed.
func (r *AccessRepo) GetByMAC(ctx context.Context, mac string) (*access.WhitelistEntry, error) {
	var row whitelistRow
	err := r.db.WithContext(ctx).Where("mac = ?", utils.NormalizeMac(mac)).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toDomain(), nil
}

// Claim redeems a registration code for the device with this MAC and adds it
// to the whitelist, all in one transaction. It returns the device ID the
// device is listed under:
//   - a MAC the platform already knows (e.g. the same device connected before
//     with a token) keeps its existing ID, so its record carries over;
//   - otherwise the requested ID is used, unless another device holds it.
//
// Ownership is checked before the code is consumed, so a rejected claim does
// not burn the code.
func (r *AccessRepo) Claim(ctx context.Context, codeHash, mac, devid string, now int64) (string, error) {
	mac = utils.NormalizeMac(mac)
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var known string
		if err := tx.Raw(`SELECT ddns FROM devices WHERE mac = ?`, mac).Scan(&known).Error; err != nil {
			return err
		}
		if known != "" {
			devid = known
		}

		var owners []string
		if err := tx.Raw(
			`SELECT mac FROM device_whitelist WHERE devid = ?
			 UNION SELECT mac FROM devices WHERE ddns = ?`, devid, devid,
		).Scan(&owners).Error; err != nil {
			return err
		}
		for _, o := range owners {
			if utils.NormalizeMac(o) != mac {
				return access.ErrDevidTaken
			}
		}

		res := tx.Exec(
			`UPDATE registration_codes SET used_at = ?, used_mac = ?
			  WHERE code_hash = ? AND used_at IS NULL AND revoked_at IS NULL AND expires_at > ?`,
			now, mac, codeHash, now,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected != 1 {
			return access.ErrCodeInvalid
		}

		return tx.Exec(
			`INSERT INTO device_whitelist (mac, devid, code_id, created_at)
			 VALUES (?, ?, (SELECT id FROM registration_codes WHERE code_hash = ?), ?)
			 ON CONFLICT(mac) DO UPDATE SET devid = excluded.devid, code_id = excluded.code_id`,
			mac, devid, codeHash, now,
		).Error
	})
	if err != nil {
		return "", err
	}
	return devid, nil
}
