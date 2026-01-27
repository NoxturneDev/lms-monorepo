package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/noxturnedev/lms-monorepo/gateway/utils"
	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
)

func (gw *Gateway) LoginTeacher(c *gin.Context) {
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

	resp, err := gw.TeacherClient.LoginTeacher(ctx, &teacherpb.LoginTeacherRequest{
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

	token, err := utils.GenerateToken(resp.TeacherId, resp.Email, "teacher")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":    token,
		"user_id":  resp.TeacherId,
		"email":    resp.Email,
		"name":     resp.FullName,
		"userType": "teacher",
	})
}

func (gw *Gateway) LoginStudent(c *gin.Context) {
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

	resp, err := gw.StudentClient.LoginStudent(ctx, &studentpb.LoginStudentRequest{
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

	token, err := utils.GenerateToken(resp.StudentId, resp.Email, "student")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":          token,
		"user_id":        resp.StudentId,
		"email":          resp.Email,
		"name":           resp.FullName,
		"student_number": resp.StudentNumber,
		"userType":       "student",
	})
}
