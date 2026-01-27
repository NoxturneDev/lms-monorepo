package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sony/gobreaker"
	"golang.org/x/sync/errgroup"

	"github.com/noxturnedev/lms-monorepo/gateway/utils"
	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
)

func (gw *Gateway) CreateStudent(c *gin.Context) {
	var req struct {
		Email         string `json:"email"`
		FullName      string `json:"full_name"`
		Password      string `json:"password"`
		StudentNumber string `json:"student_number"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.StudentClient.CreateStudent(ctx, &studentpb.CreateStudentRequest{
		Email:         req.Email,
		FullName:      req.FullName,
		Password:      req.Password,
		StudentNumber: req.StudentNumber,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (gw *Gateway) DeleteStudent(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := gw.StudentClient.DeleteStudent(ctx, &studentpb.DeleteStudentRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student deleted and cleanup scheduled"})
}

func (gw *Gateway) GetAllStudents(c *gin.Context) {
	var req struct {
		ClassID string `json:"class_id"`
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.StudentClient.GetAllStudents(ctx, &studentpb.ListStudentRequest{
		ClassId: req.ClassID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) GetStudentDetails(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	student, err := gw.StudentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"student_profile": student,
		"source":          "Aggregation Gateway",
	})
}

func (gw *Gateway) GetStudentReportCard(c *gin.Context) {
	studentID := c.Param("id")
	g, ctx := errgroup.WithContext(c.Request.Context())

	var (
		studentResp *studentpb.StudentResponse
		gradesResp  *teacherpb.StudentGradesResponse
	)

	g.Go(func() error {
		result, err := utils.ExecuteWithBreaker(utils.StudentBreaker, func() (interface{}, error) {
			return gw.StudentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{Id: studentID})
		})
		if err != nil {
			return err
		}

		studentResp = result.(*studentpb.StudentResponse)
		return nil
	})

	g.Go(func() error {
		result, err := utils.ExecuteWithBreaker(utils.TeacherBreaker, func() (interface{}, error) {
			return gw.TeacherClient.GetStudentGrades(ctx, &teacherpb.GetStudentGradesRequest{StudentId: studentID})
		})
		if err != nil {
			return err
		}

		gradesResp = result.(*teacherpb.StudentGradesResponse)
		return nil
	})

	if err := g.Wait(); err != nil {
		if err == gobreaker.ErrOpenState {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "System overloaded. Please try again later."})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch data: " + err.Error()})
		return
	}

	response := gin.H{
		"student_info": gin.H{
			"name":           studentResp.FullName,
			"email":          studentResp.Email,
			"student_number": studentResp.StudentNumber,
		},
		"academic_record": gradesResp.Grades,
		"generated_at":    time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

func (gw *Gateway) UpdateStudent(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Email         string `json:"email"`
		FullName      string `json:"full_name"`
		StudentNumber string `json:"student_number"`
		Password      string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.StudentClient.UpdateStudent(ctx, &studentpb.UpdateStudentRequest{
		Id:            id,
		Email:         req.Email,
		FullName:      req.FullName,
		StudentNumber: req.StudentNumber,
		Password:      req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
