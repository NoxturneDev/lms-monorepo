# Stats Service Sync Issue - Root Cause Analysis

**Status**: ❌ BROKEN - Event Publishing Missing
**Date**: 2026-02-23
**Impact**: Stats database is not receiving grade updates from teacher-service

---

## Problem Statement

When you save a new grade in the teacher-service, the stats-service does not receive or process it. The stats database remains out of sync with the teacher database.

---

## Root Cause

**The teacher-service does NOT publish GradeAssignedEvent to RabbitMQ after saving grades.**

### Current (Broken) Flow

```
Teacher Service                    Stats Service
     ↓                                    ↓
AssignGrade RPC
     ↓
Save to teacher_db ✅
     ↓
[MISSING] Publish event to RabbitMQ ❌
     ↓
Stats service waiting...                (Nothing arrives)
     ↓
Stats database out of sync ❌
```

### What Should Happen (Correct Flow)

```
Teacher Service                    RabbitMQ              Stats Service
     ↓                              ↓                          ↓
AssignGrade RPC
     ↓
Save to teacher_db ✅
     ↓
Publish GradeAssignedEvent ✅  ──→ grades.assigned queue
                                     ↓
                              Stats Service consumes ✅
                                     ↓
                              Save to stats_db ✅
```

---

## Code Analysis

### Current Teacher-Service Implementation

**File**: `teacher-service/main.go`

**Server Struct** (lines 25-30):
```go
type server struct {
    teacherpb.UnimplementedTeacherServiceServer
    db            *sql.DB
    studentClient studentpb.StudentServiceClient
    schoolClient  schoolpb.SchoolServiceClient
    // ❌ MISSING: rabbitChannel *amqp.Channel
}
```

**AssignGrade Function** (lines 89-127):
```go
func (s *server) AssignGrade(ctx context.Context, req *teacherpb.AssignGradeRequest) (*teacherpb.GradeResponse, error) {
    log.Printf("Assigning Grade for Student %v on Assignment %v", req.StudentId, req.AssignmentId)

    // 1. Validate student exists
    _, err := s.studentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{
        Id: req.StudentId,
    })
    if err != nil {
        return nil, fmt.Errorf("student not found: %v", err)
    }

    // 2. Look up assignment
    var courseID string
    var maxScore int32
    err = s.db.QueryRow("SELECT course_id, max_score FROM assignments WHERE id = $1", req.AssignmentId).Scan(&courseID, &maxScore)
    if err != nil {
        return nil, err
    }

    if req.Score > maxScore {
        return nil, fmt.Errorf("score %d exceeds max score %d", req.Score, maxScore)
    }

    // 3. Save to DB
    query := `INSERT INTO grades (assignment_id, student_id, score) VALUES ($1, $2, $3) RETURNING id`
    var id string
    err = s.db.QueryRow(query, req.AssignmentId, req.StudentId, req.Score).Scan(&id)
    if err != nil {
        return nil, fmt.Errorf("failed to assign grade: %v", err)
    }

    // ❌ MISSING: Publish GradeAssignedEvent to RabbitMQ!

    return &teacherpb.GradeResponse{Id: id, Success: true}, nil
}
```

**What's Missing**:
- After line 124 (successful DB insert), should publish event
- No `rabbitChannel` field in server struct
- No event publishing code anywhere in the file

---

## Comparison: Student-Service (Working Example)

The student-service has the correct pattern:

**Server Struct** (includes rabbitChannel):
```go
type server struct {
    pb.UnimplementedStudentServiceServer
    db            *sql.DB
    rabbitChannel *amqp.Channel          // ✅ Has RabbitMQ channel
    schoolClient  schoolpb.SchoolServiceClient
}
```

**DeleteStudent Function** (fire-and-forget pattern):
```go
func (s *server) DeleteStudent(ctx context.Context, req *pb.DeleteStudentRequest) (*pb.DeleteStudentResponse, error) {
    log.Printf("Deleting Student: %v", req.Id)

    // 1. Delete from DB
    _, err := s.db.Exec("DELETE FROM students WHERE id = $1", req.Id)
    if err != nil {
        return nil, fmt.Errorf("failed to delete student: %v", err)
    }

    // 2. FIRE AND FORGET (Async Event) ✅
    go func() {
        err := s.PublishEvent("StudentDeleted", map[string]string{"id": req.Id})
        if err != nil {
            log.Printf("Failed to publish event: %v", err)
        }
    }()

    return &pb.DeleteStudentResponse{Success: true}, nil
}
```

