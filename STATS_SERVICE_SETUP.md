# Stats Service Setup Guide

## Overview

The Stats Service is an **event-driven analytics microservice** that provides real-time insights into student performance. It uses a **Read Model** pattern (CQRS) to maintain a local projection of grade data consumed from RabbitMQ events.

### Architecture Highlights

- **Event-Driven**: Consumes `GradeAssigned`, `StudentDeleted` events from RabbitMQ
- **Read Model**: Local PostgreSQL database with denormalized data for fast queries
- **Idempotent**: Handles duplicate events gracefully using grade_id as unique constraint
- **Statistical Analysis**: Provides bell curves, at-risk identification, and category mastery insights

---

## Prerequisites

1. **Generate Proto Files**:
   ```bash
   cd /mnt/workspace/projects/lms-ziad/lms-monorepo/proto
   protoc --go_out=. --go_opt=paths=source_relative \
     --go-grpc_out=. --go-grpc_opt=paths=source_relative \
     stats.proto
   ```

2. **Initialize Go Modules**:
   ```bash
   cd /mnt/workspace/projects/lms-ziad/lms-monorepo/stats-service
   go mod download
   ```

---

## Starting the Service

### Option 1: Start Stats Service Only
```bash
docker compose --profile stats up -d
```

This starts:
- `stats-service` (port 8084)
- `postgres-stats` (port 5436)
- `rabbitmq` (ports 5672, 15672)
- `jaeger` (ports 16686, 4317)

### Option 2: Start Everything
```bash
docker compose --profile all up -d
```

### Option 3: Development Stack (Gateway + All Services)
```bash
docker compose --profile gateway --profile teacher --profile student --profile school --profile stats up -d
```

---

## Verifying the Service

### 1. Check Service Health
```bash
# Check if stats service is running
docker ps | grep stats_app

# View logs
docker logs -f stats_app
```

You should see:
```
✅ Connected to stats database
✅ RabbitMQ queues declared
📊 [Stats Service] Listening for GradeAssigned events...
🗑️  [Stats Service] Listening for StudentDeleted events...
📊 Stats Service listening on :8080
```

### 2. Verify Database Initialization
```bash
docker exec postgres_stats psql -U stats_admin -d stats_db -c "SELECT COUNT(*) FROM grades;"
```

Should return `4` (seed data).

### 3. Test Event Processing

Assign a new grade through the teacher service - it will publish an event that stats-service consumes:

```bash
# 1. Login as teacher
TOKEN=$(curl -s -X POST http://localhost:3000/api/v1/auth/teacher/login \
  -H "Content-Type: application/json" \
  -d '{"email":"turing@uni.edu","password":"secret"}' | jq -r '.token')

# 2. Assign a grade
curl -X POST http://localhost:3000/api/v1/grades \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "teacher_id": "d290f1ee-6c54-4b01-90e6-d701748f0851",
    "assignment_id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
    "student_id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
    "score": 85
  }'
```

**Check stats-service logs**:
```bash
docker logs stats_app | tail -5
```

You should see:
```
📈 Received GradeAssigned: grade_id=..., student=..., score=85/100
```

---

## Using the Stats API

### 1. Get Performance Distribution (Bell Curve)

```bash
curl -X GET "http://localhost:3000/api/v1/courses/c100f1ee-6c54-4b01-90e6-d701748f0851/stats/distribution" \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "buckets": [
    {"range": "0-10", "count": 0, "percentage": 0},
    {"range": "11-20", "count": 0, "percentage": 0},
    ...
    {"range": "91-100", "count": 2, "percentage": 50}
  ],
  "mean": 91.5,
  "median": 91.5,
  "std_deviation": 3.5,
  "total_students": 2
}
```

**What it tells you**: Distribution of student scores across 10-point buckets

---

### 2. Get At-Risk Students

```bash
curl -X GET "http://localhost:3000/api/v1/courses/c100f1ee-6c54-4b01-90e6-d701748f0851/stats/at-risk" \
  -H "Authorization: Bearer $TOKEN"
```

**Optional Query Parameters**:
- `missing_threshold=3` - Number of missing assignments to flag (default: 3)
- `std_dev_threshold=2.0` - Standard deviations below mean (default: 2.0)

**Response**:
```json
{
  "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "at_risk_students": [
    {
      "student_id": "a888f1ee-6c54-4b01-90e6-d701748f0852",
      "student_name": "Jane Smith",
      "student_number": "STD-2026-002",
      "current_average": 44.0,
      "class_mean": 63.33,
      "deviation_from_mean": 2.5,
      "missing_assignments": 1,
      "total_assignments": 2,
      "risk_factors": ["Low Performance"]
    }
  ],
  "class_mean": 63.33,
  "class_std_deviation": 7.5,
  "total_students": 2,
  "at_risk_count": 1
}
```

**What it tells you**:
- Students performing >2σ below class average
- Students missing 3+ assignments
- Specific risk factors for intervention

---

### 3. Get Category Mastery

