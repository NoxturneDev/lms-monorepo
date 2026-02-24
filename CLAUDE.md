# LMS Monorepo - Project Context

Last Updated: 2026-02-23

---

## Project Overview

A Learning Management System (LMS) built with microservices architecture using Go, gRPC, and PostgreSQL. The system provides a complete API for managing students, teachers, courses, enrollments, grading, schools, and administrative functions.

### Architecture

- **Gateway**: HTTP REST API gateway (Gin framework) that translates REST to gRPC
- **Student Service**: gRPC microservice for student management
- **Teacher Service**: gRPC microservice for teacher, course, enrollment, and grade management
- **School Service**: gRPC microservice for school, admin, and class management
- **Proto**: Shared protocol buffer definitions
- **Infrastructure**: PostgreSQL databases via Docker Compose with profile-based deployment

---

## Recent Session Work - 2026-02-23

### Production & Data Integrity

1. **Production Docker Compose** (`docker-compose.prod.yml`)
   - Single PostgreSQL container with 4 databases via `infra/postgres/init-all.sql`
   - Production binaries (15MB vs 1.5GB dev images)
   - Resource limits, healthchecks, proper startup ordering

2. **Stats-Service Sync Fixed**
   - Added RabbitMQ event publishing to teacher-service `AssignGrade()`
   - Teacher-service now publishes `GradeAssignedEvent` with all fields including category
   - Stats-service auto-reconciles missing category data every 30 seconds
   - Added `ReconcileGradesSync()` RPC to detect/fix inconsistencies

3. **Data Integrity - Category Field**
   - Added `category` to `assignments` table (teacher_db, stats_db, init-all.sql)
   - Updated teacher-service proto messages: `CreateAssignmentRequest`, `AssignmentResponse`, `AssignmentDetailResponse`, `UpdateAssignmentRequest`
   - All CRUD operations now preserve category field across DBs
   - Stats-service category-mastery queries now always have categorized data

4. **Better Error Detection**
   - Improved all entrypoint scripts (teacher, student, school, stats services)
   - Pre-build check before Air starts - shows Go compilation errors clearly
   - Won't try to run missing binary if build fails
   - Creates `tmp` directory before build

### Code Changes Summary
- Teacher-service: Added RabbitMQ channel, PublishEvent() method, category handling in all assignment functions
- Gateway: JWT_SECRET from env var, added production Dockerfile stages
- All services: RABBITMQ_URL from env var
- Fixed Dockerfile bugs: teacher/school services now copy correct .air.toml paths
- Stats-service: Reconciliation job, ReconcileGradesSync() RPC, time.import added

### Files Modified
- Proto: `teacher.proto` (category fields), `stats.proto` (ReconcileSync messages)
- Teacher-service: `main.go` (event publishing, category handling), `.entrypoint.sh`, `Dockerfile`
- Student-service: `.entrypoint.sh` (better error detection)
- School-service: `.entrypoint.sh` (better error detection)
- Stats-service: `main.go` (reconciliation job), `.entrypoint.sh`
- Gateway: `Dockerfile` (production stages), `utils/jwt.go` (env var)
- DB: `init-all.sql`, `teacher-init.sql` (category field, seed data)

---

## Previous Session Work - 2026-01-29

### What We Did

1. **New School Service Microservice** (`/mnt/workspace/projects/lms-ziad/lms-monorepo/school-service/`)
   - Created complete gRPC service implementing 16 RPCs across three domains:
     - **Admin Management**: LoginAdmin, CreateAdmin, GetAdmin, UpdateAdmin, DeleteAdmin, ListAdmins (6 RPCs)
     - **School Management**: CreateSchool, GetSchool, UpdateSchool, DeleteSchool, ListSchools (5 RPCs)
     - **Class Management**: CreateClass, GetClass, UpdateClass, DeleteClass, ListClasses (5 RPCs)
   - Implemented `main.go` with full business logic:
     - PostgreSQL integration with connection pooling
     - JWT-based authentication for admin login
     - Cross-domain validation (classes belong to schools)
     - Error handling and logging
   - Created `telemetry.go` with OpenTelemetry integration (Jaeger tracing)
   - Set up module: `github.com/noxturnedev/lms-monorepo/school-service`
   - Created multi-stage Dockerfile (dev + production builds)
   - Port assignment: 8083 (gRPC)

