package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"ringover/internal/adapter/http/dto"
	"ringover/internal/adapter/http/mapper"
	"ringover/internal/adapter/http/middleware"
	"ringover/internal/adapter/http/validation"
	"ringover/internal/core/domain"
	"ringover/internal/core/ports"
	"ringover/pkg/apierrors"
	"strconv"

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
		zap.L().Error("failed to parse root task id", zap.Error(err))
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
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		zap.L().Error("failed binding payload create task", zap.Error(err))
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	var raw map[string]json.RawMessage
	if err := c.ShouldBindBodyWith(&raw, binding.JSON); err != nil {
		zap.L().Error("failed binding payload create task", zap.Error(err))
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	input, err := validation.BuildCreateTaskInput(req, raw)
	if err != nil {
		zap.L().Error("failed build payload create task", zap.Error(err))
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	task, err := h.taskService.CreateTask(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			zap.L().Error("failed create task, parent task not found", zap.Error(err))
			c.JSON(
				http.StatusNotFound,
				apierrors.CreateError(http.StatusNotFound, apierrors.MsgTaskNotFound, lang),
			)
			return
		}
		if errors.Is(err, domain.ErrCategoryNotFound) {
			zap.L().Error("failed create task, category not found", zap.Error(err))
			c.JSON(
				http.StatusNotFound,
				apierrors.CreateError(http.StatusNotFound, apierrors.MsgCategoryNotFound, lang),
			)
			return
		}
		if errors.Is(err, domain.ErrTaskHierarchyCycle) {
			zap.L().Error("failed create task", zap.Error(err))
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
		zap.L().Error("failed to parse task id", zap.Error(err))
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskID, lang),
		)
		return
	}

	var req dto.UpdateTaskRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		zap.L().Error("failed to binding task payload", zap.Error(err))
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	var raw map[string]json.RawMessage
	if err := c.ShouldBindBodyWith(&raw, binding.JSON); err != nil {
		zap.L().Error("failed to binding task payload", zap.Error(err))
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	input, err := validation.BuildUpdateTaskInput(req, raw)
	if err != nil {
		zap.L().Error("failed to building task payload", zap.Error(err))
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskPayload, lang),
		)
		return
	}

	task, err := h.taskService.UpdateTask(c.Request.Context(), taskID, input)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			zap.L().Error("failed to updating task,not found", zap.Error(err))
			c.JSON(
				http.StatusNotFound,
				apierrors.CreateError(http.StatusNotFound, apierrors.MsgTaskNotFound, lang),
			)
			return
		}
		if errors.Is(err, domain.ErrCategoryNotFound) {
			zap.L().Error("failed to updating task, category not found", zap.Error(err))
			c.JSON(
				http.StatusNotFound,
				apierrors.CreateError(http.StatusNotFound, apierrors.MsgCategoryNotFound, lang),
			)
			return
		}
		if errors.Is(err, domain.ErrTaskHierarchyCycle) {
			zap.L().Error("failed to updating task", zap.Error(err))
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
		zap.L().Error("failed to parsing task id", zap.Error(err))
		c.JSON(
			http.StatusBadRequest,
			apierrors.CreateError(http.StatusBadRequest, apierrors.MsgInvalidTaskID, lang),
		)
		return
	}

	if err := h.taskService.DeleteTask(c.Request.Context(), taskID); err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			zap.L().Error("failed to deleteing task, task not found", zap.Error(err))
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
