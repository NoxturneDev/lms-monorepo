package web

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
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

// ============================================
// TEACHER CRUD
// ============================================

func (gw *Gateway) CreateTeacher(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.CreateTeacher(ctx, &teacherpb.CreateTeacherRequest{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (gw *Gateway) GetTeacher(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.GetTeacher(ctx, &teacherpb.GetTeacherRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Teacher not found"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) UpdateTeacher(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Email    string `json:"email"`
		FullName string `json:"full_name"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.UpdateTeacher(ctx, &teacherpb.UpdateTeacherRequest{
		Id:       id,
		Email:    req.Email,
		FullName: req.FullName,
		Password: req.Password,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) DeleteTeacher(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.DeleteTeacher(ctx, &teacherpb.DeleteTeacherRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) ListTeachers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.ListTeachers(ctx, &teacherpb.ListTeachersRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ============================================
// COURSE MANAGEMENT
// ============================================

func (gw *Gateway) GetCourses(c *gin.Context) {
	teacherID := c.Query("teacher_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.GetCourses(ctx, &teacherpb.GetCoursesRequest{
		TeacherId: teacherID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) GetCourse(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.GetCourse(ctx, &teacherpb.GetCourseRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Course not found"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) UpdateCourse(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.UpdateCourse(ctx, &teacherpb.UpdateCourseRequest{
		Id:          id,
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) DeleteCourse(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.DeleteCourse(ctx, &teacherpb.DeleteCourseRequest{Id: id})
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

// ============================================
// ENROLLMENT
// ============================================

func (gw *Gateway) EnrollStudent(c *gin.Context) {
	var req struct {
		StudentID string `json:"student_id"`
		CourseID  string `json:"course_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.EnrollStudent(ctx, &teacherpb.EnrollStudentRequest{
		StudentId: req.StudentID,
		CourseId:  req.CourseID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !resp.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": resp.Message})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (gw *Gateway) GetStudentCoursesByID(c *gin.Context) {
	studentID := c.Param("id")
	log.Println(studentID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.StudentClient.GetStudentCourses(ctx, &studentpb.GetStudentCoursesRequest{
		StudentId: studentID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ============================================
// GRADEBOOK
// ============================================

func (gw *Gateway) GetCourseGrades(c *gin.Context) {
	courseID := c.Param("course_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.GetCourseGrades(ctx, &teacherpb.GetCourseGradesRequest{
		CourseId: courseID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ============================================
// REPORTING
// ============================================

func (gw *Gateway) GetTeacherDashboard(c *gin.Context) {
	teacherID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.TeacherClient.GetTeacherDashboard(ctx, &teacherpb.GetTeacherDashboardRequest{
		TeacherId: teacherID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
