package domain

import "time"

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

type Task struct {
	ID          uint64
	Title       string
	Description *string
	Status      TaskStatus
	Priority    int
	DueDate     *time.Time
	CompletedAt *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Category    *Category
	Subtasks    []Task
}

type CreateTaskInput struct {
	Title        string
	Description  *string
	Status       TaskStatus
	Priority     int
	DueDate      *time.Time
	ParentTaskID *uint64
	CategoryID   *uint64
}
