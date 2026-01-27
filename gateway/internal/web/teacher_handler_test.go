package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
	"google.golang.org/grpc"
)

type mockTeacherClientFull struct {
	teacherpb.TeacherServiceClient
}

func (m *mockTeacherClientFull) CreateTeacher(ctx context.Context, req *teacherpb.CreateTeacherRequest, opts ...grpc.CallOption) (*teacherpb.TeacherResponse, error) {
	return &teacherpb.TeacherResponse{
		Id:       "teacher-123",
		Email:    req.Email,
		FullName: req.FullName,
	}, nil
}

func (m *mockTeacherClientFull) GetTeacher(ctx context.Context, req *teacherpb.GetTeacherRequest, opts ...grpc.CallOption) (*teacherpb.TeacherResponse, error) {
	return &teacherpb.TeacherResponse{
		Id:       req.Id,
		Email:    "turing@uni.edu",
		FullName: "Alan Turing",
	}, nil
}

func (m *mockTeacherClientFull) UpdateTeacher(ctx context.Context, req *teacherpb.UpdateTeacherRequest, opts ...grpc.CallOption) (*teacherpb.TeacherResponse, error) {
	return &teacherpb.TeacherResponse{
		Id:       req.Id,
		Email:    req.Email,
		FullName: req.FullName,
	}, nil
}

func (m *mockTeacherClientFull) DeleteTeacher(ctx context.Context, req *teacherpb.DeleteTeacherRequest, opts ...grpc.CallOption) (*teacherpb.DeleteTeacherResponse, error) {
	return &teacherpb.DeleteTeacherResponse{
		Success: true,
	}, nil
}

func (m *mockTeacherClientFull) ListTeachers(ctx context.Context, req *teacherpb.ListTeachersRequest, opts ...grpc.CallOption) (*teacherpb.ListTeachersResponse, error) {
	return &teacherpb.ListTeachersResponse{
		Teachers: []*teacherpb.TeacherResponse{
			{
				Id:       "teacher-123",
				Email:    "turing@uni.edu",
				FullName: "Alan Turing",
			},
		},
	}, nil
}

func (m *mockTeacherClientFull) CreateCourse(ctx context.Context, req *teacherpb.CreateCourseRequest, opts ...grpc.CallOption) (*teacherpb.CourseResponse, error) {
	return &teacherpb.CourseResponse{
		Id:    "course-123",
		Title: req.Title,
	}, nil
}

func (m *mockTeacherClientFull) GetCourses(ctx context.Context, req *teacherpb.GetCoursesRequest, opts ...grpc.CallOption) (*teacherpb.GetCoursesResponse, error) {
	return &teacherpb.GetCoursesResponse{
		Courses: []*teacherpb.CourseDetailResponse{
			{
				Id:          "course-123",
				TeacherId:   "teacher-123",
				Title:       "Advanced Algorithms",
				Description: "P vs NP",
				TeacherName: "Alan Turing",
			},
		},
	}, nil
}

func (m *mockTeacherClientFull) GetCourse(ctx context.Context, req *teacherpb.GetCourseRequest, opts ...grpc.CallOption) (*teacherpb.CourseDetailResponse, error) {
	return &teacherpb.CourseDetailResponse{
		Id:          req.Id,
		TeacherId:   "teacher-123",
		Title:       "Advanced Algorithms",
		Description: "P vs NP",
		TeacherName: "Alan Turing",
	}, nil
}

func (m *mockTeacherClientFull) UpdateCourse(ctx context.Context, req *teacherpb.UpdateCourseRequest, opts ...grpc.CallOption) (*teacherpb.CourseResponse, error) {
	return &teacherpb.CourseResponse{
		Id:    req.Id,
		Title: req.Title,
	}, nil
}

func (m *mockTeacherClientFull) DeleteCourse(ctx context.Context, req *teacherpb.DeleteCourseRequest, opts ...grpc.CallOption) (*teacherpb.DeleteCourseResponse, error) {
	return &teacherpb.DeleteCourseResponse{
		Success: true,
		Message: "Course deleted successfully",
	}, nil
}

func (m *mockTeacherClientFull) EnrollStudent(ctx context.Context, req *teacherpb.EnrollStudentRequest, opts ...grpc.CallOption) (*teacherpb.EnrollmentResponse, error) {
	return &teacherpb.EnrollmentResponse{
		Id:      "enrollment-123",
		Success: true,
		Message: "Enrolled successfully",
	}, nil
}

