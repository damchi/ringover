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

var _ ports.TaskService = (*TaskService)(nil)

func (s *TaskService) ListRootTasks(ctx context.Context) ([]domain.Task, error) {
	return s.taskRepository.ListRootTasks(ctx)
}

func (s *TaskService) ListRootSubtasks(ctx context.Context, taskID uint64) ([]domain.Task, error) {
	return s.taskRepository.ListRootSubTasks(ctx, taskID)
}

func (s *TaskService) CreateTask(ctx context.Context, input domain.CreateTaskInput) (domain.Task, error) {
	return s.taskRepository.CreateTask(ctx, input)
}