2. **New Proto Definition** (`/mnt/workspace/projects/lms-ziad/lms-monorepo/proto/school.proto`)
   - Defined SchoolService with 16 RPC methods
   - Message types:
     - `Admin` - id, email, password, full_name, created_at
     - `School` - id, name, address, created_at
     - `Class` - id, name, school_id, created_at
   - Request/Response messages for all CRUD operations
   - Compiled to `/mnt/workspace/projects/lms-ziad/lms-monorepo/proto/school/` package

3. **Database Schema** (`/mnt/workspace/projects/lms-ziad/lms-monorepo/infra/postgres/school-init.sql`)
   - Created three tables:
     - `schools` (id, name, address, created_at)
     - `admins` (id, email, password, full_name, created_at)
     - `classes` (id, name, school_id FK, created_at)
   - Seed data:
     - 2 schools: "Springfield Elementary", "Shelbyville High"
     - 2 admins: admin@school.edu, principal@school.edu
     - 3 classes: "Grade 5A", "Grade 5B", "Grade 10C"
   - Foreign key constraint: classes.school_id references schools.id
   - New PostgreSQL instance: postgres-school (port 5435)

4. **Docker Compose Profile System**
   - Implemented profile-based service orchestration in `docker-compose.yml`
   - Profiles added to ALL services:
     - `gateway` profile: gateway service only
     - `student` profile: student-service + postgres-student
     - `teacher` profile: teacher-service + postgres-teacher
     - `school` profile: school-service + postgres-school
     - `all` profile: all services (gateway + all microservices + all databases + Jaeger)
   - Benefits:
     - Start only required services during development
     - Faster iteration cycles
     - Reduced resource consumption
     - Clear service dependencies
   - New services in compose:
     - `school-service` (port 8083, depends on postgres-school)
     - `postgres-school` (port 5435, database: school_db)

5. **Gateway Integration**
   - Updated `gateway/internal/web/gateway.go`:
     - Added `SchoolClient pb_school.SchoolServiceClient` to Gateway struct
   - Created `gateway/internal/web/school_handler.go` with 16 handlers:
     - `LoginAdmin()` - Admin authentication
     - `CreateAdmin()`, `GetAdmin()`, `UpdateAdmin()`, `DeleteAdmin()`, `ListAdmins()` - Admin CRUD
     - `CreateSchool()`, `GetSchool()`, `UpdateSchool()`, `DeleteSchool()`, `ListSchools()` - School CRUD
     - `CreateClass()`, `GetClass()`, `UpdateClass()`, `DeleteClass()`, `ListClasses()` - Class CRUD
   - Enhanced `gateway/internal/web/auth_middleware.go`:
     - Added `AdminOnly()` middleware for admin-restricted routes
     - Checks `user_type == "admin"` from JWT claims
   - Updated `gateway/main.go`:
     - Added school-service gRPC client connection
     - New environment variable: `SCHOOL_SERVICE_URL` (default: school-service:8083)
     - Registered 16 new routes across three categories
   - Route organization:
     - Public: 1 route (admin login)
     - Protected (any authenticated user): 4 routes (view schools/classes)
     - Admin-only: 11 routes (all admin/school/class write operations)

6. **New REST API Endpoints** (16 total)
   - **Authentication** (1 endpoint):
     - POST `/api/v1/auth/admin/login` - Admin login (public)

   - **Admin Management** (5 endpoints, admin-only):
     - POST `/api/v1/admins` - Create admin
     - GET `/api/v1/admins` - List admins
     - GET `/api/v1/admins/:id` - Get admin details
     - PUT `/api/v1/admins/:id` - Update admin
     - DELETE `/api/v1/admins/:id` - Delete admin

   - **School Management** (5 endpoints):
     - POST `/api/v1/schools` - Create school (admin-only)
     - GET `/api/v1/schools` - List schools (protected)
     - GET `/api/v1/schools/:id` - Get school details (protected)
     - PUT `/api/v1/schools/:id` - Update school (admin-only)
     - DELETE `/api/v1/schools/:id` - Delete school (admin-only)

   - **Class Management** (5 endpoints):
     - POST `/api/v1/classes` - Create class (admin-only)
     - GET `/api/v1/classes` - List classes (protected)
     - GET `/api/v1/classes/:id` - Get class details (protected)
     - PUT `/api/v1/classes/:id` - Update class (admin-only)
     - DELETE `/api/v1/classes/:id` - Delete class (admin-only)