func (m *mockTeacherClientFull) AssignGrade(ctx context.Context, req *teacherpb.AssignGradeRequest, opts ...grpc.CallOption) (*teacherpb.GradeResponse, error) {
	return &teacherpb.GradeResponse{
		Id:      "grade-123",
		Success: true,
	}, nil
}

func (m *mockTeacherClientFull) GetCourseGrades(ctx context.Context, req *teacherpb.GetCourseGradesRequest, opts ...grpc.CallOption) (*teacherpb.CourseGradesResponse, error) {
	return &teacherpb.CourseGradesResponse{
		CourseId:    req.CourseId,
		CourseTitle: "Advanced Algorithms",
		Grades: []*teacherpb.StudentGradeItem{
			{
				GradeId:       "grade-123",
				StudentId:     "student-123",
				StudentName:   "John Doe",
				StudentNumber: "STD-2026-001",
				Score:         95,
			},
		},
	}, nil
}

func (m *mockTeacherClientFull) GetTeacherDashboard(ctx context.Context, req *teacherpb.GetTeacherDashboardRequest, opts ...grpc.CallOption) (*teacherpb.TeacherDashboardResponse, error) {
	return &teacherpb.TeacherDashboardResponse{
		TeacherId:             req.TeacherId,
		TeacherName:           "Alan Turing",
		TotalCourses:          3,
		TotalStudentsEnrolled: 150,
		Courses: []*teacherpb.CourseSummary{
			{
				CourseId:      "course-123",
				Title:         "Advanced Algorithms",
				EnrolledCount: 45,
			},
		},
	}, nil
}

func setupTeacherRouter() (*gin.Engine, *Gateway) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	gw := &Gateway{
		StudentClient: &mockStudentClient{},
		TeacherClient: &mockTeacherClientFull{},
	}

	return r, gw
}

func TestCreateTeacher(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.POST("/teachers", gw.CreateTeacher)

	reqBody := map[string]string{
		"email":     "turing@uni.edu",
		"password":  "secret",
		"full_name": "Alan Turing",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/teachers", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestGetTeacher(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.GET("/teachers/:id", gw.GetTeacher)

	req, _ := http.NewRequest("GET", "/teachers/teacher-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUpdateTeacher(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.PUT("/teachers/:id", gw.UpdateTeacher)

	reqBody := map[string]string{
		"email":     "a.turing@uni.edu",
		"full_name": "Alan M. Turing",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/teachers/teacher-123", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestDeleteTeacher(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.DELETE("/teachers/:id", gw.DeleteTeacher)

	req, _ := http.NewRequest("DELETE", "/teachers/teacher-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestListTeachers(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.GET("/teachers", gw.ListTeachers)

	req, _ := http.NewRequest("GET", "/teachers", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCreateCourse(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.POST("/courses", gw.CreateCourse)

	reqBody := map[string]string{
		"teacher_id":  "teacher-123",
		"title":       "Intro to Go",
		"description": "Learn Go programming",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/courses", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestGetCourses(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.GET("/courses", gw.GetCourses)

	req, _ := http.NewRequest("GET", "/courses", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetCourse(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.GET("/courses/:id", gw.GetCourse)

	req, _ := http.NewRequest("GET", "/courses/course-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestUpdateCourse(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.PUT("/courses/:id", gw.UpdateCourse)

	reqBody := map[string]string{
		"title":       "Advanced Algorithms - Updated",
		"description": "New syllabus",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("PUT", "/courses/course-123", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestDeleteCourse(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.DELETE("/courses/:id", gw.DeleteCourse)

	req, _ := http.NewRequest("DELETE", "/courses/course-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestEnrollStudent(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.POST("/enrollments", gw.EnrollStudent)

	reqBody := map[string]string{
		"student_id": "student-123",
		"course_id":  "course-123",
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/enrollments", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}
}

func TestAssignGrade(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.POST("/grades", gw.AssignGrade)

	reqBody := map[string]interface{}{
		"teacher_id": "teacher-123",
		"course_id":  "course-123",
		"student_id": "student-123",
		"score":      95,
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/grades", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetCourseGrades(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.GET("/courses/:course_id/grades", gw.GetCourseGrades)

	req, _ := http.NewRequest("GET", "/courses/course-123/grades", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestGetTeacherDashboard(t *testing.T) {
	r, gw := setupTeacherRouter()
	r.GET("/dashboard/teacher/:id", gw.GetTeacherDashboard)

	req, _ := http.NewRequest("GET", "/dashboard/teacher/teacher-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["total_courses"].(float64) != 3 {
		t.Errorf("Expected 3 total courses, got %v", resp["total_courses"])
	}
}
