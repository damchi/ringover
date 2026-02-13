package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	dbadapter "ringover/internal/adapter/db"
	httpadapter "ringover/internal/adapter/http"
	"ringover/internal/adapter/http/dto"
	"ringover/internal/adapter/http/handlers"
	appservice "ringover/internal/app/service"
	"ringover/pkg/apierrors"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type TasksIntegrationSuite struct {
	IntegrationSuiteBase
	router *gin.Engine
}

func TestTasksIntegrationSuite(t *testing.T) {
	suite.Run(t, new(TasksIntegrationSuite))
}

func (s *TasksIntegrationSuite) SetupTest() {
	s.ResetDatabase()

	router := gin.New()
	healthHandler := handlers.NewHealthHandler(s.DB)
	taskRepository := dbadapter.NewTaskRepository(s.DB)
	taskService := appservice.NewTaskService(taskRepository)
	taskHandler := handlers.NewTaskHandler(taskService)
	httpadapter.RegisterRoutes(router, healthHandler, taskHandler)

	s.router = router
}

func (s *TasksIntegrationSuite) TestGetTasks_ReturnsRootTasksOnly() {
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)

	var got []dto.TaskItem
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Len(got, 3)

	for _, item := range got {
		s.Require().NotZero(item.ID)
		s.Require().NotEmpty(item.Title)
		s.Require().NotEmpty(item.Status)
		s.Require().NotEmpty(item.CreatedAt)
		s.Require().NotEmpty(item.UpdatedAt)
		s.Require().NotNil(item.Category)
	}

	// Ensure only root tasks are returned (subtasks from seed data are excluded).
	s.Require().Equal(uint64(1), got[0].ID)
	s.Require().Equal(uint64(2), got[1].ID)
	s.Require().Equal(uint64(3), got[2].ID)
}

func (s *TasksIntegrationSuite) TestGetTasks_ReturnsEmptyListWhenNoRootTasks() {
	_, err := s.DB.Exec("DELETE FROM tasks WHERE parent_task_id IS NULL")
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)

	var got []dto.TaskItem
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Len(got, 0)
}

func (s *TasksIntegrationSuite) TestGetTasks_ReturnsInternalServerErrorWhenQueryFails() {
	_, err := s.DB.Exec("DROP TABLE tasks")
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusInternalServerError, rec.Code)

	var got apierrors.JsonErr
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Equal(http.StatusInternalServerError, got.ErrDetails.Code)
	s.Require().Equal("failed to list root tasks", got.ErrDetails.Message)
}
