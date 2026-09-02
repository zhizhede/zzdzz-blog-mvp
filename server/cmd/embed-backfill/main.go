// embed-backfill 全量(重)嵌入所有非 draft 文章的向量块, 幂等可重跑.
// 用途: 1) 首次开启 AI 回顾后的存量回填  2) 换 embedding 模型后的全量重嵌.
// 用法(在 server/ 目录下):
//
//	go run ./cmd/embed-backfill -config config/config.yaml
//
// 私人笔记是否纳入由 ai.recall.index_private 决定(IndexArticle 内部处理).
package main

import (
	"context"
	"flag"
	"log"
	"time"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/database"
	"zzdzz-blog/server/internal/model"
	"zzdzz-blog/server/internal/service"
)

func main() {
	cfgPath := flag.String("config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	db, err := database.New(cfg.Database.DSN())
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}

	embed, err := service.NewOpenAIEmbedding(
		cfg.AI.Embedding.BaseURL, cfg.AI.Embedding.APIKey, cfg.AI.Embedding.Model, cfg.AI.Embedding.Dims,
		cfg.AI.Embedding.Asymmetric, cfg.AI.Embedding.Style,
	)
	if err != nil {
		log.Fatalf("embedding: %v", err)
	}
	recall := service.NewRecallService(db, embed, cfg.AI.Recall)

	var ids []uint64
	if err := db.Model(&model.Article{}).
		Where("visibility != ?", "draft").
		Order("id ASC").
		Pluck("id", &ids).Error; err != nil {
		log.Fatalf("list articles: %v", err)
	}
	log.Printf("backfill %d articles (index_private=%v)", len(ids), cfg.AI.Recall.IndexPrivate)

	var failed int
	for i, id := range ids {
		var a model.Article
		if err := db.First(&a, id).Error; err != nil {
			log.Printf("[%d/%d] load article %d: %v", i+1, len(ids), id, err)
			failed++
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		if err := recall.IndexArticle(ctx, &a); err != nil {
			log.Printf("[%d/%d] article %d (%s): %v", i+1, len(ids), a.ID, a.Title, err)
			failed++
		} else {
			log.Printf("[%d/%d] article %d (%s) ok", i+1, len(ids), a.ID, a.Title)
		}
		cancel()
	}
	if failed > 0 {
		log.Fatalf("done with %d failures", failed)
	}
	log.Printf("backfill complete: %d articles", len(ids))
}
