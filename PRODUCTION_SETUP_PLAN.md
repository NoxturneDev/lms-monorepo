# Production Docker Compose & Code Optimization Plan

## Overview

This plan transforms the LMS monorepo from a **development-focused** multi-database setup to a **production-prototyping** architecture optimized for minimal resource usage and binary deployments.

### Current State (Development)
- 4 separate PostgreSQL containers (one per service)
- All services use `development` Dockerfile stage (Air hot reload, 1.5GB+ images)
- Hardcoded values: JWT secret, RabbitMQ URLs
- Environment-specific configuration scattered across compose files

### Target State (Production-Ready)
- **Single PostgreSQL container** with 4 databases (teacher_db, student_db, school_db, stats_db)
- **Production binaries** - optimized builds, ~15MB Alpine images instead of 1.5GB
- **Environment variables** - JWT_SECRET, RABBITMQ_URL from compose
- **Resource limits** - memory capping, healthchecks for reliability
- **Minimal footprint** - faster startup, lower disk/memory usage

---

## Implementation Steps

### 1. Create `infra/postgres/init-all.sql`

**Purpose**: Single consolidated database initialization script for production Postgres

**Structure**:
```sql
-- Create 4 databases
CREATE DATABASE teacher_db;
CREATE DATABASE student_db;
CREATE DATABASE school_db;
CREATE DATABASE stats_db;

-- Create per-service users with scoped permissions
CREATE USER teacher_admin WITH PASSWORD 'teacher_password';
CREATE USER student_admin WITH PASSWORD 'student_password';
CREATE USER school_admin WITH PASSWORD 'school_password';
CREATE USER stats_admin WITH PASSWORD 'stats_password';

-- Grant database-level permissions (least privilege)
GRANT ALL PRIVILEGES ON DATABASE teacher_db TO teacher_admin;
GRANT ALL PRIVILEGES ON DATABASE student_db TO student_admin;
GRANT ALL PRIVILEGES ON DATABASE school_db TO school_admin;
GRANT ALL PRIVILEGES ON DATABASE stats_db TO stats_admin;

-- Run all schema initializations under their respective DBs
\c teacher_db
  [contents of teacher-init.sql]

\c student_db
  [contents of student-init.sql]

\c school_db
  [contents of school-init.sql]

\c stats_db
  [contents of stats-init.sql]
```

**Benefits**:
- Single container startup
- All schemas created atomically
- Consistent seeding across all databases
- Easier database backup/restore

---

### 2. Create `docker-compose.prod.yml`

**Purpose**: Production-optimized compose file for prototyping

**Key Changes**:
| Aspect | Dev | Prod |
|--------|-----|------|
| **Postgres** | 4 containers | 1 container |
| **Dockerfile target** | `development` (Air) | `production` (binary) |
| **Image size** | ~1.5GB each | ~15MB each |
| **Volume mounts** | Yes (hot reload) | No |
| **Restart policy** | `on-failure` | `unless-stopped` |
| **Network** | Named network | Named network |
| **Healthchecks** | None | Yes (Postgres, RabbitMQ) |
| **Resource limits** | None | 64-256MB per service |
| **Environment** | Dev URLs | Prod URLs + secrets |

**Service Breakdown**:

| Service | Image | Port | Memory | Notes |
|---------|-------|------|--------|-------|
| postgres | postgres:15-alpine | 5432 | 256MB | Single instance, all 4 DBs |
| rabbitmq | rabbitmq:3-alpine | 5672 | 128MB | No management plugin (prod) |
| gateway | production binary | 3000 | 64MB | Compiled binary |
| teacher-service | production binary | 8080 | 64MB | Compiled binary |
| student-service | production binary | 8080 | 64MB | Compiled binary |
| school-service | production binary | 8080 | 64MB | Compiled binary |
| stats-service | production binary | 8080 | 64MB | Compiled binary |

