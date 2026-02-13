package db

import (
	"context"
	"database/sql"
	"ringover/internal/core/ports"
	"time"

	"github.com/jmoiron/sqlx"

	"ringover/internal/core/domain"
)

const listRootTasksQuery = `
SELECT
  t.*,
  c.name AS category_name
FROM tasks t
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.parent_task_id IS NULL
ORDER BY t.id;
`

type TaskRepository struct {
	db *sqlx.DB
}

type taskRow struct {
	ID           uint64         `db:"id"`
	ParentTaskID sql.NullInt64  `db:"parent_task_id"`
	Title        string         `db:"title"`
	Description  sql.NullString `db:"description"`
	Status       string         `db:"status"`
	Priority     int            `db:"priority"`
	DueDate      sql.NullTime   `db:"due_date"`
	CompletedAt  sql.NullTime   `db:"completed_at"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
	CategoryID   sql.NullInt64  `db:"category_id"`
	CategoryName sql.NullString `db:"category_name"`
}

var _ ports.TaskRepository = (*TaskRepository)(nil)

func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) ListRootTasks(ctx context.Context) ([]domain.Task, error) {
	var rows []taskRow
	if err := r.db.SelectContext(ctx, &rows, listRootTasksQuery); err != nil {
		return nil, err
	}

	tasks := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		tasks = append(tasks, mapTaskRowToDomainTask(row))
	}

	return tasks, nil
}

func mapTaskRowToDomainTask(row taskRow) domain.Task {
	task := domain.Task{
		ID:        row.ID,
		Title:     row.Title,
		Status:    domain.TaskStatus(row.Status),
		Priority:  row.Priority,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}

	if row.Description.Valid {
		value := row.Description.String
		task.Description = &value
	}

	if row.DueDate.Valid {
		value := row.DueDate.Time
		task.DueDate = &value
	}

	if row.CompletedAt.Valid {
		value := row.CompletedAt.Time
		task.CompletedAt = &value
	}

	if row.CategoryID.Valid && row.CategoryName.Valid {
		task.Category = &domain.Category{
			ID:   uint64(row.CategoryID.Int64),
			Name: row.CategoryName.String,
		}
	}

	return task
}
