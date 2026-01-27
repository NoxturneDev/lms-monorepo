package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure" // IMPORTANT: For non-TLS connections
	"google.golang.org/grpc/reflection"

	// Alias the imports so they don't conflict
	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type server struct {
	teacherpb.UnimplementedTeacherServiceServer
	db *sql.DB
	// NEW: We hold a client to talk to the Student Service
	studentClient studentpb.StudentServiceClient
}

func startEventConsumer(db *sql.DB) {
	// Connect to Rabbit (Copy the retry logic from Student Service)
	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		log.Printf("Consumer: Failed to connect to RabbitMQ: %v", err)
		return
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Consumer: Failed to open channel: %v", err)
		return
	}

	// Declare the SAME queue to ensure it exists
	q, _ := ch.QueueDeclare("student_events", true, false, false, false, nil)

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer tag
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)

	// Background Worker
	go func() {
		log.Println("🐰 RabbitMQ Consumer Started. Listening for events...")
		for d := range msgs {
			log.Printf("Received Event: %s", d.Type)

			if d.Type == "StudentDeleted" {
				var payload map[string]string
				json.Unmarshal(d.Body, &payload)
				studentID := payload["id"]

				log.Printf("⚠️  Orphan Cleanup: Deleting grades for student %s", studentID)

				// EXECUTE CLEANUP
				_, err := db.Exec("DELETE FROM grades WHERE student_id = $1", studentID)
				if err != nil {
					log.Printf("Failed to delete grades: %v", err)
				} else {
					log.Printf("✅ Grades deleted successfully.")
				}
			}
		}
	}()
}

// ... CreateCourse stays the same ...
func (s *server) CreateCourse(ctx context.Context, req *teacherpb.CreateCourseRequest) (*teacherpb.CourseResponse, error) {
	// (Keep your existing code here)
	log.Printf("Creating Course: %v", req.Title)
	query := `INSERT INTO courses (teacher_id, title, description) VALUES ($1, $2, $3) RETURNING id`
	var id string
	err := s.db.QueryRow(query, req.TeacherId, req.Title, req.Description).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create course: %v", err)
	}
	return &teacherpb.CourseResponse{Id: id, Title: req.Title}, nil
}

// UPDATE THIS FUNCTION
func (s *server) AssignGrade(ctx context.Context, req *teacherpb.AssignGradeRequest) (*teacherpb.GradeResponse, error) {
	log.Printf("Assigning Grade for Student %v in Course %v", req.StudentId, req.CourseId)

	// ---------------------------------------------------------
	// 1. THE CALL (East-West Traffic)
	// ---------------------------------------------------------
	log.Println("Validating student with Student Service...")

	// We call the other microservice just like a local function!
	_, err := s.studentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{
		Id: req.StudentId,
	})
	if err != nil {
		// If Student Service returns error (e.g., "not found"), we reject the request.
		log.Printf("Student validation failed: %v", err)
		return nil, fmt.Errorf("student not found: %v", err)
	}

	log.Println("Student verified! Proceeding to save grade.")

	// ---------------------------------------------------------
	// 2. Save to DB (Only if validation passed)
	// ---------------------------------------------------------
	query := `INSERT INTO grades (course_id, student_id, score) VALUES ($1, $2, $3) RETURNING id`

	var id string
	err = s.db.QueryRow(query, req.CourseId, req.StudentId, req.Score).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to assign grade: %v", err)
	}

	return &teacherpb.GradeResponse{Id: id, Success: true}, nil
}

func (s *server) GetStudentGrades(ctx context.Context, req *teacherpb.GetStudentGradesRequest) (*teacherpb.StudentGradesResponse, error) {
	log.Printf("Fetching grades for student: %v", req.StudentId)

	// A simple JOIN to get the course title along with the score
	query := `
        SELECT c.title, g.score 
        FROM grades g
        JOIN courses c ON g.course_id = c.id
        WHERE g.student_id = $1
    `

	rows, err := s.db.Query(query, req.StudentId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grades: %v", err)
	}
	defer rows.Close()

	var gradeList []*teacherpb.GradeItem

	for rows.Next() {
		var title string
		var score int32
		if err := rows.Scan(&title, &score); err != nil {
			continue
		}
		gradeList = append(gradeList, &teacherpb.GradeItem{
			CourseTitle: title,
			Score:       score,
		})
	}

	return &teacherpb.StudentGradesResponse{Grades: gradeList}, nil
}

// ============================================
// AUTHENTICATION
// ============================================

