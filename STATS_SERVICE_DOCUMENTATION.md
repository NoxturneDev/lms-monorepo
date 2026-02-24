# Stats Service - Comprehensive Documentation

**Version**: 1.0
**Last Updated**: 2026-02-22
**Service Port**: 8080
**Protocol**: gRPC
**Database**: PostgreSQL (stats_db)
**Message Broker**: RabbitMQ

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Architecture Overview](#architecture-overview)
3. [Statistical Theory & Foundations](#statistical-theory--foundations)
4. [Core Statistical Concepts](#core-statistical-concepts)
5. [API Endpoints](#api-endpoints)
6. [Database Schema](#database-schema)
7. [Event-Driven Architecture](#event-driven-architecture)
8. [Important Code Snippets](#important-code-snippets)
9. [Real-World Applications](#real-world-applications)
10. [Usage Examples](#usage-examples)

---

## Executive Summary

The **Stats Service** is a specialized microservice that provides advanced statistical analysis for Learning Management System (LMS) course performance data. It implements a **read model** using the **Event Sourcing** pattern, consuming grade assignment and student deletion events from other services to maintain an optimized analytics database.

### Key Features

✅ **Performance Distribution Analysis** - Histogram of student scores with bell curve statistics
✅ **At-Risk Student Detection** - Multi-factor analysis to identify struggling students
✅ **Category Mastery Analysis** - Track performance across assignment types (exams, projects, labs)
✅ **Course Overview Statistics** - Comprehensive course-level metrics
✅ **Event-Driven Updates** - Real-time data aggregation via RabbitMQ
✅ **Idempotent Processing** - Safe duplicate event handling

### Data Flow

```
Teacher Service               Student Service
      ↓                              ↓
  GradeAssigned            StudentDeleted
      ↓                              ↓
      └─────── RabbitMQ ───────────┘
                  ↓
          Stats Service (Consumer)
                  ↓
           stats_db (Read Model)
                  ↓
         Analytics API Endpoints
```

---

## Architecture Overview

### Service Components

```go
type server struct {
    statspb.UnimplementedStatsServiceServer
    db *sql.DB  // PostgreSQL connection to stats_db
}
```

### Design Pattern: CQRS + Event Sourcing

The Stats Service implements a **Command Query Responsibility Segregation (CQRS)** pattern:

| Aspect | Implementation |
|--------|----------------|
| **Commands** | Grade assignments, student deletions (handled by other services) |
| **Queries** | Four RPC endpoints for analytics (read-only) |
| **Event Log** | RabbitMQ queues (grades.assigned, students.deleted) |
| **Read Model** | PostgreSQL tables optimized for fast analytics queries |

### Key Architectural Principles

1. **Eventual Consistency**: Data synced asynchronously from events
2. **Idempotency**: Safe to reprocess duplicate events
3. **Tombstone Pattern**: Soft deletes track removed students
4. **Denormalization**: Flattened schema for analytics queries
5. **Isolation**: Separate database prevents coupling with transactional systems

---

## Statistical Theory & Foundations

### Overview

The Stats Service employs **Descriptive Statistics** principles to provide actionable insights into course performance. All calculations are based on **population statistics** (not sample statistics) since we analyze all grades in a course.

### Why These Statistics Matter for Education

| Statistic | Purpose | Decision Support |
|-----------|---------|------------------|
| **Mean** | Central tendency | Is the class performing well overall? |
| **Median** | Robust central tendency | What's the typical student's score? (unaffected by outliers) |
| **Standard Deviation** | Spread/Dispersion | How varied are student performances? |
| **Distribution Shape** | Pattern analysis | Are grades clustered or spread? |
| **Z-scores** | Standardized comparison | How far below average is each student? |
| **Categories** | Skill areas | Which topics need more focus? |

---

## Core Statistical Concepts

### 1. Mean (Average)

**Mathematical Definition**:
```
μ = (Σx) / n

Where:
  μ = mean
  Σx = sum of all values
  n = number of values
```

**Implementation**:
```go
func calculateMean(values []float64) float64 {
    if len(values) == 0 {
        return 0
    }
    sum := 0.0
    for _, v := range values {
        sum += v
    }
    return sum / float64(len(values))
}
```

**Use Cases**:
- Overall course average (70.5%)
- Category performance (Exams: 75%, Labs: 82%)
- Class-wide benchmark

**Example**:
```
Student Scores: [85, 90, 78, 92, 88]
Mean = (85+90+78+92+88) / 5 = 433 / 5 = 86.6%
```

---

### 2. Median

**Mathematical Definition**:
```
- Sort values in ascending order
- If odd count: median = middle value
- If even count: median = average of two middle values
```

**Implementation**:
```go
func calculateMedian(values []float64) float64 {
    if len(values) == 0 {
        return 0
    }
    sorted := make([]float64, len(values))
    copy(sorted, values)
    sort.Float64s(sorted)

    mid := len(sorted) / 2
    if len(sorted)%2 == 0 {
        return (sorted[mid-1] + sorted[mid]) / 2
    }
    return sorted[mid]
}
```

**Why It Matters**:
- **Robust to outliers**: Unaffected by extreme scores
- **Typical student**: Represents the middle performer
- **Income-like distribution**: Better than mean for skewed data

**Example**:
```
Sorted Scores: [78, 85, 88, 90, 92]
Median = 88% (middle value)

If one student had 5% (failure):
[5, 78, 85, 88, 90, 92]
Median = (85+88)/2 = 86.5% (much more stable than mean!)
```

---

### 3. Standard Deviation (σ)

**Mathematical Definition**:
```
σ = √[Σ(x - μ)² / n]

Where:
  σ = standard deviation
  x = individual value
  μ = mean
  n = number of values
```

**Implementation**:
```go
func calculateStdDev(values []float64, mean float64) float64 {
    if len(values) == 0 {
        return 0
    }
    sumSquaredDiff := 0.0
    for _, v := range values {
        diff := v - mean
        sumSquaredDiff += diff * diff
    }
    variance := sumSquaredDiff / float64(len(values))
    return math.Sqrt(variance)
}
```

**Interpretation**:
```
If μ = 80% and σ = 5%:
  ±1σ → 75%-85% (68.3% of students typically fall here)
  ±2σ → 70%-90% (95.4% of students typically fall here)
  ±3σ → 65%-95% (99.7% of students typically fall here)
```

**Why It Matters**:
- **Consistency metric**: Low σ = homogeneous class (similar ability levels)
- **Risk indicator**: High σ = diverse performance levels
- **Intervention target**: Students >2σ below mean need help

**Example**:
```
Class 1: [80, 80, 80, 80] → σ = 0% (perfectly uniform)
Class 2: [50, 75, 85, 100] → σ = 17.6% (highly varied)

Interpretation: Class 2 has much wider performance spread
```

---

### 4. Z-Score (Standardized Deviation)

**Mathematical Definition**:
```
z = (x - μ) / σ

Where:
  z = standard deviations from mean
  x = individual value
  μ = mean
  σ = standard deviation
```

**Application in At-Risk Detection**:
```go
// From GetAtRiskStudents (line 361)
deviationFromMean := (classMean - sd.avgPercentage) / classStdDev
if deviationFromMean > stdDevThreshold {  // Default: 2.0
    riskFactors = append(riskFactors, "Low Performance")
    isAtRisk = true
}
```

**Interpretation**:
```
z = -2.0 means: Student is 2 standard deviations BELOW mean
                This is statistically "unusual" and warrants intervention

Example:
  Class Mean: 75%
  Class StdDev: 5%
  Student Average: 65%
  z = (75 - 65) / 5 = 2.0

  Interpretation: Student is 2σ below mean → At-Risk!
```

**Why Z-Scores Are Used**:
- **Normalized comparison**: Works regardless of course difficulty
- **Statistical rigor**: Standard threshold (2σ) = 5% false positive rate
- **Contextual**: Accounts for class variability

---

### 5. Distribution Analysis (Histogram)

**Concept**: Organize scores into ranges and count frequency

**Implementation** (from GetPerformanceDistribution, lines 241-268):
```go
// Create 10-point buckets (0-10, 11-20, ..., 91-100)
buckets := make([]*statspb.ScoreBucket, 10)
bucketCounts := make([]int32, 10)

for _, score := range scores {
    bucketIndex := int(score) / 10
    if bucketIndex > 9 {
        bucketIndex = 9  // Handle perfect 100
    }
    bucketCounts[bucketIndex]++
}

// Convert counts to percentages
for i := 0; i < 10; i++ {
    buckets[i] = &statspb.ScoreBucket{
        Range:      fmt.Sprintf("%d-%d", i*10, i*10+10),
        Count:      bucketCounts[i],
        Percentage: (float64(bucketCounts[i]) / float64(totalStudents)) * 100,
    }
}
```

**Visualization Example**:
```
Score Distribution for "Advanced Algorithms":

0-10:   ░░░ (3 students,  6%)
11-20:  █████ (5 students, 10%)
21-30:  ████████ (8 students, 16%)
31-40:  ██████████ (10 students, 20%)
41-50:  ███████ (7 students, 14%)
51-60:  ████████ (8 students, 16%)
61-70:  ██ (2 students, 4%)
71-80:  ██ (2 students, 4%)
81-90:  (0 students, 0%)
91-100: (0 students, 0%)

Mean: 42%, Median: 40%, StdDev: 18.5%
Pattern: Left-skewed distribution (concentration in lower ranges)
```

**What Bell Curves Tell Us**:
- **Normal/Bell Curve** (μ ± σ): Course is well-designed, balanced difficulty
- **Left-Skewed** (pile-up on right): Course is too easy or students well-prepared
- **Right-Skewed** (pile-up on left): Course is too hard or students underprepared
- **Bimodal** (two peaks): Two distinct student populations (e.g., pre-trained vs new)

---

### 6. Grouped Statistics (Category Mastery)

**Concept**: Aggregate statistics by groups (assignment type)

**SQL Implementation** (from GetCategoryMastery, lines 404-415):
```sql
SELECT
    category,
    AVG(percentage) as avg_percentage,          -- Mean per category
    STDDEV(percentage) as std_dev,              -- Std Dev per category
    COUNT(DISTINCT assignment_id) as total_assignments,
    COUNT(*) as total_submissions
FROM grades
WHERE course_id = $1
  AND student_id NOT IN (SELECT student_id FROM deleted_students)
  AND category IS NOT NULL
GROUP BY category
```

**Purpose**: Identify skill gaps

**Example Output**:
```
Category        Avg%    StdDev   Assignments   Mastery Level
────────────────────────────────────────────────────────────
Exams           72.5%   8.2%     3             ⚠️  Below target (70%)
Labs            85.3%   5.1%     4             ✅ Strong (>80%)
Projects        68.1%   12.4%    2             ❌ Needs work (<70%)
Quizzes         79.8%   6.7%     5             ✅ On track (70-80%)
```

**Pedagogical Interpretation**:
- Students struggle with Projects (low avg, high variance)
- Labs are strength area → Use as model for teaching other areas
- Low StdDev in Labs → Consistent methodology works well
- High StdDev in Projects → Needs clearer rubric or more structure

---

### 7. Multi-Factor Risk Assessment

**Concept**: Combine multiple indicators to identify at-risk students

**Implementation** (from GetAtRiskStudents, lines 353-387):

```go
for _, sd := range students {
    var riskFactors []string
    isAtRisk := false

    // Factor 1: Academic Performance (Statistical)
    deviationFromMean := (classMean - sd.avgPercentage) / classStdDev
    if deviationFromMean > stdDevThreshold {  // Default: 2.0σ
        riskFactors = append(riskFactors, "Low Performance")
        isAtRisk = true
    }

    // Factor 2: Assignment Completion (Behavioral)
    missingAssignments := totalAssignments - sd.completedAssignments
    if missingAssignments >= missingThreshold {  // Default: 3 missing
        riskFactors = append(riskFactors, "Missing Assignments")
        isAtRisk = true
    }

    if isAtRisk {
        atRiskStudents = append(atRiskStudents, &statspb.AtRiskStudent{
            StudentId:          sd.studentID,
            CurrentAverage:     sd.avgPercentage,
            ClassMean:          classMean,
            DeviationFromMean:  deviationFromMean,
            MissingAssignments: missingAssignments,
            RiskFactors:        riskFactors,  // Both factors listed
        })
    }
}
```

**Risk Factor Categories**:

| Factor | Type | Threshold | Interpretation |
|--------|------|-----------|-----------------|
| Low Performance | Statistical | μ - 2σ | Student significantly below class average |
| Missing Assignments | Behavioral | ≥3 | Student not keeping up with coursework |

**Example Risk Matrix**:
```
Student    Avg%    σ from mean    Missing    Risk Factors
───────────────────────────────────────────────────────────
Alice      92%     +0.5           0          None → Not at-risk
Bob        45%     -2.8           5          Low Performance, Missing Assignments
Carol      71%     -2.0           1          Low Performance
Diana      55%     -1.5           4          Missing Assignments
Eve        88%     +0.3           0          None → Not at-risk
```

**Intervention Recommendations**:
- **Low Performance Only**: Subject tutoring, study groups
- **Missing Assignments Only**: Time management coaching, deadline reminders
- **Both**: Comprehensive intervention (academic + behavioral support)

---

## API Endpoints

### 1. GetPerformanceDistribution

**Purpose**: Generate histogram of score distribution with bell curve statistics

**Request**:
```protobuf
message PerformanceDistributionRequest {
  string course_id = 1;
  string assignment_id = 2;  // Optional: specific assignment, if empty = all
}
```

**Response**:
```protobuf
message PerformanceDistributionResponse {
  string course_id = 1;
  string assignment_id = 2;
  repeated ScoreBucket buckets = 3;    // 10 histogram buckets
  double mean = 4;                      // Arithmetic mean
  double median = 5;                    // 50th percentile
  double std_deviation = 6;             // Population std dev
  int32 total_students = 7;
}

message ScoreBucket {
  string range = 1;                    // "0-10", "11-20", etc.
  int32 min_score = 2;
  int32 max_score = 3;
  int32 count = 4;                     // # students in range
  double percentage = 5;               // % of total
}
```

**Code Implementation** (lines 196-279):
```go
func (s *server) GetPerformanceDistribution(ctx context.Context,
    req *statspb.PerformanceDistributionRequest)
    (*statspb.PerformanceDistributionResponse, error) {

    // Query all grades for course (excluding deleted students)
    query := `
        SELECT score, max_score
        FROM grades
        WHERE course_id = $1
          AND student_id NOT IN (SELECT student_id FROM deleted_students)
    `

    // Optional: filter by specific assignment
    if req.AssignmentId != "" {
        query += " AND assignment_id = $2"
    }

    // Execute query and normalize scores to percentages
    var scores []float64
    for rows.Next() {
        var score, maxScore int32
        rows.Scan(&score, &maxScore)
        percentage := (float64(score) / float64(maxScore)) * 100
        scores = append(scores, percentage)
    }

    // Calculate central tendency measures
    mean := calculateMean(scores)
    median := calculateMedian(scores)
    stdDev := calculateStdDev(scores, mean)

    // Organize into 10 histogram buckets
    buckets := make([]*statspb.ScoreBucket, 10)
    for _, score := range scores {
        bucketIndex := int(score) / 10
        if bucketIndex > 9 { bucketIndex = 9 }
        bucketCounts[bucketIndex]++
    }

    // Return distribution data
    return &statspb.PerformanceDistributionResponse{
        CourseId:     req.CourseId,
        Mean:         mean,
        Median:       median,
        StdDeviation: stdDev,
        Buckets:      buckets,
        TotalStudents: int32(len(scores)),
    }, nil
}
```

**Use Cases**:
1. **Course Difficulty Assessment**: If mean < 60%, course may be too hard
2. **Grading Curve Justification**: Visualize if curve is needed
3. **Bimodal Detection**: Identify if two distinct student populations
4. **Trend Monitoring**: Compare same course across semesters

---

### 2. GetAtRiskStudents

**Purpose**: Identify students who need intervention based on performance and engagement

**Request**:
```protobuf
message AtRiskStudentsRequest {
  string course_id = 1;
  int32 missing_assignment_threshold = 2;   // Default: 3
  double std_deviation_threshold = 3;       // Default: 2.0
}
```

**Response**:
```protobuf
message AtRiskStudentsResponse {
  string course_id = 1;
  repeated AtRiskStudent at_risk_students = 2;
  double class_mean = 3;
  double class_std_deviation = 4;
  int32 total_students = 5;
  int32 at_risk_count = 6;
}

message AtRiskStudent {
  string student_id = 1;
  string student_name = 2;
  string student_number = 3;
  double current_average = 4;
  double class_mean = 5;
  double deviation_from_mean = 6;        // Z-score (σ units)
  int32 missing_assignments = 7;
  int32 total_assignments = 8;
  repeated string risk_factors = 9;      // ["Low Performance", "Missing Assignments"]
}
```

**Code Implementation** (lines 285-397):
```go
func (s *server) GetAtRiskStudents(ctx context.Context,
    req *statspb.AtRiskStudentsRequest)
    (*statspb.AtRiskStudentsResponse, error) {

    // Set thresholds (customizable via request)
    missingThreshold := req.MissingAssignmentThreshold
    if missingThreshold == 0 { missingThreshold = 3 }
    stdDevThreshold := req.StdDeviationThreshold
    if stdDevThreshold == 0 { stdDevThreshold = 2.0 }

    // Get total assignments for course
    var totalAssignments int32
    s.db.QueryRowContext(ctx,
        "SELECT COUNT(*) FROM assignments WHERE course_id = $1",
        req.CourseId,
    ).Scan(&totalAssignments)

    // Calculate per-student statistics
    query := `
        SELECT
            student_id,
            AVG(percentage) as avg_percentage,
            COUNT(*) as completed_assignments
        FROM grades
        WHERE course_id = $1
          AND student_id NOT IN (SELECT student_id FROM deleted_students)
        GROUP BY student_id
    `

    // Calculate class-level statistics
    classMean := calculateMean(allAverages)
    classStdDev := calculateStdDev(allAverages, classMean)

    // Identify at-risk students using Z-scores
    var atRiskStudents []*statspb.AtRiskStudent
    for _, sd := range students {
        var riskFactors []string
        isAtRisk := false

        // Statistical Risk: Performance > 2σ below mean
        deviationFromMean := (classMean - sd.avgPercentage) / classStdDev
        if deviationFromMean > stdDevThreshold {
            riskFactors = append(riskFactors, "Low Performance")
            isAtRisk = true
        }

        // Behavioral Risk: Missing too many assignments
        missingAssignments := totalAssignments - sd.completedAssignments
        if missingAssignments >= missingThreshold {
            riskFactors = append(riskFactors, "Missing Assignments")
            isAtRisk = true
        }

        if isAtRisk {
            atRiskStudents = append(atRiskStudents, &statspb.AtRiskStudent{
                StudentId:          sd.studentID,
                CurrentAverage:     sd.avgPercentage,
                ClassMean:          classMean,
                DeviationFromMean:  deviationFromMean,
                MissingAssignments: missingAssignments,
                RiskFactors:        riskFactors,
            })
        }
    }

    return &statspb.AtRiskStudentsResponse{
        CourseId:          req.CourseId,
        AtRiskStudents:    atRiskStudents,
        ClassMean:         classMean,
        ClassStdDeviation: classStdDev,
        TotalStudents:     int32(len(students)),
        AtRiskCount:       int32(len(atRiskStudents)),
    }, nil
}
```

**Use Cases**:
1. **Early Warning System**: Identify struggling students early
2. **Targeted Support**: Allocate tutoring resources to highest-need students
3. **Progress Monitoring**: Track students over time
4. **Intervention Effectiveness**: Measure if interventions help students move off at-risk list

---

### 3. GetCategoryMastery

**Purpose**: Analyze performance by assignment type to identify skill gaps

**Request**:
```protobuf
message CategoryMasteryRequest {
  string course_id = 1;
}
```

**Response**:
```protobuf
message CategoryMasteryResponse {
  string course_id = 1;
  repeated CategoryStats categories = 2;
  string strongest_category = 3;    // Highest average
  string weakest_category = 4;      // Lowest average
}

message CategoryStats {
  string category = 1;                    // "Lab", "Exam", "Quiz", "Project"
  double average_score = 2;
  double average_percentage = 3;          // Normalized 0-100
  int32 total_assignments = 4;
  int32 total_submissions = 5;
  double std_deviation = 6;               // Consistency across students
}
```

**Code Implementation** (lines 403-460):
```go
func (s *server) GetCategoryMastery(ctx context.Context,
    req *statspb.CategoryMasteryRequest)
    (*statspb.CategoryMasteryResponse, error) {

    // Group by category and calculate statistics
    query := `
        SELECT
            category,
            AVG(percentage) as avg_percentage,
            STDDEV(percentage) as std_dev,
            COUNT(DISTINCT assignment_id) as total_assignments,
            COUNT(*) as total_submissions
        FROM grades
        WHERE course_id = $1
          AND student_id NOT IN (SELECT student_id FROM deleted_students)
          AND category IS NOT NULL
        GROUP BY category
    `

    var categories []*statspb.CategoryStats
    var strongest, weakest string
    maxAvg, minAvg := 0.0, 100.0

    for rows.Next() {
        var cat statspb.CategoryStats
        var stdDev sql.NullFloat64
        rows.Scan(&cat.Category, &cat.AveragePercentage, &stdDev,
                  &cat.TotalAssignments, &cat.TotalSubmissions)

        if stdDev.Valid {
            cat.StdDeviation = stdDev.Float64
        }

        categories = append(categories, &cat)

        // Track strongest and weakest
        if cat.AveragePercentage > maxAvg {
            maxAvg = cat.AveragePercentage
            strongest = cat.Category
        }
        if cat.AveragePercentage < minAvg {
            minAvg = cat.AveragePercentage
            weakest = cat.Category
        }
    }

    return &statspb.CategoryMasteryResponse{
        CourseId:          req.CourseId,
        Categories:        categories,
        StrongestCategory: strongest,
        WeakestCategory:   weakest,
    }, nil
}
```

**Pedagogical Analysis**:

```
Category     Avg%    StdDev   Interpretation
─────────────────────────────────────────────
Labs         87%     4.2%     ✅ Strength - High avg, Low variance
Exams        65%     18.3%    ⚠️  Weakness - Low avg, High variance
Projects     72%     8.1%     📊 Mixed - Moderate avg, Moderate variance

Insights:
1. Labs are successful → Teaching method works well
2. Exams show inconsistency (high σ) → Needs clearer instruction
3. Projects are a pain point → May need scaffolding/rubric revision
```

**Action Items**:
- Replicate lab-style teaching for exams
- Provide project templates/scaffolding
- Offer exam prep sessions for struggling students

---

### 4. GetCourseStats

**Purpose**: High-level overview of course performance

**Request**:
```protobuf
message CourseStatsRequest {
  string course_id = 1;
}
```

**Response**:
```protobuf
message CourseStatsResponse {
  string course_id = 1;
  int32 total_students = 2;
  int32 total_assignments = 3;
  double overall_average = 4;
  double overall_std_deviation = 5;
  int32 at_risk_count = 6;
  int32 total_grades_recorded = 7;
  string highest_performing_category = 8;
  string lowest_performing_category = 9;
}
```

**Code Implementation** (lines 466-565):
```go
func (s *server) GetCourseStats(ctx context.Context,
    req *statspb.CourseStatsRequest)
    (*statspb.CourseStatsResponse, error) {

    // Overall course statistics
    query := `
        SELECT
            COUNT(DISTINCT student_id) as total_students,
            COUNT(DISTINCT assignment_id) as total_assignments,
            COUNT(*) as total_grades,
            AVG(percentage) as overall_avg,
            STDDEV(percentage) as overall_std_dev
        FROM grades
        WHERE course_id = $1
          AND student_id NOT IN (SELECT student_id FROM deleted_students)
    `

    var totalStudents, totalAssignments, totalGrades int32
    var overallAvg, overallStdDev sql.NullFloat64

    s.db.QueryRowContext(ctx, query, req.CourseId).Scan(
        &totalStudents, &totalAssignments, &totalGrades,
        &overallAvg, &overallStdDev)

    // Count at-risk students (>2σ below mean)
    var atRiskCount int32
    if overallStdDev.Valid && overallStdDev.Float64 > 0 {
        riskRows, _ := s.db.QueryContext(ctx, `
            SELECT AVG(percentage)
            FROM grades
            WHERE course_id = $1
              AND student_id NOT IN (SELECT student_id FROM deleted_students)
            GROUP BY student_id
        `, req.CourseId)

        for riskRows.Next() {
            var avg float64
            riskRows.Scan(&avg)
            deviation := (overallAvg.Float64 - avg) / overallStdDev.Float64
            if deviation > 2.0 {
                atRiskCount++
            }
        }
    }

    // Category performance
    var highestCat, lowestCat string
    catRows, _ := s.db.QueryContext(ctx, `
        SELECT category, AVG(percentage) as avg_perc
        FROM grades
        WHERE course_id = $1 AND category IS NOT NULL
          AND student_id NOT IN (SELECT student_id FROM deleted_students)
        GROUP BY category
        ORDER BY avg_perc DESC
    `, req.CourseId)

    first := true
    for catRows.Next() {
        var cat string
        var avg float64
        catRows.Scan(&cat, &avg)
        if first { highestCat = cat; first = false }
        lowestCat = cat
    }

    return &statspb.CourseStatsResponse{
        CourseId:                  req.CourseId,
        TotalStudents:             totalStudents,
        TotalAssignments:          totalAssignments,
        OverallAverage:            overallAvg.Float64,
        OverallStdDeviation:       overallStdDev.Float64,
        AtRiskCount:               atRiskCount,
        TotalGradesRecorded:       totalGrades,
        HighestPerformingCategory: highestCat,
        LowestPerformingCategory:  lowestCat,
    }, nil
}
```

**Dashboard Example**:
```
COURSE STATISTICS: Advanced Algorithms
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Enrollment:
  • Total Students: 47
  • Graded: 45 (95.7%)
  • Active: 43 (91.5%)

Performance:
  • Overall Average: 72.3%
  • Standard Deviation: 12.8%
  • Students At-Risk: 6 (12.8%)

Assignment Categories:
  • Highest Performing: Labs (84%)
  • Lowest Performing: Midterm Exam (61%)

Quality Metrics:
  • Total Grades Recorded: 189
  • Grades per Student: 4.2
  • Submission Rate: 96%
```

---

## Database Schema

### Overview

The stats_db uses a **denormalized, event-driven schema** optimized for analytics queries:

```sql
-- 1. COURSES (from CourseCreated events)
CREATE TABLE courses (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. ASSIGNMENTS (from AssignmentCreated events)
CREATE TABLE assignments (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    category VARCHAR(100),  -- "Lab", "Exam", "Quiz", "Project"
    max_score INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. GRADES (Fully denormalized for fast queries)
CREATE TABLE grades (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL,
    assignment_id UUID NOT NULL,
    student_id UUID NOT NULL,
    score INTEGER NOT NULL,
    max_score INTEGER NOT NULL,
    category VARCHAR(100),  -- Denormalized from assignments
    percentage DECIMAL(5,2) GENERATED ALWAYS AS
        ((score::DECIMAL / max_score) * 100) STORED,  -- Computed column
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(id)  -- Idempotency key
);

-- 4. STUDENT_ENROLLMENTS (Track which students are in which courses)
CREATE TABLE student_enrollments (
    course_id UUID NOT NULL,
    student_id UUID NOT NULL,
    enrolled_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (course_id, student_id)
);

-- 5. DELETED_STUDENTS (Tombstone pattern)
CREATE TABLE deleted_students (
    student_id UUID PRIMARY KEY,
    deleted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Performance Indexes
CREATE INDEX idx_grades_course ON grades(course_id);
CREATE INDEX idx_grades_assignment ON grades(assignment_id);
CREATE INDEX idx_grades_student ON grades(student_id);
CREATE INDEX idx_grades_course_student ON grades(course_id, student_id);
CREATE INDEX idx_grades_category ON grades(category);
CREATE INDEX idx_assignments_course ON assignments(course_id);
```

### Key Design Decisions

| Feature | Rationale |
|---------|-----------|
| **Denormalization** | Pre-computed percentage field speeds up analytics queries |
| **Computed Columns** | `percentage` is automatically calculated and stored |
| **Tombstone Pattern** | Soft delete tracks removed students without breaking FK relations |
| **UNIQUE(id)** | Ensures idempotent event processing |
| **Multiple Indexes** | Fast filtering by course, assignment, student, or combination |

### Example Data

```sql
-- Sample: "Advanced Algorithms" Course

grades table (sample):
┌──────────────────┬──────────────────┬──────────────────┬──────┬────┬────────┐
│ id               │ course_id        │ assignment_id    │ score│ max│ percent│
├──────────────────┼──────────────────┼──────────────────┼──────┼────┼────────┤
│ grade-001        │ course-alg       │ assign-mt1       │  85  │ 100│ 85.00  │
│ grade-002        │ course-alg       │ assign-mt1       │  92  │ 100│ 92.00  │
│ grade-003        │ course-alg       │ assign-proj1     │ 180  │ 200│ 90.00  │
└──────────────────┴──────────────────┴──────────────────┴──────┴────┴────────┘

assignments table:
┌──────────────────┬──────────────────┬─────────────────┬──────────┐
│ id               │ course_id        │ title           │ category │
├──────────────────┼──────────────────┼─────────────────┼──────────┤
│ assign-mt1       │ course-alg       │ Midterm Exam    │ Exam     │
│ assign-proj1     │ course-alg       │ Final Project   │ Project  │
└──────────────────┴──────────────────┴─────────────────┴──────────┘
```

---

## Event-Driven Architecture

### Event Types

#### 1. GradeAssignedEvent

**Source**: Teacher Service (when grade is assigned)
**Queue**: `grades.assigned`
**Payload**:

```json
{
  "grade_id": "grade-001",
  "course_id": "course-alg",
  "assignment_id": "assign-mt1",
  "student_id": "student-001",
  "score": 85,
  "max_score": 100,
  "category": "Exam",
  "teacher_id": "teacher-001",
  "timestamp": "2026-02-22T10:30:00Z"
}
```

**Processing** (lines 62-99):

```go
func (s *server) consumeGradeAssignedEvents(ch *amqp.Channel) {
    msgs, _ := ch.Consume("grades.assigned", "", false, false, false, false, nil)

    for msg := range msgs {
        var event GradeAssignedEvent
        json.Unmarshal(msg.Body, &event)

        log.Printf("Processing: student=%s, score=%d/%d",
                   event.StudentID, event.Score, event.MaxScore)

        // Idempotent insert - handles duplicate events gracefully
        if err := s.processGradeAssigned(event); err != nil {
            msg.Nack(false, true)  // Requeue on failure
            continue
        }

        msg.Ack(false)  // Acknowledge on success
    }
}

func (s *server) processGradeAssigned(event GradeAssignedEvent) error {
    // INSERT with conflict resolution (idempotency)
    query := `
        INSERT INTO grades (id, course_id, assignment_id, student_id,
                           score, max_score, category)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (id) DO NOTHING  -- Safe reprocessing!
    `
    _, err := s.db.Exec(query, event.GradeID, event.CourseID,
                        event.AssignmentID, event.StudentID,
                        event.Score, event.MaxScore, event.Category)

    // Ensure student enrollment exists
    s.db.Exec(`
        INSERT INTO student_enrollments (course_id, student_id)
        VALUES ($1, $2)
        ON CONFLICT (course_id, student_id) DO NOTHING
    `, event.CourseID, event.StudentID)

    return err
}
```

**Failure Handling**:
- **Parsing Error**: Message ACK'd, logged as bad event (dead letter)
- **Database Error**: Message NACK'd with requeue flag (will retry)
- **Success**: Message ACK'd, data persisted

#### 2. StudentDeletedEvent

**Source**: Student Service (when student is deleted)
**Queue**: `students.deleted`
**Payload**:

```json
{
  "student_id": "student-001",
  "timestamp": "2026-02-22T11:00:00Z"
}
```

**Processing** (lines 101-135):

```go
func (s *server) consumeStudentDeletedEvents(ch *amqp.Channel) {
    msgs, _ := ch.Consume("students.deleted", "", false, false, false, false, nil)

    for msg := range msgs {
        var event StudentDeletedEvent
        json.Unmarshal(msg.Body, &event)

        log.Printf("Student deleted: %s", event.StudentID)

        if err := s.processStudentDeleted(event); err != nil {
            msg.Nack(false, true)
            continue
        }

        msg.Ack(false)
    }
}

func (s *server) processStudentDeleted(event StudentDeletedEvent) error {
    // Tombstone pattern: mark student as deleted
    query := `
        INSERT INTO deleted_students (student_id)
        VALUES ($1)
        ON CONFLICT (student_id) DO NOTHING
    `
    s.db.Exec(query, event.StudentID)

    // Remove from enrollments
    return s.db.Exec("DELETE FROM student_enrollments WHERE student_id = $1",
                     event.StudentID).Error
}
```

**Effect**: All analytics queries automatically exclude deleted students via:
```sql
WHERE student_id NOT IN (SELECT student_id FROM deleted_students)
```

---

## Important Code Snippets

### 1. Core Statistical Calculations

```go
// Mean (Average)
func calculateMean(values []float64) float64 {
    if len(values) == 0 { return 0 }
    sum := 0.0
    for _, v := range values { sum += v }
    return sum / float64(len(values))
}

// Median (50th Percentile)
func calculateMedian(values []float64) float64 {
    if len(values) == 0 { return 0 }
    sorted := make([]float64, len(values))
    copy(sorted, values)
    sort.Float64s(sorted)

    mid := len(sorted) / 2
    if len(sorted)%2 == 0 {
        return (sorted[mid-1] + sorted[mid]) / 2
    }
    return sorted[mid]
}

// Standard Deviation (Population)
func calculateStdDev(values []float64, mean float64) float64 {
    if len(values) == 0 { return 0 }
    sumSquaredDiff := 0.0
    for _, v := range values {
        diff := v - mean
        sumSquaredDiff += diff * diff
    }
    variance := sumSquaredDiff / float64(len(values))
    return math.Sqrt(variance)
}
```

### 2. Performance Distribution (Bell Curve)

```go
// Get normalized scores
var scores []float64
rows, _ := s.db.QueryContext(ctx, `
    SELECT score, max_score
    FROM grades
    WHERE course_id = $1
      AND student_id NOT IN (SELECT student_id FROM deleted_students)
`, req.CourseId)

for rows.Next() {
    var score, maxScore int32
    rows.Scan(&score, &maxScore)
    percentage := (float64(score) / float64(maxScore)) * 100
    scores = append(scores, percentage)
}

// Calculate statistics
mean := calculateMean(scores)
median := calculateMedian(scores)
stdDev := calculateStdDev(scores, mean)

// Create histogram buckets
bucketCounts := make([]int32, 10)
for _, score := range scores {
    bucketIndex := int(score) / 10
    if bucketIndex > 9 { bucketIndex = 9 }
    bucketCounts[bucketIndex]++
}

// Convert to response format
totalStudents := int32(len(scores))
for i := 0; i < 10; i++ {
    buckets[i] = &statspb.ScoreBucket{
        Range:      fmt.Sprintf("%d-%d", i*10, i*10+10),
        Count:      bucketCounts[i],
        Percentage: (float64(bucketCounts[i]) / float64(totalStudents)) * 100,
    }
}
```

### 3. Z-Score Risk Detection

```go
// Calculate class statistics
classMean := calculateMean(allAverages)
classStdDev := calculateStdDev(allAverages, classMean)

// For each student, calculate Z-score
for _, sd := range students {
    var riskFactors []string

    // Z-score = (value - mean) / stddev
    deviationFromMean := (classMean - sd.avgPercentage) / classStdDev

    // Threshold: > 2σ below mean (top 2.5% with lowest scores)
    if deviationFromMean > 2.0 {
        riskFactors = append(riskFactors, "Low Performance")
    }

    // Also check for missing assignments
    missingAssignments := totalAssignments - sd.completedAssignments
    if missingAssignments >= 3 {
        riskFactors = append(riskFactors, "Missing Assignments")
    }

    if len(riskFactors) > 0 {
        atRiskStudents = append(atRiskStudents, &statspb.AtRiskStudent{
            StudentId:          sd.studentID,
            CurrentAverage:     sd.avgPercentage,
            ClassMean:          classMean,
            DeviationFromMean:  deviationFromMean,
            RiskFactors:        riskFactors,
        })
    }
}
```

### 4. Category Mastery SQL Aggregation

```sql
-- Group by assignment type (category)
SELECT
    category,                           -- Group dimension
    AVG(percentage) as avg_percentage,  -- Central tendency
    STDDEV(percentage) as std_dev,      -- Spread
    COUNT(DISTINCT assignment_id) as total_assignments,  -- Count
    COUNT(*) as total_submissions       -- Detail submissions
FROM grades
WHERE course_id = $1
  AND student_id NOT IN (SELECT student_id FROM deleted_students)
  AND category IS NOT NULL
GROUP BY category
ORDER BY avg_percentage DESC  -- Strongest first
```

### 5. Idempotent Event Processing

```go
// Event 1 arrives: INSERT with unique ID
INSERT INTO grades (id, course_id, student_id, score, ...)
VALUES ($1, $2, $3, $4, ...)
ON CONFLICT (id) DO NOTHING;  -- Unique constraint on id column

// Event 1 arrives AGAIN (duplicate):
// Conflict silently ignored - same result!
INSERT INTO grades (id, course_id, student_id, score, ...)
VALUES ($1, $2, $3, $4, ...)
ON CONFLICT (id) DO NOTHING;  -- No error, no duplicate insert
```

---

## Real-World Applications

### 1. Instructor Dashboard

**Use Case**: Teacher views course performance at a glance

```
Course: Advanced Algorithms
Semester: Spring 2026

┌─────────────────────────────────────────────┐
│ QUICK STATS                                 │
├─────────────────────────────────────────────┤
│ Enrollment: 47 students                     │
│ Average Score: 72.3% (±12.8%)              │
│ At-Risk Students: 6 (13%)                  │
│ Strongest Area: Labs (84%)                 │
│ Weakest Area: Midterm Exam (61%)           │
└─────────────────────────────────────────────┘

┌─────────────────────────────────────────────┐
│ SCORE DISTRIBUTION                          │
├─────────────────────────────────────────────┤
│ 0-10:   ░░░░░                 (5 students)  │
│ 11-20:  ░░░░░░░░               (8 students)  │
│ 21-30:  ░░░░░░░░░░░░░░░        (14 students) │  ← Peak (Mode)
│ 31-40:  ░░░░░░░░░░            (10 students)  │
│ 41-50:  ░░░░░░                 (7 students)  │
│ 51-60:  ░░░░                   (4 students)  │
│ 61-70:  (0 students)                        │
│ 71-80:  (0 students)                        │
│ 81-90:  (0 students)                        │
│ 91-100: (0 students)                        │
├─────────────────────────────────────────────┤
│ Interpretation: RIGHT-SKEWED distribution   │
│ Suggests: Course too difficult or students  │
│           underprepared. Consider review.   │
└─────────────────────────────────────────────┘

┌──────────────────────────────────────────────────┐
│ AT-RISK STUDENTS                                 │
├──────────────────────────────────────────────────┤
│ 1. Bob Smith (STD-2026-015)                      │
│    • Average: 45%  (2.1σ below mean)            │
│    • Missing 4/6 assignments                    │
│    • Risk Factors: Low Performance, Missing     │
│    → Recommend: One-on-one tutoring             │
│                                                  │
│ 2. Carol Jones (STD-2026-018)                    │
│    • Average: 52%  (1.6σ below mean)            │
│    • Missing 0/6 assignments                    │
│    • Risk Factors: Low Performance              │
│    → Recommend: Study group + office hours      │
└──────────────────────────────────────────────────┘
```

**Action Items Generated**:
- Schedule tutoring for at-risk students
- Review midterm exam difficulty
- Highlight lab success as model for other assessments

### 2. Early Warning System

**Use Case**: Automated alert when students show warning signs

```python
# Pseudo-code: Early Warning Logic
students = get_at_risk_students(course_id)
for student in students:
    if len(student.risk_factors) > 1:  # Both academic + behavioral
        send_alert(teacher, f"HIGH RISK: {student.name}")
        send_notification(student, "Please meet with instructor this week")
    elif student.deviation_from_mean > 2.5:  # More than 2.5σ below
        send_alert(teacher, f"URGENT: {student.name} needs intervention")
    else:
        send_email(teacher, f"Monitor: {student.name} showing signs")
```

### 3. Curriculum Improvement

**Use Case**: Identify which topics/skills need reteaching

```
Category Analysis:
┌────────────┬──────┬────────┬──────────────────────┐
│ Category   │ Avg% │ StdDev │ Interpretation       │
├────────────┼──────┼────────┼──────────────────────┤
│ Labs       │ 84%  │ 4%     │ ✅ Success - Keep    │
│ Exams      │ 68%  │ 16%    │ ❌ Problem - High    │
│            │      │        │    variance, low avg │
│ Quizzes    │ 76%  │ 8%     │ ⚠️  Mixed - Moderate │
│ Projects   │ 71%  │ 12%    │ ⚠️  Needs review     │
└────────────┴──────┴────────┴──────────────────────┘

High StdDev in Exams = Inconsistent understanding
→ Action: Add exam review sessions, provide worked examples

Labs succeed with low variance
→ Action: Use lab format for other topics (projects, quizzes)

Projects have moderate performance + variance
→ Action: Provide clearer rubric, break into milestones
```

### 4. Grade Distribution Analysis

**Use Case**: Detect grading issues or course misalignment

```
Distribution Patterns Indicate:

✅ NORMAL DISTRIBUTION (Bell curve around 70-80%)
   Interpretation: Well-designed course, aligned to learning objectives
   Action: Continue current approach

⚠️ LEFT-SKEWED (Cluster at top, tail on left)
   Interpretation: Course too easy OR students over-prepared
   Action: Increase difficulty, add advanced content

⚠️ RIGHT-SKEWED (Cluster at bottom, tail on right)
   Interpretation: Course too hard OR students under-prepared
   Action: Add prerequisite review, slow pacing, more scaffolding

❌ BIMODAL (Two distinct peaks)
   Interpretation: Two student populations (e.g., majors vs non-majors)
   Action: Consider differentiated instruction or separate sections
```

---

## Usage Examples

### Example 1: Getting Performance Distribution

**Request** (gRPC):
```go
resp, err := statsClient.GetPerformanceDistribution(ctx,
    &statspb.PerformanceDistributionRequest{
        CourseId: "course-algorithms",
        AssignmentId: "",  // All assignments
    })

if err != nil {
    log.Fatalf("RPC error: %v", err)
}

// Use response
fmt.Printf("Mean Score: %.1f%%\n", resp.Mean)           // 72.3%
fmt.Printf("Median Score: %.1f%%\n", resp.Median)       // 71.0%
fmt.Printf("Std Dev: %.1f%%\n", resp.StdDeviation)      // 12.8%

for _, bucket := range resp.Buckets {
    fmt.Printf("%s: %d students (%.1f%%)\n",
        bucket.Range, bucket.Count, bucket.Percentage)
}
```

**Response**:
```
Mean Score: 72.3%
Median Score: 71.0%
Std Dev: 12.8%
0-10: 5 students (10.6%)
11-20: 8 students (17.0%)
21-30: 14 students (29.8%)
31-40: 10 students (21.3%)
41-50: 7 students (14.9%)
51-60: 3 students (6.4%)
61-70: 0 students (0.0%)
71-80: 0 students (0.0%)
81-90: 0 students (0.0%)
91-100: 0 students (0.0%)
```

---

### Example 2: Identifying At-Risk Students

**Request** (gRPC):
```go
resp, err := statsClient.GetAtRiskStudents(ctx,
    &statspb.AtRiskStudentsRequest{
        CourseId: "course-algorithms",
        MissingAssignmentThreshold: 3,
        StdDeviationThreshold: 2.0,
    })

if err != nil {
    log.Fatalf("RPC error: %v", err)
}

fmt.Printf("At-Risk Count: %d / %d students (%.1f%%)\n",
    resp.AtRiskCount, resp.TotalStudents,
    float64(resp.AtRiskCount)/float64(resp.TotalStudents)*100)

for i, student := range resp.AtRiskStudents {
    fmt.Printf("\n#%d. %s (ID: %s)\n", i+1, student.StudentName, student.StudentId)
    fmt.Printf("  Average: %.1f%% (class mean: %.1f%%)\n",
        student.CurrentAverage, student.ClassMean)
    fmt.Printf("  Deviation: %.2fσ below mean\n", student.DeviationFromMean)
    fmt.Printf("  Missing: %d/%d assignments\n",
        student.MissingAssignments, student.TotalAssignments)
    fmt.Printf("  Risk Factors: %v\n", student.RiskFactors)
}
```

**Response**:
```
At-Risk Count: 6 / 47 students (12.8%)

#1. Bob Smith (ID: student-015)
  Average: 45.2% (class mean: 72.3%)
  Deviation: 2.08σ below mean
  Missing: 4/6 assignments
  Risk Factors: [Low Performance Missing Assignments]

#2. Carol Jones (ID: student-018)
  Average: 51.3% (class mean: 72.3%)
  Deviation: 1.62σ below mean
  Missing: 0/6 assignments
  Risk Factors: [Low Performance]

#3. Diana Chen (ID: student-022)
  Average: 55.8% (class mean: 72.3%)
  Deviation: 1.31σ below mean
  Missing: 3/6 assignments
  Risk Factors: [Missing Assignments]
...
```

---

### Example 3: Category Mastery Analysis

**Request** (gRPC):
```go
resp, err := statsClient.GetCategoryMastery(ctx,
    &statspb.CategoryMasteryRequest{
        CourseId: "course-algorithms",
    })

fmt.Printf("Strongest Category: %s\n", resp.StrongestCategory)
fmt.Printf("Weakest Category: %s\n\n", resp.WeakestCategory)

for _, cat := range resp.Categories {
    fmt.Printf("%-10s: %.1f%% avg, σ=%.1f%% (%d assignments, %d submissions)\n",
        cat.Category,
        cat.AveragePercentage,
        cat.StdDeviation,
        cat.TotalAssignments,
        cat.TotalSubmissions)
}
```

**Response**:
```
Strongest Category: Labs
Weakest Category: Exams

Labs      : 84.2% avg, σ=4.1% (4 assignments, 47 submissions)
Quizzes   : 76.8% avg, σ=8.3% (5 assignments, 45 submissions)
Projects  : 71.5% avg, σ=12.1% (2 assignments, 40 submissions)
Exams     : 65.3% avg, σ=16.8% (2 assignments, 38 submissions)
```

---

### Example 4: Course Overview

**Request** (gRPC):
```go
resp, err := statsClient.GetCourseStats(ctx,
    &statspb.CourseStatsRequest{
        CourseId: "course-algorithms",
    })

fmt.Printf("COURSE STATISTICS: %s\n", resp.CourseId)
fmt.Printf("Enrolled: %d students\n", resp.TotalStudents)
fmt.Printf("Assignments: %d total\n", resp.TotalAssignments)
fmt.Printf("Grades Recorded: %d\n", resp.TotalGradesRecorded)
fmt.Printf("\n")
fmt.Printf("Overall Average: %.1f%% (±%.1f%%)\n",
    resp.OverallAverage, resp.OverallStdDeviation)
fmt.Printf("At-Risk Students: %d (%.1f%%)\n",
    resp.AtRiskCount,
    float64(resp.AtRiskCount)/float64(resp.TotalStudents)*100)
fmt.Printf("\n")
fmt.Printf("Top Category: %s\n", resp.HighestPerformingCategory)
fmt.Printf("Bottom Category: %s\n", resp.LowestPerformingCategory)
```

**Response**:
```
COURSE STATISTICS: course-algorithms
Enrolled: 47 students
Assignments: 13 total
Grades Recorded: 189

Overall Average: 72.3% (±12.8%)
At-Risk Students: 6 (12.8%)

Top Category: Labs
Bottom Category: Exams
```

---

## Summary Table: Statistical Metrics

| Metric | Formula | Range | Interpretation | Use Case |
|--------|---------|-------|-----------------|----------|
| **Mean** | Σx/n | 0-100% | Average performance | Course benchmark |
| **Median** | Middle value | 0-100% | Typical score (robust) | Skewed distributions |
| **StdDev** | √[Σ(x-μ)²/n] | 0-50% | Performance spread | Homogeneity indicator |
| **Z-Score** | (x-μ)/σ | -∞ to +∞ | Std devs from mean | At-risk detection |
| **Percentile** | Rank % | 0-100% | Position in distribution | Comparative ranking |
| **Skewness** | Σ(x-μ)³/(n·σ³) | -1 to +1 | Distribution shape | Course difficulty |
| **IQR** | Q3-Q1 | 0-100% | Middle 50% range | Outlier detection |

---

## References

- **Descriptive Statistics**: Used for summarizing and describing data patterns
- **Z-Scores**: Part of **Standardization** techniques from **Probability Theory**
- **Histogram/Distribution**: From **Frequency Analysis** and **Empirical CDF**
- **Grouped Statistics**: **Stratified Analysis** technique
- **At-Risk Detection**: Multi-factor **Decision Analysis** approach
- **Idempotency**: **Functional Programming** and **Distributed Systems** pattern

---

## Conclusion

The Stats Service provides educators with powerful, statistically-grounded insights into student performance. By combining descriptive statistics with domain knowledge (assignment categories, completion rates), it enables:

✅ **Objective Assessment**: Data-driven rather than subjective evaluation
✅ **Early Intervention**: Identify struggling students before failure
✅ **Curriculum Improvement**: Identify which topics need reteaching
✅ **Fair Grading**: Statistical context for grade distribution decisions
✅ **Skill Gap Analysis**: Category mastery shows which skills need focus

All calculations are transparent, repeatable, and mathematically rigorous for use in official reports.