7. **Go Workspace Configuration**
   - Created `/mnt/workspace/projects/lms-ziad/lms-monorepo/go.work` file
   - Unified workspace for all Go modules:
     - `./gateway`
     - `./proto`
     - `./student-service`
     - `./teacher-service`
     - `./school-service`
   - Benefits:
     - Single dependency graph across all modules
     - Easier cross-module development
     - Consistent tooling (gopls, go test)
     - Simplified builds

### What We Accomplished

- Extended LMS to support multi-school architecture
- Added complete administrative layer for school management
- Implemented third microservice following established patterns
- Created profile-based Docker Compose system for flexible deployment
- Unified Go workspace for improved developer experience
- Expanded REST API from 24+ to 40+ endpoints
- Three-tier user system: Students, Teachers, Admins

### Key Takeaways

1. **Microservice Pattern Consistency**:
   - School service follows exact same architecture as teacher/student services
   - Same telemetry integration (OpenTelemetry/Jaeger)
   - Same error handling patterns
   - Same Dockerfile structure (multi-stage build)
   - Pattern reusability accelerates new service development

2. **Docker Compose Profiles**:
   - Profiles enable selective service deployment: `docker compose --profile school up`
   - `--profile all` starts entire system
   - Multiple profiles can be combined: `docker compose --profile student --profile teacher up`
   - Each profile is self-contained with its dependencies
   - Gateway profile typically used with specific service profiles

3. **JWT Role-Based Access Control**:
   - Three user types now supported: student, teacher, admin
   - Middleware hierarchy: `AuthMiddleware` -> `TeacherOnly/StudentOnly/AdminOnly`
   - JWT payload includes `user_type` field for role identification
   - Admins have separate login flow and token generation
   - Role enforcement at both gateway and service levels

4. **Domain-Driven Service Design**:
   - School service manages three related domains: admins, schools, classes
   - Clear domain boundaries with foreign key relationships
   - Classes belong to schools (enforced at DB level)
   - Admins manage schools and classes
   - Single service for related business logic reduces inter-service communication

5. **Go Workspace Benefits**:
   - `go.work` file enables multi-module development
   - All modules share common dependencies
   - Cross-module imports work seamlessly
   - Single `go mod tidy` across workspace
   - Better IDE support (autocomplete, navigation)

### Technical Details

**New Dependencies**:
- School service uses same stack as other services:
  - gRPC & Protocol Buffers
  - PostgreSQL driver (lib/pq)
  - OpenTelemetry/Jaeger
  - JWT-go (for admin authentication)

**New Environment Variables**:
- `SCHOOL_SERVICE_URL` - gRPC address for school service (default: school-service:8083)
- School service database config:
  - `DB_HOST=postgres-school`
  - `DB_PORT=5432`
  - `DB_USER=school_user`
  - `DB_PASSWORD=school_pass`
  - `DB_NAME=school_db`

**Updated File Structure**:
```
lms-monorepo/
├── gateway/
│   ├── main.go                          # Updated: school-service client + routes
│   ├── internal/web/
│   │   ├── gateway.go                   # Updated: SchoolClient field
│   │   ├── school_handler.go            # NEW: 16 school/admin/class handlers
│   │   └── auth_middleware.go           # Updated: AdminOnly middleware
├── school-service/                      # NEW SERVICE
│   ├── main.go                          # gRPC server + business logic
│   ├── telemetry.go                     # OpenTelemetry integration
│   ├── go.mod                           # Module definition
│   └── Dockerfile                       # Multi-stage build
├── proto/
│   ├── school.proto                     # NEW: SchoolService definition
│   └── school/                          # NEW: Generated Go code
│       └── school.pb.go
│       └── school_grpc.pb.go
├── infra/postgres/
│   └── school-init.sql                  # NEW: School database schema
├── go.work                              # NEW: Workspace configuration
└── docker-compose.yml                   # Updated: profiles + school service
```

**Port Allocation**:
- Gateway: 3000 (HTTP REST)
- Student Service: 50051 (gRPC)
- Teacher Service: 50052 (gRPC)
- School Service: 8083 (gRPC) - NEW
- Postgres Student: 5433
- Postgres Teacher: 5434
- Postgres School: 5435 - NEW
- Jaeger: 16686 (UI), 6831 (agent)

**Docker Compose Profiles Usage**:
```bash
# Start only school service
docker compose --profile school up

# Start student and teacher services
docker compose --profile student --profile teacher up

# Start everything
docker compose --profile all up

# Start gateway + school (for admin development)
docker compose --profile gateway --profile school up
```

