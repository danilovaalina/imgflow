package repository

import (
	"context"
	"time"

	"imgflow/internal/model"

	sq "github.com/Masterminds/squirrel"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Metadata struct {
	pool    *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewMetadata(pool *pgxpool.Pool) *Metadata {
	return &Metadata{
		pool:    pool,
		builder: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

type CreateImageOptions struct {
	ID       uuid.UUID
	Filename string
	Format   model.ImageFormat
	Status   model.ImageStatus
	Created  time.Time
}

func (r *Metadata) CreateImage(ctx context.Context, opts CreateImageOptions) error {
	query := `insert into images (id, filename, format, status, created) values ($1, $2, $3, $4, $5)`
	_, err := r.pool.Exec(ctx, query, opts.ID, opts.Filename, opts.Format, opts.Status, opts.Created)
	return errors.WithStack(err)
}

type UpdateImageOptions struct {
	ID           uuid.UUID
	Status       model.ImageStatus
	OriginalURL  string
	ProcessedURL string
	Updated      time.Time
}

func (r *Metadata) UpdateStatus(ctx context.Context, opts UpdateImageOptions) error {
	b := r.builder.Update("images").
		Where(sq.Eq{"id": opts.ID}).
		Set("updated", opts.Updated)

	if opts.Status != "" {
		b = b.Set("status", opts.Status)
	}
	if opts.OriginalURL != "" {
		b = b.Set("original_url", opts.OriginalURL)
	}
	if opts.ProcessedURL != "" {
		b = b.Set("processed_url", opts.ProcessedURL)
	}

	query, args, err := b.ToSql()
	if err != nil {
		return errors.WithStack(err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (r *Metadata) Image(ctx context.Context, id uuid.UUID) (model.Image, error) {
	query := `
		select id, filename, format, status, original_url, processed_url, created 
		from images 
		where id = $1
	`
	rows, err := r.pool.Query(ctx, query, id)
	if err != nil {
		return model.Image{}, errors.WithStack(err)
	}
	defer rows.Close()

	row, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByNameLax[imageRow])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Image{}, model.ErrNotFound
		}
		return model.Image{}, errors.WithStack(err)
	}

	return r.imageModel(row), nil
}

func (r *Metadata) DeleteImage(ctx context.Context, id uuid.UUID) error {
	query := `delete from images where id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return errors.WithStack(err)
	}

	if result.RowsAffected() == 0 {
		return model.ErrNotFound
	}

	return nil
}

func (r *Metadata) imageModel(row imageRow) model.Image {
	return model.Image{
		ID:           row.ID,
		Filename:     row.Filename,
		Format:       model.ImageFormat(row.Format),
		Status:       model.ImageStatus(row.Status),
		OriginalURL:  row.OriginalURL,
		ProcessedURL: row.ProcessedURL,
		Created:      row.Created,
	}
}

type imageRow struct {
	ID           uuid.UUID `db:"id"`
	Filename     string    `db:"filename"`
	Format       string    `db:"format"`
	Status       string    `db:"status"`
	OriginalURL  string    `db:"original_url"`
	ProcessedURL string    `db:"processed_url"`
	Created      time.Time `db:"created"`
}
