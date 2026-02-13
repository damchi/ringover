package handlers

import (
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	if !hasTaskUpdateFields(req) {
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	var title *string
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
	if req.Status != nil {
		value := domain.TaskStatus(*req.Status)
		status = &value
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

	task, err := h.taskService.UpdateTask(c.Request.Context(), taskID, domain.UpdateTaskInput{
		Title:        title,
		Description:  req.Description,
		Status:       status,
		Priority:     req.Priority,
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

		zap.L().Error("failed to update task", zap.Uint64("task_id", taskID), zap.Error(err))
		c.JSON(
			http.StatusInternalServerError,
			apierrors.CreateError(http.StatusInternalServerError, apierrors.MsgFailUpdateTask, lang),
		)
		return
	}

	c.JSON(http.StatusOK, mapper.ToTaskItem(task))
}

func hasTaskUpdateFields(req dto.UpdateTaskRequest) bool {
	return req.Title != nil ||
		req.Description != nil ||
		req.Status != nil ||
		req.Priority != nil ||
		req.DueDate != nil ||
		req.ParentTaskID != nil ||
		req.CategoryID != nil
}
