package store

import "gorm.io/gorm"

func init() {
	RegisterGORMMigration(func(db *gorm.DB) error {
		if db.Name() != "sqlite" {
			return nil
		}
		return db.Exec(`ALTER TABLE credentials ADD COLUMN pool_provider TEXT DEFAULT NULL`).Error
	})
}
