package ports

import (
	"context"

	"ringover/internal/core/domain"
)

type TaskRepository interface {
	ListRootTasks(ctx context.Context) ([]domain.Task, error)
}

type TaskService interface {
	ListRootTasks(ctx context.Context) ([]domain.Task, error)
}
