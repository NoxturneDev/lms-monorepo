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
	schoolpb "github.com/noxturnedev/lms-monorepo/proto/school"
	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type server struct {
	teacherpb.UnimplementedTeacherServiceServer
	db            *sql.DB
	studentClient studentpb.StudentServiceClient
	schoolClient  schoolpb.SchoolServiceClient
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

func (s *server) AssignGrade(ctx context.Context, req *teacherpb.AssignGradeRequest) (*teacherpb.GradeResponse, error) {
	log.Printf("Assigning Grade for Student %v on Assignment %v", req.StudentId, req.AssignmentId)

	// 1. Validate student exists via Student Service
	log.Println("Validating student with Student Service...")
	_, err := s.studentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{
		Id: req.StudentId,
	})
	if err != nil {
		log.Printf("Student validation failed: %v", err)
		return nil, fmt.Errorf("student not found: %v", err)
	}
	log.Println("Student verified! Proceeding to save grade.")

	// 2. Look up the assignment and validate score <= max_score
	var courseID string
	var maxScore int32
	err = s.db.QueryRow("SELECT course_id, max_score FROM assignments WHERE id = $1", req.AssignmentId).Scan(&courseID, &maxScore)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("assignment not found")
		}
		return nil, err
	}

	if req.Score > maxScore {
		return nil, fmt.Errorf("score %d exceeds max score %d for this assignment", req.Score, maxScore)
	}

	// 3. Save to DB (enrollment validation is now handled by school service)
	query := `INSERT INTO grades (assignment_id, student_id, score) VALUES ($1, $2, $3) RETURNING id`
	var id string
	err = s.db.QueryRow(query, req.AssignmentId, req.StudentId, req.Score).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to assign grade: %v", err)
	}

	return &teacherpb.GradeResponse{Id: id, Success: true}, nil
}

