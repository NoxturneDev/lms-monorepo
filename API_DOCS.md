# LMS API Documentation

Base URL: `http://localhost:3000`

All endpoints use `Content-Type: application/json`

## Authentication

This API uses **JWT (JSON Web Token)** for authentication.

### Getting a Token

Use the login endpoints to get your JWT token:
- **Teachers**: `POST /api/v1/auth/teacher/login`
- **Students**: `POST /api/v1/auth/student/login`
- **Admins**: `POST /api/v1/auth/admin/login`

### Using the Token

Include the token in the `Authorization` header for protected endpoints:

```
Authorization: Bearer <your-jwt-token>
```

### Route Access Levels

**Public Routes** (No authentication required):
- `POST /api/v1/auth/teacher/login` - Teacher login
- `POST /api/v1/auth/student/login` - Student login
- `POST /api/v1/auth/admin/login` - Admin login
- `POST /api/v1/students` - Student registration
- `POST /api/v1/teachers` - Teacher registration

**Protected Routes** (Any authenticated user):
- Student CRUD (GET, PUT, DELETE)
- Student report card, courses, enrollments
- Teacher listing and details
- Course listing and details
- Course teachers listing
- Assignment listing and details
- Student course grade
- Enrollment creation and removal
- Course enrollments and student enrollments
- School listing and details
- Class listing and details

**Teacher-Only Routes**:
- Grade assignment (`POST /api/v1/grades`)
- Gradebook viewing (`GET /api/v1/courses/:id/grades`)
- Teacher dashboard (`GET /api/v1/dashboard/teacher/:id`)
- Assignment creation, update, deletion

**Admin-Only Routes**:
- Admin management (CRUD)
- School management (create, update, delete)
- Class management (create, update, delete)
- Course management (create, update, delete)
- Course-teacher assignment and unassignment

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
  "email": "admin@greenwood.edu",
  "password": "secret"
}
```

**Response:** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
  "email": "admin@greenwood.edu",
  "name": "Alice Greenwood",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "school_name": "Greenwood Academy",
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

## Student APIs

### Create Student
**POST** `/api/v1/students` (Public)

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
**GET** `/api/v1/students` (Protected)

**Query Params:** `?class_id=UUID` (optional)

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
**GET** `/api/v1/students/:id` (Protected)

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
**PUT** `/api/v1/students/:id` (Protected)

**Request:**
```json
{
  "email": "john.doe@student.edu",
  "full_name": "John Doe Updated",
  "student_number": "STD-2026-001",
  "password": "newpassword"
}
```

The `password` field is optional. If omitted, the password is not changed.

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
**DELETE** `/api/v1/students/:id` (Protected)

Deletes the student and asynchronously publishes a `StudentDeleted` event to RabbitMQ for grade cleanup in the teacher service.

**Response:** `200 OK`
```json
{
  "message": "Student deleted and cleanup scheduled"
}
```

---

### Get Student Report Card
**GET** `/api/v1/students/:id/report-card` (Protected)

Aggregates student info from the student service and grades from the teacher service in parallel using circuit breakers.

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
      "score": 95,
      "assignment_title": "Midterm Exam",
      "max_score": 100,
      "assignment_id": "a100f1ee-6c54-4b01-90e6-d701748f0001"
    },
    {
      "course_title": "Operating Systems",
      "score": 45,
      "assignment_title": "Lab Report 1",
      "max_score": 50,
      "assignment_id": "a200f1ee-6c54-4b01-90e6-d701748f0003"
    }
  ],
  "generated_at": "2026-01-28T10:30:00Z"
}
```

**Error Response:** `503 Service Unavailable` (circuit breaker open)
```json
{
  "error": "System overloaded. Please try again later."
}
```

---

### Get Student Courses
**GET** `/api/v1/students/:id/courses` (Protected)

Returns courses the student is enrolled in. Data is fetched from the school service's enrollment records.

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
**POST** `/api/v1/teachers` (Public)

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
**GET** `/api/v1/teachers` (Protected)

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
**GET** `/api/v1/teachers/:id` (Protected)

**Response:** `200 OK`
```json
{
  "id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "turing@uni.edu",
  "full_name": "Alan Turing"
}
```

**Error Response:** `404 Not Found`
```json
{
  "error": "Teacher not found"
}
```

---

### Update Teacher
**PUT** `/api/v1/teachers/:id` (Protected)

