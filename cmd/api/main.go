package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"imgflow/internal/api"
	"imgflow/internal/client"
	"imgflow/internal/config"
	"imgflow/internal/db/minio"
	"imgflow/internal/db/postgres"
	"imgflow/internal/kafka"
	"imgflow/internal/repository"
	"imgflow/internal/service"

	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Pool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}
	defer db.Close()

	s3, err := minio.Client(minio.ClientOptions{
		Endpoint:  cfg.MinIOEndpoint,
		AccessKey: cfg.MinIOAccessKey,
		SecretKey: cfg.MinIOSecretKey,
		Bucket:    cfg.MinIOBucket,
	})
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}

	prod := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer func() { _ = prod.Close() }()

	meta := repository.NewMetadata(db)
	file := repository.NewFile(s3, cfg.MinIOBucket)

	pub := client.NewPublisher(prod)
	svc := service.New(meta, file, pub)

	a := api.New(svc)
	if err = a.Start(cfg.Addr); err != nil {
		log.Fatal().Stack().Err(err).Send()
	}
}
