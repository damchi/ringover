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