**Request:**
```json
{
  "email": "a.turing@uni.edu",
  "full_name": "Alan M. Turing",
  "password": "newsecret"
}
```

The `password` field is optional.

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
**DELETE** `/api/v1/teachers/:id` (Protected)

**Response:** `200 OK`
```json
{
  "success": true
}
```

---

## Admin APIs

### Create Admin
**POST** `/api/v1/admins` (Admin Only)

**Request:**
```json
{
  "email": "newadmin@greenwood.edu",
  "password": "adminsecret",
  "full_name": "New Admin",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001"
}
```

**Response:** `201 Created`
```json
{
  "id": "a300f1ee-6c54-4b01-90e6-d701748f0003",
  "email": "newadmin@greenwood.edu",
  "full_name": "New Admin",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "school_name": "Greenwood Academy"
}
```

---

### Get All Admins
**GET** `/api/v1/admins` (Admin Only)

**Response:** `200 OK`
```json
{
  "admins": [
    {
      "id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
      "email": "admin@greenwood.edu",
      "full_name": "Alice Greenwood",
      "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
      "school_name": "Greenwood Academy"
    },
    {
      "id": "a200f1ee-6c54-4b01-90e6-d701748f0002",
      "email": "admin@riverside.edu",
      "full_name": "Bob Riverside",
      "school_id": "b200f1ee-6c54-4b01-90e6-d701748f0002",
      "school_name": "Riverside High School"
    }
  ]
}
```

---

### Get Admin Details
**GET** `/api/v1/admins/:id` (Admin Only)