func (s *server) LoginTeacher(ctx context.Context, req *teacherpb.LoginTeacherRequest) (*teacherpb.LoginTeacherResponse, error) {
	log.Printf("Login attempt for teacher: %v", req.Email)

	query := `SELECT id, email, full_name, password_hash FROM teachers WHERE email = $1`
	var id, email, fullName, passwordHash string
	err := s.db.QueryRow(query, req.Email).Scan(&id, &email, &fullName, &passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return &teacherpb.LoginTeacherResponse{
				Success: false,
				Message: "Invalid email or password",
			}, nil
		}
		return nil, err
	}

	if passwordHash != req.Password {
		return &teacherpb.LoginTeacherResponse{
			Success: false,
			Message: "Invalid email or password",
		}, nil
	}

	return &teacherpb.LoginTeacherResponse{
		Success:   true,
		Message:   "Login successful",
		TeacherId: id,
		Email:     email,
		FullName:  fullName,
	}, nil
}

// ============================================
// TEACHER CRUD
// ============================================

func (s *server) CreateTeacher(ctx context.Context, req *teacherpb.CreateTeacherRequest) (*teacherpb.TeacherResponse, error) {
	log.Printf("Creating Teacher: %v", req.Email)
	query := `INSERT INTO teachers (email, password_hash, full_name) VALUES ($1, $2, $3) RETURNING id`
	var id string
	err := s.db.QueryRow(query, req.Email, req.Password, req.FullName).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create teacher: %v", err)
	}
	return &teacherpb.TeacherResponse{Id: id, Email: req.Email, FullName: req.FullName}, nil
}

func (s *server) GetTeacher(ctx context.Context, req *teacherpb.GetTeacherRequest) (*teacherpb.TeacherResponse, error) {
	log.Printf("Getting Teacher: %v", req.Id)
	query := `SELECT id, email, full_name FROM teachers WHERE id = $1`
	var teacher teacherpb.TeacherResponse
	err := s.db.QueryRow(query, req.Id).Scan(&teacher.Id, &teacher.Email, &teacher.FullName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("teacher not found")
		}
		return nil, err
	}
	return &teacher, nil
}

func (s *server) UpdateTeacher(ctx context.Context, req *teacherpb.UpdateTeacherRequest) (*teacherpb.TeacherResponse, error) {
	log.Printf("Updating Teacher: %v", req.Id)

	if req.Password != "" {
		query := `UPDATE teachers SET email = $1, full_name = $2, password_hash = $3 WHERE id = $4 RETURNING id, email, full_name`
		var teacher teacherpb.TeacherResponse
		err := s.db.QueryRow(query, req.Email, req.FullName, req.Password, req.Id).Scan(&teacher.Id, &teacher.Email, &teacher.FullName)
		if err != nil {
			return nil, fmt.Errorf("failed to update teacher: %v", err)
		}
		return &teacher, nil
	}

	query := `UPDATE teachers SET email = $1, full_name = $2 WHERE id = $3 RETURNING id, email, full_name`
	var teacher teacherpb.TeacherResponse
	err := s.db.QueryRow(query, req.Email, req.FullName, req.Id).Scan(&teacher.Id, &teacher.Email, &teacher.FullName)
	if err != nil {
		return nil, fmt.Errorf("failed to update teacher: %v", err)
	}
	return &teacher, nil
}

func (s *server) DeleteTeacher(ctx context.Context, req *teacherpb.DeleteTeacherRequest) (*teacherpb.DeleteTeacherResponse, error) {
	log.Printf("Deleting Teacher: %v", req.Id)
	_, err := s.db.Exec("DELETE FROM teachers WHERE id = $1", req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete teacher: %v", err)
	}
	return &teacherpb.DeleteTeacherResponse{Success: true}, nil
}

func (s *server) ListTeachers(ctx context.Context, req *teacherpb.ListTeachersRequest) (*teacherpb.ListTeachersResponse, error) {
	log.Println("Listing all teachers")
	query := `SELECT id, email, full_name FROM teachers`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teachers []*teacherpb.TeacherResponse
	for rows.Next() {
		var teacher teacherpb.TeacherResponse
		if err := rows.Scan(&teacher.Id, &teacher.Email, &teacher.FullName); err != nil {
			continue
		}
		teachers = append(teachers, &teacher)
	}
	return &teacherpb.ListTeachersResponse{Teachers: teachers}, nil
}

// ============================================
// COURSE MANAGEMENT
// ============================================

