package model

import (
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
)

type ImageStatus string

const (
	StatusPending    ImageStatus = "pending"
	StatusProcessing ImageStatus = "processing"
	StatusCompleted  ImageStatus = "completed"
	StatusFailed     ImageStatus = "failed"
)

type ImageFormat string

const (
	FormatJPG  ImageFormat = "jpg"
	FormatPNG  ImageFormat = "png"
	FormatWEBP ImageFormat = "webp"
	FormatGIF  ImageFormat = "gif"
)

type Image struct {
	ID           uuid.UUID
	Filename     string
	Format       ImageFormat
	Status       ImageStatus
	OriginalURL  string
	ProcessedURL string
	Created      time.Time
	Updated      time.Time
}

type ImageTask struct {
	ID       uuid.UUID
	Filename string
	Format   ImageFormat
}

var ErrNotFound = errors.New("image not found")

func ParseFormat(s string) (ImageFormat, bool) {
	f := ImageFormat(strings.TrimPrefix(strings.ToLower(s), "."))
	switch f {
	case FormatJPG, "jpeg":
		return FormatJPG, true
	case FormatPNG, FormatWEBP, FormatGIF:
		return f, true
	}
	return "", false
}
