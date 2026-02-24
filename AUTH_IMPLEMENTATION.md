# Authentication Implementation

## Overview

Basic JWT-based authentication and authorization system for the LMS platform.

## Features

✅ **JWT Token Authentication**
- 24-hour token expiration
- HS256 signing algorithm
- User type differentiation (teacher/student)

✅ **Two User Types**
- Teachers: Can create courses, assign grades, view dashboards
- Students: Can enroll in courses, view their grades and courses

✅ **Route Protection**
- Public routes: Login, registration
- Protected routes: Require valid JWT token
- Teacher-only routes: Additional authorization check

---

## Architecture

### 1. Proto Layer
- Added `LoginTeacher` RPC in `teacher.proto`
- Added `LoginStudent` RPC in `student.proto`
- Both return user info on successful authentication

### 2. Service Layer
**Teacher Service** (`teacher-service/main.go`):
- `LoginTeacher()`: Validates email/password against database
- Plain text password comparison (should use bcrypt in production)

**Student Service** (`student-service/main.go`):
- `LoginStudent()`: Validates email/password against database
- Plain text password comparison (should use bcrypt in production)

### 3. Gateway Layer

**JWT Utilities** (`gateway/utils/jwt.go`):
- `GenerateToken()`: Creates JWT with user_id, email, user_type
- `ValidateToken()`: Parses and validates JWT

**Middleware** (`gateway/internal/web/auth_middleware.go`):
- `AuthMiddleware()`: Validates JWT from Authorization header
- `TeacherOnly()`: Ensures user is a teacher
- `StudentOnly()`: Ensures user is a student

**Auth Handlers** (`gateway/internal/web/auth_handler.go`):
- `LoginTeacher()`: HTTP handler for teacher login
- `LoginStudent()`: HTTP handler for student login

---

## Endpoints

### Public Endpoints (No Auth)
```
POST /api/v1/auth/teacher/login   - Teacher login
POST /api/v1/auth/student/login   - Student login
POST /api/v1/students              - Student registration
POST /api/v1/teachers              - Teacher registration
```

### Protected Endpoints (Auth Required)
```
All student CRUD operations (except create)
All teacher CRUD operations (except create)
Course viewing
Enrollment
```

### Teacher-Only Endpoints
```
POST   /api/v1/courses                  - Create course
PUT    /api/v1/courses/:id              - Update course
DELETE /api/v1/courses/:id              - Delete course
POST   /api/v1/grades                   - Assign grade
GET    /api/v1/courses/:course_id/grades - View gradebook
GET    /api/v1/dashboard/teacher/:id    - Teacher dashboard
```

---

## Usage Flow

### 1. Login
```bash
# Teacher login
curl -X POST http://localhost:3000/api/v1/auth/teacher/login \
  -H "Content-Type: application/json" \
  -d '{"email":"turing@uni.edu","password":"secret"}'
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
  "email": "turing@uni.edu",
  "name": "Alan Turing",
  "userType": "teacher"
}
```

### 2. Use Token
```bash
# Include token in Authorization header
curl -X GET http://localhost:3000/api/v1/students \
  -H "Authorization: Bearer <token>"
```

---

## Security Considerations

### ⚠️ Current Implementation (Development Only)

1. **Plain Text Passwords**: Passwords stored as plain text
   - **Production Fix**: Use bcrypt for password hashing

2. **Static JWT Secret**: Hardcoded in `jwt.go`
   - **Production Fix**: Use environment variable

3. **No Refresh Tokens**: Tokens expire after 24 hours
   - **Production Fix**: Implement refresh token mechanism

4. **No Rate Limiting**: No protection against brute force
   - **Production Fix**: Add rate limiting on login endpoints

5. **No HTTPS**: Running on HTTP
   - **Production Fix**: Use HTTPS in production

---

## Files Modified/Created

### Proto Files
- `proto/teacher.proto` - Added LoginTeacher RPC
- `proto/student.proto` - Added LoginStudent RPC

### Services
- `teacher-service/main.go` - Added LoginTeacher handler
- `student-service/main.go` - Added LoginStudent handler

### Gateway
- `gateway/utils/jwt.go` - JWT utilities (NEW)
- `gateway/internal/web/auth_middleware.go` - Auth middleware (NEW)
- `gateway/internal/web/auth_handler.go` - Login handlers (NEW)
- `gateway/main.go` - Updated routes with auth protection

### Documentation
- `API_DOCS.md` - Added authentication section
- `AUTH_IMPLEMENTATION.md` - This file (NEW)

---

## Testing Authentication

### Test with Seed Data

**Teachers:**
```json
{"email":"turing@uni.edu","password":"secret"}
{"email":"hopper@uni.edu","password":"secret"}
{"email":"ritchie@uni.edu","password":"secret"}
```

**Students:**
```json
{"email":"john@student.edu","password":"secret"}
{"email":"jane@student.edu","password":"secret"}
{"email":"bob@student.edu","password":"secret"}
{"email":"alice@student.edu","password":"secret"}
```

### Test Flow

1. Login as teacher:
   ```bash
   curl -X POST http://localhost:3000/api/v1/auth/teacher/login \
     -H "Content-Type: application/json" \
     -d '{"email":"turing@uni.edu","password":"secret"}'
   ```

2. Try to access protected route without token (should fail):
   ```bash
   curl -X GET http://localhost:3000/api/v1/students
   ```

3. Access with token (should succeed):
   ```bash
   curl -X GET http://localhost:3000/api/v1/students \
     -H "Authorization: Bearer <token>"
   ```

4. Try student accessing teacher-only route (should fail):
   ```bash
   # Login as student first
   # Then try to create course - should return 403 Forbidden
   curl -X POST http://localhost:3000/api/v1/courses \
     -H "Authorization: Bearer <student-token>" \
     -H "Content-Type: application/json" \
     -d '{"teacher_id":"xxx","title":"Course"}'
   ```

---

## Token Structure

JWT payload contains:
```json
{
  "user_id": "uuid",
  "email": "user@example.com",
  "user_type": "teacher|student",
  "exp": 1234567890,
  "iat": 1234567890
}
```

---

## Next Steps for Production

1. **Password Hashing**
   - Implement bcrypt in CreateTeacher/CreateStudent
   - Update LoginTeacher/LoginStudent to use bcrypt.CompareHashAndPassword

2. **Environment Configuration**
   - Move JWT secret to environment variable
   - Add JWT_SECRET to docker-compose

3. **Refresh Tokens**
   - Implement refresh token mechanism
   - Add refresh endpoint

4. **Rate Limiting**
   - Add rate limiter middleware
   - Limit login attempts per IP

5. **HTTPS**
   - Configure TLS certificates
   - Force HTTPS in production

6. **Audit Logging**
   - Log all authentication attempts
   - Track failed login attempts