### API Expansion Summary

**Before this session**: 24 endpoints (students, teachers, courses, enrollments, grades)
**After this session**: 40 endpoints (added 16 school/admin/class endpoints)

**New authentication endpoint**:
- POST `/api/v1/auth/admin/login` (3rd login type alongside teacher/student)

**New resource management**:
- Full CRUD for admins (5 endpoints)
- Full CRUD for schools (5 endpoints)
- Full CRUD for classes (5 endpoints)

**Authorization model**:
- Public routes: 4 (student/teacher/admin registration + 3 login endpoints)
- Protected routes: All authenticated users can view schools/classes
- Admin-only routes: Admin/school/class write operations
- Teacher-only routes: Course/grade management (from previous session)
- Student-only routes: None yet defined

### Known Issues & Limitations

1. **No Student-Class Association**: Students not yet linked to classes
2. **No Teacher-Class Assignment**: Teachers not assigned to specific classes
3. **No Cross-Service School Validation**: Student/teacher creation doesn't validate school_id
4. **Admin Self-Management**: Admins can delete themselves (potential issue)
5. **No School Hierarchy**: Single-level school structure (no districts/regions)
6. **No Class Capacity Limits**: Classes have no student count constraints
7. **Same Security Issues**: Plain text passwords still used (development only)

### Next Steps

**Integration & Validation**:
1. Add school_id field to students and teachers tables
2. Implement cross-service validation (validate school exists when creating students/teachers)
3. Add student-class enrollment (many-to-many relationship)
4. Add teacher-class assignment (many-to-many relationship)
5. Implement class roster endpoints (list students in a class)
6. Add school-level reporting (student count, teacher count per school)

**Profile-Based Testing**:
1. Document profile usage in API_DOCS.md
2. Create profile-specific test scripts
3. Add docker-compose.dev.yml for development overrides
4. Document minimal startup profiles for faster development

**Administrative Features**:
1. School dashboard endpoint (statistics, class count, student count)
2. Admin audit logs (track all admin actions)
3. Bulk operations (bulk student import, bulk class creation)
4. School settings management (academic year, terms, schedules)

**Workspace & Tooling**:
1. Add Makefile with workspace-aware commands
2. Create shared testing utilities across services
3. Implement common proto types (timestamps, pagination)
4. Add workspace-level CI/CD configuration

---

## Previous Session Work - 2026-01-28

### What We Did

1. **Refactored Gateway Architecture**
   - Extracted all API handlers from `/mnt/workspace/projects/lms-ziad/lms-monorepo/gateway/main.go` into organized handler files
   - Created `/mnt/workspace/projects/lms-ziad/lms-monorepo/gateway/internal/web/` directory structure:
     - `gateway.go` - Core Gateway struct
     - `student_handler.go` - 7 student-related endpoints
     - `teacher_handler.go` - 17 teacher/course/grade endpoints
     - `auth_handler.go` - Authentication endpoints
     - `auth_middleware.go` - JWT middleware
   - Moved utility files to `/mnt/workspace/projects/lms-ziad/lms-monorepo/gateway/utils/`:
     - `jwt.go` - JWT generation and validation
     - `breaker.go` - Circuit breaker implementation
     - `telemetry.go` - OpenTelemetry/Jaeger integration

2. **Complete REST API Implementation** (24+ endpoints)
   - **Student APIs** (7 endpoints):
     - POST `/api/v1/students` - Create student (public)
     - GET `/api/v1/students` - List students (protected)
     - GET `/api/v1/students/:id` - Get student details (protected)
     - PUT `/api/v1/students/:id` - Update student (protected)
     - DELETE `/api/v1/students/:id` - Delete student (protected)
     - GET `/api/v1/students/:id/report-card` - Get report card (protected)
     - GET `/api/v1/students/:id/courses` - Get student courses (protected)

   - **Teacher APIs** (5 endpoints):
     - POST `/api/v1/teachers` - Create teacher (public)
     - GET `/api/v1/teachers` - List teachers (protected)
     - GET `/api/v1/teachers/:id` - Get teacher details (protected)
     - PUT `/api/v1/teachers/:id` - Update teacher (protected)
     - DELETE `/api/v1/teachers/:id` - Delete teacher (protected)

   - **Course APIs** (5 endpoints):
     - POST `/api/v1/courses` - Create course (teacher-only)
     - GET `/api/v1/courses` - List courses (protected)
     - GET `/api/v1/courses/:id` - Get course details (protected)
     - PUT `/api/v1/courses/:id` - Update course (teacher-only)
     - DELETE `/api/v1/courses/:id` - Delete course (teacher-only)

   - **Enrollment APIs** (1 endpoint):
     - POST `/api/v1/enrollments` - Enroll student in course (protected)

   - **Grading APIs** (2 endpoints):
     - POST `/api/v1/grades` - Assign grade (teacher-only)
     - GET `/api/v1/courses/:course_id/grades` - View gradebook (teacher-only)

   - **Dashboard APIs** (1 endpoint):
     - GET `/api/v1/dashboard/teacher/:id` - Teacher dashboard (teacher-only)

   - **Authentication APIs** (2 endpoints):
     - POST `/api/v1/auth/teacher/login` - Teacher login (public)
     - POST `/api/v1/auth/student/login` - Student login (public)

