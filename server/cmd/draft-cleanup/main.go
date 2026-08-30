// draft-cleanup 删除 30 天没动过的 draft 文章.
//
// 用法:
//   draft-cleanup                        使用默认 30 天
//   draft-cleanup -days 7 -dry-run       只打印不删
//
// 部署建议: 宝塔计划任务每天 04:00 跑一次 (带绝对路径, 不要用相对).
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/database"
)

func main() {
	days := flag.Int("days", 30, "删除多少天没动过的 draft")
	dryRun := flag.Bool("dry-run", false, "只打印待删记录, 不真删")
	flag.Parse()

	cfgPath := resolveConfig()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	db, err := database.New(cfg.Database.DSN())
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	cutoff := time.Now().UTC().Add(-time.Duration(*days) * 24 * time.Hour)
	var affected int64
	if *dryRun {
		var count int64
		if err := db.Model(&struct{}{}).
			Table("articles").
			Where("visibility = ? AND (last_autosaved_at IS NULL OR last_autosaved_at < ?)", "draft", cutoff).
			Count(&count).Error; err != nil {
			log.Fatalf("count: %v", err)
		}
		fmt.Printf("dry-run: 满足条件的 draft 共 %d 条 (cutoff=%s, days=%d)\n", count, cutoff.Format(time.RFC3339), *days)
		return
	}
	res := db.Exec(
		"DELETE FROM articles WHERE visibility = ? AND (last_autosaved_at IS NULL OR last_autosaved_at < ?)",
		"draft", cutoff,
	)
	if res.Error != nil {
		log.Fatalf("delete: %v", res.Error)
	}
	affected = res.RowsAffected
	fmt.Printf("已删除 %d 条过期 draft (cutoff=%s, days=%d)\n", affected, cutoff.Format(time.RFC3339), *days)
}

func resolveConfig() string {
	if env := os.Getenv("ZZDZZ_CONFIG"); env != "" {
		return env
	}
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
			return p
		}
	}
	return "config/config.yaml"
}
