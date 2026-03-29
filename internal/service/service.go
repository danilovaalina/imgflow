package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"imgflow/internal/model"
	"imgflow/internal/repository"

	"github.com/cockroachdb/errors"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type MetadataRepository interface {
	CreateImage(ctx context.Context, opts repository.CreateImageOptions) error
	UpdateStatus(ctx context.Context, opts repository.UpdateImageOptions) error
	Image(ctx context.Context, id uuid.UUID) (model.Image, error)
	DeleteImage(ctx context.Context, id uuid.UUID) error
}

type FileRepository interface {
	Put(ctx context.Context, params repository.PutOptions) error
	Remove(ctx context.Context, objectName string) error
	Get(ctx context.Context, objectName string) (io.ReadCloser, error)
}

type Queue interface {
	Publish(ctx context.Context, msg model.ImageTask) error
}

type Service struct {
	metadataRepo MetadataRepository
	fileRepo     FileRepository
	queue        Queue
}

func New(metadataRepo MetadataRepository, fileRepo FileRepository, queue Queue) *Service {
	return &Service{
		metadataRepo: metadataRepo,
		fileRepo:     fileRepo,
		queue:        queue,
	}
}

type UploadImageOptions struct {
	Filename    string
	Content     io.Reader
	Size        int64
	ContentType string
}

func (s *Service) UploadImage(ctx context.Context, opts UploadImageOptions) (uuid.UUID, error) {
	id := uuid.New()
	objectName := id.String() + "_" + opts.Filename

	// Кладем файл в S3
	err := s.fileRepo.Put(ctx, repository.PutOptions{
		ObjectName:  objectName,
		Reader:      opts.Content,
		Size:        opts.Size,
		ContentType: opts.ContentType,
	})
	if err != nil {
		return uuid.Nil, err
	}

	err = s.metadataRepo.CreateImage(ctx, repository.CreateImageOptions{
		ID:       id,
		Filename: opts.Filename,
		Status:   model.StatusPending,
		Created:  time.Now(),
	})
	if err != nil {
		return uuid.Nil, err
	}

	// Отправляем ID задачи в Kafka
	err = s.queue.Publish(ctx, model.ImageTask{
		ID:       id,
		Filename: objectName, // Передаем имя объекта, чтобы воркер знал, что скачивать
	})
	if err != nil {
		log.Error().Err(err).Send()
		return uuid.Nil, err
	}

	return id, nil
}

// Image возвращает информацию об изображении
func (s *Service) Image(ctx context.Context, id uuid.UUID) (model.Image, error) {
	img, err := s.metadataRepo.Image(ctx, id)
	if err != nil {
		return model.Image{}, err
	}
	return img, nil
}

// DeleteImage удаляет всё: и запись в БД, и файлы в S3
func (s *Service) DeleteImage(ctx context.Context, id uuid.UUID) error {
	// Сначала находим запись, чтобы узнать имена файлов в S3
	img, err := s.metadataRepo.Image(ctx, id)
	if err != nil {
		return err
	}

	// Удаляем оригинал из S3
	// Имя объекта у нас: ID + "_" + Filename
	objectName := img.ID.String() + "_" + img.Filename
	err = s.fileRepo.Remove(ctx, objectName)
	if err != nil {
		log.Warn().Err(err).Str("task_id", img.ID.String()).Str("file", objectName).Send()
	}

	// Если есть обработанное фото, удаляем и его
	if img.Status == model.StatusCompleted || img.ProcessedURL != "" {
		procObjectName := "proc_" + img.ID.String() + "_" + img.Filename
		err = s.fileRepo.Remove(ctx, procObjectName)
		if err != nil {
			log.Warn().Err(err).Str("task_id", img.ID.String()).Str("file", procObjectName).Send()
		}
	}

	err = s.metadataRepo.DeleteImage(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

// Process основной метод воркера для обработки изображения
func (s *Service) Process(ctx context.Context, task model.ImageTask) error {
	// Устанавливаем статус "В обработке"
	err := s.metadataRepo.UpdateStatus(ctx, repository.UpdateImageOptions{
		ID:      task.ID,
		Status:  model.StatusProcessing,
		Updated: time.Now(),
	})
	if err != nil {
		return err
	}

	// Получаем оригинал из MinIO
	// Имя файла мы формировали при загрузке: ID_Filename
	objectName := task.ID.String() + "_" + task.Filename
	reader, err := s.fileRepo.Get(ctx, objectName)
	if err != nil {
		err = s.metadataRepo.UpdateStatus(ctx, repository.UpdateImageOptions{
			ID:     task.ID,
			Status: model.StatusFailed,
		})
		if err != nil {
			return err
		}
		return err
	}
	defer func() { _ = reader.Close() }()

	// Обработка (Resize + Watermark)
	processedReader, err := s.processImage(reader)
	if err != nil {
		err = s.metadataRepo.UpdateStatus(ctx, repository.UpdateImageOptions{
			ID:     task.ID,
			Status: model.StatusFailed,
		})
		if err != nil {
			return err
		}
		return err
	}

	// Загружаем обработанный результат с префиксом "proc_"
	procObjectName := "proc_" + objectName
	err = s.fileRepo.Put(ctx, repository.PutOptions{
		ObjectName:  procObjectName,
		Reader:      processedReader,
		ContentType: "image/jpeg",
	})
	if err != nil {
		return err
	}

	// Финализируем: обновляем ссылки и статус в БД
	// В реальном проекте здесь будут полные URL к MinIO или через Cloudfront
	origURL := fmt.Sprintf("/images/%s", objectName)
	procURL := fmt.Sprintf("/images/%s", procObjectName)

	err = s.metadataRepo.UpdateStatus(ctx, repository.UpdateImageOptions{
		ID:           task.ID,
		Status:       model.StatusCompleted,
		OriginalURL:  origURL,
		ProcessedURL: procURL,
		Updated:      time.Now(),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) processImage(r io.Reader) (io.Reader, error) {
	// Декодируем изображение
	src, err := imaging.Decode(r)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Ресайз: уменьшаем до 800px по ширине с сохранением пропорций
	dst := imaging.Resize(src, 800, 0, imaging.Lanczos)

	// Водяной знак: для примера сделаем небольшое размытие краев
	// или наложим полупрозрачный фильтр (имитация watermark)
	// В реальной задаче здесь можно использовать imaging.Overlay для логотипа
	dst = imaging.PasteCenter(dst, imaging.Grayscale(imaging.Thumbnail(src, 100, 100, imaging.Lanczos)))

	// Используем Pipe для передачи результата без лишнего буфера в памяти
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		// Кодируем в JPEG и пишем прямо в Pipe
		if err = imaging.Encode(pw, dst, imaging.JPEG); err != nil {
			log.Error().Err(err).Msg("failed to encode image to pipe")
		}
	}()

	return pr, nil
}
