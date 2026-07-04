package store

import "gorm.io/gorm"

func init() {
	RegisterGORMMigration(func(db *gorm.DB) error {
		if db.Name() != "sqlite" {
			return nil
		}
		return db.Exec(`ALTER TABLE request_logs ADD COLUMN account_id TEXT NOT NULL DEFAULT ''`).Error
	})
}
