package api

import (
	"context"
	"io"
	"net/http"

	"imgflow/internal/model"
	"imgflow/internal/service"

	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Service interface {
	UploadImage(ctx context.Context, params service.UploadImageOptions) (uuid.UUID, error)
	Image(ctx context.Context, id uuid.UUID) (model.Image, error)
	DeleteImage(ctx context.Context, id uuid.UUID) error
	File(ctx context.Context, name string) (io.ReadCloser, error)
}

type API struct {
	*echo.Echo
	service Service
}

func New(service Service) *API {
	a := &API{
		Echo:    echo.New(),
		service: service,
	}

	a.Static("/", "web")

	a.POST("/upload", a.upload)
	a.GET("/image/:id", a.image)
	a.DELETE("/image/:id", a.delete)
	a.GET("/images/:name", a.serveFile)
	
	return a
}

func (a *API) upload(c echo.Context) error {
	file, err := c.FormFile("image")
	if err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "image file is required"})
	}

	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "failed to open file"})
	}
	defer src.Close()

	id, err := a.service.UploadImage(c.Request().Context(), service.UploadImageOptions{
		Filename:    file.Filename,
		Content:     src,
		Size:        file.Size,
		ContentType: file.Header.Get("Content-Type"),
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

	return c.JSON(http.StatusAccepted, echo.Map{"id": id.String()})
}

func (a *API) image(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, "invalid uuid")
	}

	task, err := a.service.Image(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "image not found"})
		}

		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, a.taskToResponse(task))
}

func (a *API) delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid uuid")
	}

	err = a.service.DeleteImage(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "image not found"})
		}

		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "internal server error"})
	}

	return c.NoContent(http.StatusNoContent)
}

func (a *API) serveFile(c echo.Context) error {
	name := c.Param("name")

	// Запрашиваем поток байтов у сервиса
	reader, err := a.service.File(c.Request().Context(), name)
	if err != nil {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "file not found"})
	}
	defer func() { _ = reader.Close() }()

	// Стримим данные напрямую в ответ.
	// Content-Type можно определять динамически, но для начала хватит image/jpeg
	return c.Stream(http.StatusOK, "image/jpeg", reader)
}

type taskResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	OriginalURL  string `json:"original_url,omitempty"`
	ProcessedURL string `json:"processed_url,omitempty"`
}

func (a *API) taskToResponse(t model.Image) taskResponse {
	return taskResponse{
		ID:           t.ID.String(),
		Status:       string(t.Status),
		OriginalURL:  t.OriginalURL,
		ProcessedURL: t.ProcessedURL,
	}
}
