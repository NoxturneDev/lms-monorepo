package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sony/gobreaker"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type Gateway struct {
	studentClient studentpb.StudentServiceClient
	teacherClient teacherpb.TeacherServiceClient
}

func main() {
	// 1. Connect to Student Service
	shutdown := initTracer()
	defer shutdown(context.Background())

	initBreakers()

	studentConn, err := grpc.NewClient(
		getEnv("STUDENT_SERVICE_URL", "localhost:8082"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()), // <--- NEW
	)
	if err != nil {
		log.Fatalf("Failed to connect to Student Service: %v", err)
	}
	defer studentConn.Close()

	teacherConn, err := grpc.NewClient(
		getEnv("TEACHER_SERVICE_URL", "localhost:8081"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()), // <--- NEW
	)
	if err != nil {
		log.Fatalf("Failed to connect to Teacher Service: %v", err)
	}
	defer teacherConn.Close()

	gw := &Gateway{
		studentClient: studentpb.NewStudentServiceClient(studentConn),
		teacherClient: teacherpb.NewTeacherServiceClient(teacherConn),
	}

	r := gin.Default()
	r.Use(otelgin.Middleware("gateway")) // <--- NEW

	api := r.Group("/api/v1")
	{
		api.POST("/students", gw.CreateStudent)
		api.DELETE("/students/:id", gw.DeleteStudent)

		api.POST("/courses", gw.CreateCourse)

		api.POST("/grades", gw.AssignGrade)

		api.GET("/students", gw.GetAllStudents)
		api.GET("/students/:id", gw.GetStudentDetails)
		api.GET("/students/:id/report-card", gw.GetStudentReportCard)
	}

	log.Println("API Gateway running on port 3000")
	r.Run(":3000")
}

// --- HANDLERS ---
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

	resp, err := gw.studentClient.GetAllStudents(ctx, &studentpb.ListStudentRequest{
		ClassId: req.ClassID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

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

	// Call gRPC
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := gw.studentClient.CreateStudent(ctx, &studentpb.CreateStudentRequest{
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

	_, err := gw.studentClient.DeleteStudent(ctx, &studentpb.DeleteStudentRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student deleted and cleanup scheduled"})
}

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

	resp, err := gw.teacherClient.CreateCourse(ctx, &teacherpb.CreateCourseRequest{
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

	resp, err := gw.teacherClient.AssignGrade(ctx, &teacherpb.AssignGradeRequest{
		TeacherId: req.TeacherID,
		CourseId:  req.CourseID,
		StudentId: req.StudentID,
		Score:     req.Score,
	})
	if err != nil {
		// This will catch the "Student Not Found" error we added earlier!
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (gw *Gateway) GetStudentReportCard(c *gin.Context) {
	studentID := c.Param("id")
	g, ctx := errgroup.WithContext(c.Request.Context())

	var (
		studentResp *studentpb.StudentResponse
		gradesResp  *teacherpb.StudentGradesResponse
	)

	// --- Routine 1: Fetch Student (PROTECTED) ---
	g.Go(func() error {
		result, err := executeWithBreaker(studentBreaker, func() (interface{}, error) {
			return gw.studentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{Id: studentID})
		})
		if err != nil {
			return err
		}

		studentResp = result.(*studentpb.StudentResponse)
		return nil
	})

	g.Go(func() error {
		result, err := executeWithBreaker(teacherBreaker, func() (interface{}, error) {
			return gw.teacherClient.GetStudentGrades(ctx, &teacherpb.GetStudentGradesRequest{StudentId: studentID})
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
		"academic_record": gradesResp.Grades, // The list from Teacher Service
		"generated_at":    time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

func (gw *Gateway) GetStudentDetails(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Call Student Service
	student, err := gw.studentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{Id: id})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}

	// In the future, we would ALSO call TeacherService.GetGrades(id) here
	// and merge the JSON. For now, we return the student info.
	c.JSON(http.StatusOK, gin.H{
		"student_profile": student,
		"source":          "Aggregation Gateway",
	})
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