**Response:** `200 OK`
```json
{
  "id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
  "email": "admin@greenwood.edu",
  "full_name": "Alice Greenwood",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "school_name": "Greenwood Academy"
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
**PUT** `/api/v1/admins/:id` (Admin Only)

**Request:**
```json
{
  "email": "admin.updated@greenwood.edu",
  "full_name": "Alice Greenwood Updated",
  "password": "newadminsecret",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001"
}
```

The `password` field is optional.

**Response:** `200 OK`
```json
{
  "id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
  "email": "admin.updated@greenwood.edu",
  "full_name": "Alice Greenwood Updated",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "school_name": "Greenwood Academy"
}
```

---

### Delete Admin
**DELETE** `/api/v1/admins/:id` (Admin Only)

**Response:** `200 OK`
```json
{
  "success": true
}
```

---

## School APIs

### List Schools
**GET** `/api/v1/schools` (Protected)

**Response:** `200 OK`
```json
{
  "schools": [
    {
      "id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
      "name": "Greenwood Academy",
      "address": "123 Elm Street, Springfield"
    },
    {
      "id": "b200f1ee-6c54-4b01-90e6-d701748f0002",
      "name": "Riverside High School",
      "address": "456 Oak Avenue, Shelbyville"
    }
  ]
}
```

---

### Get School Details
**GET** `/api/v1/schools/:id` (Protected)

**Response:** `200 OK`
```json
{
  "id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "name": "Greenwood Academy",
  "address": "123 Elm Street, Springfield"
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
**POST** `/api/v1/schools` (Admin Only)

**Request:**
```json
{
  "name": "New Academy",
  "address": "789 Pine Road, New City"
}
```

**Response:** `201 Created`
```json
{
  "id": "b300f1ee-6c54-4b01-90e6-d701748f0003",
  "name": "New Academy",
  "address": "789 Pine Road, New City"
}
```

---

### Update School
**PUT** `/api/v1/schools/:id` (Admin Only)

**Request:**
```json
{
  "name": "Greenwood Academy - Updated",
  "address": "123 Elm Street, Springfield (New Wing)"
}
```

**Response:** `200 OK`
```json
{
  "id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "name": "Greenwood Academy - Updated",
  "address": "123 Elm Street, Springfield (New Wing)"
}
```

---

### Delete School
**DELETE** `/api/v1/schools/:id` (Admin Only)

Validates that the school has no admins or classes before deletion. Courses belonging to the school are cascade-deleted via FK constraint.

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "School deleted successfully"
}
```

**Response:** `200 OK` (blocked by admins)
```json
{
  "success": false,
  "message": "Cannot delete school with 2 admins"
}
```

**Response:** `200 OK` (blocked by classes)
```json
{
  "success": false,
  "message": "Cannot delete school with 3 classes"
}
```

---

## Class APIs

### List Classes
**GET** `/api/v1/classes` (Protected)

**Query Params:** `?school_id=UUID` (optional, filter by school)

**Response:** `200 OK`
```json
{
  "classes": [
    {
      "id": "cl00f1ee-6c54-4b01-90e6-d701748f0001",
      "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
      "school_name": "Greenwood Academy",
      "name": "Mathematics A",
      "grade_level": "10th Grade"
    },
    {
      "id": "cl00f1ee-6c54-4b01-90e6-d701748f0002",
      "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
      "school_name": "Greenwood Academy",
      "name": "Science B",
      "grade_level": "11th Grade"
    }
  ]
}
```

---

### Get Class Details
**GET** `/api/v1/classes/:id` (Protected)

**Response:** `200 OK`
```json
{
  "id": "cl00f1ee-6c54-4b01-90e6-d701748f0001",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "school_name": "Greenwood Academy",
  "name": "Mathematics A",
  "grade_level": "10th Grade"
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
**POST** `/api/v1/classes` (Admin Only)

**Request:**
```json
{
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "name": "Physics C",
  "grade_level": "12th Grade"
}
```

**Response:** `201 Created`
```json
{
  "id": "cl00f1ee-6c54-4b01-90e6-d701748f0004",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "school_name": "Greenwood Academy",
  "name": "Physics C",
  "grade_level": "12th Grade"
}
```

---

### Update Class
**PUT** `/api/v1/classes/:id` (Admin Only)

**Request:**
```json
{
  "name": "Advanced Mathematics A",
  "grade_level": "10th Grade"
}
```

**Response:** `200 OK`
```json
{
  "id": "cl00f1ee-6c54-4b01-90e6-d701748f0001",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "school_name": "Greenwood Academy",
  "name": "Advanced Mathematics A",
  "grade_level": "10th Grade"
}
```

---

### Delete Class
**DELETE** `/api/v1/classes/:id` (Admin Only)

**Response:** `200 OK`
```json
{
  "success": true
}
```

---

## Course APIs

Courses are school-owned resources managed by admins. Each course belongs to a school and can have multiple teachers assigned to it.

### List Courses
**GET** `/api/v1/courses` (Protected)

**Query Params:** `?school_id=UUID` (optional, filter by school)

**Response:** `200 OK`
```json
{
  "courses": [
    {
      "id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
      "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
      "school_name": "Greenwood Academy",
      "title": "Advanced Algorithms",
      "description": "P vs NP and beyond",
      "teacher_ids": ["d290f1ee-6c54-4b01-90e6-d701748f0851"],
      "teacher_names": ["Alan Turing"]
    },
    {
      "id": "c200f1ee-6c54-4b01-90e6-d701748f0852",
      "school_id": "b200f1ee-6c54-4b01-90e6-d701748f0002",
      "school_name": "Riverside High School",
      "title": "Operating Systems",
      "description": "Compilers and Cobol",
      "teacher_ids": ["e390f1ee-6c54-4b01-90e6-d701748f0852"],
      "teacher_names": ["Grace Hopper"]
    }
  ]
}
```

---

### Get Course Details
**GET** `/api/v1/courses/:id` (Protected)

**Response:** `200 OK`
```json
{
  "id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "school_name": "Greenwood Academy",
  "title": "Advanced Algorithms",
  "description": "P vs NP and beyond",
  "teacher_ids": ["d290f1ee-6c54-4b01-90e6-d701748f0851"],
  "teacher_names": ["Alan Turing"]
}
```

**Error Response:** `404 Not Found`
```json
{
  "error": "Course not found"
}
```

---

### Create Course
**POST** `/api/v1/courses` (Admin Only)

**Request:**
```json
{
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
  "title": "Intro to Go",
  "description": "Learn Go programming"
}
```

**Response:** `201 Created`
```json
{
  "id": "c300f1ee-6c54-4b01-90e6-d701748f0853",
  "title": "Intro to Go",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001"
}
```

---

### Update Course
**PUT** `/api/v1/courses/:id` (Admin Only)

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
  "title": "Advanced Algorithms - Updated",
  "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001"
}
```

---

### Delete Course
**DELETE** `/api/v1/courses/:id` (Admin Only)

Validates that the course has no enrollments and no assignments before deletion. Teacher assignments are cascade-deleted via FK constraint.

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "Course deleted successfully"
}
```

**Response:** `200 OK` (blocked by enrollments)
```json
{
  "success": false,
  "message": "Cannot delete course with 15 enrollments"
}
```

