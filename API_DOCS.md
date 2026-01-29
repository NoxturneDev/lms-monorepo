# LMS API Documentation

Base URL: `http://localhost:3000`

All endpoints use `Content-Type: application/json`

## 🔐 Authentication

This API uses **JWT (JSON Web Token)** for authentication.

### Getting a Token

Use the login endpoints to get your JWT token:
- **Teachers**: `POST /api/v1/auth/teacher/login`
- **Students**: `POST /api/v1/auth/student/login`

### Using the Token

Include the token in the `Authorization` header for protected endpoints:

```
Authorization: Bearer <your-jwt-token>
```

### Public vs Protected Routes

**Public Routes** (No authentication required):
- `POST /api/v1/auth/teacher/login` - Teacher login
- `POST /api/v1/auth/student/login` - Student login
- `POST /api/v1/auth/admin/login` - Admin login
- `POST /api/v1/students` - Student registration
- `POST /api/v1/teachers` - Teacher registration

**Protected Routes** (Authentication required):
- All other endpoints require a valid JWT token

**Teacher-Only Routes**:
- Course creation, update, deletion
- Grade assignment
- Gradebook viewing
- Teacher dashboard

**Admin-Only Routes**:
- Admin management (CRUD)
- School management (CRUD)
- Class management (CRUD)

---

## Authentication APIs

### Teacher Login
**POST** `/api/v1/auth/teacher/login`

**Request:**
```json
{
  "email": "turing@uni.edu",
  "password": "secret"
}
```

**Response:** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "turing@uni.edu",
  "name": "Alan Turing",
  "userType": "teacher"
}
```

**Error Response:** `401 Unauthorized`
```json
{
  "error": "Invalid email or password"
}
```

---

### Student Login
**POST** `/api/v1/auth/student/login`

**Request:**
```json
{
  "email": "john@student.edu",
  "password": "secret"
}
```

**Response:** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "john@student.edu",
  "name": "John Doe",
  "student_number": "STD-2026-001",
  "userType": "student"
}
```

**Error Response:** `401 Unauthorized`
```json
{
  "error": "Invalid email or password"
}
```

---

### Admin Login
**POST** `/api/v1/auth/admin/login`

**Request:**
```json
{
  "email": "admin@school.edu",
  "password": "adminsecret"
}
```

**Response:** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "f290f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "admin@school.edu",
  "name": "Admin User",
  "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
  "school_name": "Central High School",
  "userType": "admin"
}
```

**Error Response:** `401 Unauthorized`
```json
{
  "error": "Invalid email or password"
}
```

---

## Admin APIs

### Create Admin
**POST** `/api/v1/admins`

**Admin Only**

**Request:**
```json
{
  "email": "admin@school.edu",
  "password": "adminsecret",
  "full_name": "Admin User",
  "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851"
}
```

**Response:** `201 Created`
```json
{
  "id": "f290f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "admin@school.edu",
  "full_name": "Admin User",
  "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
  "school_name": "Central High School"
}
```

---

### Get All Admins
**GET** `/api/v1/admins`

**Admin Only**

**Response:** `200 OK`
```json
{
  "admins": [
    {
      "id": "f290f1ee-6c54-4b01-90e6-d701748f0851",
      "email": "admin@school.edu",
      "full_name": "Admin User",
      "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
      "school_name": "Central High School"
    }
  ]
}
```

---

### Get Admin Details
**GET** `/api/v1/admins/:id`

**Admin Only**

**Response:** `200 OK`
```json
{
  "id": "f290f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "admin@school.edu",
  "full_name": "Admin User",
  "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
  "school_name": "Central High School"
}
```

**Error Response:** `404 Not Found`
```json
{
  "error": "Admin not found"
}
```

---

### Update Admin
**PUT** `/api/v1/admins/:id`

**Admin Only**

**Request:**
```json
{
  "email": "admin.updated@school.edu",
  "full_name": "Admin User Updated",
  "password": "newadminsecret",
  "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851"
}
```

**Response:** `200 OK`
```json
{
  "id": "f290f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "admin.updated@school.edu",
  "full_name": "Admin User Updated",
  "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
  "school_name": "Central High School"
}
```

---

### Delete Admin
**DELETE** `/api/v1/admins/:id`

**Admin Only**

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "Admin deleted successfully"
}
```

---

## School APIs