**Environment Variables**:
```yaml
gateway:
  - JWT_SECRET=your-production-secret-here
  - GIN_MODE=release
  - STUDENT_SERVICE_URL=student-service:8080
  - TEACHER_SERVICE_URL=teacher-service:8080
  - SCHOOL_SERVICE_URL=school-service:8080
  - STATS_SERVICE_URL=stats-service:8080

services:
  - DATABASE_URL=postgres://[user]:[pass]@postgres:5432/[db]
  - RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
```

**Healthchecks**:
```yaml
postgres:
  test: ["CMD-SHELL", "pg_isready -U lms_admin"]
  interval: 10s
  timeout: 5s
  retries: 5

rabbitmq:
  test: ["CMD", "rabbitmq-diagnostics", "ping"]
  interval: 10s
  timeout: 5s
  retries: 5
```

**Startup Dependencies**:
```yaml
depends_on:
  postgres:
    condition: service_healthy
  rabbitmq:
    condition: service_healthy
```

---

### 3. Add Production Stage to Gateway Dockerfile

**File**: `gateway/Dockerfile`

**Current**: Only has `development` stage

**Add**:
```dockerfile
# Stage 2: Builder
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY proto/ proto/
COPY gateway/ gateway/
WORKDIR /app/gateway
ENV GOWORK=off
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gateway-server .

# Stage 3: Production
FROM alpine:latest AS production
WORKDIR /root/
RUN apk --no-cache add ca-certificates
COPY --from=builder /gateway-server .
EXPOSE 3000
CMD ["./gateway-server"]
```

**Note**: Gateway uses `CGO_ENABLED=0` (no cgo dependencies), so no sqlite-libs needed

---

### 4. Code Changes for Environment Variables

#### 4a. `gateway/utils/jwt.go`

**Change**: Read JWT_SECRET from environment variable

```go
// Before:
var jwtSecret = []byte("your-secret-key-change-in-production")

// After:
var jwtSecret []byte

func init() {
    secret := os.Getenv("JWT_SECRET")
    if secret == "" {
        secret = "your-secret-key-change-in-production"
    }
    jwtSecret = []byte(secret)
}
```

**Impact**: JWT signing now uses environment-provided secret in production

---

#### 4b. `student-service/main.go` (line ~296)

**Change**: Read RABBITMQ_URL from environment variable

```go
// Before:
rabbitConn, rabbitErr = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")

// After:
rabbitURL := os.Getenv("RABBITMQ_URL")
if rabbitURL == "" {
    rabbitURL = "amqp://guest:guest@rabbitmq:5672/"
}
// ... then use rabbitURL in retry loop
```

**Impact**: Event publishing now uses environment-provided RabbitMQ URL

---

#### 4c. `teacher-service/main.go` (line ~34)

**Change**: Read RABBITMQ_URL from environment variable

```go
// Before:
conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")

// After:
rabbitURL := os.Getenv("RABBITMQ_URL")
if rabbitURL == "" {
    rabbitURL = "amqp://guest:guest@rabbitmq:5672/"
}
conn, err := amqp.Dial(rabbitURL)
```

**Impact**: Event consumption now uses environment-provided RabbitMQ URL

---

#### 4d. `gateway/main.go`

**Status**: No code change needed! Gin already reads `GIN_MODE` environment variable automatically.

Just set `GIN_MODE=release` in compose.

---

### 5. Fix Dockerfile Bugs

#### Bug 1: `teacher-service/Dockerfile` (lines 22-23)

**Current** (Wrong):
```dockerfile
COPY student-service/.air.toml .
COPY student-service/entrypoint.sh .
```

**Fix** (Correct):
```dockerfile
COPY teacher-service/.air.toml .
COPY teacher-service/entrypoint.sh .
```

#### Bug 2: `school-service/Dockerfile` (lines 22-23)

**Current** (Wrong):
```dockerfile
COPY student-service/.air.toml .
COPY student-service/entrypoint.sh .
```

**Fix** (Correct):
```dockerfile
COPY school-service/.air.toml .
COPY school-service/entrypoint.sh .
```

**Impact**: Development stage now copies correct service-specific files

---

## File Changes Summary