func (s *server) GetStudentGrades(ctx context.Context, req *teacherpb.GetStudentGradesRequest) (*teacherpb.StudentGradesResponse, error) {
	log.Printf("Fetching grades for student: %v", req.StudentId)

	query := `
		SELECT g.score, a.title, a.max_score, a.id, a.course_id
		FROM grades g
		JOIN assignments a ON g.assignment_id = a.id
		WHERE g.student_id = $1
	`

	rows, err := s.db.Query(query, req.StudentId)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch grades: %v", err)
	}
	defer rows.Close()

	var gradeList []*teacherpb.GradeItem
	courseCache := make(map[string]string) // Cache course titles

	for rows.Next() {
		var assignmentTitle, assignmentID, courseID string
		var score, maxScore int32
		if err := rows.Scan(&score, &assignmentTitle, &maxScore, &assignmentID, &courseID); err != nil {
			continue
		}

		// Get course title from cache or school service
		courseTitle, exists := courseCache[courseID]
		if !exists {
			courseResp, err := s.schoolClient.GetCourse(ctx, &schoolpb.GetCourseRequest{Id: courseID})
			if err == nil {
				courseTitle = courseResp.Title
				courseCache[courseID] = courseTitle
			} else {
				courseTitle = "Unknown Course"
			}
		}

		gradeList = append(gradeList, &teacherpb.GradeItem{
			CourseTitle:     courseTitle,
			Score:           score,
			AssignmentTitle: assignmentTitle,
			MaxScore:        maxScore,
			AssignmentId:    assignmentID,
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
// ASSIGNMENT MANAGEMENT
// ============================================

func (s *server) CreateAssignment(ctx context.Context, req *teacherpb.CreateAssignmentRequest) (*teacherpb.AssignmentResponse, error) {
	log.Printf("Creating Assignment: %v for course %v", req.Title, req.CourseId)

	// Validate course exists via school service
	courseResp, err := s.schoolClient.ValidateCourseExists(ctx, &schoolpb.ValidateCourseRequest{
		CourseId: req.CourseId,
	})
	if err != nil || !courseResp.Exists {
		return nil, fmt.Errorf("course not found or school service unavailable")
	}

	maxScore := req.MaxScore
	if maxScore <= 0 {
		maxScore = 100
	}

	query := `INSERT INTO assignments (course_id, title, description, max_score) VALUES ($1, $2, $3, $4) RETURNING id`
	var id string
	err = s.db.QueryRow(query, req.CourseId, req.Title, req.Description, maxScore).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create assignment: %v", err)
	}

	return &teacherpb.AssignmentResponse{Id: id, Title: req.Title, MaxScore: maxScore}, nil
}

func (s *server) GetAssignment(ctx context.Context, req *teacherpb.GetAssignmentRequest) (*teacherpb.AssignmentDetailResponse, error) {
	log.Printf("Getting Assignment: %v", req.Id)

	query := `SELECT id, course_id, title, description, max_score FROM assignments WHERE id = $1`

	var assignment teacherpb.AssignmentDetailResponse
	err := s.db.QueryRow(query, req.Id).Scan(
		&assignment.Id, &assignment.CourseId,
		&assignment.Title, &assignment.Description, &assignment.MaxScore,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("assignment not found")
		}
		return nil, err
	}

	// Get course details from school service
	courseResp, err := s.schoolClient.GetCourse(ctx, &schoolpb.GetCourseRequest{
		Id: assignment.CourseId,
	})
	if err == nil {
		assignment.CourseTitle = courseResp.Title
	}

	return &assignment, nil
}

func (s *server) UpdateAssignment(ctx context.Context, req *teacherpb.UpdateAssignmentRequest) (*teacherpb.AssignmentResponse, error) {
	log.Printf("Updating Assignment: %v", req.Id)

	maxScore := req.MaxScore
	if maxScore <= 0 {
		maxScore = 100
	}

	query := `UPDATE assignments SET title = $1, description = $2, max_score = $3 WHERE id = $4 RETURNING id, title, max_score`
	var assignment teacherpb.AssignmentResponse
	err := s.db.QueryRow(query, req.Title, req.Description, maxScore, req.Id).Scan(&assignment.Id, &assignment.Title, &assignment.MaxScore)
	if err != nil {
		return nil, fmt.Errorf("failed to update assignment: %v", err)
	}
	return &assignment, nil
}

func (s *server) DeleteAssignment(ctx context.Context, req *teacherpb.DeleteAssignmentRequest) (*teacherpb.DeleteAssignmentResponse, error) {
	log.Printf("Deleting Assignment: %v", req.Id)

	// Check for existing grades
	var gradeCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM grades WHERE assignment_id = $1", req.Id).Scan(&gradeCount)
	if err != nil {
		return nil, err
	}

	if gradeCount > 0 {
		return &teacherpb.DeleteAssignmentResponse{
			Success: false,
			Message: fmt.Sprintf("Cannot delete assignment with %d existing grades", gradeCount),
		}, nil
	}

	_, err = s.db.Exec("DELETE FROM assignments WHERE id = $1", req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete assignment: %v", err)
	}
	return &teacherpb.DeleteAssignmentResponse{Success: true, Message: "Assignment deleted successfully"}, nil
}

func (s *server) ListAssignments(ctx context.Context, req *teacherpb.ListAssignmentsRequest) (*teacherpb.ListAssignmentsResponse, error) {
	log.Printf("Listing assignments for course: %v", req.CourseId)

	query := `SELECT id, course_id, title, description, max_score FROM assignments WHERE course_id = $1`

	rows, err := s.db.Query(query, req.CourseId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get course details from school service
	var courseTitle string
	courseResp, err := s.schoolClient.GetCourse(ctx, &schoolpb.GetCourseRequest{
		Id: req.CourseId,
	})
	if err == nil {
		courseTitle = courseResp.Title
	}

	var assignments []*teacherpb.AssignmentDetailResponse
	for rows.Next() {
		var a teacherpb.AssignmentDetailResponse
		if err := rows.Scan(&a.Id, &a.CourseId, &a.Title, &a.Description, &a.MaxScore); err != nil {
			continue
		}
		a.CourseTitle = courseTitle
		assignments = append(assignments, &a)
	}
	return &teacherpb.ListAssignmentsResponse{Assignments: assignments}, nil
}

// ============================================
// VALIDATION (used by School Service)
// ============================================

func (s *server) ValidateCourseHasAssignments(ctx context.Context, req *teacherpb.ValidateCourseAssignmentsRequest) (*teacherpb.ValidateCourseAssignmentsResponse, error) {
	var count int32
	err := s.db.QueryRow("SELECT COUNT(*) FROM assignments WHERE course_id = $1", req.CourseId).Scan(&count)
	if err != nil {
		return nil, err
	}
	return &teacherpb.ValidateCourseAssignmentsResponse{
		HasAssignments:  count > 0,
		AssignmentCount: count,
	}, nil
}

// ============================================
// STUDENT COURSE GRADE
// ============================================

func (s *server) GetStudentCourseGrade(ctx context.Context, req *teacherpb.GetStudentCourseGradeRequest) (*teacherpb.StudentCourseGradeResponse, error) {
	log.Printf("Getting student course grade: student=%v course=%v", req.StudentId, req.CourseId)

	// Get course title from school service
	courseResp, err := s.schoolClient.GetCourse(ctx, &schoolpb.GetCourseRequest{
		Id: req.CourseId,
	})
	if err != nil {
		return nil, fmt.Errorf("course not found or school service unavailable")
	}

	// Get per-assignment breakdown
	query := `
		SELECT a.id, a.title, g.score, a.max_score
		FROM grades g
		JOIN assignments a ON g.assignment_id = a.id
		WHERE a.course_id = $1 AND g.student_id = $2
	`
	rows, err := s.db.Query(query, req.CourseId, req.StudentId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*teacherpb.AssignmentGradeItem
	var totalScore, totalMaxScore int32

	for rows.Next() {
		var item teacherpb.AssignmentGradeItem
		if err := rows.Scan(&item.AssignmentId, &item.AssignmentTitle, &item.Score, &item.MaxScore); err != nil {
			continue
		}
		totalScore += item.Score
		totalMaxScore += item.MaxScore
		items = append(items, &item)
	}

	var overallScore float64
	if totalMaxScore > 0 {
		overallScore = float64(totalScore) / float64(totalMaxScore) * 100
	}

	return &teacherpb.StudentCourseGradeResponse{
		CourseId:      req.CourseId,
		CourseTitle:   courseResp.Title,
		StudentId:     req.StudentId,
		OverallScore:  overallScore,
		TotalScore:    totalScore,
		TotalMaxScore: totalMaxScore,
		Assignments:   items,
	}, nil
}

// ============================================
// GRADEBOOK
// ============================================
func (s *server) GetCourseGrades(ctx context.Context, req *teacherpb.GetCourseGradesRequest) (*teacherpb.CourseGradesResponse, error) {
	log.Printf("Getting grades for course: %v", req.CourseId)

	// Get course title from school service
	courseResp, err := s.schoolClient.GetCourse(ctx, &schoolpb.GetCourseRequest{
		Id: req.CourseId,
	})
	if err != nil {
		return nil, fmt.Errorf("course not found or school service unavailable")
	}

	query := `SELECT g.id, g.student_id, g.score, a.title, a.max_score, a.id
			FROM grades g
			JOIN assignments a ON g.assignment_id = a.id
			WHERE a.course_id = $1`
	rows, err := s.db.Query(query, req.CourseId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grades []*teacherpb.StudentGradeItem
	for rows.Next() {
		var gradeId, studentId, assignmentTitle, assignmentId string
		var score, maxScore int32
		if err := rows.Scan(&gradeId, &studentId, &score, &assignmentTitle, &maxScore, &assignmentId); err != nil {
			continue
		}

		studentResp, err := s.studentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{Id: studentId})
		if err != nil {
			grades = append(grades, &teacherpb.StudentGradeItem{
				GradeId:         gradeId,
				StudentId:       studentId,
				StudentName:     "Unknown",
				StudentNumber:   "N/A",
				Score:           score,
				AssignmentTitle: assignmentTitle,
				MaxScore:        maxScore,
				AssignmentId:    assignmentId,
			})
			continue
		}

		grades = append(grades, &teacherpb.StudentGradeItem{
			GradeId:         gradeId,
			StudentId:       studentId,
			StudentName:     studentResp.FullName,
			StudentNumber:   studentResp.StudentNumber,
			Score:           score,
			AssignmentTitle: assignmentTitle,
			MaxScore:        maxScore,
			AssignmentId:    assignmentId,
		})
	}

	return &teacherpb.CourseGradesResponse{
		CourseId:    req.CourseId,
		CourseTitle: courseResp.Title,
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

	// Get courses assigned to this teacher from school service
	// Note: This requires the school service to provide a method to get courses by teacher
	// For now, we'll return basic teacher info without course statistics
	// This functionality should be implemented in school service

	return &teacherpb.TeacherDashboardResponse{
		TeacherId:             req.TeacherId,
		TeacherName:           teacherName,
		TotalCourses:          0,
		TotalStudentsEnrolled: 0,
		Courses:               []*teacherpb.CourseSummary{},
	}, nil
}

func (s *server) GetTeacherCourseList(ctx context.Context, req *teacherpb.GetTeacherCourseList) (*teacherpb.GetTeacherCourseListResponse, error)

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

	studentConn, err := grpc.NewClient(
		studentServiceUrl,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("did not connect to student service: %v", err)
	}
	defer studentConn.Close()

	studentClient := studentpb.NewStudentServiceClient(studentConn)

	schoolServiceUrl := os.Getenv("SCHOOL_SERVICE_URL")
	if schoolServiceUrl == "" {
		schoolServiceUrl = "localhost:8083"
	}

	schoolConn, err := grpc.NewClient(
		schoolServiceUrl,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("did not connect to school service: %v", err)
	}
	defer schoolConn.Close()

	schoolClient := schoolpb.NewSchoolServiceClient(schoolConn)

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
		schoolClient:  schoolClient,
	})

	startEventConsumer(db)
	reflection.Register(s)

	log.Println("Teacher Service listening on port 8080...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