**Response:** `200 OK` (blocked by assignments)
```json
{
  "success": false,
  "message": "Cannot delete course with existing assignments"
}
```

---

## Course-Teacher Assignment APIs

Manages the many-to-many relationship between courses and teachers. A course can have multiple teachers assigned to it.

### Assign Teacher to Course
**POST** `/api/v1/courses/:id/teachers` (Admin Only)

Validates both the course (local DB) and teacher (via Teacher Service) exist before creating the assignment.

**Request:**
```json
{
  "teacher_id": "d290f1ee-6c54-4b01-90e6-d701748f0851"
}
```

**Response:** `201 Created`
```json
{
  "id": "cta-100f1ee-6c54-4b01-90e6-d701748f0001",
  "success": true,
  "message": "Teacher assigned successfully"
}
```

**Error Response:** `400 Bad Request`
```json
{
  "error": "Course not found"
}
```

```json
{
  "error": "Teacher not found"
}
```

---

### Unassign Teacher from Course
**DELETE** `/api/v1/courses/:id/teachers/:teacher_id` (Admin Only)

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "Teacher unassigned successfully"
}
```

**Error Response:** `400 Bad Request`
```json
{
  "error": "Assignment not found"
}
```

---

### Get Course Teachers
**GET** `/api/v1/courses/:id/teachers` (Protected)

Returns all teachers assigned to a course. Teacher details are fetched from the Teacher Service.

**Response:** `200 OK`
```json
{
  "teachers": [
    {
      "teacher_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
      "teacher_name": "Alan Turing",
      "teacher_email": "turing@uni.edu"
    }
  ]
}
```

---

## Enrollment APIs

Manages student-course enrollments. Enrollment data is stored in the school service.

### Enroll Student
**POST** `/api/v1/enrollments` (Protected)

Validates both the course (local DB) and student (via Student Service) exist before enrolling.

**Request:**
```json
{
  "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "student_id": "a999f1ee-6c54-4b01-90e6-d701748f0851"
}
```

**Response:** `201 Created`
```json
{
  "id": "e100f1ee-6c54-4b01-90e6-d701748f0001",
  "success": true,
  "message": "Student enrolled successfully"
}
```

**Error Response:** `400 Bad Request`
```json
{
  "error": "Course not found"
}
```

```json
{
  "error": "Student not found"
}
```

---

### Unenroll Student
**DELETE** `/api/v1/enrollments` (Protected)

**Request:**
```json
{
  "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "student_id": "a999f1ee-6c54-4b01-90e6-d701748f0851"
}
```

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "Student unenrolled successfully"
}
```

**Error Response:** `400 Bad Request`
```json
{
  "error": "Enrollment not found"
}
```

---

### Get Course Enrollments
**GET** `/api/v1/courses/:id/enrollments` (Protected)

Returns all students enrolled in a course. Student details are fetched from the Student Service.

**Response:** `200 OK`
```json
{
  "students": [
    {
      "student_id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
      "student_name": "John Doe",
      "student_number": "STD-2026-001"
    },
    {
      "student_id": "a888f1ee-6c54-4b01-90e6-d701748f0852",
      "student_name": "Jane Smith",
      "student_number": "STD-2026-002"
    }
  ]
}
```

---

### Get Student Enrollments
**GET** `/api/v1/students/:id/enrollments` (Protected)

Returns all courses a student is enrolled in, with full course details including assigned teachers.

**Response:** `200 OK`
```json
{
  "courses": [
    {
      "id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
      "school_id": "b100f1ee-6c54-4b01-90e6-d701748f0001",
      "school_name": "Greenwood Academy",
      "title": "Advanced Algorithms",
      "description": "P vs NP and beyond",
      "teacher_ids": ["d290f1ee-6c54-4b01-90e6-d701748f0851"],
      "teacher_names": ["Alan Turing"]
    }
  ]
}
```

---

## Assignment APIs

Assignments belong to courses and are managed by teachers. Grades are assigned per assignment.

### Create Assignment
**POST** `/api/v1/courses/:id/assignments` (Teacher Only)

Validates the course exists via the School Service before creating the assignment.

**Request:**
```json
{
  "title": "Midterm Exam",
  "description": "Covers sorting and graph algorithms",
  "max_score": 100
}
```

The `max_score` defaults to 100 if not provided or <= 0. The course ID comes from the URL path parameter.

**Response:** `201 Created`
```json
{
  "id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
  "title": "Midterm Exam",
  "max_score": 100
}
```

