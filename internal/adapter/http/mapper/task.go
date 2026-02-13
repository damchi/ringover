package mapper

import (
	"ringover/internal/adapter/http/dto"
	"ringover/internal/core/domain"
	"time"
)

func ToTaskItems(tasks []domain.Task) []dto.TaskItem {
	items := make([]dto.TaskItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, ToTaskItem(task))
	}
	return items
}

func ToTaskItem(task domain.Task) dto.TaskItem {
	item := dto.TaskItem{
		ID:        task.ID,
		Title:     task.Title,
		Status:    string(task.Status),
		Priority:  task.Priority,
		CreatedAt: task.CreatedAt.Format(time.RFC3339),
		UpdatedAt: task.UpdatedAt.Format(time.RFC3339),
	}

	if task.Description != nil {
		value := *task.Description
		item.Description = &value
	}

	if task.DueDate != nil {
		value := task.DueDate.Format("2006-01-02")
		item.DueDate = &value
	}

	if task.CompletedAt != nil {
		value := task.CompletedAt.Format("2006-01-02")
		item.CompletedAt = &value
	}

	if task.Category != nil {
		item.Category = &dto.Category{
			ID:   task.Category.ID,
			Name: task.Category.Name,
		}
	}

	if len(task.Subtasks) > 0 {
		item.Subtasks = ToTaskItems(task.Subtasks)
	}

	return item
}
