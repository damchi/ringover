package tests

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ringover/internal/adapter/http/dto"
	"ringover/internal/adapter/http/handlers"
	"ringover/internal/adapter/http/middleware"
	"ringover/internal/core/domain"
	"ringover/pkg/apierrors"
	"ringover/pkg/translator"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type taskServiceMock struct {
	mock.Mock
}

func (m *taskServiceMock) ListRootTasks(ctx context.Context) ([]domain.Task, error) {
	args := m.Called(ctx)

	var tasks []domain.Task
	if value := args.Get(0); value != nil {
		tasks = value.([]domain.Task)
	}
	return tasks, args.Error(1)
}

func (m *taskServiceMock) ListRootSubtasks(ctx context.Context, taskID uint64) ([]domain.Task, error) {
	args := m.Called(ctx, taskID)

	var tasks []domain.Task
	if value := args.Get(0); value != nil {
		tasks = value.([]domain.Task)
	}
	return tasks, args.Error(1)
}

func (m *taskServiceMock) CreateTask(ctx context.Context, input domain.CreateTaskInput) (domain.Task, error) {
	args := m.Called(ctx, input)

	var task domain.Task
	if value := args.Get(0); value != nil {
		task = value.(domain.Task)
	}
	return task, args.Error(1)
}

func (m *taskServiceMock) UpdateTask(ctx context.Context, taskID uint64, input domain.UpdateTaskInput) (domain.Task, error) {
	args := m.Called(ctx, taskID, input)

	var task domain.Task
	if value := args.Get(0); value != nil {
		task = value.(domain.Task)
	}
	return task, args.Error(1)
}

func TestTaskHandler_ListRootTasks_Success(t *testing.T) {
	description := "ship endpoint"
	dueDate := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 2, 13, 10, 20, 30, 0, time.UTC)
	updatedAt := time.Date(2026, 2, 13, 11, 20, 30, 0, time.UTC)
	completedAt := time.Date(2026, 2, 19, 11, 20, 30, 0, time.UTC)

	serviceMock := new(taskServiceMock)
	serviceMock.On("ListRootTasks", mock.Anything).Return(
		[]domain.Task{
			{
				ID:          1,
				Title:       "Build interview API",
				Description: &description,
				Status:      domain.TaskStatusInProgress,
				Priority:    3,
				DueDate:     &dueDate,
				CreatedAt:   createdAt,
				CompletedAt: &completedAt,
				UpdatedAt:   updatedAt,
				Category: &domain.Category{
					ID:   1,
					Name: "Backend",
				},
			},
		},
		nil,
	).Once()
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.GET("/api/tasks", middleware.LanguageMiddleware(), handler.ListRootTasks)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var got []dto.TaskItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 1)

	require.Equal(t, uint64(1), got[0].ID)
	require.Equal(t, "Build interview API", got[0].Title)
	require.Equal(t, "ship endpoint", *got[0].Description)
	require.Equal(t, "in_progress", got[0].Status)
	require.Equal(t, 3, got[0].Priority)
	require.Equal(t, "2026-02-20", *got[0].DueDate)
	require.Equal(t, "2026-02-19", *got[0].CompletedAt)
	require.Equal(t, "2026-02-13T10:20:30Z", got[0].CreatedAt)
	require.Equal(t, "2026-02-13T11:20:30Z", got[0].UpdatedAt)
	require.NotNil(t, got[0].Category)
	require.Equal(t, uint64(1), got[0].Category.ID)
	require.Equal(t, "Backend", got[0].Category.Name)
	serviceMock.AssertExpectations(t)
}

func TestTaskHandler_ListRootTasks_Error(t *testing.T) {
	serviceMock := new(taskServiceMock)
	serviceMock.On("ListRootTasks", mock.Anything).Return(nil, errors.New("db is down")).Once()
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.GET("/api/tasks", middleware.LanguageMiddleware(), handler.ListRootTasks)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusInternalServerError, got.ErrDetails.Code)
	require.Equal(t, "failed to list root tasks", got.ErrDetails.Message)
	serviceMock.AssertExpectations(t)
}

func TestTaskHandler_ListRootSubTasks_Success(t *testing.T) {
	createdAt := time.Date(2026, 2, 13, 10, 20, 30, 0, time.UTC)
	updatedAt := time.Date(2026, 2, 13, 11, 20, 30, 0, time.UTC)

	serviceMock := new(taskServiceMock)
	serviceMock.On("ListRootSubtasks", mock.Anything, uint64(1)).Return(
		[]domain.Task{
			{
				ID:        4,
				Title:     "Ajouter OAuth2",
				Status:    domain.TaskStatusTodo,
				Priority:  2,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				Subtasks: []domain.Task{
					{
						ID:        7,
						Title:     "Configurer provider",
						Status:    domain.TaskStatusInProgress,
						Priority:  1,
						CreatedAt: createdAt,
						UpdatedAt: updatedAt,
					},
				},
			},
			{
				ID:        5,
				Title:     "Configurer JWT",
				Status:    domain.TaskStatusTodo,
				Priority:  3,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
		},
		nil,
	).Once()
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.GET("/api/tasks/:id/subtasks", middleware.LanguageMiddleware(), handler.ListRootSubTasks)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1/subtasks", nil)
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var got []dto.TaskItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 2)
	require.Equal(t, uint64(4), got[0].ID)
	require.Len(t, got[0].Subtasks, 1)
	require.Equal(t, uint64(7), got[0].Subtasks[0].ID)
	require.Equal(t, uint64(5), got[1].ID)
	require.Len(t, got[1].Subtasks, 0)
	serviceMock.AssertExpectations(t)
}