---

### List Assignments
**GET** `/api/v1/courses/:id/assignments` (Protected)

**Response:** `200 OK`
```json
{
  "assignments": [
    {
      "id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
      "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
      "course_title": "Advanced Algorithms",
      "title": "Midterm Exam",
      "description": "Covers sorting and graph algorithms",
      "max_score": 100
    },
    {
      "id": "a100f1ee-6c54-4b01-90e6-d701748f0002",
      "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
      "course_title": "Advanced Algorithms",
      "title": "Final Project",
      "description": "Implement a novel algorithm",
      "max_score": 200
    }
  ]
}
```

---

### Get Assignment Details
**GET** `/api/v1/assignments/:id` (Protected)

**Response:** `200 OK`
```json
{
  "id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
  "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "course_title": "Advanced Algorithms",
  "title": "Midterm Exam",
  "description": "Covers sorting and graph algorithms",
  "max_score": 100
}
```

**Error Response:** `404 Not Found`
```json
{
  "error": "Assignment not found"
}
```

---

### Update Assignment
**PUT** `/api/v1/assignments/:id` (Teacher Only)

**Request:**
```json
{
  "title": "Midterm Exam - Updated",
  "description": "Updated scope: sorting, graphs, and dynamic programming",
  "max_score": 150
}
```

**Response:** `200 OK`
```json
{
  "id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
  "title": "Midterm Exam - Updated",
  "max_score": 150
}
```

---

### Delete Assignment
**DELETE** `/api/v1/assignments/:id` (Teacher Only)

Cannot delete assignments that have existing grades.

**Response:** `200 OK`
```json
{
  "success": true,
  "message": "Assignment deleted successfully"
}
```

**Response:** `409 Conflict` (has existing grades)
```json
{
  "error": "Cannot delete assignment with 12 existing grades"
}
```

---

## Grading APIs

### Assign Grade
**POST** `/api/v1/grades` (Teacher Only)

Grades are assigned per assignment, per student. The service validates:
- Student exists (via Student Service)
- Assignment exists
- Score does not exceed the assignment's `max_score`

**Request:**
```json
{
  "teacher_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "assignment_id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
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

**Error Response:** `502 Bad Gateway`
- Student not found
- Assignment not found
- Score exceeds max_score

---

### Get Course Gradebook
**GET** `/api/v1/courses/:id/grades` (Teacher Only)

Returns all grades for a course. Student names and numbers are fetched from the Student Service.

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
      "score": 95,
      "assignment_title": "Midterm Exam",
      "max_score": 100,
      "assignment_id": "a100f1ee-6c54-4b01-90e6-d701748f0001"
    },
    {
      "grade_id": "g101f1ee-6c54-4b01-90e6-d701748f0851",
      "student_id": "a888f1ee-6c54-4b01-90e6-d701748f0852",
      "student_name": "Jane Smith",
      "student_number": "STD-2026-002",
      "score": 88,
      "assignment_title": "Midterm Exam",
      "max_score": 100,
      "assignment_id": "a100f1ee-6c54-4b01-90e6-d701748f0001"
    }
  ]
}
```

---

### Get Student Course Grade
**GET** `/api/v1/courses/:id/student-grade?student_id=UUID` (Protected)

Computes the overall course grade for a student on the fly. The overall score is a weighted percentage: `SUM(score) / SUM(max_score) * 100`.

**Query Params:** `student_id` (required)

**Response:** `200 OK`
```json
{
  "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "course_title": "Advanced Algorithms",
  "student_id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
  "overall_score": 63.33,
  "total_score": 190,
  "total_max_score": 300,
  "assignments": [
    {
      "assignment_id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
      "assignment_title": "Midterm Exam",
      "score": 95,
      "max_score": 100
    },
    {
      "assignment_id": "a100f1ee-6c54-4b01-90e6-d701748f0002",
      "assignment_title": "Final Project",
      "score": 95,
      "max_score": 200
    }
  ]
}
```

**Error Response:** `400 Bad Request` (missing student_id)
```json
{
  "error": "student_id query parameter is required"
}
```

---

## Reporting APIs

### Get Teacher Dashboard
**GET** `/api/v1/dashboard/teacher/:id` (Teacher Only)

Returns basic teacher info. Course statistics are not yet populated (returns zeros) pending a "get courses by teacher" RPC in the school service.

