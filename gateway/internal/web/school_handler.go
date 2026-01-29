package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/noxturnedev/lms-monorepo/gateway/utils"
	schoolpb "github.com/noxturnedev/lms-monorepo/proto/school"
)

// ============================================
// ADMIN AUTH
// ============================================

func (gw *Gateway) LoginAdmin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.LoginAdmin(ctx, &schoolpb.LoginAdminRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if !resp.Success {
		c.JSON(http.StatusUnauthorized, gin.H{"error": resp.Message})
		return
	}

	token, err := utils.GenerateToken(resp.AdminId, resp.Email, "admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":       token,
		"user_id":     resp.AdminId,
		"email":       resp.Email,
		"name":        resp.FullName,
		"school_id":   resp.SchoolId,
		"school_name": resp.SchoolName,
		"userType":    "admin",
	})
}

// ============================================
// ADMIN CRUD
// ============================================

func (gw *Gateway) CreateAdmin(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
		SchoolID string `json:"school_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.CreateAdmin(ctx, &schoolpb.CreateAdminRequest{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		SchoolId: req.SchoolID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (gw *Gateway) GetAdmin(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.GetAdmin(ctx, &schoolpb.GetAdminRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Admin not found"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) UpdateAdmin(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Email    string `json:"email"`
		FullName string `json:"full_name"`
		Password string `json:"password"`
		SchoolID string `json:"school_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.UpdateAdmin(ctx, &schoolpb.UpdateAdminRequest{
		Id:       id,
		Email:    req.Email,
		FullName: req.FullName,
		Password: req.Password,
		SchoolId: req.SchoolID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) DeleteAdmin(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.DeleteAdmin(ctx, &schoolpb.DeleteAdminRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) ListAdmins(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.ListAdmins(ctx, &schoolpb.ListAdminsRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ============================================
// SCHOOL CRUD
// ============================================

func (gw *Gateway) CreateSchool(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.CreateSchool(ctx, &schoolpb.CreateSchoolRequest{
		Name:    req.Name,
		Address: req.Address,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (gw *Gateway) GetSchool(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.GetSchool(ctx, &schoolpb.GetSchoolRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "School not found"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) UpdateSchool(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.UpdateSchool(ctx, &schoolpb.UpdateSchoolRequest{
		Id:      id,
		Name:    req.Name,
		Address: req.Address,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) DeleteSchool(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.DeleteSchool(ctx, &schoolpb.DeleteSchoolRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !resp.Success {
		c.JSON(http.StatusConflict, gin.H{"error": resp.Message})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) ListSchools(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.ListSchools(ctx, &schoolpb.ListSchoolsRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ============================================
// CLASS CRUD
// ============================================

func (gw *Gateway) CreateClass(c *gin.Context) {
	var req struct {
		SchoolID   string `json:"school_id"`
		Name       string `json:"name"`
		GradeLevel string `json:"grade_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.CreateClass(ctx, &schoolpb.CreateClassRequest{
		SchoolId:   req.SchoolID,
		Name:       req.Name,
		GradeLevel: req.GradeLevel,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (gw *Gateway) GetClass(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.GetClass(ctx, &schoolpb.GetClassRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Class not found"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) UpdateClass(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Name       string `json:"name"`
		GradeLevel string `json:"grade_level"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.UpdateClass(ctx, &schoolpb.UpdateClassRequest{
		Id:         id,
		Name:       req.Name,
		GradeLevel: req.GradeLevel,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) DeleteClass(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.DeleteClass(ctx, &schoolpb.DeleteClassRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) ListClasses(c *gin.Context) {
	schoolID := c.Query("school_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.ListClasses(ctx, &schoolpb.ListClassesRequest{
		SchoolId: schoolID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