### Get All Schools
**GET** `/api/v1/schools`

**Protected (Any authenticated user)**

**Response:** `200 OK`
```json
{
  "schools": [
    {
      "id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
      "name": "Central High School",
      "address": "123 Education Ave, City, State 12345"
    },
    {
      "id": "s101f1ee-6c54-4b01-90e6-d701748f0851",
      "name": "Downtown Elementary",
      "address": "456 Learning Blvd, City, State 12346"
    }
  ]
}
```

---

### Get School Details
**GET** `/api/v1/schools/:id`

**Protected (Any authenticated user)**

**Response:** `200 OK`
```json
{
  "id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
  "name": "Central High School",
  "address": "123 Education Ave, City, State 12345"
}
```

**Error Response:** `404 Not Found`
```json
{
  "error": "School not found"
}
```

---

### Create School
**POST** `/api/v1/schools`

**Admin Only**

**Request:**
```json
{
  "name": "Central High School",
  "address": "123 Education Ave, City, State 12345"
}
```

**Response:** `201 Created`
```json
{
  "id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
  "name": "Central High School",
  "address": "123 Education Ave, City, State 12345"
}
```

---

### Update School
**PUT** `/api/v1/schools/:id`

**Admin Only**

**Request:**
```json
{
  "name": "Central High School - Updated",
  "address": "123 Education Ave, New City, State 12345"
}
```

**Response:** `200 OK`
```json
{
  "id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
  "name": "Central High School - Updated",
  "address": "123 Education Ave, New City, State 12345"
}
```

---

### Delete School
**DELETE** `/api/v1/schools/:id`

**Admin Only**

**Response:** `200 OK` (Success)
```json
{
  "success": true,
  "message": "School deleted successfully"
}
```

**Response:** `409 Conflict` (School has admins or classes)
```json
{
  "error": "Cannot delete school with existing admins or classes"
}
```

---

## Class APIs

### Get All Classes
**GET** `/api/v1/classes`

**Protected (Any authenticated user)**

**Query Params:** `?school_id=UUID` (optional, filter by school)

**Response:** `200 OK`
```json
{
  "classes": [
    {
      "id": "c200f1ee-6c54-4b01-90e6-d701748f0851",
      "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
      "school_name": "Central High School",
      "name": "Class 10-A",
      "grade_level": 10
    },
    {
      "id": "c201f1ee-6c54-4b01-90e6-d701748f0851",
      "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
      "school_name": "Central High School",
      "name": "Class 10-B",
      "grade_level": 10
    }
  ]
}
```

---

### Get Class Details
**GET** `/api/v1/classes/:id`

**Protected (Any authenticated user)**

**Response:** `200 OK`
```json
{
  "id": "c200f1ee-6c54-4b01-90e6-d701748f0851",
  "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
  "school_name": "Central High School",
  "name": "Class 10-A",
  "grade_level": 10
}
```

**Error Response:** `404 Not Found`
```json
{
  "error": "Class not found"
}
```

---

### Create Class
**POST** `/api/v1/classes`

**Admin Only**

**Request:**
```json
{
  "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
  "name": "Class 10-A",
  "grade_level": 10
}
```

**Response:** `201 Created`
```json
{
  "id": "c200f1ee-6c54-4b01-90e6-d701748f0851",
  "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
  "school_name": "Central High School",
  "name": "Class 10-A",
  "grade_level": 10
}
```

---

### Update Class
**PUT** `/api/v1/classes/:id`

**Admin Only**

**Request:**
```json
{
  "name": "Class 10-A-Advanced",
  "grade_level": 10
}
```

**Response:** `200 OK`
```json
{
  "id": "c200f1ee-6c54-4b01-90e6-d701748f0851",
  "school_id": "s100f1ee-6c54-4b01-90e6-d701748f0851",
  "school_name": "Central High School",
  "name": "Class 10-A-Advanced",
  "grade_level": 10
}
```

---

### Delete Class
**DELETE** `/api/v1/classes/:id`