3. **Microservices Implementation**
   - **Student Service** extended with:
     - Full CRUD operations for students
     - Course enrollment tracking
     - Cross-service validation via gRPC to teacher-service

   - **Teacher Service** extended with:
     - Full CRUD for teachers
     - Complete course management (CRUD)
     - Enrollment management with student validation
     - Grade assignment and gradebook retrieval
     - Teacher dashboard with course statistics
     - Cross-service validation via gRPC to student-service

4. **JWT Authentication & Authorization**
   - Implemented JWT-based authentication system
   - Token expiration: 24 hours
   - Signing algorithm: HS256
   - Token payload includes: `user_id`, `email`, `user_type` (teacher/student)
   - Three middleware layers:
     - `AuthMiddleware()` - Validates JWT for protected routes
     - `TeacherOnly()` - Restricts access to teacher-only routes
     - `StudentOnly()` - Restricts access to student-only routes
   - Login flows for both teachers and students
   - Password validation (plain text - development only)
   - Route protection applied to all sensitive endpoints

5. **Comprehensive Documentation**
   - Created `/mnt/workspace/projects/lms-ziad/lms-monorepo/API_DOCS.md`:
     - All 24+ endpoints with request/response examples
     - Authentication flow documentation
     - Error response patterns
     - curl examples for testing
   - Created `/mnt/workspace/projects/lms-ziad/lms-monorepo/AUTH_IMPLEMENTATION.md`:
     - Architecture overview
     - Security considerations
     - Production recommendations
     - Test credentials
   - Created `/mnt/workspace/projects/lms-ziad/lms-monorepo/gateway/TEST_INSTRUCTIONS.md`:
     - Step-by-step test running instructions
   - Created `/mnt/workspace/projects/lms-ziad/lms-monorepo/TEST_SUMMARY.md`:
     - Test results and coverage metrics

6. **Unit Tests** (20 tests, 60.4% coverage)
   - Created `/mnt/workspace/projects/lms-ziad/lms-monorepo/gateway/internal/web/student_handler_test.go`:
     - 6 tests covering all student CRUD operations
     - Mock StudentServiceClient and TeacherServiceClient
   - Created `/mnt/workspace/projects/lms-ziad/lms-monorepo/gateway/internal/web/teacher_handler_test.go`:
     - 14 tests covering teacher, course, enrollment, grade, and dashboard endpoints
     - Comprehensive mock gRPC client implementations
   - All tests pass with isolated unit testing (no database/services required)
   - Execution time: ~15ms total

### What We Accomplished

- Fully functional LMS REST API with 24+ endpoints
- Clean separation of concerns with organized handler structure
- Complete JWT authentication and role-based authorization
- Cross-service communication between student and teacher microservices
- Comprehensive API documentation for developers
- Automated unit tests with good coverage
- Production-ready structure (though security needs hardening for production)

### Key Takeaways

1. **Code Organization Pattern**:
   - Gateway uses a struct-based approach with dependency injection
   - Handlers are methods on the `Gateway` struct
   - All gRPC clients are injected at initialization
   - This pattern makes testing easier with mock clients

2. **Authentication Flow**:
   - Public routes: Login and registration endpoints
   - Protected routes: All other endpoints require valid JWT
   - Teacher-only routes: Course/grade management requires teacher role
   - Middleware chain: `gin.Default() -> AuthMiddleware -> TeacherOnly/StudentOnly`

3. **Cross-Service Validation**:
   - Teacher service validates student existence by calling student service
   - Student service validates course/teacher existence by calling teacher service
   - This prevents orphaned records and maintains referential integrity

