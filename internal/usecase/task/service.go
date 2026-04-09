package task

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		Title:            normalized.Title,
		Description:      normalized.Description,
		Status:           taskdomain.StatusNew,
		PeriodicityType:  normalized.PeriodicityType,
		Periodicity:      normalized.Periodicity,
		PeriodicityDates: normalized.PeriodicityDates,
	}
	now := s.now()
	model.CreatedAt = now
	model.UpdatedAt = now

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) SetInProgress(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.repo.SetStatus(ctx, id, taskdomain.StatusInProgress)
}

func (s *Service) SetDone(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.repo.SetStatus(ctx, id, taskdomain.StatusDone)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Task{
		ID:               id,
		Title:            normalized.Title,
		Description:      normalized.Description,
		UpdatedAt:        s.now(),
		PeriodicityType:  normalized.PeriodicityType,
		Periodicity:      normalized.Periodicity,
		PeriodicityDates: normalized.PeriodicityDates,
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) ClosePeriodicTask(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.ClosePeriodicTask(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetTasksForUpdate(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.GetTasksForUpdate(ctx)
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	return input, nil
}

func (s *Service) CreatePeriodicTask(ctx context.Context, task *taskdomain.Task) error {
	if task.PeriodicityClosed {
		return fmt.Errorf("task is already closed")
	}

	var shouldCreate bool
	var closeNewPeriodicTask bool
	now := s.now()

	switch task.PeriodicityType {
	case taskdomain.PeriodicityDaily:
		if task.Periodicity == nil {
			return fmt.Errorf("periodicity is required for daily periodic task")
		}
		daysToAdd := int(*task.Periodicity)
		if daysToAdd <= 0 {
			daysToAdd = 1
		}

		nextCreationDate := task.CreatedAt.AddDate(0, 0, daysToAdd)
		if now.After(nextCreationDate) || now.Equal(nextCreationDate) {
			shouldCreate = true
		}

	case taskdomain.PeriodicityMonthly:
		if task.Periodicity == nil {
			return fmt.Errorf("periodicity is required for daily periodic task")
		}
		targetDay := int(*task.Periodicity)
		if targetDay <= 0 || targetDay > 31 {
			targetDay = 1
		}
		currentDay := now.Day()

		if currentDay >= targetDay {
			lastCreated := task.CreatedAt
			if lastCreated.Month() != now.Month() || lastCreated.Year() != now.Year() {
				shouldCreate = true
			}
		}

	case taskdomain.PeriodicitySpecificDates:
		for _, date := range task.PeriodicityDates {
			if isSameDay(date, now) {
				shouldCreate = true
				break
			}
		}
		if shouldCreate && isLastDate(task.PeriodicityDates, now) {
			closeNewPeriodicTask = true
		}

	case taskdomain.PeriodicityParity:
		if task.Periodicity == nil {
			return fmt.Errorf("periodicity is required for daily periodic task")
		}
		targetParity := int(*task.Periodicity)
		currentDay := now.Day()
		isEven := currentDay%2 == 0

		if (targetParity == 0 && isEven) || (targetParity == 1 && !isEven) {
			lastCreated := task.UpdatedAt
			if !isSameDay(lastCreated, now) {
				shouldCreate = true
			}
		}

	default:
		return fmt.Errorf("unknown periodicity type: %s", task.PeriodicityType)
	}

	if !shouldCreate {
		return nil
	}

	if err := s.repo.ClosePeriodicTask(ctx, task.ID); err != nil {
		return fmt.Errorf("failed to close task: %w", err)
	}

	newTask := &taskdomain.Task{
		Title:             task.Title,
		Description:       task.Description,
		Status:            "new",
		PeriodicityType:   task.PeriodicityType,
		Periodicity:       task.Periodicity,
		PeriodicityDates:  task.PeriodicityDates,
		PeriodicityClosed: false,
		ParentId:          &task.ID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if closeNewPeriodicTask {
		newTask.PeriodicityClosed = true
	}

	createdTask, err := s.repo.CreatePeriodicTask(ctx, newTask)
	if err != nil {
		return fmt.Errorf("failed to create new task: %w", err)
	}

	logger := slog.Default()
	logger.Info("periodic task created",
		"parent_task_id", task.ID,
		"new_task_id", createdTask.ID,
		"periodicity_type", task.PeriodicityType,
		"periodicity", task.Periodicity,
		"created_at", createdTask.CreatedAt)

	return nil
}

func isSameDay(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func isLastDate(dates []time.Time, currentDate time.Time) bool {
	if len(dates) == 0 {
		return false
	}

	maxDate := dates[0]
	for _, date := range dates {
		if date.After(maxDate) {
			maxDate = date
		}
	}

	return isSameDay(currentDate, maxDate)
}
