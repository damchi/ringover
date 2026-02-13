package dto

type TaskItem struct {
	ID          uint64     `json:"id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	Priority    int        `json:"priority"`
	DueDate     *string    `json:"due_date,omitempty"`
	CompletedAt *string    `json:"completed_at,omitempty"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
	Category    *Category  `json:"category,omitempty"`
	Subtasks    []TaskItem `json:"subtasks,omitempty"`
}

type CreateTaskRequest struct {
	Title        string  `json:"title" binding:"required,max=255"`
	Description  *string `json:"description" binding:"omitempty,max=65535"`
	Status       *string `json:"status" binding:"omitempty,oneof=todo in_progress done"`
	Priority     *int    `json:"priority" binding:"omitempty,gte=0,lte=127"`
	DueDate      *string `json:"due_date" binding:"omitempty,datetime=2006-01-02"`
	ParentTaskID *uint64 `json:"parent_task_id" binding:"omitempty,gt=0"`
	CategoryID   *uint64 `json:"category_id" binding:"omitempty,gt=0"`
}