func (s *server) GetCourses(ctx context.Context, req *teacherpb.GetCoursesRequest) (*teacherpb.GetCoursesResponse, error) {
	log.Printf("Getting courses, filter: %v", req.TeacherId)

	var query string
	var rows *sql.Rows
	var err error

	if req.TeacherId != "" {
		query = `SELECT c.id, c.teacher_id, c.title, c.description, t.full_name
				FROM courses c
				JOIN teachers t ON c.teacher_id = t.id
				WHERE c.teacher_id = $1`
		rows, err = s.db.Query(query, req.TeacherId)
	} else {
		query = `SELECT c.id, c.teacher_id, c.title, c.description, t.full_name
				FROM courses c
				JOIN teachers t ON c.teacher_id = t.id`
		rows, err = s.db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []*teacherpb.CourseDetailResponse
	for rows.Next() {
		var course teacherpb.CourseDetailResponse
		if err := rows.Scan(&course.Id, &course.TeacherId, &course.Title, &course.Description, &course.TeacherName); err != nil {
			continue
		}
		courses = append(courses, &course)
	}
	return &teacherpb.GetCoursesResponse{Courses: courses}, nil
}

func (s *server) GetCourse(ctx context.Context, req *teacherpb.GetCourseRequest) (*teacherpb.CourseDetailResponse, error) {
	log.Printf("Getting course: %v", req.Id)
	query := `SELECT c.id, c.teacher_id, c.title, c.description, t.full_name
			FROM courses c
			JOIN teachers t ON c.teacher_id = t.id
			WHERE c.id = $1`

	var course teacherpb.CourseDetailResponse
	err := s.db.QueryRow(query, req.Id).Scan(&course.Id, &course.TeacherId, &course.Title, &course.Description, &course.TeacherName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("course not found")
		}
		return nil, err
	}
	return &course, nil
}

func (s *server) UpdateCourse(ctx context.Context, req *teacherpb.UpdateCourseRequest) (*teacherpb.CourseResponse, error) {
	log.Printf("Updating course: %v", req.Id)
	query := `UPDATE courses SET title = $1, description = $2 WHERE id = $3 RETURNING id, title`
	var course teacherpb.CourseResponse
	err := s.db.QueryRow(query, req.Title, req.Description, req.Id).Scan(&course.Id, &course.Title)
	if err != nil {
		return nil, fmt.Errorf("failed to update course: %v", err)
	}
	return &course, nil
}

func (s *server) DeleteCourse(ctx context.Context, req *teacherpb.DeleteCourseRequest) (*teacherpb.DeleteCourseResponse, error) {
	log.Printf("Deleting course: %v", req.Id)

	var enrollmentCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE course_id = $1", req.Id).Scan(&enrollmentCount)
	if err != nil {
		return nil, err
	}

	if enrollmentCount > 0 {
		return &teacherpb.DeleteCourseResponse{
			Success: false,
			Message: fmt.Sprintf("Cannot delete course with %d enrolled students", enrollmentCount),
		}, nil
	}

	_, err = s.db.Exec("DELETE FROM courses WHERE id = $1", req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete course: %v", err)
	}
	return &teacherpb.DeleteCourseResponse{Success: true, Message: "Course deleted successfully"}, nil
}

// ============================================
// ENROLLMENT
// ============================================

func (s *server) EnrollStudent(ctx context.Context, req *teacherpb.EnrollStudentRequest) (*teacherpb.EnrollmentResponse, error) {
	log.Printf("Enrolling student %v in course %v", req.StudentId, req.CourseId)

	_, err := s.studentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{Id: req.StudentId})
	if err != nil {
		return &teacherpb.EnrollmentResponse{Success: false, Message: "Student not found"}, nil
	}

	var courseExists bool
	err = s.db.QueryRow("SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1)", req.CourseId).Scan(&courseExists)
	if err != nil || !courseExists {
		return &teacherpb.EnrollmentResponse{Success: false, Message: "Course not found"}, nil
	}

	query := `INSERT INTO enrollments (student_id, course_id) VALUES ($1, $2) RETURNING id`
	var id string
	err = s.db.QueryRow(query, req.StudentId, req.CourseId).Scan(&id)
	if err != nil {
		return &teacherpb.EnrollmentResponse{Success: false, Message: "Already enrolled or enrollment failed"}, nil
	}

	return &teacherpb.EnrollmentResponse{Id: id, Success: true, Message: "Enrolled successfully"}, nil
}

func (s *server) GetCourseEnrollments(ctx context.Context, req *teacherpb.GetCourseEnrollmentsRequest) (*teacherpb.GetCourseEnrollmentsResponse, error) {
	log.Printf("Getting enrollments for course: %v", req.CourseId)

	query := `SELECT student_id FROM enrollments WHERE course_id = $1`
	rows, err := s.db.Query(query, req.CourseId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var enrollments []*teacherpb.EnrollmentItem
	for rows.Next() {
		var studentId string
		if err := rows.Scan(&studentId); err != nil {
			continue
		}

		studentResp, err := s.studentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{Id: studentId})
		if err != nil {
			continue
		}

		enrollments = append(enrollments, &teacherpb.EnrollmentItem{
			StudentId:     studentId,
			StudentName:   studentResp.FullName,
			StudentNumber: studentResp.StudentNumber,
		})
	}
	return &teacherpb.GetCourseEnrollmentsResponse{Enrollments: enrollments}, nil
}

