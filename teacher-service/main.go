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
	_, err := s.studentClient.GetStudent(ctx, &studentpb.GetStudentRequest{
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

func main() {
	// 1. DB Connection
	dbHost := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", dbHost)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	// 2. NEW: Connect to Student Service
	studentServiceUrl := os.Getenv("STUDENT_SERVICE_URL")
	if studentServiceUrl == "" {
		studentServiceUrl = "localhost:8082" // Fallback for local dev without Docker
	}

	// We use "WithTransportCredentials(insecure.NewCredentials())" because we don't have SSL certs
	conn, err := grpc.NewClient(studentServiceUrl, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect to student service: %v", err)
	}
	defer conn.Close()

	// Create the Client
	studentClient := studentpb.NewStudentServiceClient(conn)

	// 3. Listener
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()

	// Register Server with the Client injected
	teacherpb.RegisterTeacherServiceServer(s, &server{
		db:            db,
		studentClient: studentClient, // Inject client here
	})

	startEventConsumer(db)
	reflection.Register(s)

	log.Println("Teacher Service listening on port 8080...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
