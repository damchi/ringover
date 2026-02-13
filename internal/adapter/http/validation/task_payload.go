package validation

import (
	"bytes"
	"encoding/json"
	"errors"
	"ringover/internal/adapter/http/dto"
	"ringover/internal/core/domain"
	"strings"
	"time"
)

var ErrInvalidTaskPayload = errors.New("invalid task payload")

func BuildCreateTaskInput(req dto.CreateTaskRequest, raw map[string]json.RawMessage) (domain.CreateTaskInput, error) {
	if hasJSONField(raw, "status") && req.Status == nil {
		return domain.CreateTaskInput{}, ErrInvalidTaskPayload
	}
	if hasJSONField(raw, "priority") && req.Priority == nil {
		return domain.CreateTaskInput{}, ErrInvalidTaskPayload
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return domain.CreateTaskInput{}, ErrInvalidTaskPayload
	}

	status := domain.TaskStatusTodo
	if req.Status != nil {
		status = domain.TaskStatus(*req.Status)
	}

	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}

	var dueDate *time.Time
	if req.DueDate != nil {
		parsedDueDate, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			return domain.CreateTaskInput{}, ErrInvalidTaskPayload
		}
		dueDate = &parsedDueDate
	}

	return domain.CreateTaskInput{
		Title:        title,
		Description:  req.Description,
		Status:       status,
		Priority:     priority,
		DueDate:      dueDate,
		ParentTaskID: req.ParentTaskID,
		CategoryID:   req.CategoryID,
	}, nil
}

func BuildUpdateTaskInput(req dto.UpdateTaskRequest, raw map[string]json.RawMessage) (domain.UpdateTaskInput, error) {
	if !hasTaskUpdateFields(raw) {
		return domain.UpdateTaskInput{}, ErrInvalidTaskPayload
	}

	var title *string
	if hasJSONField(raw, "title") && req.Title == nil {
		return domain.UpdateTaskInput{}, ErrInvalidTaskPayload
	}
	if req.Title != nil {
		value := strings.TrimSpace(*req.Title)
		if value == "" {
			return domain.UpdateTaskInput{}, ErrInvalidTaskPayload
		}
		title = &value
	}

	var status *domain.TaskStatus
	if hasJSONField(raw, "status") && req.Status == nil {
		return domain.UpdateTaskInput{}, ErrInvalidTaskPayload
	}
	if req.Status != nil {
		value := domain.TaskStatus(*req.Status)
		status = &value
	}

	if hasJSONField(raw, "priority") && req.Priority == nil {
		return domain.UpdateTaskInput{}, ErrInvalidTaskPayload
	}

	descriptionSet := hasJSONField(raw, "description")
	if descriptionSet && !isJSONNull(raw["description"]) && req.Description == nil {
		return domain.UpdateTaskInput{}, ErrInvalidTaskPayload
	}

	var dueDate *time.Time
	dueDateSet := hasJSONField(raw, "due_date")
	if dueDateSet && !isJSONNull(raw["due_date"]) {
		if req.DueDate == nil {
			return domain.UpdateTaskInput{}, ErrInvalidTaskPayload
		}
		parsedDueDate, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			return domain.UpdateTaskInput{}, ErrInvalidTaskPayload
		}
		dueDate = &parsedDueDate
	}

	parentTaskIDSet := hasJSONField(raw, "parent_task_id")
	if parentTaskIDSet && !isJSONNull(raw["parent_task_id"]) && req.ParentTaskID == nil {
		return domain.UpdateTaskInput{}, ErrInvalidTaskPayload
	}

	categoryIDSet := hasJSONField(raw, "category_id")
	if categoryIDSet && !isJSONNull(raw["category_id"]) && req.CategoryID == nil {
		return domain.UpdateTaskInput{}, ErrInvalidTaskPayload
	}

	return domain.UpdateTaskInput{
		Title:           title,
		Description:     req.Description,
		DescriptionSet:  descriptionSet,
		Status:          status,
		Priority:        req.Priority,
		DueDate:         dueDate,
		DueDateSet:      dueDateSet,
		ParentTaskID:    req.ParentTaskID,
		ParentTaskIDSet: parentTaskIDSet,
		CategoryID:      req.CategoryID,
		CategoryIDSet:   categoryIDSet,
	}, nil
}

func hasTaskUpdateFields(raw map[string]json.RawMessage) bool {
	return hasJSONField(raw, "title") ||
		hasJSONField(raw, "description") ||
		hasJSONField(raw, "status") ||
		hasJSONField(raw, "priority") ||
		hasJSONField(raw, "due_date") ||
		hasJSONField(raw, "parent_task_id") ||
		hasJSONField(raw, "category_id")
}

func hasJSONField(raw map[string]json.RawMessage, field string) bool {
	_, ok := raw[field]
	return ok
}

func isJSONNull(value json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}
