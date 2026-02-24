package web

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	statspb "github.com/noxturnedev/lms-monorepo/proto/stats"
	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
)

// ============================================
// PERFORMANCE DISTRIBUTION (BELL CURVE)
// ============================================

func (gw *Gateway) GetPerformanceDistribution(c *gin.Context) {
	courseID := c.Param("id")
	assignmentID := c.Query("assignment_id") // Optional

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := gw.StatsClient.GetPerformanceDistribution(ctx, &statspb.PerformanceDistributionRequest{
		CourseId:     courseID,
		AssignmentId: assignmentID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ============================================
// AT-RISK STUDENTS (per-course)
// ============================================

func (gw *Gateway) GetAtRiskStudents(c *gin.Context) {
	courseID := c.Param("id")

	// Optional query parameters
	missingThreshold := int32(3)
	if val := c.Query("missing_threshold"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			missingThreshold = int32(parsed)
		}
	}

	stdDevThreshold := 2.0
	if val := c.Query("std_dev_threshold"); val != "" {
		if parsed, err := strconv.ParseFloat(val, 64); err == nil {
			stdDevThreshold = parsed
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := gw.StatsClient.GetAtRiskStudents(ctx, &statspb.AtRiskStudentsRequest{
		CourseId:                   courseID,
		MissingAssignmentThreshold: missingThreshold,
		StdDeviationThreshold:      stdDevThreshold,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Enrich with student names from student service
	for _, student := range resp.AtRiskStudents {
		studentResp, err := gw.StudentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{
			Id: student.StudentId,
		})
		if err == nil {
			student.StudentName = studentResp.FullName
			student.StudentNumber = studentResp.StudentNumber
		}
	}

	c.JSON(http.StatusOK, resp)
}

// ============================================
// CATEGORY MASTERY
// ============================================

func (gw *Gateway) GetCategoryMastery(c *gin.Context) {
	courseID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := gw.StatsClient.GetCategoryMastery(ctx, &statspb.CategoryMasteryRequest{
		CourseId: courseID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ============================================
// COURSE STATS OVERVIEW
// ============================================

func (gw *Gateway) GetCourseStats(c *gin.Context) {
	courseID := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := gw.StatsClient.GetCourseStats(ctx, &statspb.CourseStatsRequest{
		CourseId: courseID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ============================================
// EARLY WARNING SYSTEM (all courses)
// ============================================

func (gw *Gateway) GetStudentsAtRisk(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := gw.StatsClient.GetStudentsAtRisk(ctx, &statspb.StudentsAtRiskRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Enrich with student names from student service
	for _, risk := range resp.AtRiskStudents {
		studentResp, err := gw.StudentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{
			Id: risk.StudentId,
		})
		if err == nil {
			risk.StudentName = studentResp.FullName
		}
	}

	c.JSON(http.StatusOK, resp)
}

// ============================================
// ENROLLMENT FORECASTING (Linear Regression)
// ============================================

func (gw *Gateway) ForecastEnrollment(c *gin.Context) {
	forecastYears := int32(1)
	if val := c.Query("years"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			forecastYears = int32(parsed)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := gw.StatsClient.ForecastEnrollment(ctx, &statspb.EnrollmentForecastRequest{
		ForecastYears: forecastYears,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
