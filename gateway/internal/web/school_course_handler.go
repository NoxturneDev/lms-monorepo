package web

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	schoolpb "github.com/noxturnedev/lms-monorepo/proto/school"
)

// ============================================
// COURSE MANAGEMENT (uses SchoolClient)
// ============================================

func (gw *Gateway) CreateCourse(c *gin.Context) {
	var req struct {
		SchoolID    string `json:"school_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.CreateCourse(ctx, &schoolpb.CreateCourseRequest{
		SchoolId:    req.SchoolID,
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (gw *Gateway) GetCourse(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.GetCourse(ctx, &schoolpb.GetCourseRequest{Id: id})
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

	resp, err := gw.SchoolClient.UpdateCourse(ctx, &schoolpb.UpdateCourseRequest{
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

	resp, err := gw.SchoolClient.DeleteCourse(ctx, &schoolpb.DeleteCourseRequest{Id: id})
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

func (gw *Gateway) ListCourses(c *gin.Context) {
	schoolID := c.Query("school_id") // optional filter

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.ListCourses(ctx, &schoolpb.ListCoursesRequest{
		SchoolId: schoolID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ============================================
// COURSE-TEACHER ASSIGNMENT
// ============================================

func (gw *Gateway) AssignTeacherToCourse(c *gin.Context) {
	courseID := c.Param("id")
	var req struct {
		TeacherID string `json:"teacher_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.AssignTeacherToCourse(ctx, &schoolpb.AssignTeacherToCourseRequest{
		CourseId:  courseID,
		TeacherId: req.TeacherID,
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

func (gw *Gateway) UnassignTeacherFromCourse(c *gin.Context) {
	courseID := c.Param("id")
	teacherID := c.Param("teacher_id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.UnassignTeacherFromCourse(ctx, &schoolpb.UnassignTeacherRequest{
		CourseId:  courseID,
		TeacherId: teacherID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !resp.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": resp.Message})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) GetCourseTeachers(c *gin.Context) {
	courseID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.GetCourseTeachers(ctx, &schoolpb.GetCourseTeachersRequest{
		CourseId: courseID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ============================================
// ENROLLMENT MANAGEMENT
// ============================================

func (gw *Gateway) EnrollStudent(c *gin.Context) {
	var req struct {
		CourseID  string `json:"course_id"`
		StudentID string `json:"student_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.EnrollStudent(ctx, &schoolpb.EnrollStudentRequest{
		CourseId:  req.CourseID,
		StudentId: req.StudentID,
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

func (gw *Gateway) UnenrollStudent(c *gin.Context) {
	var req struct {
		CourseID  string `json:"course_id"`
		StudentID string `json:"student_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.UnenrollStudent(ctx, &schoolpb.UnenrollStudentRequest{
		CourseId:  req.CourseID,
		StudentId: req.StudentID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !resp.Success {
		c.JSON(http.StatusBadRequest, gin.H{"error": resp.Message})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) GetCourseEnrollments(c *gin.Context) {
	courseID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.GetCourseEnrollments(ctx, &schoolpb.GetCourseEnrollmentsRequest{
		CourseId: courseID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) GetStudentEnrollments(c *gin.Context) {
	studentID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.GetStudentEnrollments(ctx, &schoolpb.GetStudentEnrollmentsRequest{
		StudentId: studentID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// ============================================
// TEACHER COURSE LIST
// ============================================

func (gw *Gateway) GetTeacherCourseList(c *gin.Context) {
	teacherID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.SchoolClient.GetTeacherCourseList(ctx, &schoolpb.GetTeacherCourseListRequest{
		TeacherId: teacherID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}
