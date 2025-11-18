package main

import (
	"context"
	"flag"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/zanmajeric/reporadar-go-ingest/api_server"
	"github.com/zanmajeric/reporadar-go-ingest/config"
	"github.com/zanmajeric/reporadar-go-ingest/embedder"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	configFiles := flag.String("configFiles", "config.yaml", "Comma separated list of config files to load")
	flag.Parse()

	configuration := config.LoadConfig(strings.Split(*configFiles, ","))

	embedderClient := embedder.NewClient(configuration.EmbedderUrl)

	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}
	log.Println("Connected to Postgres")

	s := api_server.NewServer(configuration.HttpPort, pool, embedderClient)
	log.Printf("Go ingest service listening on :%d", configuration.HttpPort)
	s.Run()
}
