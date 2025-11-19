package main

import (
	"context"
	"flag"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/zanmajeric/reporadar-go-ingest/api_server"
	"github.com/zanmajeric/reporadar-go-ingest/config"
	"github.com/zanmajeric/reporadar-go-ingest/embedder"
	"github.com/zanmajeric/reporadar-go-ingest/internal/search"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	configFiles := flag.String("configFiles", "config.yaml", "Comma separated list of config files to load")
	flag.Parse()
	cfg := config.LoadConfig(strings.Split(*configFiles, ","))

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseUrl)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}
	log.Println("Connected to Postgres")

	embedderClient := embedder.NewClient(cfg.EmbedderUrl)
	issueRep := search.NewPgRepository(pool)
	searchSrv := search.New(embedderClient, issueRep, *cfg)
	s := api_server.NewServer(cfg, pool, searchSrv)
	log.Printf("Go ingest service listening on :%d", cfg.HttpPort)
	s.Run()
}
