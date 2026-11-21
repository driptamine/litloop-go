package config

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	RedisURL                                                                                                string
	Addr, DatabaseURL, SQLitePath, JWTSecret, MediaDir, PublicURL                                           string
	AutoMigrate                                                                                             bool
	Origins                                                                                                 []string
	AccessTTL, RefreshTTL                                                                                   time.Duration
	S3Endpoint, S3Region, S3Bucket, S3AccessKey, S3SecretKey, S3PublicURL                                   string
	R2Endpoint, R2Bucket, R2AccessKey, R2SecretKey, R2PublicURL                                             string
	RequireR2Media                                                                                          bool
	GCSBucket, GCSAccessKey, GCSSecretKey, GoogleCredentials                                                string
	GoogleClientID, VKClientID, VKClientSecret                                                              string
	GoogleAdditionalRedirectURIs                                                                            []string
	GeminiKey, GeminiModel, BraveKey, BingKey, GoogleSearchKey, GoogleSearchCX, TMDBKey                     string
	SMTPAddr, SMTPUser, SMTPPassword, SMTPFrom, FrontendURL, TranscriptionURL, TranscriptionSecret, SpotURL string
	ICEURLs, ICEUsername, ICECredential                                                                     string
	AstraDatabaseID, AstraToken, AstraKeyspace                                                              string
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func Load() Config {
	_ = godotenv.Load()
	secret := env("JWT_SECRET", os.Getenv("SECRET_KEY"))
	if secret == "" {
		b := make([]byte, 32)
		if _, e := rand.Read(b); e != nil {
			panic(e)
		}
		secret = hex.EncodeToString(b)
	}
	hours := func(k string, d int) time.Duration {
		v, _ := strconv.Atoi(env(k, strconv.Itoa(d)))
		if v < 1 {
			v = d
		}
		return time.Duration(v) * time.Hour
	}
	origins := strings.Split(env("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173"), ",")
	if additional := os.Getenv("CORS_ADDITIONAL_ORIGINS"); additional != "" {
		origins = append(origins, strings.Split(additional, ",")...)
	}
	return Config{
		RedisURL: os.Getenv("REDIS_URL"),
		Addr:     env("HTTP_ADDR", ":8000"), DatabaseURL: os.Getenv("DATABASE_URL"), SQLitePath: env("SQLITE_PATH", "litloop.db"), JWTSecret: secret,
		MediaDir: env("MEDIA_DIR", "var/media"), PublicURL: env("PUBLIC_URL", "http://localhost:8000"), AutoMigrate: env("AUTO_MIGRATE", "false") == "true", Origins: origins, AccessTTL: hours("ACCESS_TTL_HOURS", 1), RefreshTTL: hours("REFRESH_TTL_HOURS", 720),
		S3Endpoint: os.Getenv("S3_ENDPOINT"), S3Region: env("S3_REGION", "us-east-1"), S3Bucket: os.Getenv("S3_BUCKET"), S3AccessKey: os.Getenv("AWS_ACCESS_KEY_ID"), S3SecretKey: os.Getenv("AWS_SECRET_ACCESS_KEY"), S3PublicURL: os.Getenv("S3_PUBLIC_URL"),
		R2Endpoint: os.Getenv("R2_ENDPOINT"), R2Bucket: os.Getenv("R2_BUCKET_NAME"), R2AccessKey: os.Getenv("R2_ACCESS_KEY_ID"), R2SecretKey: os.Getenv("R2_SECRET_ACCESS_KEY"), R2PublicURL: os.Getenv("R2_PUBLIC_URL"),
		RequireR2Media: env("REQUIRE_R2_MEDIA", "false") == "true",
		GCSBucket:      os.Getenv("GCS_BUCKET_NAME"), GCSAccessKey: os.Getenv("GCS_ACCESS_KEY_ID"), GCSSecretKey: os.Getenv("GCS_SECRET_ACCESS_KEY"), GoogleCredentials: os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"), GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"), VKClientID: os.Getenv("VK_CLIENT_ID"), VKClientSecret: os.Getenv("VK_CLIENT_SECRET"),
		GoogleAdditionalRedirectURIs: strings.FieldsFunc(os.Getenv("GOOGLE_OAUTH_ADDITIONAL_REDIRECT_URIS"), func(r rune) bool { return r == ',' || r == ' ' }),
		GeminiKey:                    os.Getenv("GEMINI_API_KEY"), GeminiModel: env("GEMINI_MODEL", "gemini-2.5-flash"), BraveKey: os.Getenv("BRAVE_API_KEY"), BingKey: os.Getenv("BING_API_KEY"), GoogleSearchKey: os.Getenv("GOOGLE_SEARCH_API_KEY"), GoogleSearchCX: os.Getenv("GOOGLE_SEARCH_CX"), TMDBKey: os.Getenv("TMDB_API_KEY"),
		SMTPAddr: os.Getenv("SMTP_ADDR"), SMTPUser: os.Getenv("SMTP_USER"), SMTPPassword: os.Getenv("SMTP_PASSWORD"), SMTPFrom: os.Getenv("SMTP_FROM"), FrontendURL: env("FRONTEND_URL", "http://localhost:3000"), TranscriptionURL: os.Getenv("TRANSCRIPTION_URL"), TranscriptionSecret: os.Getenv("TRANSCRIPTION_SECRET"), SpotURL: os.Getenv("SPOT_LAUNCH_URL"),
		ICEURLs: env("ICE_URLS", "stun:stun.l.google.com:19302"), ICEUsername: os.Getenv("ICE_USERNAME"), ICECredential: os.Getenv("ICE_CREDENTIAL"),
		AstraDatabaseID: os.Getenv("ASTRA_DB_ID"), AstraToken: os.Getenv("ASTRA_DB_APPLICATION_TOKEN"), AstraKeyspace: env("ASTRA_DB_KEYSPACE", "engagement"),
	}
}

func (c Config) UsesR2ForMedia() bool { return c.RequireR2Media || c.R2Bucket != "" }
