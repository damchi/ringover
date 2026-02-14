package tests

// Mock generation example for handler tests.
//
// Usage:
//   go generate ./internal/adapter/http/handlers/tests
//
//go:generate mockery --name TaskService --dir ../../../../core/ports --output ./mocks --outpkg mocks --filename task_service_mock.go --with-expecter
