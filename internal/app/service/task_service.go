package service

import (
	"context"

	"ringover/internal/core/domain"
	"ringover/internal/core/ports"
)

type TaskService struct {
	taskRepository ports.TaskRepository
}

func NewTaskService(taskRepository ports.TaskRepository) *TaskService {
	return &TaskService{taskRepository: taskRepository}
}

func (s *TaskService) ListRootTasks(ctx context.Context) ([]domain.Task, error) {
	return s.taskRepository.ListRootTasks(ctx)
}

var _ ports.TaskService = (*TaskService)(nil)
