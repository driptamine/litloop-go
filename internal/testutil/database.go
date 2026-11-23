// Package testutil isolates each SQL test in a temporary database namespace.
package testutil

import (
	"crypto/rand"
	"encoding/hex"
	"gorm.io/gorm"
	"litloop/backend/internal/config"
	"litloop/backend/internal/database"
	"net/url"
	"os"
	"testing"
)

func Open(tester *testing.T, appConfig *config.Config) *gorm.DB {
	tester.Helper()
	var admin *gorm.DB
	schema := ""
	if raw := os.Getenv("TEST_DATABASE_URL"); raw != "" {
		var err error
		admin, err = database.Open(config.Config{DatabaseURL: raw})
		if err != nil {
			tester.Fatal(err)
		}
		randomBytes := make([]byte, 8)
		_, _ = rand.Read(randomBytes)
		schema = "test_" + hex.EncodeToString(randomBytes)
		if err = admin.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
			tester.Fatal(err)
		}
		uri, err := url.Parse(raw)
		if err != nil {
			tester.Fatal(err)
		}
		query := uri.Query()
		query.Set("search_path", schema)
		uri.RawQuery = query.Encode()
		appConfig.DatabaseURL = uri.String()
	}
	db, err := database.Open(*appConfig)
	if err != nil {
		tester.Fatal(err)
	}
	tester.Cleanup(func() {
		pool, _ := db.DB()
		_ = pool.Close()
		if admin != nil {
			_ = admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`).Error
			pool, _ = admin.DB()
			_ = pool.Close()
		}
	})
	if err = database.Migrate(db); err != nil {
		tester.Fatal(err)
	}
	return db
}
