package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/database"
	"zzdzz-blog/server/internal/router"
)

func main() {
	cfgPath := os.Getenv("ZZDZZ_CONFIG")
	if cfgPath == "" {
		exe, err := os.Executable()
		if err != nil {
			exe = "."
		}
		dir := filepath.Dir(exe)
		candidates := []string{
			filepath.Join(dir, "config", "config.production.yaml"),
			filepath.Join(dir, "config", "config.local.yaml"),
			filepath.Join(dir, "config", "config.yaml"),
			"config/config.production.yaml",
			"config/config.local.yaml",
			"config/config.yaml",
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				cfgPath = p
				break
			}
		}
		if cfgPath == "" {
			cfgPath = "config/config.yaml"
		}
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.New(cfg.Database.DSN())
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	r := router.New(db, cfg)

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("server listening on %s (mode=%s)", addr, cfg.Server.Mode)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}