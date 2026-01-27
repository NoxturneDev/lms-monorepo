package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/noxturnedev/lms-monorepo/gateway/internal/web"
	"github.com/noxturnedev/lms-monorepo/gateway/utils"
	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

func main() {
	shutdown := utils.InitTracer()
	defer shutdown(context.Background())

	utils.InitBreakers()

	studentConn, err := grpc.NewClient(
		getEnv("STUDENT_SERVICE_URL", "localhost:8082"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to Student Service: %v", err)
	}
	defer studentConn.Close()

	teacherConn, err := grpc.NewClient(
		getEnv("TEACHER_SERVICE_URL", "localhost:8081"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to Teacher Service: %v", err)
	}
	defer teacherConn.Close()

	gw := web.NewGateway(
		studentpb.NewStudentServiceClient(studentConn),
		teacherpb.NewTeacherServiceClient(teacherConn),
	)

	r := gin.Default()
	r.Use(otelgin.Middleware("gateway"))

	api := r.Group("/api/v1")
	{
		// Student CRUD
		api.POST("/students", gw.CreateStudent)
		api.GET("/students", gw.GetAllStudents)
		api.GET("/students/:id", gw.GetStudentDetails)
		api.PUT("/students/:id", gw.UpdateStudent)
		api.DELETE("/students/:id", gw.DeleteStudent)
		api.GET("/students/:id/report-card", gw.GetStudentReportCard)
		api.GET("/students/:id/courses", gw.GetStudentCoursesByID)

		// Teacher CRUD
		api.POST("/teachers", gw.CreateTeacher)
		api.GET("/teachers", gw.ListTeachers)
		api.GET("/teachers/:id", gw.GetTeacher)
		api.PUT("/teachers/:id", gw.UpdateTeacher)
		api.DELETE("/teachers/:id", gw.DeleteTeacher)

		// Course Management
		api.POST("/courses", gw.CreateCourse)
		api.GET("/courses", gw.GetCourses)

		// api.GET("/courses/:course_id/grades", gw.GetCourseGrades)
		api.GET("/courses/:id", gw.GetCourse)
		api.PUT("/courses/:id", gw.UpdateCourse)
		api.DELETE("/courses/:id", gw.DeleteCourse)

		// Enrollment
		api.POST("/enrollments", gw.EnrollStudent)
		// Grading
		api.POST("/grades", gw.AssignGrade)

		// Reporting
		api.GET("/dashboard/teacher/:id", gw.GetTeacherDashboard)
	}

	log.Println("API Gateway running on port 3000")
	r.Run(":3000")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
