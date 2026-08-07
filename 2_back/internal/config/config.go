package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"rkw_hatcher/internal/db"
)

type Config struct {
	HTTPAddr string
	DBPath   string
}

func Load() Config {
	if exe, err := os.Executable(); err == nil {
		_ = godotenv.Load(filepath.Join(filepath.Dir(exe), ".env"))
	}
	_ = godotenv.Load()
	path := getenv("DB_PATH", "")
	if path == "" {
		path = db.ResolveDBPath()
	}
	return Config{
		HTTPAddr: getenv("HTTP_ADDR", ":3070"),
		DBPath:   path,
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
