package admin

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// DepartmentHandler handles admin department management
type DepartmentHandler struct {
	deptService service.DepartmentService
}

// NewDepartmentHandler creates a new admin department handler
func NewDepartmentHandler(deptService service.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{deptService: deptService}
}

type CreateDepartmentRequest struct {
	Name        string `json:"name" binding:"required"`
	Code        string `json:"code" binding:"omitempty,max=50"`
	Description string `json:"description"`
	ParentID    *int64 `json:"parent_id"`
	SortOrder   int    `json:"sort_order"`
	Status      string `json:"status" binding:"omitempty,oneof=active disabled"`
}

type UpdateDepartmentRequest struct {
	Name        *string `json:"name"`
	Code        *string `json:"code"`
	Description *string `json:"description"`
	ParentID    *int64  `json:"parent_id"`
	SortOrder   *int    `json:"sort_order"`
	Status      *string `json:"status" binding:"omitempty,oneof=active disabled"`
}

// List handles listing all departments
// GET /api/v1/admin/departments
func (h *DepartmentHandler) List(c *gin.Context) {
	depts, err := h.deptService.List(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.DepartmentsFromService(depts))
}

// Get handles getting a single department
// GET /api/v1/admin/departments/:id
func (h *DepartmentHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid department ID")
		return
	}

	dept, err := h.deptService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.DepartmentFromService(dept))
}

// Create handles creating a new department
// POST /api/v1/admin/departments
func (h *DepartmentHandler) Create(c *gin.Context) {
	var req CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	dept, err := h.deptService.Create(c.Request.Context(), &service.CreateDepartmentInput{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		ParentID:    req.ParentID,
		SortOrder:   req.SortOrder,
		Status:      req.Status,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.DepartmentFromService(dept))
}

// Update handles updating a department
// PUT /api/v1/admin/departments/:id
func (h *DepartmentHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid department ID")
		return
	}

	var req UpdateDepartmentRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 第二次绑定为 raw map，检测 parent_id 是否显式存在
	var raw map[string]json.RawMessage
	if err := c.ShouldBindBodyWith(&raw, binding.JSON); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := &service.UpdateDepartmentInput{
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		Status:      req.Status,
	}
	if rawParent, ok := raw["parent_id"]; ok {
		input.ParentIDSet = true
		if !bytes.Equal(bytes.TrimSpace(rawParent), []byte("null")) {
			input.ParentID = req.ParentID
		}
	}

	dept, err := h.deptService.Update(c.Request.Context(), id, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.DepartmentFromService(dept))
}

// Delete handles deleting a department
// DELETE /api/v1/admin/departments/:id
func (h *DepartmentHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid department ID")
		return
	}

	force := c.Query("force") == "true"

	if err := h.deptService.Delete(c.Request.Context(), id, force); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, nil)
}