| File | Action | Purpose |
|------|--------|---------|
| `infra/postgres/init-all.sql` | **CREATE** | Consolidated DB init for single Postgres |
| `docker-compose.prod.yml` | **CREATE** | Production compose (optimized, single DB) |
| `gateway/Dockerfile` | **EDIT** | Add builder + production stages |
| `gateway/utils/jwt.go` | **EDIT** | JWT_SECRET from env var |
| `student-service/main.go` | **EDIT** | RABBITMQ_URL from env var |
| `teacher-service/main.go` | **EDIT** | RABBITMQ_URL from env var |
| `teacher-service/Dockerfile` | **EDIT** | Fix .air.toml copy path |
| `school-service/Dockerfile` | **EDIT** | Fix .air.toml copy path |

---

## Usage

### Development (Current)
```bash
# Start specific services with hot reload
docker compose --profile student up
docker compose --profile teacher --profile student up
docker compose --profile all up
```

### Production (New)
```bash
# Start optimized production stack
docker compose -f docker-compose.prod.yml up --build -d

# Verify all containers running
docker compose -f docker-compose.prod.yml ps

# View logs
docker compose -f docker-compose.prod.yml logs -f gateway

# Stop
docker compose -f docker-compose.prod.yml down
```

### Testing Production
```bash
# Create teacher
curl -X POST http://localhost:3000/api/v1/teachers \
  -H "Content-Type: application/json" \
  -d '{"email":"turing@uni.edu","password":"secret","full_name":"Alan Turing"}'

# Login
curl -X POST http://localhost:3000/api/v1/auth/teacher/login \
  -H "Content-Type: application/json" \
  -d '{"email":"turing@uni.edu","password":"secret"}'

# Test with token
TOKEN="<token-from-login>"
curl -X GET http://localhost:3000/api/v1/students \
  -H "Authorization: Bearer $TOKEN"
```

---

## Benefits

### Resource Efficiency
- ✅ Single Postgres container (not 4)
- ✅ Production binaries (~15MB vs 1.5GB)
- ✅ Memory limits prevent runaway usage
- ✅ Alpine base images (~5MB vs Debian)

### Reliability
- ✅ Healthchecks ensure proper startup ordering
- ✅ `unless-stopped` restart policy
- ✅ Dependency ordering via `condition: service_healthy`

### Security
- ✅ JWT secret from environment (not hardcoded)
- ✅ RabbitMQ URL from environment
- ✅ Per-database users with scoped permissions
- ✅ No management plugins in production

### Maintainability
- ✅ Separate compose files for dev vs prod
- ✅ Consolidated DB schema in single file
- ✅ Dockerfile bugs fixed
- ✅ Clear environment variable strategy

---

## Migration Path

### Keep Development Workflow
- `docker-compose.yml` remains unchanged (with Air, separate DBs)
- Use for local development with hot reload

### Add Production Option
- `docker-compose.prod.yml` for optimized prototyping/demo deployments
- Use for performance testing, docker registry builds

### In the Future
- Consider Kubernetes manifests for true production
- Add CI/CD pipeline to build and push production images
- Implement proper secret management (Vault, AWS Secrets Manager)
- Add monitoring and alerting

---

## Verification Checklist

After implementation:

- [ ] `init-all.sql` creates all 4 databases with correct schemas
- [ ] `docker-compose.prod.yml` builds all services successfully
- [ ] Gateway Dockerfile has builder and production stages
- [ ] `jwt.go` reads JWT_SECRET from env
- [ ] `student-service/main.go` reads RABBITMQ_URL from env
- [ ] `teacher-service/main.go` reads RABBITMQ_URL from env
- [ ] Teacher-service Dockerfile copies correct .air.toml path
- [ ] School-service Dockerfile copies correct .air.toml path
- [ ] Production stack starts: `docker compose -f docker-compose.prod.yml up --build`
- [ ] All containers healthy: `docker compose -f docker-compose.prod.yml ps`
- [ ] API responds: `curl http://localhost:3000/api/v1/auth/teacher/login`
- [ ] Database contains seed data: `docker exec lms_postgres psql -U teacher_admin -d teacher_db -c "SELECT * FROM teachers;"`