```bash
curl -X GET "http://localhost:3000/api/v1/courses/c100f1ee-6c54-4b01-90e6-d701748f0851/stats/category-mastery" \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "categories": [
    {
      "category": "Exam",
      "average_score": 91.5,
      "average_percentage": 91.5,
      "total_assignments": 1,
      "total_submissions": 2,
      "std_deviation": 3.5
    },
    {
      "category": "Project",
      "average_score": 47.5,
      "average_percentage": 47.5,
      "total_assignments": 1,
      "total_submissions": 1,
      "std_deviation": 0
    }
  ],
  "strongest_category": "Exam",
  "weakest_category": "Project"
}
```

**What it tells you**:
- Students excel at exams but struggle with projects
- Identifies gaps between theoretical knowledge and practical application

---

### 4. Get Course Stats Overview

```bash
curl -X GET "http://localhost:3000/api/v1/courses/c100f1ee-6c54-4b01-90e6-d701748f0851/stats" \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "course_id": "c100f1ee-6c54-4b01-90e6-d701748f0851",
  "total_students": 2,
  "total_assignments": 2,
  "overall_average": 63.33,
  "overall_std_deviation": 25.4,
  "at_risk_count": 1,
  "total_grades_recorded": 3,
  "highest_performing_category": "Exam",
  "lowest_performing_category": "Lab"
}
```

---

## Event Flow Diagram

```
Teacher assigns grade via Gateway
         ↓
Teacher Service saves to DB
         ↓
Teacher Service publishes GradeAssigned event to RabbitMQ
         ↓
Stats Service consumes event
         ↓
Stats Service writes to local Read Model (idempotent)
         ↓
Stats API queries Read Model (fast!)
```

---

## Database Schema

### grades (Main Table)
- Stores denormalized grade data from events
- `percentage` is a **computed column**: `(score / max_score) * 100`
- Unique constraint on `id` (grade_id) ensures idempotency

### assignments
- Tracks assignment metadata (title, category, max_score)
- Built from events or seeded

### student_enrollments
- Auto-populated when first grade is assigned
- Tracks which students are in which courses

### deleted_students (Tombstone Pattern)
- Marks students as deleted without removing data
- Queries exclude deleted students from stats

---

## Troubleshooting

### Stats not updating after grade assignment

1. **Check RabbitMQ**:
   ```bash
   # Open RabbitMQ Management UI
   open http://localhost:15672
   # Login: guest/guest
   # Check if "grades.assigned" queue has messages
   ```

2. **Check Stats Service Logs**:
   ```bash
   docker logs stats_app | grep "GradeAssigned"
   ```

3. **Manually publish test event**:
   ```bash
   docker exec rabbitmq_broker rabbitmqadmin publish \
     routing_key=grades.assigned \
     payload='{"grade_id":"test-123","course_id":"c100f1ee-6c54-4b01-90e6-d701748f0851","assignment_id":"a100f1ee-6c54-4b01-90e6-d701748f0001","student_id":"a999f1ee-6c54-4b01-90e6-d701748f0851","score":95,"max_score":100,"category":"Exam"}'
   ```

### Empty stats returned

- **Cause**: No grades exist for the course
- **Solution**: Assign grades through teacher API first

### Duplicate grade events

- **Expected**: Stats service handles this with `ON CONFLICT DO NOTHING`
- **No action needed**: Idempotency is built-in

---

## Production Recommendations

1. **Separate Read/Write Models**: Already implemented ✅
2. **Event Sourcing**: Consider storing raw events for replay
3. **Caching**: Add Redis for frequently accessed stats
4. **Pagination**: Add limit/offset to stats queries
5. **Real-time Updates**: Add WebSocket support for live dashboards
6. **Data Retention**: Archive old grades after academic year ends
7. **Monitoring**: Add Prometheus metrics for event processing lag

---

## API Endpoints Summary

| Endpoint | Method | Access | Description |
|----------|--------|--------|-------------|
| `/api/v1/courses/:id/stats` | GET | Teacher Only | Course overview stats |
| `/api/v1/courses/:id/stats/distribution` | GET | Teacher Only | Performance bell curve |
| `/api/v1/courses/:id/stats/at-risk` | GET | Teacher Only | At-risk student list |
| `/api/v1/courses/:id/stats/category-mastery` | GET | Teacher Only | Category performance breakdown |

---

## Next Steps

1. **Frontend Integration**: Build charts using Chart.js or D3.js
2. **Email Alerts**: Send weekly reports to teachers about at-risk students
3. **Student View**: Allow students to see their own performance vs. class average
4. **Predictive Analytics**: ML model to predict final grades based on current trajectory
5. **Export Reports**: Generate PDF reports for parent-teacher conferences

---

## Questions?

Check the logs:
```bash
docker logs stats_app -f
```

Inspect the database:
```bash
docker exec -it postgres_stats psql -U stats_admin -d stats_db
```

Test event publishing:
```bash
# The teacher service automatically publishes events when grades are assigned
# No manual intervention needed!
```
