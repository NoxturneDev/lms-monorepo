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
		// ===== PUBLIC ROUTES (No Auth Required) =====

		// Authentication
		api.POST("/auth/teacher/login", gw.LoginTeacher)
		api.POST("/auth/student/login", gw.LoginStudent)

		// Public registration
		api.POST("/students", gw.CreateStudent)
		api.POST("/teachers", gw.CreateTeacher)

		// ===== PROTECTED ROUTES (Auth Required) =====
		protected := api.Group("")
		protected.Use(web.AuthMiddleware())
		{
			// Student CRUD (Protected)
			protected.GET("/students", gw.GetAllStudents)
			protected.GET("/students/:id", gw.GetStudentDetails)
			protected.PUT("/students/:id", gw.UpdateStudent)
			protected.DELETE("/students/:id", gw.DeleteStudent)
			protected.GET("/students/:id/report-card", gw.GetStudentReportCard)
			protected.GET("/students/:id/courses", gw.GetStudentCoursesByID)

			// Teacher CRUD (Protected)
			protected.GET("/teachers", gw.ListTeachers)
			protected.GET("/teachers/:id", gw.GetTeacher)
			protected.PUT("/teachers/:id", gw.UpdateTeacher)
			protected.DELETE("/teachers/:id", gw.DeleteTeacher)

			// Course Management (Teacher Only)
			teacherRoutes := protected.Group("")
			teacherRoutes.Use(web.TeacherOnly())
			{
				teacherRoutes.POST("/courses", gw.CreateCourse)
				teacherRoutes.PUT("/courses/:id", gw.UpdateCourse)
				teacherRoutes.DELETE("/courses/:id", gw.DeleteCourse)
				teacherRoutes.POST("/grades", gw.AssignGrade)
				// teacherRoutes.GET("/courses/:course_id/grades", gw.GetCourseGrades)
				teacherRoutes.GET("/dashboard/teacher/:id", gw.GetTeacherDashboard)
			}

			// Course Viewing (All authenticated users)
			protected.GET("/courses", gw.GetCourses)
			protected.GET("/courses/:id", gw.GetCourse)

			// Enrollment (All authenticated users)
			protected.POST("/enrollments", gw.EnrollStudent)
		}
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
