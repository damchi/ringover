package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"ringover/internal/adapter/http/dto"
	"ringover/internal/adapter/http/mapper"
	"ringover/internal/adapter/http/middleware"
	"ringover/internal/core/domain"
	"ringover/internal/core/ports"
	"ringover/pkg/apierrors"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/zap"
)

type TaskHandler struct {
	taskService ports.TaskService
}

func NewTaskHandler(taskService ports.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

func (h *TaskHandler) ListRootTasks(c *gin.Context) {
	lang := middleware.GetLang(c)
	tasks, err := h.taskService.ListRootTasks(c.Request.Context())
	if err != nil {
		zap.L().Error("failed to list root tasks", zap.Error(err))
		c.JSON(
			http.StatusInternalServerError,
			apierrors.CreateError(http.StatusInternalServerError, apierrors.MsgFailListTask, lang),
		)
		return
	}

	c.JSON(http.StatusOK, mapper.ToTaskItems(tasks))
}

func (h *TaskHandler) ListRootSubTasks(c *gin.Context) {
	lang := middleware.GetLang(c)

	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || taskID == 0 {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskID, lang),
		)
		return
	}

	subtasks, err := h.taskService.ListRootSubtasks(c.Request.Context(), taskID)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			c.JSON(
				http.StatusNotFound,
				apierrors.CreateError(http.StatusNotFound, apierrors.MsgTaskNotFound, lang),
			)
			return
		}

		zap.L().Error("failed to list subtasks hierarchy", zap.Uint64("task_id", taskID), zap.Error(err))
		c.JSON(
			http.StatusInternalServerError,
			apierrors.CreateError(http.StatusInternalServerError, apierrors.MsgFailListSubtasks, lang),
		)
		return
	}

	c.JSON(http.StatusOK, mapper.ToTaskItems(subtasks))
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	lang := middleware.GetLang(c)

	var req dto.CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
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
			c.JSON(
				http.StatusBadRequest,
				apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
			)
			return
		}
		dueDate = &parsedDueDate
	}

	task, err := h.taskService.CreateTask(c.Request.Context(), domain.CreateTaskInput{
		Title:        title,
		Description:  req.Description,
		Status:       status,
		Priority:     priority,
		DueDate:      dueDate,
		ParentTaskID: req.ParentTaskID,
		CategoryID:   req.CategoryID,
	})
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			c.JSON(
				http.StatusNotFound,
				apierrors.CreateError(http.StatusNotFound, apierrors.MsgTaskNotFound, lang),
			)
			return
		}
		if errors.Is(err, domain.ErrCategoryNotFound) {
			c.JSON(
				http.StatusNotFound,
				apierrors.CreateError(http.StatusNotFound, apierrors.MsgCategoryNotFound, lang),
			)
			return
		}
		if errors.Is(err, domain.ErrTaskHierarchyCycle) {
			c.JSON(
				http.StatusBadRequest,
				apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskHierarchy, lang),
			)
			return
		}

		zap.L().Error("failed to create task", zap.Error(err))
		c.JSON(
			http.StatusInternalServerError,
			apierrors.CreateError(http.StatusInternalServerError, apierrors.MsgFailCreateTask, lang),
		)
		return
	}

	c.JSON(http.StatusCreated, mapper.ToTaskItem(task))
}

func (h *TaskHandler) UpdateTask(c *gin.Context) {
	lang := middleware.GetLang(c)

	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || taskID == 0 {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskID, lang),
		)
		return
	}

	var req dto.UpdateTaskRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	var raw map[string]json.RawMessage
	if err := c.ShouldBindBodyWith(&raw, binding.JSON); err != nil {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	if !hasTaskUpdateFields(raw) {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	var title *string
	if hasJSONField(raw, "title") && req.Title == nil {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}
	if req.Title != nil {
		value := strings.TrimSpace(*req.Title)
		if value == "" {
			c.JSON(
				http.StatusBadRequest,
				apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
			)
			return
		}
		title = &value
	}

	var status *domain.TaskStatus
	if hasJSONField(raw, "status") && req.Status == nil {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}
	if req.Status != nil {
		value := domain.TaskStatus(*req.Status)
		status = &value
	}

	if hasJSONField(raw, "priority") && req.Priority == nil {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	descriptionSet := hasJSONField(raw, "description")
	if descriptionSet && !isJSONNull(raw["description"]) && req.Description == nil {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	var dueDate *time.Time
	dueDateSet := hasJSONField(raw, "due_date")
	if dueDateSet && !isJSONNull(raw["due_date"]) {
		if req.DueDate == nil {
			c.JSON(
				http.StatusBadRequest,
				apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
			)
			return
		}
		parsedDueDate, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
			)
			return
		}
		dueDate = &parsedDueDate
	}

	parentTaskIDSet := hasJSONField(raw, "parent_task_id")
	if parentTaskIDSet && !isJSONNull(raw["parent_task_id"]) && req.ParentTaskID == nil {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	categoryIDSet := hasJSONField(raw, "category_id")
	if categoryIDSet && !isJSONNull(raw["category_id"]) && req.CategoryID == nil {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	task, err := h.taskService.UpdateTask(c.Request.Context(), taskID, domain.UpdateTaskInput{
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
	})
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			c.JSON(
				http.StatusNotFound,
				apierrors.CreateError(http.StatusNotFound, apierrors.MsgTaskNotFound, lang),
			)
			return
		}
		if errors.Is(err, domain.ErrCategoryNotFound) {
			c.JSON(
				http.StatusNotFound,
				apierrors.CreateError(http.StatusNotFound, apierrors.MsgCategoryNotFound, lang),
			)
			return
		}
		if errors.Is(err, domain.ErrTaskHierarchyCycle) {
			c.JSON(
				http.StatusBadRequest,
				apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskHierarchy, lang),
			)
			return
		}

		zap.L().Error("failed to update task", zap.Uint64("task_id", taskID), zap.Error(err))
		c.JSON(
			http.StatusInternalServerError,
			apierrors.CreateError(http.StatusInternalServerError, apierrors.MsgFailUpdateTask, lang),
		)
		return
	}

	c.JSON(http.StatusOK, mapper.ToTaskItem(task))
}

func (h *TaskHandler) DeleteTask(c *gin.Context) {
	lang := middleware.GetLang(c)

	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || taskID == 0 {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskID, lang),
		)
		return
	}

	if err := h.taskService.DeleteTask(c.Request.Context(), taskID); err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			c.JSON(
				http.StatusNotFound,
				apierrors.CreateError(http.StatusNotFound, apierrors.MsgTaskNotFound, lang),
			)
			return
		}

		zap.L().Error("failed to delete task", zap.Uint64("task_id", taskID), zap.Error(err))
		c.JSON(
			http.StatusInternalServerError,
			apierrors.CreateError(http.StatusInternalServerError, apierrors.MsgFailDeleteTask, lang),
		)
		return
	}

	c.Status(http.StatusNoContent)
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
