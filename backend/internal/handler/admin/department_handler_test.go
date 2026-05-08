package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubDepartmentService implements service.DepartmentService for tests.
type stubDepartmentService struct {
	departments []service.Department
	createErr   error
	updateErr   error
	deleteErr   error
	lastCreate  *service.CreateDepartmentInput
	lastUpdate  *service.UpdateDepartmentInput
	lastDelete  struct {
		id    int64
		force bool
	}
}

func newStubDepartmentService() *stubDepartmentService {
	return &stubDepartmentService{
		departments: []service.Department{
			{
				ID: 1, Name: "Engineering", Code: "eng",
				Description: "Engineering team", Status: "active", IsDefault: true,
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			},
			{
				ID: 2, Name: "Marketing", Code: "mkt",
				Description: "Marketing team", Status: "active",
				CreatedAt: time.Now(), UpdatedAt: time.Now(),
			},
		},
	}
}

func (s *stubDepartmentService) List(_ context.Context) ([]service.Department, error) {
	return s.departments, nil
}

func (s *stubDepartmentService) GetByID(_ context.Context, id int64) (*service.Department, error) {
	for i := range s.departments {
		if s.departments[i].ID == id {
			return &s.departments[i], nil
		}
	}
	return nil, service.ErrDepartmentNotFound
}

func (s *stubDepartmentService) Create(_ context.Context, input *service.CreateDepartmentInput) (*service.Department, error) {
	s.lastCreate = input
	if s.createErr != nil {
		return nil, s.createErr
	}
	dept := &service.Department{
		ID:          99,
		Name:        input.Name,
		Code:        input.Code,
		Description: input.Description,
		ParentID:    input.ParentID,
		SortOrder:   input.SortOrder,
		Status:      input.Status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	return dept, nil
}

func (s *stubDepartmentService) Update(_ context.Context, id int64, input *service.UpdateDepartmentInput) (*service.Department, error) {
	s.lastUpdate = input
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	for i := range s.departments {
		if s.departments[i].ID == id {
			d := s.departments[i]
			if input.Name != nil {
				d.Name = *input.Name
			}
			if input.Code != nil {
				d.Code = *input.Code
			}
			return &d, nil
		}
	}
	return nil, service.ErrDepartmentNotFound
}

func (s *stubDepartmentService) Delete(_ context.Context, id int64, force bool) error {
	s.lastDelete.id = id
	s.lastDelete.force = force
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return nil
}

func (s *stubDepartmentService) GetDefaultDepartmentID(_ context.Context) (int64, error) {
	for _, d := range s.departments {
		if d.IsDefault {
			return d.ID, nil
		}
	}
	return 0, service.ErrDepartmentNotFound
}

func (s *stubDepartmentService) ListDescendantIDs(_ context.Context, _ int64) ([]int64, error) {
	return nil, nil
}

func setupDepartmentRouter() (*gin.Engine, *stubDepartmentService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc := newStubDepartmentService()
	h := NewDepartmentHandler(svc)

	router.GET("/api/v1/admin/departments", h.List)
	router.GET("/api/v1/admin/departments/:id", h.Get)
	router.POST("/api/v1/admin/departments", h.Create)
	router.PUT("/api/v1/admin/departments/:id", h.Update)
	router.DELETE("/api/v1/admin/departments/:id", h.Delete)

	return router, svc
}

func TestDepartmentHandler_List(t *testing.T) {
	router, _ := setupDepartmentRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/departments", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data := resp["data"].([]any)
	assert.Len(t, data, 2)
}

func TestDepartmentHandler_Get(t *testing.T) {
	router, _ := setupDepartmentRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/departments/1", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data := resp["data"].(map[string]any)
	assert.Equal(t, "Engineering", data["name"])
}

func TestDepartmentHandler_Get_NotFound(t *testing.T) {
	router, _ := setupDepartmentRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/departments/999", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDepartmentHandler_Get_InvalidID(t *testing.T) {
	router, _ := setupDepartmentRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/departments/abc", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDepartmentHandler_Create(t *testing.T) {
	router, svc := setupDepartmentRouter()

	body, _ := json.Marshal(map[string]any{
		"name":       "Sales",
		"code":       "sales",
		"sort_order": 10,
		"status":     "active",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/departments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "Sales", svc.lastCreate.Name)
	assert.Equal(t, "sales", svc.lastCreate.Code)
	assert.Equal(t, 10, svc.lastCreate.SortOrder)
}

func TestDepartmentHandler_Create_MissingName(t *testing.T) {
	router, _ := setupDepartmentRouter()

	// name is still required — missing it should return 400
	body, _ := json.Marshal(map[string]any{
		"code": "SALES",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/departments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestDepartmentHandler_Create_EmptyCode(t *testing.T) {
	router, svc := setupDepartmentRouter()

	// code is optional — empty string should succeed
	body, _ := json.Marshal(map[string]any{
		"name": "Sales",
		"code": "",
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/departments", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", svc.lastCreate.Code)
}

func TestDepartmentHandler_Update(t *testing.T) {
	router, svc := setupDepartmentRouter()

	newName := "Eng Updated"
	body, _ := json.Marshal(map[string]any{
		"name": newName,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/departments/1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, svc.lastUpdate.Name)
	assert.Equal(t, newName, *svc.lastUpdate.Name)
}

func TestDepartmentHandler_Update_NotFound(t *testing.T) {
	router, svc := setupDepartmentRouter()
	svc.updateErr = service.ErrDepartmentNotFound

	body, _ := json.Marshal(map[string]any{"name": "x"})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/departments/999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDepartmentHandler_Delete(t *testing.T) {
	router, svc := setupDepartmentRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/departments/2", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int64(2), svc.lastDelete.id)
	assert.False(t, svc.lastDelete.force)
}

func TestDepartmentHandler_Delete_Force(t *testing.T) {
	router, svc := setupDepartmentRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/departments/2?force=true", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int64(2), svc.lastDelete.id)
	assert.True(t, svc.lastDelete.force)
}

func TestDepartmentHandler_Delete_Error(t *testing.T) {
	router, svc := setupDepartmentRouter()
	svc.deleteErr = service.ErrCannotDeleteDefaultDepartment

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/departments/1", nil)
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}
