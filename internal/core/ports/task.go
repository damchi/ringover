package ports

import (
	"context"

	"ringover/internal/core/domain"
)

type TaskRepository interface {
	ListRootTasks(ctx context.Context) ([]domain.Task, error)
	ListRootSubTasks(ctx context.Context, taskID uint64) ([]domain.Task, error)
}

type TaskService interface {
	ListRootTasks(ctx context.Context) ([]domain.Task, error)
	ListRootSubtasks(ctx context.Context, taskID uint64) ([]domain.Task, error)
}
