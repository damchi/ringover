package handlers

import (
	"errors"
	"net/http"
	"ringover/internal/adapter/http/mapper"
	"ringover/internal/adapter/http/middleware"
	"ringover/internal/core/domain"
	"ringover/internal/core/ports"
	"ringover/pkg/apierrors"
	"strconv"

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
