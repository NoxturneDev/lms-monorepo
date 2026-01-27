package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
	"google.golang.org/grpc"
)

type mockStudentClient struct {
	studentpb.StudentServiceClient
}

func (m *mockStudentClient) CreateStudent(ctx context.Context, req *studentpb.CreateStudentRequest, opts ...grpc.CallOption) (*studentpb.StudentResponse, error) {
	return &studentpb.StudentResponse{
		Id:            "test-id-123",
		Email:         req.Email,
		FullName:      req.FullName,
		StudentNumber: req.StudentNumber,
	}, nil
}

func (m *mockStudentClient) GetAllStudents(ctx context.Context, req *studentpb.ListStudentRequest, opts ...grpc.CallOption) (*studentpb.ListStudentResponse, error) {
	return &studentpb.ListStudentResponse{
		Students: []*studentpb.StudentResponse{
			{
				Id:            "test-id-123",
				Email:         "john@student.edu",
				FullName:      "John Doe",
				StudentNumber: "STD-2026-001",
			},
		},
	}, nil
}

func (m *mockStudentClient) GetStudentById(ctx context.Context, req *studentpb.GetStudentByIdRequest, opts ...grpc.CallOption) (*studentpb.StudentResponse, error) {
	return &studentpb.StudentResponse{
		Id:            req.Id,
		Email:         "john@student.edu",
		FullName:      "John Doe",
		StudentNumber: "STD-2026-001",
	}, nil
}

func (m *mockStudentClient) UpdateStudent(ctx context.Context, req *studentpb.UpdateStudentRequest, opts ...grpc.CallOption) (*studentpb.StudentResponse, error) {
	return &studentpb.StudentResponse{
		Id:            req.Id,
		Email:         req.Email,
		FullName:      req.FullName,
		StudentNumber: req.StudentNumber,
	}, nil
}

func (m *mockStudentClient) DeleteStudent(ctx context.Context, req *studentpb.DeleteStudentRequest, opts ...grpc.CallOption) (*studentpb.DeleteStudentResponse, error) {
	return &studentpb.DeleteStudentResponse{
		Success: true,
	}, nil
}

func (m *mockStudentClient) GetStudentCourses(ctx context.Context, req *studentpb.GetStudentCoursesRequest, opts ...grpc.CallOption) (*studentpb.GetStudentCoursesResponse, error) {
	return &studentpb.GetStudentCoursesResponse{
		Courses: []*studentpb.CourseItem{
			{
				CourseId:    "course-123",
				Title:       "Advanced Algorithms",
				Description: "P vs NP",
				TeacherName: "Alan Turing",
			},
		},
	}, nil
}

type mockTeacherClient struct {
	teacherpb.TeacherServiceClient
}

func (m *mockTeacherClient) GetStudentGrades(ctx context.Context, req *teacherpb.GetStudentGradesRequest, opts ...grpc.CallOption) (*teacherpb.StudentGradesResponse, error) {
	return &teacherpb.StudentGradesResponse{
		Grades: []*teacherpb.GradeItem{
			{
				CourseTitle: "Advanced Algorithms",
				Score:       95,
			},
		},
	}, nil
}

func setupRouter() (*gin.Engine, *Gateway) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	gw := &Gateway{
		StudentClient: &mockStudentClient{},
		TeacherClient: &mockTeacherClient{},
	}

	return r, gw
}

func TestCreateStudent(t *testing.T) {
	r, gw := setupRouter()
	r.POST("/students", gw.CreateStudent)

	reqBody := map[string]string{
		"email":          "john@student.edu",
		"full_name":      "John Doe",
		"password":       "secret",
		"student_number": "STD-2026-001",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/students", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["id"] != "test-id-123" {
		t.Errorf("Expected id test-id-123, got %v", resp["id"])
	}
}

func TestGetAllStudents(t *testing.T) {
	r, gw := setupRouter()
	r.GET("/students", gw.GetAllStudents)

	req, _ := http.NewRequest("GET", "/students", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	students := resp["students"].([]interface{})
	if len(students) != 1 {
		t.Errorf("Expected 1 student, got %d", len(students))
	}
}

func TestGetStudentDetails(t *testing.T) {
	r, gw := setupRouter()
	r.GET("/students/:id", gw.GetStudentDetails)

	req, _ := http.NewRequest("GET", "/students/test-id-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUpdateStudent(t *testing.T) {
	r, gw := setupRouter()
	r.PUT("/students/:id", gw.UpdateStudent)

	reqBody := map[string]string{
		"email":          "john.updated@student.edu",
		"full_name":      "John Updated",
		"student_number": "STD-2026-001",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/students/test-id-123", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestDeleteStudent(t *testing.T) {
	r, gw := setupRouter()
	r.DELETE("/students/:id", gw.DeleteStudent)

	req, _ := http.NewRequest("DELETE", "/students/test-id-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetStudentCourses(t *testing.T) {
	r, gw := setupRouter()
	r.GET("/students/:id/courses", gw.GetStudentCoursesByID)

	req, _ := http.NewRequest("GET", "/students/test-id-123/courses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	courses := resp["courses"].([]interface{})
	if len(courses) != 1 {
		t.Errorf("Expected 1 course, got %d", len(courses))
	}
}