4. **Testing Strategy**:
   - Unit tests use mock gRPC clients
   - No external dependencies required for tests
   - Mock implementations return predefined success responses
   - Focus on happy path scenarios in unit tests

5. **Proto Pattern**:
   - All RPCs return consistent message types
   - Success responses include relevant entity data
   - Services handle business logic and data validation
   - Gateway translates between HTTP and gRPC

### Technical Details

**Dependencies**:
- Gin Web Framework: HTTP routing and middleware
- gRPC: Microservice communication
- JWT-go: Token generation and validation
- PostgreSQL: Data persistence (via docker-compose)
- OpenTelemetry/Jaeger: Distributed tracing
- Circuit Breaker: Fault tolerance

**Environment Variables** (defined in docker-compose.yml):
- `STUDENT_SERVICE_URL` - gRPC address for student service
- `TEACHER_SERVICE_URL` - gRPC address for teacher service
- `SCHOOL_SERVICE_URL` - gRPC address for school service (NEW)
- `JWT_SECRET` - Secret key for JWT signing (hardcoded in dev)
- Database credentials for each service

**File Structure**:
```
lms-monorepo/
├── gateway/
│   ├── main.go                          # Entry point, route registration
│   ├── internal/web/
│   │   ├── gateway.go                   # Gateway struct
│   │   ├── student_handler.go           # Student endpoints
│   │   ├── teacher_handler.go           # Teacher/course/grade endpoints
│   │   ├── auth_handler.go              # Login endpoints
│   │   ├── auth_middleware.go           # JWT middleware
│   │   ├── student_handler_test.go      # Student tests
│   │   └── teacher_handler_test.go      # Teacher tests
│   └── utils/
│       ├── jwt.go                       # JWT utilities
│       ├── breaker.go                   # Circuit breaker
│       └── telemetry.go                 # Tracing
├── student-service/
│   └── main.go                          # Student microservice
├── teacher-service/
│   └── main.go                          # Teacher microservice
├── school-service/                      # NEW SERVICE
│   ├── main.go                          # School microservice
│   └── telemetry.go                     # OpenTelemetry integration
├── proto/
│   ├── student.proto                    # Student service contract
│   ├── teacher.proto                    # Teacher service contract
│   └── school.proto                     # School service contract (NEW)
├── go.work                              # Go workspace configuration (NEW)
├── API_DOCS.md                          # Complete API documentation
├── AUTH_IMPLEMENTATION.md               # Auth system details
└── TEST_SUMMARY.md                      # Test results
```

**Code Patterns**:
- Error handling: All handlers return JSON error responses with appropriate HTTP status codes
- Context propagation: Request context passed through middleware to handlers
- JWT claims: Stored in gin.Context with key "user_id", "email", "user_type"
- Mock testing: Interfaces allow easy mocking of gRPC clients

### Security Considerations (Development vs Production)

**Current Development Setup**:
- Plain text password storage (NOT production-safe)
- Hardcoded JWT secret
- No refresh token mechanism
- No rate limiting on login endpoints
- HTTP only (no HTTPS)
- No audit logging

**Production Requirements** (documented in AUTH_IMPLEMENTATION.md):
1. Implement bcrypt password hashing
2. Move JWT secret to environment variable
3. Add refresh token mechanism
4. Implement rate limiting on auth endpoints
5. Configure HTTPS/TLS
6. Add comprehensive audit logging
7. Add token blacklisting for logout
8. Implement password complexity requirements

### Known Issues & Limitations

1. **No Logout Endpoint**: JWT tokens remain valid until expiration (24h)
2. **No Password Reset**: Not yet implemented
3. **No Email Verification**: Registration doesn't verify emails
4. **Limited Error Handling**: Some edge cases not covered in handlers
5. **No Pagination**: List endpoints return all results
6. **No Filtering**: Limited query parameter support
7. **Integration Tests Missing**: Only unit tests implemented

### Next Steps

**Immediate (Production Readiness)**:
1. Implement bcrypt password hashing
2. Add environment-based JWT secret configuration
3. Add rate limiting middleware
4. Implement HTTPS/TLS
5. Add comprehensive error handling
6. Add audit logging for authentication events

**Short-term Features**:
1. Add pagination to list endpoints
2. Implement logout functionality with token blacklisting
3. Add password reset flow with email verification
4. Add profile picture upload for students/teachers
5. Implement course search and filtering
6. Add bulk enrollment operations

