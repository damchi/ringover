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

const categoryExistsQuery = `
SELECT id
FROM categories
WHERE id = ?
LIMIT 1;
`

const taskParentIDQuery = `
SELECT parent_task_id
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

const deleteTaskByIDQuery = `
DELETE FROM tasks
WHERE id = ?;
`

const (
	mysqlErrorNoReferencedRow = uint16(1452)
	parentTaskFKConstraint    = "fk_task_parent"
	categoryFKConstraint      = "fk_task_category"
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
	if input.CategoryID != nil {
		exists, err := r.categoryExists(ctx, *input.CategoryID)
		if err != nil {
			return domain.Task{}, err
		}
		if !exists {
			return domain.Task{}, domain.ErrCategoryNotFound
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
		if isForeignKeyConstraintError(err, parentTaskFKConstraint) {
			return domain.Task{}, domain.ErrTaskNotFound
		}
		// Handle race condition where category was deleted between existence check and insert.
		if isForeignKeyConstraintError(err, categoryFKConstraint) {
			return domain.Task{}, domain.ErrCategoryNotFound
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

	if input.ParentTaskIDSet && input.ParentTaskID != nil {
		if *input.ParentTaskID == taskID {
			return domain.Task{}, domain.ErrTaskHierarchyCycle
		}

		exists, err := r.taskExists(ctx, *input.ParentTaskID)
		if err != nil {
			return domain.Task{}, err
		}
		if !exists {
			return domain.Task{}, domain.ErrTaskNotFound
		}

		wouldCreateCycle, err := r.wouldCreateTaskHierarchyCycle(ctx, taskID, *input.ParentTaskID)
		if err != nil {
			return domain.Task{}, err
		}
		if wouldCreateCycle {
			return domain.Task{}, domain.ErrTaskHierarchyCycle
		}
	}

	if input.CategoryIDSet && input.CategoryID != nil {
		exists, err := r.categoryExists(ctx, *input.CategoryID)
		if err != nil {
			return domain.Task{}, err
		}
		if !exists {
			return domain.Task{}, domain.ErrCategoryNotFound
		}
	}

	setClauses := make([]string, 0, 7)
	args := make([]any, 0, 8)

	if input.Title != nil {
		setClauses = append(setClauses, "title = ?")
		args = append(args, *input.Title)
	}
	if input.DescriptionSet {
		setClauses = append(setClauses, "description = ?")
		if input.Description == nil {
			args = append(args, nil)
		} else {
			args = append(args, *input.Description)
		}
	}
	if input.Status != nil {
		setClauses = append(setClauses, "status = ?")
		args = append(args, string(*input.Status))
	}
	if input.Priority != nil {
		setClauses = append(setClauses, "priority = ?")
		args = append(args, *input.Priority)
	}
	if input.DueDateSet {
		setClauses = append(setClauses, "due_date = ?")
		if input.DueDate == nil {
			args = append(args, nil)
		} else {
			args = append(args, *input.DueDate)
		}
	}
	if input.ParentTaskIDSet {
		setClauses = append(setClauses, "parent_task_id = ?")
		if input.ParentTaskID == nil {
			args = append(args, nil)
		} else {
			args = append(args, *input.ParentTaskID)
		}
	}
	if input.CategoryIDSet {
		setClauses = append(setClauses, "category_id = ?")
		if input.CategoryID == nil {
			args = append(args, nil)
		} else {
			args = append(args, *input.CategoryID)
		}
	}

	if len(setClauses) == 0 {
		return r.getTaskByID(ctx, taskID)
	}

	updateQuery := "UPDATE tasks SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	args = append(args, taskID)

	if _, err := r.db.ExecContext(ctx, updateQuery, args...); err != nil {
		// Handle race condition where parent was deleted between existence check and update.
		if isForeignKeyConstraintError(err, parentTaskFKConstraint) {
			return domain.Task{}, domain.ErrTaskNotFound
		}
		// Handle race condition where category was deleted between existence check and update.
		if isForeignKeyConstraintError(err, categoryFKConstraint) {
			return domain.Task{}, domain.ErrCategoryNotFound
		}
		return domain.Task{}, err
	}

	return r.getTaskByID(ctx, taskID)
}

func (r *TaskRepository) DeleteTask(ctx context.Context, taskID uint64) error {
	result, err := r.db.ExecContext(ctx, deleteTaskByIDQuery, taskID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrTaskNotFound
	}

	return nil
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

func (r *TaskRepository) categoryExists(ctx context.Context, categoryID uint64) (bool, error) {
	var id uint64
	if err := r.db.GetContext(ctx, &id, categoryExistsQuery, categoryID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *TaskRepository) wouldCreateTaskHierarchyCycle(ctx context.Context, taskID uint64, newParentID uint64) (bool, error) {
	visited := map[uint64]struct{}{
		taskID: {},
	}

	current := newParentID
	for {
		if _, seen := visited[current]; seen {
			return true, nil
		}
		visited[current] = struct{}{}

		parentID, hasParent, err := r.getParentTaskID(ctx, current)
		if err != nil {
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, err
		}
		if !hasParent {
			return false, nil
		}

		current = parentID
	}
}

func (r *TaskRepository) getParentTaskID(ctx context.Context, taskID uint64) (uint64, bool, error) {
	var parentID sql.NullInt64
	if err := r.db.GetContext(ctx, &parentID, taskParentIDQuery, taskID); err != nil {
		return 0, false, err
	}
	if !parentID.Valid {
		return 0, false, nil
	}

	return uint64(parentID.Int64), true, nil
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

func isForeignKeyConstraintError(err error, constraintName string) bool {
	var mysqlErr *mysqlDriver.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	if mysqlErr.Number != mysqlErrorNoReferencedRow {
		return false
	}

	message := strings.ToLower(mysqlErr.Message)
	return strings.Contains(message, strings.ToLower(constraintName))
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
