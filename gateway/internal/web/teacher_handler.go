package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
)

func (gw *Gateway) CreateCourse(c *gin.Context) {
	var req struct {
		TeacherID   string `json:"teacher_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.CreateCourse(ctx, &teacherpb.CreateCourseRequest{
		TeacherId:   req.TeacherID,
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (gw *Gateway) AssignGrade(c *gin.Context) {
	var req struct {
		TeacherID string `json:"teacher_id"`
		CourseID  string `json:"course_id"`
		StudentID string `json:"student_id"`
		Score     int32  `json:"score"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.AssignGrade(ctx, &teacherpb.AssignGradeRequest{
		TeacherId: req.TeacherID,
		CourseId:  req.CourseID,
		StudentId: req.StudentID,
		Score:     req.Score,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
