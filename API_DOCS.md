# LMS API Documentation

Base URL: `http://localhost:3000`

All endpoints use `Content-Type: application/json`

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