**Admin Only**

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "Class deleted successfully"
}
```

---

## Student APIs

### Create Student
**POST** `/api/v1/students`

**Request:**
```json
{
  "email": "john@student.edu",
  "full_name": "John Doe",
  "password": "secret123",
  "student_number": "STD-2026-001"
}
```

**Response:** `201 Created`
```json
{
  "id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "john@student.edu",
  "full_name": "John Doe",
  "student_number": "STD-2026-001"
}
```

---

### Get All Students
**GET** `/api/v1/students`

**Query Params:** `?class_id=xxx` (optional)

**Response:** `200 OK`
```json
{
  "students": [
    {
      "id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
      "full_name": "John Doe",
      "email": "john@student.edu",
      "student_number": "STD-2026-001"
    }
  ]
}
```

---

### Get Student Details
**GET** `/api/v1/students/:id`

**Response:** `200 OK`
```json
{
  "student_profile": {
    "id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
    "full_name": "John Doe",
    "email": "john@student.edu",
    "student_number": "STD-2026-001"
  },
  "source": "Aggregation Gateway"
}
```

---

### Update Student
**PUT** `/api/v1/students/:id`

**Request:**
```json
{
  "email": "john.doe@student.edu",
  "full_name": "John Doe Updated",
  "student_number": "STD-2026-001",
  "password": "newpassword" // optional
}
```

**Response:** `200 OK`
```json
{
  "id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "john.doe@student.edu",
  "full_name": "John Doe Updated",
  "student_number": "STD-2026-001"
}
```

---

### Delete Student
**DELETE** `/api/v1/students/:id`

**Response:** `200 OK`
```json
{
  "message": "Student deleted and cleanup scheduled"
}
```

---

### Get Student Report Card
**GET** `/api/v1/students/:id/report-card`

**Response:** `200 OK`
```json
{
  "student_info": {
    "name": "John Doe",
    "email": "john@student.edu",
    "student_number": "STD-2026-001"
  },
  "academic_record": [
    {
      "course_title": "Advanced Algorithms",
      "score": 95
    },
    {
      "course_title": "Operating Systems",
      "score": 88
    }
  ],
  "generated_at": "2026-01-28T10:30:00Z"
}
```

---

### Get Student Courses
**GET** `/api/v1/students/:id/courses`

**Response:** `200 OK`
```json
{
  "courses": [
    {
      "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
      "title": "Advanced Algorithms",
      "description": "P vs NP and beyond",
      "teacher_name": "Alan Turing"
    }
  ]
}
```

---

## Teacher APIs

### Create Teacher
**POST** `/api/v1/teachers`

**Request:**
```json
{
  "email": "turing@uni.edu",
  "password": "secret",
  "full_name": "Alan Turing"
}
```

**Response:** `201 Created`
```json
{
  "id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "turing@uni.edu",
  "full_name": "Alan Turing"
}
```

---

### Get All Teachers
**GET** `/api/v1/teachers`

**Response:** `200 OK`
```json
{
  "teachers": [
    {
      "id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
      "email": "turing@uni.edu",
      "full_name": "Alan Turing"
    }
  ]
}
```

---

### Get Teacher
**GET** `/api/v1/teachers/:id`

**Response:** `200 OK`
```json
{
  "id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "turing@uni.edu",
  "full_name": "Alan Turing"
}
```

---

### Update Teacher
**PUT** `/api/v1/teachers/:id`

**Request:**
```json
{
  "email": "a.turing@uni.edu",
  "full_name": "Alan M. Turing",
  "password": "newsecret" // optional
}
```

**Response:** `200 OK`
```json
{
  "id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "a.turing@uni.edu",
  "full_name": "Alan M. Turing"
}
```

---

### Delete Teacher
**DELETE** `/api/v1/teachers/:id`

**Response:** `200 OK`
```json
{
  "success": true
}
```

---

## Course APIs

### Create Course
**POST** `/api/v1/courses`

**Request:**
```json
{
  "teacher_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "title": "Intro to Go",
  "description": "Learn Go programming"
}
```

**Response:** `201 Created`
```json
{
  "id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "title": "Intro to Go"
}
```

---

### Get All Courses
**GET** `/api/v1/courses`

**Query Params:** `?teacher_id=xxx` (optional, filter by teacher)

**Response:** `200 OK`
```json
{
  "courses": [
    {
      "id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
      "teacher_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
      "title": "Advanced Algorithms",
      "description": "P vs NP and beyond",
      "teacher_name": "Alan Turing"
    }
  ]
}
```

---

### Get Course Details
**GET** `/api/v1/courses/:id`

**Response:** `200 OK`
```json
{
  "id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "teacher_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "title": "Advanced Algorithms",
  "description": "P vs NP and beyond",
  "teacher_name": "Alan Turing"
}
```

---

### Update Course
**PUT** `/api/v1/courses/:id`

**Request:**
```json
{
  "title": "Advanced Algorithms - Updated",
  "description": "New syllabus for 2026"
}
```

**Response:** `200 OK`
```json
{
  "id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "title": "Advanced Algorithms - Updated"
}
```

---

### Delete Course
**DELETE** `/api/v1/courses/:id`

**Response:** `200 OK` (Success)
```json
{
  "success": true,
  "message": "Course deleted successfully"
}
```

**Response:** `409 Conflict` (Has enrollments)
```json
{
  "error": "Cannot delete course with 15 enrolled students"
}
```

---

## Enrollment APIs

### Enroll Student
**POST** `/api/v1/enrollments`

**Request:**
```json
{
  "student_id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
  "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851"
}
```

**Response:** `201 Created` (Success)
```json
{
  "id": "e100f1ee-6c54-4b01-90e6-d701748f0851",
  "success": true,
  "message": "Enrolled successfully"
}
```

**Response:** `400 Bad Request` (Failed)
```json
{
  "error": "Student not found"
}
```

---

## Grading APIs

### Assign Grade
**POST** `/api/v1/grades`

**Request:**
```json
{
  "teacher_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "student_id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
  "score": 95
}
```

**Response:** `200 OK`
```json
{
  "id": "g100f1ee-6c54-4b01-90e6-d701748f0851",
  "success": true
}
```

---

### Get Course Gradebook
**GET** `/api/v1/courses/:course_id/grades`

**Response:** `200 OK`
```json
{
  "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "course_title": "Advanced Algorithms",
  "grades": [
    {
      "grade_id": "g100f1ee-6c54-4b01-90e6-d701748f0851",
      "student_id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
      "student_name": "John Doe",
      "student_number": "STD-2026-001",
      "score": 95
    },
    {
      "grade_id": "g101f1ee-6c54-4b01-90e6-d701748f0851",
      "student_id": "a888f1ee-6c54-4b01-90e6-d701748f0852",
      "student_name": "Jane Smith",
      "student_number": "STD-2026-002",
      "score": 88
    }
  ]
}
```

---

## Reporting APIs

### Get Teacher Dashboard
**GET** `/api/v1/dashboard/teacher/:id`

**Response:** `200 OK`
```json
{
  "teacher_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "teacher_name": "Alan Turing",
  "total_courses": 3,
  "total_students_enrolled": 150,
  "courses": [
    {
      "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
      "title": "Advanced Algorithms",
      "enrolled_count": 45
    },
    {
      "course_id": "c101f1ee-6c54-4b01-90e6-d701748f0851",
      "title": "Cryptography 101",
      "enrolled_count": 38
    },
    {
      "course_id": "c102f1ee-6c54-4b01-90e6-d701748f0851",
      "title": "Machine Learning",
      "enrolled_count": 67
    }
  ]
}
```

---

## Error Responses

All endpoints may return the following error responses:

**400 Bad Request**
```json
{
  "error": "Invalid request body"
}
```

**401 Unauthorized** (Missing or invalid token)
```json
{
  "error": "Invalid or expired token"
}
```

**403 Forbidden** (Insufficient permissions)
```json
{
  "error": "Teacher access only"
}
```

**404 Not Found**
```json
{
  "error": "Resource not found"
}
```

**500 Internal Server Error**
```json
{
  "error": "Internal server error message"
}
```

**503 Service Unavailable** (Circuit breaker open)
```json
{
  "error": "System overloaded. Please try again later."
}
```

---

## Example Usage with Authentication

### 1. Login as Teacher
```bash
curl -X POST http://localhost:3000/api/v1/auth/teacher/login \
  -H "Content-Type: application/json" \
  -d '{"email":"turing@uni.edu","password":"secret"}'
```

### 2. Use Token for Protected Endpoints
```bash
# Save the token from login response
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Make authenticated request
curl -X GET http://localhost:3000/api/v1/students \
  -H "Authorization: Bearer $TOKEN"
```

### 3. Access Teacher-Only Endpoint
```bash
# Only works if logged in as teacher
curl -X POST http://localhost:3000/api/v1/courses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"teacher_id":"xxx","title":"New Course","description":"Description"}'
```
