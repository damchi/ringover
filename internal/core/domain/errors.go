package domain

import "errors"

var (
	ErrTaskNotFound       = errors.New("task not found")
	ErrCategoryNotFound   = errors.New("category not found")
	ErrTaskHierarchyCycle = errors.New("task hierarchy cycle")
)
