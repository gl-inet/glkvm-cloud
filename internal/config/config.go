package config

import "time"

type Config struct {
	HTTP struct{ Addr string }
	Auth struct{ SessionTTL time.Duration }
	DB   struct{ DSN string }
}

func MustLoad() Config {
	var cfg Config
	cfg.HTTP.Addr = ":8080"
	cfg.Auth.SessionTTL = 24 * time.Hour
	cfg.DB.DSN = "file:rttys.db?_pragma=foreign_keys(1)"
	return cfg
}
