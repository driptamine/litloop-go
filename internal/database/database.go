package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"litloop/backend/internal/config"
	"litloop/backend/internal/models"
	"time"
)

func Open(c config.Config) (*gorm.DB, error) {
	var driver gorm.Dialector
	if c.DatabaseURL != "" {
		driver = postgres.New(postgres.Config{DSN: c.DatabaseURL, PreferSimpleProtocol: true})
	} else {
		driver = sqlite.Open(c.SQLitePath + "?_foreign_keys=on&_busy_timeout=10000")
	}
	db, e := gorm.Open(driver, &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		return nil, e
	}
	pool, e := db.DB()
	if e != nil {
		return nil, e
	}
	if c.DatabaseURL == "" {
		pool.SetMaxOpenConns(1)
	} else {
		pool.SetMaxOpenConns(20)
		pool.SetMaxIdleConns(5)
		pool.SetConnMaxLifetime(30 * time.Minute)
	}
	return db, nil
}
func Migrate(db *gorm.DB) error { return db.AutoMigrate(models.All()...) }
