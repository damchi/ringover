package tests

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