func TestTaskHandler_ListRootSubTasks_InvalidTaskID(t *testing.T) {
	serviceMock := new(taskServiceMock)
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.GET("/api/tasks/:id/subtasks", middleware.LanguageMiddleware(), handler.ListRootSubTasks)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/invalid/subtasks", nil)
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusBadRequest, got.ErrDetails.Code)
	require.Equal(t, "Invalid id", got.ErrDetails.Message)
}

func TestTaskHandler_ListRootSubTasks_NotFound(t *testing.T) {
	serviceMock := new(taskServiceMock)
	serviceMock.On("ListRootSubtasks", mock.Anything, uint64(999)).Return(nil, domain.ErrTaskNotFound).Once()
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.GET("/api/tasks/:id/subtasks", middleware.LanguageMiddleware(), handler.ListRootSubTasks)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/999/subtasks", nil)
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusNotFound, got.ErrDetails.Code)
	require.Equal(t, "Task not found", got.ErrDetails.Message)
	serviceMock.AssertExpectations(t)
}

func TestTaskHandler_ListRootSubTasks_Error(t *testing.T) {
	serviceMock := new(taskServiceMock)
	serviceMock.On("ListRootSubtasks", mock.Anything, uint64(1)).Return(nil, errors.New("db is down")).Once()
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.GET("/api/tasks/:id/subtasks", middleware.LanguageMiddleware(), handler.ListRootSubTasks)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1/subtasks", nil)
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusInternalServerError, got.ErrDetails.Code)
	require.Equal(t, "Error fetching the subtasks", got.ErrDetails.Message)
	serviceMock.AssertExpectations(t)
}

func TestTaskHandler_CreateTask_Success(t *testing.T) {
	createdAt := time.Date(2026, 2, 13, 10, 20, 30, 0, time.UTC)
	updatedAt := time.Date(2026, 2, 13, 11, 20, 30, 0, time.UTC)
	dueDate := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

	serviceMock := new(taskServiceMock)
	serviceMock.On("CreateTask", mock.Anything, mock.MatchedBy(func(input domain.CreateTaskInput) bool {
		if input.Title != "Build interview API" {
			return false
		}
		if input.Status != domain.TaskStatusInProgress {
			return false
		}
		if input.Priority != 3 {
			return false
		}
		if input.DueDate == nil {
			return false
		}
		return input.DueDate.Format("2006-01-02") == "2026-02-20"
	})).Return(
		domain.Task{
			ID:        10,
			Title:     "Build interview API",
			Status:    domain.TaskStatusInProgress,
			Priority:  3,
			DueDate:   &dueDate,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Category: &domain.Category{
				ID:   1,
				Name: "Backend",
			},
		},
		nil,
	).Once()

	handler := handlers.NewTaskHandler(serviceMock)
	router := gin.New()
	router.POST("/api/tasks", middleware.LanguageMiddleware(), handler.CreateTask)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"title":"Build interview API",
		"status":"in_progress",
		"priority":3,
		"due_date":"2026-02-20",
		"category_id":1
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var got dto.TaskItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, uint64(10), got.ID)
	require.Equal(t, "Build interview API", got.Title)
	require.Equal(t, "in_progress", got.Status)
	require.Equal(t, 3, got.Priority)
	require.NotNil(t, got.DueDate)
	require.Equal(t, "2026-02-20", *got.DueDate)
	require.NotNil(t, got.Category)
	require.Equal(t, uint64(1), got.Category.ID)
	serviceMock.AssertExpectations(t)
}

func TestTaskHandler_CreateTask_InvalidPayload(t *testing.T) {
	serviceMock := new(taskServiceMock)
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.POST("/api/tasks", middleware.LanguageMiddleware(), handler.CreateTask)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusBadRequest, got.ErrDetails.Code)
	require.Equal(t, "Invalid task payload", got.ErrDetails.Message)
}

func TestTaskHandler_CreateTask_InvalidDueDate(t *testing.T) {
	serviceMock := new(taskServiceMock)
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.POST("/api/tasks", middleware.LanguageMiddleware(), handler.CreateTask)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"title":"Build interview API",
		"due_date":"2026-40-99"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusBadRequest, got.ErrDetails.Code)
	require.Equal(t, "Invalid task payload", got.ErrDetails.Message)
}

func TestTaskHandler_CreateTask_InvalidStatus(t *testing.T) {
	serviceMock := new(taskServiceMock)
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.POST("/api/tasks", middleware.LanguageMiddleware(), handler.CreateTask)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"title":"Build interview API",
		"status":"blocked"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusBadRequest, got.ErrDetails.Code)
	require.Equal(t, "Invalid task payload", got.ErrDetails.Message)
}

