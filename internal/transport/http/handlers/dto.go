package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title            string                     `json:"title"`
	Description      string                     `json:"description"`
	PeriodicityType  taskdomain.PeriodicityType `json:"periodicity_type"`
	Periodicity      *int32                     `json:"periodicity"`
	PeriodicityDates []time.Time                `json:"periodicity_dates"`
}

type taskDTO struct {
	ID                int64                      `json:"id"`
	Title             string                     `json:"title"`
	Description       string                     `json:"description"`
	Status            taskdomain.Status          `json:"status"`
	CreatedAt         time.Time                  `json:"created_at"`
	UpdatedAt         time.Time                  `json:"updated_at"`
	PeriodicityType   taskdomain.PeriodicityType `json:"periodicity_type"`
	Periodicity       *int32                     `json:"periodicity"`
	PeriodicityDates  []time.Time                `json:"periodicity_dates"`
	PeriodicityClosed bool                       `json:"periodicity_closed"`
	ParentId          int64                      `json:"parent_id"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	return taskDTO{
		ID:               task.ID,
		Title:            task.Title,
		Description:      task.Description,
		Status:           task.Status,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
		PeriodicityType:  task.PeriodicityType,
		Periodicity:      task.Periodicity,
		PeriodicityDates: task.PeriodicityDates,
	}
}
