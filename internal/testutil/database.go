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

func Open(t *testing.T, c *config.Config) *gorm.DB {
	t.Helper()
	var admin *gorm.DB
	schema := ""
	if raw := os.Getenv("TEST_DATABASE_URL"); raw != "" {
		var e error
		admin, e = database.Open(config.Config{DatabaseURL: raw})
		if e != nil {
			t.Fatal(e)
		}
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		schema = "test_" + hex.EncodeToString(b)
		if e = admin.Exec(`CREATE SCHEMA "` + schema + `"`).Error; e != nil {
			t.Fatal(e)
		}
		uri, e := url.Parse(raw)
		if e != nil {
			t.Fatal(e)
		}
		query := uri.Query()
		query.Set("search_path", schema)
		uri.RawQuery = query.Encode()
		c.DatabaseURL = uri.String()
	}
	db, e := database.Open(*c)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		pool, _ := db.DB()
		_ = pool.Close()
		if admin != nil {
			_ = admin.Exec(`DROP SCHEMA "` + schema + `" CASCADE`).Error
			pool, _ = admin.DB()
			_ = pool.Close()
		}
	})
	if e = database.Migrate(db); e != nil {
		t.Fatal(e)
	}
	return db
}