**PublishEvent Method**:
```go
func (s *server) PublishEvent(event string, payload interface{}) error {
    body, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    err = s.rabbitChannel.Publish(
        "",               // exchange
        "student_events", // routing key (Queue Name)
        false,            // mandatory
        false,            // immediate
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
            Type:        event,
        },
    )
    log.Printf("Published Event: %s", event)
    return err
}
```

---

## Solution: Add Event Publishing to Teacher-Service

### Step 1: Update Server Struct

Add `rabbitChannel` field:

```go
type server struct {
    teacherpb.UnimplementedTeacherServiceServer
    db            *sql.DB
    rabbitChannel *amqp.Channel                    // ✅ ADD THIS
    studentClient studentpb.StudentServiceClient
    schoolClient  schoolpb.SchoolServiceClient
}
```

---

### Step 2: Add PublishEvent Method

```go
func (s *server) PublishEvent(event string, payload interface{}) error {
    body, err := json.Marshal(payload)
    if err != nil {
        return err
    }

    err = s.rabbitChannel.Publish(
        "",             // exchange
        "grades.assigned",  // routing key (Queue Name for grades)
        false,          // mandatory
        false,          // immediate
        amqp.Publishing{
            ContentType: "application/json",
            Body:        body,
            Type:        event,
        },
    )
    log.Printf("Published Event: %s", event)
    return err
}
```

---

### Step 3: Publish Event in AssignGrade

After successful DB insert (around line 124):

```go
func (s *server) AssignGrade(ctx context.Context, req *teacherpb.AssignGradeRequest) (*teacherpb.GradeResponse, error) {
    // ... validation code ...

    // Save to DB
    query := `INSERT INTO grades (assignment_id, student_id, score) VALUES ($1, $2, $3) RETURNING id`
    var id string
    err = s.db.QueryRow(query, req.AssignmentId, req.StudentId, req.Score).Scan(&id)
    if err != nil {
        return nil, fmt.Errorf("failed to assign grade: %v", err)
    }

    // ✅ ADD THIS: Publish event asynchronously
    go func() {
        gradeEvent := map[string]interface{}{
            "grade_id":      id,
            "course_id":     courseID,
            "assignment_id": req.AssignmentId,
            "student_id":    req.StudentId,
            "score":         req.Score,
            "max_score":     maxScore,
            "timestamp":     time.Now().UTC().Format(time.RFC3339),
        }

        err := s.PublishEvent("GradeAssigned", gradeEvent)
        if err != nil {
            log.Printf("Failed to publish GradeAssigned event: %v", err)
            // Don't fail the RPC - event can be retried later
        }
    }()

    return &teacherpb.GradeResponse{Id: id, Success: true}, nil
}
```

---

### Step 4: Initialize RabbitMQ Channel in main()

Update the `main()` function to create the channel and pass it to the server:

**Find this section** (around line 600+):
```go
srv := &server{
    db: db,
}
```

**Change to**:
```go
// Initialize RabbitMQ
rabbitURL := os.Getenv("RABBITMQ_URL")
if rabbitURL == "" {
    rabbitURL = "amqp://guest:guest@rabbitmq:5672/"
}

conn, err := amqp.Dial(rabbitURL)
if err != nil {
    log.Fatalf("Failed to connect to RabbitMQ: %v", err)
}
defer conn.Close()

ch, err := conn.Channel()
if err != nil {
    log.Fatalf("Failed to open channel: %v", err)
}
defer ch.Close()

// Declare the queue (idempotent)
_, err = ch.QueueDeclare("grades.assigned", true, false, false, false, nil)
if err != nil {
    log.Fatalf("Failed to declare queue: %v", err)
}

srv := &server{
    db:            db,
    rabbitChannel: ch,  // ✅ ADD THIS
    studentClient: studentClient,
    schoolClient:  schoolClient,
}
```

---

## Event Format

When a grade is assigned, this is what should be published:

