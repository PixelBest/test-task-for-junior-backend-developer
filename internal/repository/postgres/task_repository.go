package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, created_at, updated_at, periodicity_type, periodicity, periodicity_dates)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, title, description, status, created_at, updated_at, periodicity_type, periodicity, periodicity_dates;
	`

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt, task.PeriodicityType, task.Periodicity, task.PeriodicityDates)
	created, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *Repository) CreatePeriodicTask(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, created_at, updated_at, periodicity_type, periodicity, periodicity_dates, periodicity_closed, parent_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, title, description, status, created_at, updated_at, periodicity_type, periodicity, periodicity_dates, periodicity_closed, parent_id;
	`

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt, task.PeriodicityType, task.Periodicity, task.PeriodicityDates, task.PeriodicityClosed, task.ParentId)
	created, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at, periodicity_type, periodicity, periodicity_dates
		FROM tasks
		WHERE id = $1 and is_deleted = false;
	`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) SetStatus(ctx context.Context, id int64, status taskdomain.Status) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND is_deleted = false
		RETURNING id, title, description, status, created_at, updated_at, periodicity_type, periodicity, periodicity_dates;
	`

	row := r.pool.QueryRow(ctx, query, status, id)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}

	return updated, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title = $1,
			description = $2,
			updated_at = $3,
			periodicity_type = $4,
			periodicity = $5,
			periodicity_dates = $6
		WHERE id = $7
		RETURNING id, title, description, status, created_at, updated_at, periodicity_type, periodicity, periodicity_dates;
	`

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.UpdatedAt, task.PeriodicityType, task.Periodicity, task.PeriodicityDates, task.ID)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `
		UPDATE
			tasks
		SET
			is_deleted = true
		WHERE
		    id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) ClosePeriodicTask(ctx context.Context, id int64) error {
	const query = `
		UPDATE
			tasks
		SET
			periodicity_closed = true
		WHERE
		    id = $1
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at, periodicity_type, periodicity, periodicity_dates
		FROM tasks
		WHERE is_deleted = false
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *Repository) GetTasksForUpdate(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, created_at, updated_at, periodicity_type, periodicity, periodicity_dates, periodicity_closed, parent_id
		FROM tasks
		WHERE is_deleted = false AND periodicity_closed = false AND periodicity_type is not null
		ORDER BY id DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task            taskdomain.Task
		status          string
		periodicityType *string
	)

	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&task.CreatedAt,
		&task.UpdatedAt,
		&periodicityType,
		&task.Periodicity,
		&task.PeriodicityDates,
		&task.PeriodicityClosed,
		&task.ParentId,
	); err != nil {
		return nil, err
	}
	task.Status = taskdomain.Status(status)
	if periodicityType != nil {
		task.PeriodicityType = taskdomain.PeriodicityType(*periodicityType)
	}

	return &task, nil
}
