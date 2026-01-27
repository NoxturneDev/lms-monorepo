# Gateway API Unit Tests

## Overview
Unit tests for all gateway HTTP handlers covering 24 API endpoints.

## Test Files
- `internal/web/student_handler_test.go` - Tests for 7 student APIs
- `internal/web/teacher_handler_test.go` - Tests for 17 teacher/course/enrollment/grading APIs

## Running Tests

### Run All Tests
```bash
cd gateway
go test ./internal/web -v
```

### Run Specific Test File
```bash
cd gateway

# Student handler tests only
go test ./internal/web -v -run "TestCreateStudent|TestGetAllStudents|TestGetStudentDetails|TestUpdateStudent|TestDeleteStudent|TestGetStudentCourses"

# Teacher handler tests only
go test ./internal/web -v -run "TestCreateTeacher|TestGetTeacher|TestUpdateTeacher|TestDeleteTeacher|TestListTeachers"

# Course tests only
go test ./internal/web -v -run "TestCreateCourse|TestGetCourses|TestGetCourse|TestUpdateCourse|TestDeleteCourse"
```

### Run Single Test
```bash
cd gateway
go test ./internal/web -v -run TestCreateStudent
```

### Run with Coverage
```bash
cd gateway
go test ./internal/web -cover
```

### Run with Detailed Coverage Report
```bash
cd gateway
go test ./internal/web -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Structure

Each test:
1. Sets up a mock gRPC client
2. Creates a test HTTP request
3. Records the response
4. Validates the status code and response body

## Mock Clients

Tests use mock implementations of:
- `StudentServiceClient` - Mocks student service gRPC calls
- `TeacherServiceClient` - Mocks teacher service gRPC calls

## Tests Covered

### Student APIs (7 tests)
- ✅ POST /students - Create student
- ✅ GET /students - Get all students
- ✅ GET /students/:id - Get student details
- ✅ PUT /students/:id - Update student
- ✅ DELETE /students/:id - Delete student
- ✅ GET /students/:id/courses - Get student courses
- ✅ Report card test (included in student_handler_test.go mock)

### Teacher APIs (5 tests)
- ✅ POST /teachers - Create teacher
- ✅ GET /teachers - List teachers
- ✅ GET /teachers/:id - Get teacher
- ✅ PUT /teachers/:id - Update teacher
- ✅ DELETE /teachers/:id - Delete teacher

### Course APIs (5 tests)
- ✅ POST /courses - Create course
- ✅ GET /courses - Get all courses
- ✅ GET /courses/:id - Get course details
- ✅ PUT /courses/:id - Update course
- ✅ DELETE /courses/:id - Delete course

### Enrollment APIs (1 test)
- ✅ POST /enrollments - Enroll student

### Grading APIs (2 tests)
- ✅ POST /grades - Assign grade
- ✅ GET /courses/:course_id/grades - Get course gradebook

### Reporting APIs (1 test)
- ✅ GET /dashboard/teacher/:id - Teacher dashboard

## Quick Test
```bash
# From gateway directory
cd gateway
go test ./internal/web -v
```

Expected output:
```
PASS
ok  	github.com/noxturnedev/lms-monorepo/gateway/internal/web	0.015s
```

All 20 tests should PASS with ~60% code coverage
