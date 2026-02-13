package db

import (
	"context"
	"database/sql"
	"errors"
	"ringover/internal/core/ports"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
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

const listRootSubTasksQuery = `
SELECT
  t.*,
  c.name AS category_name
FROM tasks t
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.parent_task_id = ?
ORDER BY t.id;
`

const taskExistsQuery = `
SELECT id
FROM tasks
WHERE id = ?
LIMIT 1;
`

const createTaskQuery = `
INSERT INTO tasks (
  title,
  description,
  status,
  priority,
  due_date,
  parent_task_id,
  category_id
)
VALUES (?, ?, ?, ?, ?, ?, ?);
`

const getTaskByIDQuery = `
SELECT
  t.*,
  c.name AS category_name
FROM tasks t
LEFT JOIN categories c ON c.id = t.category_id
WHERE t.id = ?
LIMIT 1;
`

const (
	mysqlSQLStateIntegrityConstraintViolation = "23000"
	mysqlChildRowFKMessage                    = "cannot add or update a child row"
	parentTaskFKConstraintName                = "fk_task_parent"
)

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

func (r *TaskRepository) ListRootSubTasks(ctx context.Context, taskID uint64) ([]domain.Task, error) {
	exists, err := r.taskExists(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrTaskNotFound
	}

	return r.listSubtasksTree(ctx, taskID)
}

func (r *TaskRepository) CreateTask(ctx context.Context, input domain.CreateTaskInput) (domain.Task, error) {
	if input.ParentTaskID != nil {
		exists, err := r.taskExists(ctx, *input.ParentTaskID)
		if err != nil {
			return domain.Task{}, err
		}
		if !exists {
			return domain.Task{}, domain.ErrTaskNotFound
		}
	}

	result, err := r.db.ExecContext(
		ctx,
		createTaskQuery,
		input.Title,
		input.Description,
		string(input.Status),
		input.Priority,
		input.DueDate,
		input.ParentTaskID,
		input.CategoryID,
	)
	if err != nil {
		// Handle race condition where parent was deleted between existence check and insert.
		if isParentForeignKeyError(err) {
			return domain.Task{}, domain.ErrTaskNotFound
		}
		return domain.Task{}, err
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		return domain.Task{}, err
	}

	return r.getTaskByID(ctx, uint64(insertedID))
}

func (r *TaskRepository) UpdateTask(ctx context.Context, taskID uint64, input domain.UpdateTaskInput) (domain.Task, error) {
	exists, err := r.taskExists(ctx, taskID)
	if err != nil {
		return domain.Task{}, err
	}
	if !exists {
		return domain.Task{}, domain.ErrTaskNotFound
	}

	if input.ParentTaskID != nil {
		exists, err := r.taskExists(ctx, *input.ParentTaskID)
		if err != nil {
			return domain.Task{}, err
		}
		if !exists {
			return domain.Task{}, domain.ErrTaskNotFound
		}
	}

	setClauses := make([]string, 0, 7)
	args := make([]any, 0, 8)

	if input.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *input.Title)
	}
	if input.Description != nil {
		setClauses = append(setClauses, "description = ?")
		args = append(args, *input.Description)
	}
	if input.Status != nil {
		setClauses = append(setClauses, "status = ?")
		args = append(args, string(*input.Status))
	}
	if input.Priority != nil {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, *input.Priority)
	}
	if input.DueDate != nil {
		setClauses = append(setClauses, "due_date = ?")
		args = append(args, *input.DueDate)
	}
	if input.ParentTaskID != nil {
		setClauses = append(setClauses, "parent_task_id = ?")
		args = append(args, *input.ParentTaskID)
	}
	if input.CategoryID != nil {
		setClauses = append(setClauses, "category_id = ?")
		args = append(args, *input.CategoryID)
	}

	if len(setClauses) == 0 {
		return r.getTaskByID(ctx, taskID)
	}

	updateQuery := "UPDATE tasks SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	args = append(args, taskID)

	if _, err := r.db.ExecContext(ctx, updateQuery, args...); err != nil {
		// Handle race condition where parent was deleted between existence check and update.
		if isParentForeignKeyError(err) {
			return domain.Task{}, domain.ErrTaskNotFound
		}
		return domain.Task{}, err
	}

	return r.getTaskByID(ctx, taskID)
}

func (r *TaskRepository) taskExists(ctx context.Context, taskID uint64) (bool, error) {
	var id uint64
	if err := r.db.GetContext(ctx, &id, taskExistsQuery, taskID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *TaskRepository) listSubtasksTree(ctx context.Context, parentTaskID uint64) ([]domain.Task, error) {
	var rows []taskRow
	if err := r.db.SelectContext(ctx, &rows, listRootSubTasksQuery, parentTaskID); err != nil {
		return nil, err
	}

	tasks := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		task := mapTaskRowToDomainTask(row)
		subtasks, err := r.listSubtasksTree(ctx, task.ID)
		if err != nil {
			return nil, err
		}
		task.Subtasks = subtasks
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (r *TaskRepository) getTaskByID(ctx context.Context, taskID uint64) (domain.Task, error) {
	var row taskRow
	if err := r.db.GetContext(ctx, &row, getTaskByIDQuery, taskID); err != nil {
		if err == sql.ErrNoRows {
			return domain.Task{}, domain.ErrTaskNotFound
		}
		return domain.Task{}, err
	}

	return mapTaskRowToDomainTask(row), nil
}

func isParentForeignKeyError(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	if string(mysqlErr.SQLState[:]) != mysqlSQLStateIntegrityConstraintViolation {
		return false
	}

	message := strings.ToLower(mysqlErr.Message)
	if !strings.Contains(message, mysqlChildRowFKMessage) {
		return false
	}

	return strings.Contains(message, parentTaskFKConstraintName)
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
