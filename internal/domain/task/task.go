package task

import "time"

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type PeriodicityType string

const (
	PeriodicityDaily         PeriodicityType = "daily"
	PeriodicityMonthly       PeriodicityType = "monthly"
	PeriodicitySpecificDates PeriodicityType = "specific_dates"
	PeriodicityParity        PeriodicityType = "parity"
)

type Task struct {
	ID                int64           `json:"id"`
	Title             string          `json:"title"`
	Description       string          `json:"description"`
	Status            Status          `json:"status"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
	PeriodicityType   PeriodicityType `json:"periodicity_type"`
	Periodicity       *int32          `json:"periodicity"`
	PeriodicityDates  []time.Time     `json:"periodicity_dates"`
	PeriodicityClosed bool            `json:"periodicity_closed"`
	ParentId          *int64          `json:"parent_id"`
}