**Response:** `200 OK`
```json
{
  "teacher_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "teacher_name": "Alan Turing",
  "total_courses": 0,
  "total_students_enrolled": 0,
  "courses": []
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

```json
{
  "error": "Admin access only"
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

## Example Usage

### 1. Login as Admin
```bash
curl -X POST http://localhost:3000/api/v1/auth/admin/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@greenwood.edu","password":"secret"}'
```

### 2. Create a Course (Admin)
```bash
TOKEN="<admin-token>"

curl -X POST http://localhost:3000/api/v1/courses \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"school_id":"b100f1ee-6c54-4b01-90e6-d701748f0001","title":"New Course","description":"Course description"}'
```

### 3. Assign a Teacher to the Course (Admin)
```bash
curl -X POST http://localhost:3000/api/v1/courses/COURSE_ID/teachers \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"teacher_id":"d290f1ee-6c54-4b01-90e6-d701748f0851"}'
```

### 4. Enroll a Student
```bash
curl -X POST http://localhost:3000/api/v1/enrollments \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"course_id":"COURSE_ID","student_id":"a999f1ee-6c54-4b01-90e6-d701748f0851"}'
```

### 5. Login as Teacher and Create an Assignment
```bash
curl -X POST http://localhost:3000/api/v1/auth/teacher/login \
  -H "Content-Type: application/json" \
  -d '{"email":"turing@uni.edu","password":"secret"}'

TEACHER_TOKEN="<teacher-token>"

curl -X POST http://localhost:3000/api/v1/courses/COURSE_ID/assignments \
  -H "Authorization: Bearer $TEACHER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Midterm Exam","description":"Covers chapters 1-5","max_score":100}'
```

### 6. Grade a Student on an Assignment
```bash
curl -X POST http://localhost:3000/api/v1/grades \
  -H "Authorization: Bearer $TEACHER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"teacher_id":"d290f1ee-6c54-4b01-90e6-d701748f0851","assignment_id":"ASSIGNMENT_ID","student_id":"a999f1ee-6c54-4b01-90e6-d701748f0851","score":95}'
```

### 7. Get a Student's Overall Course Grade
```bash
curl -X GET "http://localhost:3000/api/v1/courses/COURSE_ID/student-grade?student_id=a999f1ee-6c54-4b01-90e6-d701748f0851" \
  -H "Authorization: Bearer $TEACHER_TOKEN"
```

### 8. View Student's Enrolled Courses
```bash
curl -X GET http://localhost:3000/api/v1/students/a999f1ee-6c54-4b01-90e6-d701748f0851/enrollments \
  -H "Authorization: Bearer $TOKEN"
```

---

## Seed Data Reference

The following seed data is available after a fresh `docker compose up`:

### Schools
| ID | Name | Address |
|----|------|---------|
| `b100f1ee-...-0001` | Greenwood Academy | 123 Elm Street, Springfield |
| `b200f1ee-...-0002` | Riverside High School | 456 Oak Avenue, Shelbyville |

### Admins
| Email | Password | School |
|-------|----------|--------|
| `admin@greenwood.edu` | `secret` | Greenwood Academy |
| `admin@riverside.edu` | `secret` | Riverside High School |

### Teachers
| Email | Password | Name |
|-------|----------|------|
| `turing@uni.edu` | `secret` | Alan Turing |
| `hopper@uni.edu` | `secret` | Grace Hopper |
| `ritchie@uni.edu` | `secret` | Dennis Ritchie |

### Students
| Email | Password | Name | Number |
|-------|----------|------|--------|
| `john@student.edu` | `secret` | John Doe | STD-2026-001 |
| `jane@student.edu` | `secret` | Jane Smith | STD-2026-002 |
| `bob@student.edu` | `secret` | Bob Martin | STD-2026-003 |
| `alice@student.edu` | `secret` | Alice Wonderland | STD-2026-004 |

### Courses
| Title | School | Assigned Teacher |
|-------|--------|-----------------|
| Advanced Algorithms | Greenwood Academy | Alan Turing |
| Cryptography 101 | Greenwood Academy | Alan Turing |
| Operating Systems | Riverside High School | Grace Hopper |

### Pre-existing Enrollments
- John Doe enrolled in Advanced Algorithms
- Jane Smith enrolled in Advanced Algorithms

### Pre-existing Grades
- John Doe: 95/100 on Algorithms Midterm
- Jane Smith: 88/100 on Algorithms Midterm
- John Doe: 45/50 on OS Lab Report 1
