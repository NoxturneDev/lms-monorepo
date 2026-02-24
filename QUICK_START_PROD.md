# Quick Start: Production Docker Compose

## One-Command Start

```bash
docker compose -f docker-compose.prod.yml up --build -d
```

## Verify Everything's Running

```bash
# Check containers
docker compose -f docker-compose.prod.yml ps

# Should see all 7 containers (postgres, rabbitmq, gateway, and 4 services)
```

## Test the API

```bash
# 1. Register a teacher
curl -X POST http://localhost:3000/api/v1/teachers \
  -H "Content-Type: application/json" \
  -d '{"email":"turing@uni.edu","password":"secret","full_name":"Alan Turing"}'

# 2. Login
curl -X POST http://localhost:3000/api/v1/auth/teacher/login \
  -H "Content-Type: application/json" \
  -d '{"email":"turing@uni.edu","password":"secret"}'
# Copy the token from response

# 3. Use the token
TOKEN="<paste-token-here>"
curl -X GET http://localhost:3000/api/v1/students \
  -H "Authorization: Bearer $TOKEN"
```

## Check Database

```bash
# Connect to teacher database
docker exec lms_postgres psql -U teacher_admin -d teacher_db -c "SELECT * FROM teachers;"

# Check all 4 databases exist
docker exec lms_postgres psql -U lms_admin -d postgres -c "\l"
```

## View Logs

```bash
# Gateway logs
docker compose -f docker-compose.prod.yml logs -f gateway

# All services
docker compose -f docker-compose.prod.yml logs -f

# Stop following
# Ctrl+C
```

## Check Resource Usage

```bash
# See memory/CPU usage
docker stats --no-stream

# Expected:
# - postgres: 50-100MB
# - rabbitmq: 20-30MB
# - services: 10-15MB each
```

## Stop Everything

```bash
docker compose -f docker-compose.prod.yml down
```

## Advanced: Check Image Sizes

```bash
# Production images should be ~15MB each
docker images | grep 'lms-monorepo.*production'

# Compare with development (1.5GB each)
docker images | grep 'lms-monorepo.*development'
```

## Environment Variables (Production Compose)

Key variables you might want to change:

```yaml
gateway:
  JWT_SECRET: "your-production-secret-change-me"
  # ↑ Change this to a strong secret in production!

postgres:
  POSTGRES_PASSWORD: "lms_password"
  # ↑ Change this in production!
```

## Development vs Production

### Development (Existing)
```bash
# Hot reload, large images, 4 separate databases
docker compose --profile all up
```

### Production (New)
```bash
# Optimized binaries, small images, single database
docker compose -f docker-compose.prod.yml up --build
```

## Files Changed

**Created**:
- `infra/postgres/init-all.sql` - Consolidated DB init
- `docker-compose.prod.yml` - Production compose

**Modified**:
- `gateway/Dockerfile` - Added production stage
- `gateway/utils/jwt.go` - JWT_SECRET from env
- `student-service/main.go` - RABBITMQ_URL from env
- `teacher-service/main.go` - RABBITMQ_URL from env
- `teacher-service/Dockerfile` - Fixed copy paths
- `school-service/Dockerfile` - Fixed copy paths

**Documentation**:
- `PRODUCTION_SETUP_PLAN.md` - Full implementation guide
- `PRODUCTION_SETUP_VERIFICATION.md` - Verification checklist

## Next: Custom Configuration

For production deployment:

1. Change `JWT_SECRET` in compose
2. Change `POSTGRES_PASSWORD` in compose
3. Consider using `.env.prod` file instead of hardcoded values
4. Add SSL certificates if deploying to cloud
5. Configure proper logging/monitoring

## Troubleshooting

**"Service Unavailable" on API call?**
```bash
# Give services time to start (30-60 seconds)
# Check Postgres is healthy
docker compose -f docker-compose.prod.yml logs postgres
```

**"connection refused" from services?**
```bash
# Check if all containers are running
docker compose -f docker-compose.prod.yml ps

# If any failed, check logs
docker compose -f docker-compose.prod.yml logs
```

**Need to rebuild images?**
```bash
docker compose -f docker-compose.prod.yml down
docker compose -f docker-compose.prod.yml build --no-cache
docker compose -f docker-compose.prod.yml up -d
```

## Performance

Production stack startup time: **~30-60 seconds**
- Postgres initialization: ~10s
- RabbitMQ startup: ~5s
- Services connecting: ~20-40s

Total memory usage: **~200-300MB** (vs 7.5GB+ for dev)
Total disk (images): **~75MB** (vs 7.5GB for dev)