**Testing & Quality**:
1. Add integration tests with real databases
2. Add end-to-end API tests
3. Implement load testing
4. Add circuit breaker behavior tests
5. Increase code coverage to 80%+

**Observability**:
1. Configure Jaeger for production
2. Add Prometheus metrics
3. Implement structured logging
4. Add health check endpoints
5. Add performance monitoring

---

## Development Workflow

### Starting the System

```bash
# Start all services with Docker Compose (using profiles)
docker compose --profile all up -d

# OR start specific services only (faster for development)
docker compose --profile student up -d        # Student service only
docker compose --profile teacher up -d        # Teacher service only
docker compose --profile school up -d         # School service only
docker compose --profile gateway --profile student up -d  # Gateway + Student

# Services will be available at:
# - Gateway: http://localhost:3000
# - Student Service: grpc://localhost:50051
# - Teacher Service: grpc://localhost:50052
# - School Service: grpc://localhost:8083
# - Jaeger UI: http://localhost:16686
```

**Profile Guide**:
- `--profile all` - Start everything (gateway + all services + databases + Jaeger)
- `--profile student` - Student service + postgres-student
- `--profile teacher` - Teacher service + postgres-teacher
- `--profile school` - School service + postgres-school
- `--profile gateway` - Gateway service only
- Combine profiles: `--profile student --profile teacher` for multiple services

### Running Tests

```bash
# Run all gateway tests
cd gateway
go test ./internal/web -v

# Run with coverage
go test ./internal/web -cover

# Expected: 20 tests, 60.4% coverage, ~15ms execution
```

### Making Code Changes

1. **Adding New Endpoints**:
   - Add RPC to relevant `.proto` file (student.proto, teacher.proto, or school.proto)
   - Regenerate proto files: `make proto`
   - Implement RPC in appropriate service (student-service, teacher-service, or school-service)
   - Add handler method to appropriate handler file in gateway
   - Register route in `gateway/main.go`
   - Apply appropriate middleware (AuthMiddleware, TeacherOnly, StudentOnly, or AdminOnly)
   - Add unit test to corresponding test file
   - Update `API_DOCS.md`

2. **Modifying Authentication**:
   - JWT logic in `/mnt/workspace/projects/lms-ziad/lms-monorepo/gateway/utils/jwt.go`
   - Middleware in `/mnt/workspace/projects/lms-ziad/lms-monorepo/gateway/internal/web/auth_middleware.go`
   - Login handlers in `/mnt/workspace/projects/lms-ziad/lms-monorepo/gateway/internal/web/auth_handler.go`
   - Service-level login in `teacher-service/main.go` and `student-service/main.go`

3. **Database Changes**:
   - Modify schema in service's SQL initialization
   - Update proto messages if response structure changes
   - Regenerate proto files
   - Update handlers if needed

### Testing the API

See `/mnt/workspace/projects/lms-ziad/lms-monorepo/API_DOCS.md` for complete endpoint documentation.

**Quick Test Flow**:
```bash
# 1. Register a teacher
curl -X POST http://localhost:3000/api/v1/teachers \
  -H "Content-Type: application/json" \
  -d '{"email":"test@uni.edu","password":"secret","full_name":"Test Teacher"}'

# 2. Login
curl -X POST http://localhost:3000/api/v1/auth/teacher/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@uni.edu","password":"secret"}'

# 3. Use token for protected endpoints
TOKEN="<token-from-step-2>"
curl -X GET http://localhost:3000/api/v1/students \
  -H "Authorization: Bearer $TOKEN"
```

---

## Troubleshooting

### Common Issues

1. **"Service Unavailable" errors**:
   - Check if microservices are running: `docker compose ps`
   - Check service logs: `docker compose logs student-service teacher-service school-service`
   - Verify service URLs in docker-compose.yml
   - Ensure required profiles are running: `docker compose --profile all up`

2. **"Invalid or expired token" errors**:
   - Token expired (24h lifetime) - login again
   - JWT secret mismatch - ensure consistent secret across restarts
   - Malformed Authorization header - format: `Bearer <token>`

3. **"Teacher access only" or "Admin access only" errors**:
   - Logged in as wrong user type trying to access restricted route
   - Use correct login endpoint and token for your user type:
     - Teacher: POST `/api/v1/auth/teacher/login`
     - Student: POST `/api/v1/auth/student/login`
     - Admin: POST `/api/v1/auth/admin/login`