func TestTaskHandler_CreateTask_InvalidPriority(t *testing.T) {
	serviceMock := new(taskServiceMock)
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.POST("/api/tasks", middleware.LanguageMiddleware(), handler.CreateTask)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"title":"Build interview API",
		"priority":-1
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusBadRequest, got.ErrDetails.Code)
	require.Equal(t, "Invalid task payload", got.ErrDetails.Message)
}

func TestTaskHandler_CreateTask_NotFound(t *testing.T) {
	serviceMock := new(taskServiceMock)
	serviceMock.On("CreateTask", mock.Anything, mock.Anything).Return(domain.Task{}, domain.ErrTaskNotFound).Once()
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.POST("/api/tasks", middleware.LanguageMiddleware(), handler.CreateTask)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"title":"Subtask",
		"parent_task_id":999
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusNotFound, got.ErrDetails.Code)
	require.Equal(t, "Task not found", got.ErrDetails.Message)
	serviceMock.AssertExpectations(t)
}

func TestTaskHandler_CreateTask_Error(t *testing.T) {
	serviceMock := new(taskServiceMock)
	serviceMock.On("CreateTask", mock.Anything, mock.Anything).Return(domain.Task{}, errors.New("db is down")).Once()
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.POST("/api/tasks", middleware.LanguageMiddleware(), handler.CreateTask)

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"title":"Build interview API"
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusInternalServerError, got.ErrDetails.Code)
	require.Equal(t, "Failed to create task", got.ErrDetails.Message)
	serviceMock.AssertExpectations(t)
}

func TestTaskHandler_UpdateTask_Success(t *testing.T) {
	createdAt := time.Date(2026, 2, 13, 10, 20, 30, 0, time.UTC)
	updatedAt := time.Date(2026, 2, 13, 11, 20, 30, 0, time.UTC)

	serviceMock := new(taskServiceMock)
	serviceMock.On("UpdateTask", mock.Anything, uint64(1), mock.MatchedBy(func(input domain.UpdateTaskInput) bool {
		if input.Title == nil || *input.Title != "Updated task title" {
			return false
		}
		if input.Status == nil || *input.Status != domain.TaskStatusDone {
			return false
		}
		return true
	})).Return(
		domain.Task{
			ID:        1,
			Title:     "Updated task title",
			Status:    domain.TaskStatusDone,
			Priority:  1,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		nil,
	).Once()

	handler := handlers.NewTaskHandler(serviceMock)
	router := gin.New()
	router.PATCH("/api/tasks/:id", middleware.LanguageMiddleware(), handler.UpdateTask)

	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{
		"title":"Updated task title",
		"status":"done",
		"priority":1
	}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var got dto.TaskItem
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, uint64(1), got.ID)
	require.Equal(t, "Updated task title", got.Title)
	require.Equal(t, "done", got.Status)
	require.Equal(t, 1, got.Priority)
	serviceMock.AssertExpectations(t)
}

func TestTaskHandler_UpdateTask_InvalidTaskID(t *testing.T) {
	serviceMock := new(taskServiceMock)
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.PATCH("/api/tasks/:id", middleware.LanguageMiddleware(), handler.UpdateTask)

	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/abc", strings.NewReader(`{"title":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusBadRequest, got.ErrDetails.Code)
	require.Equal(t, "Invalid id", got.ErrDetails.Message)
}

func TestTaskHandler_UpdateTask_InvalidPayload(t *testing.T) {
	serviceMock := new(taskServiceMock)
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.PATCH("/api/tasks/:id", middleware.LanguageMiddleware(), handler.UpdateTask)

	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusBadRequest, got.ErrDetails.Code)
	require.Equal(t, "Invalid task payload", got.ErrDetails.Message)
}

func TestTaskHandler_UpdateTask_NotFound(t *testing.T) {
	serviceMock := new(taskServiceMock)
	serviceMock.On("UpdateTask", mock.Anything, uint64(999), mock.Anything).Return(domain.Task{}, domain.ErrTaskNotFound).Once()
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.PATCH("/api/tasks/:id", middleware.LanguageMiddleware(), handler.UpdateTask)

	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/999", strings.NewReader(`{"title":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusNotFound, got.ErrDetails.Code)
	require.Equal(t, "Task not found", got.ErrDetails.Message)
	serviceMock.AssertExpectations(t)
}

func TestTaskHandler_UpdateTask_Error(t *testing.T) {
	serviceMock := new(taskServiceMock)
	serviceMock.On("UpdateTask", mock.Anything, uint64(1), mock.Anything).Return(domain.Task{}, errors.New("db is down")).Once()
	handler := handlers.NewTaskHandler(serviceMock)

	router := gin.New()
	router.PATCH("/api/tasks/:id", middleware.LanguageMiddleware(), handler.UpdateTask)

	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{"title":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", translator.LanguageEn)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)

	var got apierrors.JsonErr
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, http.StatusInternalServerError, got.ErrDetails.Code)
	require.Equal(t, "Failed to update task", got.ErrDetails.Message)
	serviceMock.AssertExpectations(t)
}