```json
{
  "grade_id": "grade-001",
  "course_id": "course-algorithms",
  "assignment_id": "assign-mt1",
  "student_id": "student-001",
  "score": 85,
  "max_score": 100,
  "timestamp": "2026-02-23T10:30:00Z"
}
```

The stats-service expects this in the `GradeAssignedEvent` struct (stats-service/main.go, lines 33-43):

```go
type GradeAssignedEvent struct {
    GradeID      string `json:"grade_id"`
    CourseID     string `json:"course_id"`
    AssignmentID string `json:"assignment_id"`
    StudentID    string `json:"student_id"`
    Score        int32  `json:"score"`
    MaxScore     int32  `json:"max_score"`
    Category     string `json:"category"`      // Note: Optional in event
    TeacherID    string `json:"teacher_id"`    // Note: Optional in event
    Timestamp    string `json:"timestamp"`
}
```

---

## RabbitMQ Queue Configuration

**Queue Name**: `grades.assigned`
**Durable**: Yes (persists across restarts)
**Auto-delete**: No
**Exclusive**: No

Both services need to declare the queue:
- ✅ **Stats Service** declares it (line 665 in stats-service/main.go)
- ❌ **Teacher Service** should declare it (missing)

---

## Testing the Fix

After implementing the changes:

### 1. Start the system
```bash
docker compose -f docker-compose.prod.yml up --build -d
```

### 2. Check RabbitMQ
```bash
# View queues
docker exec lms_rabbitmq rabbitmqctl list_queues

# Expected output:
# grades.assigned  0
# students.deleted 0
```

### 3. Assign a grade via API
```bash
# Login as teacher
TOKEN=$(curl -s -X POST http://localhost:3000/api/v1/auth/teacher/login \
  -H "Content-Type: application/json" \
  -d '{"email":"turing@uni.edu","password":"secret"}' | jq -r '.token')

# Assign grade (adjust IDs as needed)
curl -X POST http://localhost:3000/api/v1/grades \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "student_id": "a999f1ee-6c54-4b01-90e6-d701748f0851",
    "assignment_id": "a100f1ee-6c54-4b01-90e6-d701748f0001",
    "score": 95
  }'
```

### 4. Check queue message count
```bash
docker exec lms_rabbitmq rabbitmqctl list_queues

# Expected: grades.assigned should show 1 message
```

### 5. Check stats database was updated
```bash
# Query stats_db
docker exec lms_postgres psql -U stats_admin -d stats_db -c \
  "SELECT course_id, student_id, score, max_score FROM grades ORDER BY recorded_at DESC LIMIT 1;"

# Should show the new grade!
```

### 6. Query stats via API
```bash
curl -X GET 'http://localhost:3000/api/v1/courses/:course_id/stats' \
  -H "Authorization: Bearer $TOKEN"

# Should now include the new grade in calculations
```

---

## Why This Pattern?

### Fire-and-Forget Async

The event is published asynchronously (in a goroutine) for good reasons:

1. **Non-blocking**: API response doesn't wait for event publishing
2. **Resilience**: If RabbitMQ is temporarily down, the grade is still saved in DB
3. **Scale**: Multiple grades can be assigned simultaneously without queue delays
4. **Reliability**: Failed publishes are logged but don't fail the RPC

```go
go func() {
    err := s.PublishEvent("GradeAssigned", gradeEvent)
    if err != nil {
        log.Printf("Failed to publish event: %v", err)
        // Message is in DB - could be manually reprocessed if needed
    }
}()
```

---

## Summary

| Component | Current | Needed |
|-----------|---------|--------|
| **Server struct rabbitChannel** | ❌ Missing | ✅ Add field |
| **PublishEvent method** | ❌ Missing | ✅ Add method |
| **Event publishing in AssignGrade** | ❌ Missing | ✅ Add code |
| **RabbitMQ initialization** | ❌ Not done | ✅ Initialize |
| **Queue declaration** | ❌ Not declared | ✅ Declare queue |
| **Stats service consuming** | ✅ Ready | (No change) |

**Time to Fix**: ~15 minutes
**Files to Modify**: 1 (teacher-service/main.go)
**Risk Level**: Low (non-breaking change)