4. **gRPC connection errors**:
   - Services not ready - wait 10-15 seconds after docker compose up
   - Port conflicts - ensure 50051, 50052, 8083 are available
   - Network issues - check docker network connectivity
   - Wrong profile - ensure the service's profile is running

5. **Test failures**:
   - Mock implementations out of sync - update test mocks
   - Import path issues - ensure correct module path in go.mod

---

## Important Conventions

1. **Error Responses**: Always return JSON with `{"error": "message"}` format
2. **Success Responses**: Return relevant entity data, not just success flags
3. **HTTP Status Codes**:
   - 200 OK - Success
   - 201 Created - Resource created
   - 400 Bad Request - Invalid input
   - 401 Unauthorized - Missing/invalid token
   - 403 Forbidden - Insufficient permissions
   - 404 Not Found - Resource not found
   - 409 Conflict - Business logic conflict
   - 500 Internal Server Error - Server error
   - 503 Service Unavailable - Circuit breaker open
4. **Authentication**: Protected routes always check token, role-specific routes check user_type
5. **Authorization Levels**: Three tiers - public, protected (any user), role-specific (teacher/student/admin)
6. **Testing**: Use mocks for unit tests, keep tests fast and isolated
7. **Documentation**: Update API_DOCS.md whenever endpoints change
8. **Profiles**: Use Docker Compose profiles to start only needed services during development

---

## Quick Reference

### Key RabbitMQ Queues
- `grades.assigned` - Published when teacher assigns grade (teacher-service → stats-service)
- `students.deleted` - Published when student deleted (student-service → teacher-service for cleanup)

### Key Environment Variables
- `JWT_SECRET` - Gateway JWT signing key (env var, defaults to dev key)
- `RABBITMQ_URL` - RabbitMQ connection (all services publish/consume)
- `DATABASE_URL` - Service-specific database connection
- `GIN_MODE=release` - Gateway production mode

### Startup Profiles
```bash
docker compose --profile all up              # All services + all databases
docker compose --profile student up          # Student service only
docker compose -f docker-compose.prod.yml up # Production single-DB setup
```

### Known Issues & Workarounds
- Plain-text passwords in dev (for production, implement bcrypt)
- No logout endpoint (tokens valid 24h)
- Stats reconciliation runs every 30s (can cause brief DB load spikes)

---

## Git Workflow

Recent commits:
- `4ac5d58` - docs: new api docs
- `0744aab` - chore: added new enrollment table and add service url into student service container
- `74f4a7d` - feat: implemented API Handler for newest services
- `bd97fca` - feat: implemented new service proto
- `8aad769` - feat: created necessary API
- `f2db749` - refactor: moved handler to dedicated place

Branch: `master` (no main branch configured)

---

## Resources

- **API Documentation**: `API_DOCS.md`
- **Auth System**: `AUTH_IMPLEMENTATION.md`
- **Stats Service**: `STATS_SERVICE_DOCUMENTATION.md` (statistical theory, formulas, code snippets)
- **Production Setup**: `PRODUCTION_SETUP_PLAN.md` + `PRODUCTION_SETUP_VERIFICATION.md`
- **Quick Start (Prod)**: `QUICK_START_PROD.md`
- **Data Sync Issues**: `STATS_SYNC_ISSUE.md` (root cause analysis)
- **Test Results**: `TEST_SUMMARY.md`
- **Jaeger UI**: http://localhost:16686

---

## Project Status

**Current State**: Production-ready LMS with event-driven architecture & data integrity
- 40+ REST API endpoints fully functional
- 5 microservices: gateway, teacher, student, school, stats
- Event-driven stats sync (RabbitMQ: grades.assigned, students.deleted)
- Automatic data reconciliation (30s interval) for consistency
- Category-field consistency across all databases
- Production Docker Compose: single Postgres, resource limits, healthchecks
- Dev Docker Compose: profiles for selective service startup

**Data Integrity**: Multi-layer sync & reconciliation
- Teacher-service publishes GradeAssignedEvent with full payload (category included)
- Stats-service auto-reconciles missing category data via ReconcileGradesSync()
- All assignment CRUD operations preserve category field
- Periodic reconciliation job ensures eventual consistency

**Architecture**: Microservices + event sourcing + CQRS read model
- Gateway translates REST → gRPC
- Teacher/Student/School services: event publishers
- Stats service: CQRS read model consuming events
- Three user types: student, teacher, admin (JWT-based)
- All services support environment-based configuration
