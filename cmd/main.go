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

	pool, err := postgres.Pool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal().Stack().Err(err).Send()
	}
	defer pool.Close()

	zzz, err := minio.Client(cfg.MinIOEndpoint, cfg.MinIOAccessKey, cfg.MinIOSecretKey)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init minio")
	}

	producer := kafka.NewProducer(cfg.KafkaBrokers, cfg.KafkaTopic)
	defer func() { _ = producer.Close() }()
	p := client.NewPublisher(producer)
	metaRepo := repository.NewMetadata(pool)
	fileRepo := repository.NewFile(zzz, cfg.MinIOBucket)
	s := service.New(metaRepo, fileRepo, p)

	sbs := client.NewSubscriber(s)
	consumer := kafka.NewConsumer(cfg.KafkaBrokers, cfg.KafkaTopic, cfg.KafkaGroupID)
	consumer.Start(ctx, sbs.Handle)

	a := api.New(s)
	if err = a.Start(cfg.Addr); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
