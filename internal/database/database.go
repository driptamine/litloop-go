package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"litloop/backend/internal/config"
	"litloop/backend/internal/models"
)

func Open(appConfig config.Config) (*gorm.DB, error) {
	var driver gorm.Dialector
	if appConfig.DatabaseURL != "" {
		driver = postgres.New(postgres.Config{DSN: appConfig.DatabaseURL, PreferSimpleProtocol: true})
	} else {
		driver = sqlite.Open(appConfig.SQLitePath + "?_foreign_keys=on&_busy_timeout=10000")
	}
	db, err := gorm.Open(driver, &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	if appConfig.DatabaseURL == "" {
		pool.SetMaxOpenConns(1)
	} else {
		pool.SetMaxOpenConns(20)
		pool.SetMaxIdleConns(5)
		pool.SetConnMaxLifetime(30 * time.Minute)
	}
	return db, nil
}

func Migrate(db *gorm.DB) error {
	if db.Dialector.Name() == "postgres" {
		var djangoSchema bool
		if err := db.Raw(`SELECT EXISTS (SELECT 1 FROM information_schema.tables
			WHERE table_schema = current_schema() AND table_name = 'django_migrations'
			AND table_type = 'BASE TABLE')`).Scan(&djangoSchema).Error; err != nil {
			return err
		}
		if djangoSchema {
			return db.Transaction(migrateDjangoSchema)
		}
	}
	return db.AutoMigrate(models.All()...)
}

// Django records constraints and indexes under its own names. Keep those
// definitions when adding tables and columns for the Go backend.
func migrateDjangoSchema(tx *gorm.DB) error {
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext('litloop.schema'))").Error; err != nil {
		return err
	}
	var columns []struct {
		TableName	string
		ColumnName	string
	}
	if err := tx.Raw(`SELECT table_name, column_name FROM information_schema.columns
		WHERE table_schema = current_schema()`).Scan(&columns).Error; err != nil {
		return err
	}
	existing := make(map[string]map[string]bool)
	for _, column := range columns {
		if existing[column.TableName] == nil {
			existing[column.TableName] = make(map[string]bool)
		}
		existing[column.TableName][column.ColumnName] = true
	}

	migrator := tx.Migrator().(postgres.Migrator)
	for _, model := range migrator.ReorderModels(models.All(), true) {
		stmt := &gorm.Statement{DB: tx}
		if err := stmt.Parse(model); err != nil {
			return err
		}
		knownColumns := existing[stmt.Table]
		if knownColumns == nil {
			if err := migrator.CreateTable(model); err != nil {
				return fmt.Errorf("create %s: %w", stmt.Table, err)
			}
			continue
		}
		for _, name := range stmt.Schema.DBNames {
			if !knownColumns[name] && !stmt.Schema.FieldsByDBName[name].IgnoreMigration {
				if err := migrator.AddColumn(model, name); err != nil {
					return fmt.Errorf("add %s.%s: %w", stmt.Table, name, err)
				}
			}
		}
	}
	return nil
}
