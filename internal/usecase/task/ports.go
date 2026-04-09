package task

import (
	"context"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	SetStatus(ctx context.Context, id int64, status taskdomain.Status) (*taskdomain.Task, error)
	Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	ClosePeriodicTask(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
	GetTasksForUpdate(ctx context.Context) ([]taskdomain.Task, error)
	CreatePeriodicTask(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
}

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	SetInProgress(ctx context.Context, id int64) (*taskdomain.Task, error)
	SetDone(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	ClosePeriodicTask(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
	GetTasksForUpdate(ctx context.Context) ([]taskdomain.Task, error)
	CreatePeriodicTask(ctx context.Context, task *taskdomain.Task) error
}

type CreateInput struct {
	Title            string
	Description      string
	PeriodicityType  taskdomain.PeriodicityType
	Periodicity      *int32
	PeriodicityDates []time.Time
}

type UpdateInput struct {
	Title            string
	Description      string
	PeriodicityType  taskdomain.PeriodicityType
	Periodicity      *int32
	PeriodicityDates []time.Time
}
