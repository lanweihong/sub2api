package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type PreviewBatchUsersRequest struct {
	Names []string `json:"names" binding:"required,min=1,dive,required"`
}

type CreateBatchUsersRequest struct {
	Users []CreateBatchUserItemRequest `json:"users" binding:"required,min=1,dive"`
}

type CreateBatchUserItemRequest struct {
	RowNo       int     `json:"row_no"`
	SourceName  string  `json:"source_name"`
	Email       string  `json:"email"`
	Password    string  `json:"password"`
	Username    string  `json:"username"`
	Notes       string  `json:"notes"`
	Balance     float64 `json:"balance"`
	Concurrency int     `json:"concurrency"`
}

type PreviewBatchUsersResponse struct {
	Items []service.BatchUserPreviewItem `json:"items"`
}

// PreviewBatch handles previewing a batch user list from names.
// POST /api/v1/admin/users/batch/preview
func (h *UserHandler) PreviewBatch(c *gin.Context) {
	var req PreviewBatchUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	items, err := h.adminService.PreviewBatchUsers(c.Request.Context(), req.Names)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, PreviewBatchUsersResponse{Items: items})
}

// CreateBatch handles atomically creating a batch of users.
// POST /api/v1/admin/users/batch
func (h *UserHandler) CreateBatch(c *gin.Context) {
	var req CreateBatchUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := make([]service.BatchCreateUserInput, 0, len(req.Users))
	for _, item := range req.Users {
		input = append(input, service.BatchCreateUserInput{
			RowNo:       item.RowNo,
			SourceName:  item.SourceName,
			Email:       item.Email,
			Password:    item.Password,
			Username:    item.Username,
			Notes:       item.Notes,
			Balance:     item.Balance,
			Concurrency: item.Concurrency,
		})
	}

	result, err := h.adminService.CreateUsersBatch(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}
