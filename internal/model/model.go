package model

import (
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

type Image struct {
	ID           uuid.UUID   `json:"id"`
	Filename     string      `json:"filename"`
	Status       ImageStatus `json:"status"`
	OriginalURL  string      `json:"original_url,omitempty"`
	ProcessedURL string      `json:"processed_url,omitempty"`
	Created      time.Time   `json:"created"`
	Updated      time.Time   `json:"updated"`
}

type ImageTask struct {
	ID       uuid.UUID `json:"id"`
	Filename string    `json:"filename"`
}

var ErrNotFound = errors.New("image not found")
