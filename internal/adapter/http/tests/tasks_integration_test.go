package tests

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func (s *TasksIntegrationSuite) TestGetTaskSubtasks_ReturnsFullHierarchy() {
	result, err := s.DB.Exec(
		"INSERT INTO tasks (title, status, priority, due_date, parent_task_id, category_id) VALUES (?, ?, ?, ?, ?, ?)",
		"Configurer callback URL",
		"todo",
		1,
		"2025-08-21",
		4,
		1,
	)
	s.Require().NoError(err)

	grandChildID, err := result.LastInsertId()
	s.Require().NoError(err)

	req := httptest.NewRequest(http.MethodGet, "/api/tasks/1/subtasks", nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)

	var got []dto.TaskItem
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Len(got, 2)
	s.Require().Equal(uint64(4), got[0].ID)
	s.Require().Equal(uint64(5), got[1].ID)
	s.Require().Len(got[0].Subtasks, 1)
	s.Require().Equal(uint64(grandChildID), got[0].Subtasks[0].ID)
	s.Require().Len(got[1].Subtasks, 0)
}

func (s *TasksIntegrationSuite) TestGetTaskSubtasks_ReturnsNotFoundWhenTaskDoesNotExist() {
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/999999/subtasks", nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)

	var got apierrors.JsonErr
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Equal(http.StatusNotFound, got.ErrDetails.Code)
	s.Require().Equal("Task not found", got.ErrDetails.Message)
}

func (s *TasksIntegrationSuite) TestGetTaskSubtasks_ReturnsBadRequestWhenIDIsInvalid() {
	req := httptest.NewRequest(http.MethodGet, "/api/tasks/abc/subtasks", nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Equal(http.StatusBadRequest, got.ErrDetails.Code)
	s.Require().Equal("Invalid id", got.ErrDetails.Message)
}

func (s *TasksIntegrationSuite) TestPostTasks_CreatesRootTask() {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"title":"Créer endpoint POST /tasks",
		"status":"todo",
		"priority":2,
		"due_date":"2026-02-20",
		"category_id":1
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusCreated, rec.Code)

	var got dto.TaskItem
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().NotZero(got.ID)
	s.Require().Equal("Créer endpoint POST /tasks", got.Title)
	s.Require().Equal("todo", got.Status)
	s.Require().Equal(2, got.Priority)
	s.Require().NotNil(got.Category)
	s.Require().Equal(uint64(1), got.Category.ID)

	var parentID sql.NullInt64
	err := s.DB.Get(&parentID, "SELECT parent_task_id FROM tasks WHERE id = ?", got.ID)
	s.Require().NoError(err)
	s.Require().False(parentID.Valid)
}

func (s *TasksIntegrationSuite) TestPostTasks_CreatesSubTask() {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"title":"Ajouter tests endpoint",
		"parent_task_id":1,
		"category_id":1
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusCreated, rec.Code)

	var got dto.TaskItem
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().NotZero(got.ID)
	s.Require().Equal("Ajouter tests endpoint", got.Title)
	s.Require().Equal("todo", got.Status)

	var parentID sql.NullInt64
	err := s.DB.Get(&parentID, "SELECT parent_task_id FROM tasks WHERE id = ?", got.ID)
	s.Require().NoError(err)
	s.Require().True(parentID.Valid)
	s.Require().Equal(int64(1), parentID.Int64)
}

func (s *TasksIntegrationSuite) TestPostTasks_ReturnsNotFoundWhenParentTaskDoesNotExist() {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"title":"Subtask",
		"parent_task_id":999999
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)

	var got apierrors.JsonErr
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Equal(http.StatusNotFound, got.ErrDetails.Code)
	s.Require().Equal("Task not found", got.ErrDetails.Message)
}

func (s *TasksIntegrationSuite) TestPostTasks_ReturnsBadRequestWhenPayloadIsInvalid() {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Equal(http.StatusBadRequest, got.ErrDetails.Code)
	s.Require().Equal("Invalid task payload", got.ErrDetails.Message)
}

func (s *TasksIntegrationSuite) TestPostTasks_ReturnsBadRequestWhenStatusIsInvalid() {
	req := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(`{
		"title":"Task",
		"status":"blocked"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Equal(http.StatusBadRequest, got.ErrDetails.Code)
	s.Require().Equal("Invalid task payload", got.ErrDetails.Message)
}

func (s *TasksIntegrationSuite) TestPatchTasks_UpdatesTask() {
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{
		"title":"Task updated from patch",
		"status":"done",
		"priority":1
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusOK, rec.Code)

	var got dto.TaskItem
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Equal(uint64(1), got.ID)
	s.Require().Equal("Task updated from patch", got.Title)
	s.Require().Equal("done", got.Status)
	s.Require().Equal(1, got.Priority)

	var row struct {
		Title    string `db:"title"`
		Status   string `db:"status"`
		Priority int    `db:"priority"`
	}
	err := s.DB.Get(&row, "SELECT title, status, priority FROM tasks WHERE id = 1")
	s.Require().NoError(err)
	s.Require().Equal("Task updated from patch", row.Title)
	s.Require().Equal("done", row.Status)
	s.Require().Equal(1, row.Priority)
}

func (s *TasksIntegrationSuite) TestPatchTasks_ReturnsBadRequestWhenIDIsInvalid() {
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/abc", strings.NewReader(`{"title":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Equal(http.StatusBadRequest, got.ErrDetails.Code)
	s.Require().Equal("Invalid id", got.ErrDetails.Message)
}

func (s *TasksIntegrationSuite) TestPatchTasks_ReturnsBadRequestWhenPayloadIsInvalid() {
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/1", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusBadRequest, rec.Code)

	var got apierrors.JsonErr
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Equal(http.StatusBadRequest, got.ErrDetails.Code)
	s.Require().Equal("Invalid task payload", got.ErrDetails.Message)
}

func (s *TasksIntegrationSuite) TestPatchTasks_ReturnsNotFoundWhenTaskDoesNotExist() {
	req := httptest.NewRequest(http.MethodPatch, "/api/tasks/999999", strings.NewReader(`{"title":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	s.Require().Equal(http.StatusNotFound, rec.Code)

	var got apierrors.JsonErr
	s.Require().NoError(json.Unmarshal(rec.Body.Bytes(), &got))
	s.Require().Equal(http.StatusNotFound, got.ErrDetails.Code)
	s.Require().Equal("Task not found", got.ErrDetails.Message)
}