// ============================================
// GRADEBOOK
// ============================================

func (s *server) GetCourseGrades(ctx context.Context, req *teacherpb.GetCourseGradesRequest) (*teacherpb.CourseGradesResponse, error) {
	log.Printf("Getting grades for course: %v", req.CourseId)

	var courseTitle string
	err := s.db.QueryRow("SELECT title FROM courses WHERE id = $1", req.CourseId).Scan(&courseTitle)
	if err != nil {
		return nil, fmt.Errorf("course not found")
	}

	query := `SELECT g.id, g.student_id, g.score FROM grades g WHERE g.course_id = $1`
	rows, err := s.db.Query(query, req.CourseId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grades []*teacherpb.StudentGradeItem
	for rows.Next() {
		var gradeId, studentId string
		var score int32
		if err := rows.Scan(&gradeId, &studentId, &score); err != nil {
			continue
		}

		studentResp, err := s.studentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{Id: studentId})
		if err != nil {
			grades = append(grades, &teacherpb.StudentGradeItem{
				GradeId:       gradeId,
				StudentId:     studentId,
				StudentName:   "Unknown",
				StudentNumber: "N/A",
				Score:         score,
			})
			continue
		}

		grades = append(grades, &teacherpb.StudentGradeItem{
			GradeId:       gradeId,
			StudentId:     studentId,
			StudentName:   studentResp.FullName,
			StudentNumber: studentResp.StudentNumber,
			Score:         score,
		})
	}

	return &teacherpb.CourseGradesResponse{
		CourseId:    req.CourseId,
		CourseTitle: courseTitle,
		Grades:      grades,
	}, nil
}

// ============================================
// DASHBOARD
// ============================================

func (s *server) GetTeacherDashboard(ctx context.Context, req *teacherpb.GetTeacherDashboardRequest) (*teacherpb.TeacherDashboardResponse, error) {
	log.Printf("Getting dashboard for teacher: %v", req.TeacherId)

	var teacherName string
	err := s.db.QueryRow("SELECT full_name FROM teachers WHERE id = $1", req.TeacherId).Scan(&teacherName)
	if err != nil {
		return nil, fmt.Errorf("teacher not found")
	}

	var totalCourses int32
	err = s.db.QueryRow("SELECT COUNT(*) FROM courses WHERE teacher_id = $1", req.TeacherId).Scan(&totalCourses)
	if err != nil {
		return nil, err
	}

	var totalStudents int32
	query := `SELECT COUNT(DISTINCT e.student_id)
			FROM enrollments e
			JOIN courses c ON e.course_id = c.id
			WHERE c.teacher_id = $1`
	err = s.db.QueryRow(query, req.TeacherId).Scan(&totalStudents)
	if err != nil {
		return nil, err
	}

	courseSummaryQuery := `
		SELECT c.id, c.title, COUNT(e.student_id) as enrolled_count
		FROM courses c
		LEFT JOIN enrollments e ON c.id = e.course_id
		WHERE c.teacher_id = $1
		GROUP BY c.id, c.title`

	rows, err := s.db.Query(courseSummaryQuery, req.TeacherId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courseSummaries []*teacherpb.CourseSummary
	for rows.Next() {
		var summary teacherpb.CourseSummary
		if err := rows.Scan(&summary.CourseId, &summary.Title, &summary.EnrolledCount); err != nil {
			continue
		}
		courseSummaries = append(courseSummaries, &summary)
	}

	return &teacherpb.TeacherDashboardResponse{
		TeacherId:              req.TeacherId,
		TeacherName:            teacherName,
		TotalCourses:           totalCourses,
		TotalStudentsEnrolled:  totalStudents,
		Courses:                courseSummaries,
	}, nil
}

func main() {
	shutdown := initTracer()
	defer shutdown(context.Background())

	dbHost := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", dbHost)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	studentServiceUrl := os.Getenv("STUDENT_SERVICE_URL")
	if studentServiceUrl == "" {
		studentServiceUrl = "localhost:8082" // Fallback for local dev without Docker
	}

	conn, err := grpc.NewClient(
		studentServiceUrl,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("did not connect to student service: %v", err)
	}
	defer conn.Close()

	studentClient := studentpb.NewStudentServiceClient(conn)

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	teacherpb.RegisterTeacherServiceServer(s, &server{
		db:            db,
		studentClient: studentClient,
	})

	startEventConsumer(db)
	reflection.Register(s)

	log.Println("Teacher Service listening on port 8080...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
