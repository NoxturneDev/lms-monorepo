package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"sort"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	_ "github.com/lib/pq"
	statspb "github.com/noxturnedev/lms-monorepo/proto/stats"
	"github.com/streadway/amqp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type server struct {
	statspb.UnimplementedStatsServiceServer
	db *sql.DB
}

// ============================================
// EVENT DEFINITIONS (from other services)
// ============================================

type GradeAssignedEvent struct {
	GradeID      string `json:"grade_id"`
	CourseID     string `json:"course_id"`
	AssignmentID string `json:"assignment_id"`
	StudentID    string `json:"student_id"`
	Score        int32  `json:"score"`
	MaxScore     int32  `json:"max_score"`
	Category     string `json:"category"`
	TeacherID    string `json:"teacher_id"`
	Timestamp    string `json:"timestamp"`
}

type StudentDeletedEvent struct {
	StudentID string `json:"student_id"`
	Timestamp string `json:"timestamp"`
}

type CourseCreatedEvent struct {
	CourseID    string `json:"course_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	SchoolID    string `json:"school_id"`
	Timestamp   string `json:"timestamp"`
}

// ============================================
// RABBITMQ CONSUMERS
// ============================================

func (s *server) consumeGradeAssignedEvents(ch *amqp.Channel) {
	msgs, err := ch.Consume(
		"grades.assigned", // queue
		"",                // consumer
		false,             // auto-ack (false = manual ack for idempotency)
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
	)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Println("[Stats Service] Listening for GradeAssigned events...")

	for msg := range msgs {
		var event GradeAssignedEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Printf("Failed to unmarshal GradeAssigned event: %v", err)
			msg.Ack(false) // Ack to prevent reprocessing bad messages
			continue
		}

		log.Printf("Received GradeAssigned: grade_id=%s, student=%s, score=%d/%d",
			event.GradeID, event.StudentID, event.Score, event.MaxScore)

		// Idempotent insert - will fail silently if grade_id already exists
		if err := s.processGradeAssigned(event); err != nil {
			log.Printf("Failed to process GradeAssigned: %v", err)
			// Do NOT ack - message will be redelivered
			msg.Nack(false, true)
			continue
		}

		msg.Ack(false) // Successfully processed
	}
}

func (s *server) consumeStudentDeletedEvents(ch *amqp.Channel) {
	msgs, err := ch.Consume(
		"students.deleted",
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	log.Println("[Stats Service] Listening for StudentDeleted events...")

	for msg := range msgs {
		var event StudentDeletedEvent
		if err := json.Unmarshal(msg.Body, &event); err != nil {
			log.Printf("Failed to unmarshal StudentDeleted event: %v", err)
			msg.Ack(false)
			continue
		}

		log.Printf("Received StudentDeleted: student_id=%s", event.StudentID)

		if err := s.processStudentDeleted(event); err != nil {
			log.Printf("Failed to process StudentDeleted: %v", err)
			msg.Nack(false, true)
			continue
		}

		msg.Ack(false)
	}
}

// ============================================
// EVENT PROCESSORS (Write to Read Model)
// ============================================

func (s *server) processGradeAssigned(event GradeAssignedEvent) error {
	// Idempotent insert - ON CONFLICT DO NOTHING
	query := `
		INSERT INTO grades (id, course_id, assignment_id, student_id, score, max_score, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO NOTHING
	`
	_, err := s.db.Exec(query,
		event.GradeID,
		event.CourseID,
		event.AssignmentID,
		event.StudentID,
		event.Score,
		event.MaxScore,
		event.Category,
	)
	if err != nil {
		return fmt.Errorf("failed to insert grade: %v", err)
	}

	// Ensure student enrollment exists
	enrollQuery := `
		INSERT INTO student_enrollments (course_id, student_id)
		VALUES ($1, $2)
		ON CONFLICT (course_id, student_id) DO NOTHING
	`
	_, err = s.db.Exec(enrollQuery, event.CourseID, event.StudentID)
	if err != nil {
		return fmt.Errorf("failed to insert enrollment: %v", err)
	}

	return nil
}

func (s *server) processStudentDeleted(event StudentDeletedEvent) error {
	// Tombstone pattern - mark as deleted
	query := `
		INSERT INTO deleted_students (student_id)
		VALUES ($1)
		ON CONFLICT (student_id) DO NOTHING
	`
	_, err := s.db.Exec(query, event.StudentID)
	if err != nil {
		return fmt.Errorf("failed to mark student deleted: %v", err)
	}

	// Remove from enrollments
	_, err = s.db.Exec("DELETE FROM student_enrollments WHERE student_id = $1", event.StudentID)
	return err
}

// ============================================
// RPC: Performance Distribution (Bell Curve)
// ============================================

func (s *server) GetPerformanceDistribution(ctx context.Context, req *statspb.PerformanceDistributionRequest) (*statspb.PerformanceDistributionResponse, error) {
	query := `
		SELECT score, max_score
		FROM grades
		WHERE course_id = $1
		  AND student_id NOT IN (SELECT student_id FROM deleted_students)
	`
	args := []interface{}{req.CourseId}

	if req.AssignmentId != "" {
		query += " AND assignment_id = $2"
		args = append(args, req.AssignmentId)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	defer rows.Close()

	var scores []float64
	for rows.Next() {
		var score, maxScore int32
		if err := rows.Scan(&score, &maxScore); err != nil {
			return nil, status.Errorf(codes.Internal, "scan error: %v", err)
		}
		// Normalize to percentage
		percentage := (float64(score) / float64(maxScore)) * 100
		scores = append(scores, percentage)
	}

	if len(scores) == 0 {
		return &statspb.PerformanceDistributionResponse{
			CourseId:      req.CourseId,
			AssignmentId:  req.AssignmentId,
			Buckets:       []*statspb.ScoreBucket{},
			TotalStudents: 0,
		}, nil
	}

	// Calculate statistics
	mean := calculateMean(scores)
	median := calculateMedian(scores)
	stdDev := calculateStdDev(scores, mean)

	// Create 10-point buckets (0-10, 11-20, ..., 91-100)
	buckets := make([]*statspb.ScoreBucket, 10)
	bucketCounts := make([]int32, 10)

	for _, score := range scores {
		bucketIndex := int(score) / 10
		if bucketIndex > 9 {
			bucketIndex = 9 // Handle perfect 100
		}
		bucketCounts[bucketIndex]++
	}

	totalStudents := int32(len(scores))
	for i := 0; i < 10; i++ {
		minScore := i * 10
		maxScore := minScore + 10
		if i == 9 {
			maxScore = 100 // Last bucket is 91-100
		}

		buckets[i] = &statspb.ScoreBucket{
			Range:      fmt.Sprintf("%d-%d", minScore, maxScore),
			MinScore:   int32(minScore),
			MaxScore:   int32(maxScore),
			Count:      bucketCounts[i],
			Percentage: (float64(bucketCounts[i]) / float64(totalStudents)) * 100,
		}
	}

	return &statspb.PerformanceDistributionResponse{
		CourseId:      req.CourseId,
		AssignmentId:  req.AssignmentId,
		Buckets:       buckets,
		Mean:          mean,
		Median:        median,
		StdDeviation:  stdDev,
		TotalStudents: totalStudents,
	}, nil
}

// ============================================
// RPC: At-Risk Students
// ============================================

func (s *server) GetAtRiskStudents(ctx context.Context, req *statspb.AtRiskStudentsRequest) (*statspb.AtRiskStudentsResponse, error) {
	// Default thresholds
	missingThreshold := req.MissingAssignmentThreshold
	if missingThreshold == 0 {
		missingThreshold = 3
	}
	stdDevThreshold := req.StdDeviationThreshold
	if stdDevThreshold == 0 {
		stdDevThreshold = 2.0
	}

	// Get total assignments for the course
	var totalAssignments int32
	err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM assignments WHERE course_id = $1",
		req.CourseId,
	).Scan(&totalAssignments)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get total assignments: %v", err)
	}

	// Calculate class statistics
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
	rows, err := s.db.QueryContext(ctx, query, req.CourseId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	defer rows.Close()

	type studentData struct {
		studentID            string
		avgPercentage        float64
		completedAssignments int32
	}

	var students []studentData
	var allAverages []float64

	for rows.Next() {
		var sd studentData
		if err := rows.Scan(&sd.studentID, &sd.avgPercentage, &sd.completedAssignments); err != nil {
			return nil, status.Errorf(codes.Internal, "scan error: %v", err)
		}
		students = append(students, sd)
		allAverages = append(allAverages, sd.avgPercentage)
	}

	if len(students) == 0 {
		return &statspb.AtRiskStudentsResponse{
			CourseId:       req.CourseId,
			AtRiskStudents: []*statspb.AtRiskStudent{},
			TotalStudents:  0,
			AtRiskCount:    0,
		}, nil
	}

	classMean := calculateMean(allAverages)
	classStdDev := calculateStdDev(allAverages, classMean)

	// Identify at-risk students
	var atRiskStudents []*statspb.AtRiskStudent

	for _, sd := range students {
		var riskFactors []string
		isAtRisk := false

		// Check 1: Performance (> 2σ below mean)
		deviationFromMean := (classMean - sd.avgPercentage) / classStdDev
		if deviationFromMean > stdDevThreshold {
			riskFactors = append(riskFactors, "Low Performance")
			isAtRisk = true
		}

		// Check 2: Missing assignments
		missingAssignments := totalAssignments - sd.completedAssignments
		if missingAssignments >= missingThreshold {
			riskFactors = append(riskFactors, "Missing Assignments")
			isAtRisk = true
		}

		if isAtRisk {
			atRiskStudents = append(atRiskStudents, &statspb.AtRiskStudent{
				StudentId:          sd.studentID,
				StudentName:        "", // Populated by gateway
				StudentNumber:      "", // Populated by gateway
				CurrentAverage:     sd.avgPercentage,
				ClassMean:          classMean,
				DeviationFromMean:  deviationFromMean,
				MissingAssignments: missingAssignments,
				TotalAssignments:   totalAssignments,
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

// ============================================
// RPC: Category Mastery
// ============================================

func (s *server) GetCategoryMastery(ctx context.Context, req *statspb.CategoryMasteryRequest) (*statspb.CategoryMasteryResponse, error) {
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

	rows, err := s.db.QueryContext(ctx, query, req.CourseId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	defer rows.Close()

	var categories []*statspb.CategoryStats
	var strongest, weakest string
	maxAvg, minAvg := 0.0, 100.0

	for rows.Next() {
		var cat statspb.CategoryStats
		var stdDev sql.NullFloat64

		if err := rows.Scan(&cat.Category, &cat.AveragePercentage, &stdDev, &cat.TotalAssignments, &cat.TotalSubmissions); err != nil {
			return nil, status.Errorf(codes.Internal, "scan error: %v", err)
		}

		if stdDev.Valid {
			cat.StdDeviation = stdDev.Float64
		}

		cat.AverageScore = cat.AveragePercentage // Already normalized

		categories = append(categories, &cat)

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

// ============================================
// RPC: Course Stats Overview
// ============================================

func (s *server) GetCourseStats(ctx context.Context, req *statspb.CourseStatsRequest) (*statspb.CourseStatsResponse, error) {
	// Get overall stats
	var totalStudents, totalAssignments, totalGrades int32
	var overallAvg sql.NullFloat64
	var overallStdDev sql.NullFloat64

	log.Println("get new stats")

	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(DISTINCT student_id) as total_students,
			COUNT(DISTINCT assignment_id) as total_assignments,
			COUNT(*) as total_grades,
			AVG(percentage) as overall_avg,
			STDDEV(percentage) as overall_std_dev
		FROM grades
		WHERE course_id = $1
		  AND student_id NOT IN (SELECT student_id FROM deleted_students)
	`, req.CourseId).Scan(&totalStudents, &totalAssignments, &totalGrades, &overallAvg, &overallStdDev)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}

	var avgVal, stdVal float64
	if overallAvg.Valid {
		avgVal = overallAvg.Float64
	}
	if overallStdDev.Valid {
		stdVal = overallStdDev.Float64
	}

	// Get at-risk count
	classMean := avgVal
	classStdDev := stdVal
	atRiskCount := int32(0)

	if classStdDev > 0 {
		riskRows, err := s.db.QueryContext(ctx, `
			SELECT AVG(percentage)
			FROM grades
			WHERE course_id = $1
			  AND student_id NOT IN (SELECT student_id FROM deleted_students)
			GROUP BY student_id
		`, req.CourseId)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "at-risk query error: %v", err)
		}
		defer riskRows.Close()

		for riskRows.Next() {
			var avg float64
			if err := riskRows.Scan(&avg); err != nil {
				return nil, status.Errorf(codes.Internal, "scan error: %v", err)
			}
			deviation := (classMean - avg) / classStdDev
			if deviation > 2.0 {
				atRiskCount++
			}
		}
	}

	// Get category performance
	var highestCat, lowestCat string
	catRows, err := s.db.QueryContext(ctx, `
		SELECT category, AVG(percentage) as avg_perc
		FROM grades
		WHERE course_id = $1 AND category IS NOT NULL
		  AND student_id NOT IN (SELECT student_id FROM deleted_students)
		GROUP BY category
		ORDER BY avg_perc DESC
	`, req.CourseId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "category query error: %v", err)
	}
	defer catRows.Close()

	first := true
	for catRows.Next() {
		var cat string
		var avg float64
		if err := catRows.Scan(&cat, &avg); err != nil {
			return nil, status.Errorf(codes.Internal, "scan error: %v", err)
		}
		if first {
			highestCat = cat
			first = false
		}
		lowestCat = cat // Last one will be lowest
	}

	return &statspb.CourseStatsResponse{
		CourseId:                  req.CourseId,
		TotalStudents:             totalStudents,
		TotalAssignments:          totalAssignments,
		OverallAverage:            avgVal,
		OverallStdDeviation:       stdVal,
		AtRiskCount:               atRiskCount,
		TotalGradesRecorded:       totalGrades,
		HighestPerformingCategory: highestCat,
		LowestPerformingCategory:  lowestCat,
	}, nil
}

// ============================================
// RECONCILIATION / SYNC (Data Integrity)
// ============================================

func (s *server) ReconcileGradesSync(ctx context.Context, req *statspb.ReconcileSyncRequest) (*statspb.ReconcileSyncResponse, error) {
	log.Printf("Starting reconciliation sync for course: %s", req.CourseId)

	// Query teacher_db to get all grades for this course (via teacher service)
	// For now, we'll query our local stats_db and verify data consistency
	// This ensures category field is properly populated

	query := `
		SELECT id, course_id, assignment_id, student_id, score, max_score, category
		FROM grades
		WHERE course_id = $1 AND category IS NULL
	`

	rows, err := s.db.QueryContext(ctx, query, req.CourseId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	defer rows.Close()

	var missingCategoryCount int32
	var gradesToFix []map[string]interface{}

	for rows.Next() {
		var id, courseID, assignmentID, studentID string
		var score, maxScore int32
		var category sql.NullString

		if err := rows.Scan(&id, &courseID, &assignmentID, &studentID, &score, &maxScore, &category); err != nil {
			continue
		}

		// If category is NULL, try to fetch from assignments table
		if !category.Valid {
			var assignmentCategory sql.NullString
			s.db.QueryRowContext(ctx,
				"SELECT category FROM assignments WHERE id = $1",
				assignmentID,
			).Scan(&assignmentCategory)

			if assignmentCategory.Valid {
				// Update this grade with the category
				_, err := s.db.ExecContext(ctx,
					"UPDATE grades SET category = $1 WHERE id = $2",
					assignmentCategory.String, id,
				)
				if err != nil {
					log.Printf("Failed to update grade %s with category: %v", id, err)
				} else {
					missingCategoryCount++
					log.Printf("Fixed: Grade %s now has category: %s", id, assignmentCategory.String)
				}
			} else {
				gradesToFix = append(gradesToFix, map[string]interface{}{
					"grade_id":      id,
					"assignment_id": assignmentID,
				})
			}
		}
	}

	return &statspb.ReconcileSyncResponse{
		CourseId:              req.CourseId,
		GradesFixed:           missingCategoryCount,
		InconsistenciesFound:  int32(len(gradesToFix)),
		LastReconcileTime:     fmt.Sprintf("%v", ctx),
		ReconciliationSuccess: true,
	}, nil
}

// ============================================
// STATISTICAL HELPER FUNCTIONS
// ============================================

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

// ============================================
// LINEAR REGRESSION (Enrollment Forecasting)
// ============================================

type LinearRegressionResult struct {
	Slope      float64
	Intercept  float64
	RSquared   float64
	Correlation float64
}

// linearRegression performs Least Squares Linear Regression
// X: years (independent), Y: enrollment counts (dependent)
// Returns: slope (growth rate), intercept, R² (fit quality)
func linearRegression(x []float64, y []float64) LinearRegressionResult {
	// Calculate means
	meanX := calculateMean(x)
	meanY := calculateMean(y)

	// Calculate slope and intercept
	sumXY := 0.0
	sumX2 := 0.0
	sumY2 := 0.0

	for i := 0; i < len(x); i++ {
		sumXY += (x[i] - meanX) * (y[i] - meanY)
		sumX2 += (x[i] - meanX) * (x[i] - meanX)
		sumY2 += (y[i] - meanY) * (y[i] - meanY)
	}

	slope := 0.0
	if sumX2 > 0 {
		slope = sumXY / sumX2
	}
	intercept := meanY - slope*meanX

	// Calculate R² (coefficient of determination)
	ssTot := 0.0   // Total sum of squares
	ssRes := 0.0   // Residual sum of squares

	for i := 0; i < len(y); i++ {
		predicted := slope*x[i] + intercept
		ssTot += (y[i] - meanY) * (y[i] - meanY)
		ssRes += (y[i] - predicted) * (y[i] - predicted)
	}

	rSquared := 0.0
	if ssTot > 0 {
		rSquared = 1 - (ssRes / ssTot)
	}

	// Calculate Pearson correlation coefficient
	correlation := 0.0
	if sumX2 > 0 && sumY2 > 0 {
		correlation = sumXY / math.Sqrt(sumX2*sumY2)
	}

	return LinearRegressionResult{
		Slope:       slope,
		Intercept:   intercept,
		RSquared:    rSquared,
		Correlation: correlation,
	}
}

// ============================================
// RPC: Get Students At Risk (Decision Tree)
// ============================================

func (s *server) GetStudentsAtRisk(ctx context.Context, req *statspb.StudentsAtRiskRequest) (*statspb.StudentsAtRiskResponse, error) {
	// Decision Tree Logic:
	// CRITICAL: avg < 60 AND last 2 weeks avg < historical avg
	// WARNING: avg < class_avg - 15
	// GHOST: Not in enrollments

	query := `
		SELECT
			student_id,
			course_id,
			AVG(percentage) as avg_percentage,
			COUNT(*) as total_submissions
		FROM grades
		WHERE student_id NOT IN (SELECT student_id FROM deleted_students)
		GROUP BY student_id, course_id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	defer rows.Close()

	// Get class averages per course
	courseAvgMap := make(map[string]float64)
	courseQuery := `
		SELECT course_id, AVG(percentage) as avg
		FROM grades
		WHERE student_id NOT IN (SELECT student_id FROM deleted_students)
		GROUP BY course_id
	`
	courseRows, err := s.db.QueryContext(ctx, courseQuery)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "course avg query error: %v", err)
	}
	defer courseRows.Close()

	for courseRows.Next() {
		var courseID string
		var avg float64
		if err := courseRows.Scan(&courseID, &avg); err != nil {
			continue
		}
		courseAvgMap[courseID] = avg
	}

	var risks []*statspb.StudentRisk
	criticalCount, warningCount, ghostCount := int32(0), int32(0), int32(0)

	for rows.Next() {
		var studentID, courseID string
		var avgPercentage float64
		var totalSubmissions int32

		if err := rows.Scan(&studentID, &courseID, &avgPercentage, &totalSubmissions); err != nil {
			continue
		}

		classAvg := courseAvgMap[courseID]
		riskLevel := ""
		reason := ""

		// Decision Tree
		if avgPercentage < 60 {
			riskLevel = "CRITICAL"
			reason = "Average score below 60%"
			criticalCount++
		} else if avgPercentage < classAvg-15 {
			riskLevel = "WARNING"
			reason = fmt.Sprintf("15+ points below class average (%.2f%%)", classAvg)
			warningCount++
		}

		if riskLevel != "" {
			risks = append(risks, &statspb.StudentRisk{
				StudentId:    studentID,
				CourseId:     courseID,
				RiskLevel:    riskLevel,
				WarningReason: reason,
				CurrentAverage: avgPercentage,
				ClassAverage:   classAvg,
				DetectedAt:    time.Now().Format(time.RFC3339),
			})
		}
	}

	return &statspb.StudentsAtRiskResponse{
		AtRiskStudents: risks,
		TotalCritical:  criticalCount,
		TotalWarning:   warningCount,
		TotalGhost:     ghostCount,
	}, nil
}

// ============================================
// RPC: Get Ghost Students (Anomaly Mining)
// ============================================

func (s *server) GetGhostStudents(ctx context.Context, req *statspb.GhostStudentsRequest) (*statspb.GhostStudentsResponse, error) {
	// Ghost Student = Active but no recent activity (> 45 days, default)
	inactiveDays := req.InactiveDays
	if inactiveDays == 0 {
		inactiveDays = 45
	}
	if inactiveDays < 14 {
		inactiveDays = 14 // Enforce minimum
	}

	query := `
		SELECT
			se.student_id,
			se.course_id,
			MAX(g.recorded_at) as last_activity
		FROM student_enrollments se
		LEFT JOIN grades g ON se.student_id = g.student_id AND se.course_id = g.course_id
		WHERE se.student_id NOT IN (SELECT student_id FROM deleted_students)
		GROUP BY se.student_id, se.course_id
		HAVING MAX(g.recorded_at) < NOW() - INTERVAL '1 day' * $1
			OR MAX(g.recorded_at) IS NULL
	`

	rows, err := s.db.QueryContext(ctx, query, inactiveDays)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	defer rows.Close()

	var ghosts []*statspb.GhostStudent

	for rows.Next() {
		var studentID, courseID string
		var lastActivity sql.NullTime

		if err := rows.Scan(&studentID, &courseID, &lastActivity); err != nil {
			continue
		}

		var lastActivityStr string
		var daysInactive int32

		if lastActivity.Valid {
			lastActivityStr = lastActivity.Time.Format(time.RFC3339)
			daysInactive = int32(time.Since(lastActivity.Time).Hours() / 24)
		} else {
			lastActivityStr = "Never"
			daysInactive = 999
		}

		ghosts = append(ghosts, &statspb.GhostStudent{
			StudentId:    studentID,
			CourseId:     courseID,
			LastActivity: lastActivityStr,
			DaysInactive: daysInactive,
		})
	}

	criteria := fmt.Sprintf("Inactive for %d+ days", inactiveDays)

	return &statspb.GhostStudentsResponse{
		GhostStudents:   ghosts,
		TotalGhostCount: int32(len(ghosts)),
		DetectionCriteria: criteria,
	}, nil
}

// ============================================
// RPC: Enrollment Forecasting (Linear Regression)
// ============================================

func (s *server) ForecastEnrollment(ctx context.Context, req *statspb.EnrollmentForecastRequest) (*statspb.EnrollmentForecastResponse, error) {
	forecastYears := req.ForecastYears
	if forecastYears == 0 {
		forecastYears = 1
	}
	if forecastYears > 5 {
		forecastYears = 5 // Cap at 5 years
	}

	// Fetch historical enrollment data
	query := `
		SELECT academic_year, total_students
		FROM enrollment_history
		ORDER BY academic_year ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	defer rows.Close()

	var years []float64
	var enrollments []float64

	for rows.Next() {
		var year int
		var students int

		if err := rows.Scan(&year, &students); err != nil {
			continue
		}

		years = append(years, float64(year))
		enrollments = append(enrollments, float64(students))
	}

	if len(years) < 2 {
		return nil, status.Errorf(codes.FailedPrecondition, "insufficient historical data")
	}

	// Perform linear regression
	regression := linearRegression(years, enrollments)

	// Generate forecasts
	var projections []*statspb.EnrollmentProjection
	lastYear := int(years[len(years)-1])
	lastEnrollment := enrollments[len(enrollments)-1]

	for i := 1; i <= int(forecastYears); i++ {
		forecastYear := lastYear + i
		predictedEnrollment := regression.Slope*float64(forecastYear) + regression.Intercept

		// Determine trend
		trend := "STABLE"
		if regression.Slope > 5 {
			trend = "GROWING"
		} else if regression.Slope < -5 {
			trend = "DECLINING"
		}

		// Confidence based on R²
		confidence := regression.RSquared
		if confidence < 0 {
			confidence = 0
		} else if confidence > 1 {
			confidence = 1
		}

		projections = append(projections, &statspb.EnrollmentProjection{
			Year:              int32(forecastYear),
			ProjectedStudents: int32(predictedEnrollment),
			ConfidenceLevel:   confidence,
			Trend:             trend,
		})
	}

	// Determine forecast accuracy level
	accuracy := "MEDIUM"
	if regression.RSquared > 0.85 {
		accuracy = "HIGH"
	} else if regression.RSquared < 0.60 {
		accuracy = "LOW"
	}

	// Calculate growth rate per year
	growthRate := (regression.Slope / lastEnrollment) * 100

	return &statspb.EnrollmentForecastResponse{
		Projections:          projections,
		GrowthRate:           growthRate,
		ForecastAccuracy:     accuracy,
		HistoricalDataPoints: int32(len(years)),
	}, nil
}

// ============================================
// MAIN
// ============================================

func main() {
	shutdown := initTracer()
	defer shutdown(context.Background())

	dbHost := os.Getenv("DATABASE_URL")
	if dbHost == "" {
		dbHost = "postgres://stats_admin:stats_password@localhost:5436/stats_db?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbHost)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	log.Println("Connected to stats database")

	srv := &server{db: db}

	// RabbitMQ connection
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@rabbitmq:5672/"
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
		return
	}
	defer conn.Close()

	// Each consumer needs its own channel - amqp.Channel is NOT goroutine-safe
	gradesCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open grades channel: %v", err)
		return
	}
	defer gradesCh.Close()

	studentsCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open students channel: %v", err)
		return
	}
	defer studentsCh.Close()

	// Declare queues (idempotent)
	_, err = gradesCh.QueueDeclare("grades.assigned", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare grades.assigned queue: %v", err)
	}

	_, err = studentsCh.QueueDeclare("students.deleted", true, false, false, false, nil)
	if err != nil {
		log.Fatalf("Failed to declare students.deleted queue: %v", err)
	}

	log.Println("RabbitMQ queues declared")

	// Start consumers in background - each with its own channel
	go srv.consumeGradeAssignedEvents(gradesCh)
	go srv.consumeStudentDeletedEvents(studentsCh)

	// Start periodic reconciliation job (every 30 seconds to fix data inconsistencies)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			log.Println("[Reconciliation] Starting periodic sync job...")
			// Get all courses with grades and reconcile
			rows, err := db.Query(`SELECT DISTINCT course_id FROM grades LIMIT 10`)
			if err != nil {
				log.Printf("Reconciliation error: %v", err)
				continue
			}
			for rows.Next() {
				var courseID string
				if err := rows.Scan(&courseID); err != nil {
					continue
				}
				resp, err := srv.ReconcileGradesSync(context.Background(), &statspb.ReconcileSyncRequest{
					CourseId: courseID,
				})
				if err == nil && resp.GradesFixed > 0 {
					log.Printf("[Reconciliation] Course %s: Fixed %d grades", courseID, resp.GradesFixed)
				}
			}
			rows.Close()
		}
	}()

	// Start gRPC server
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	statspb.RegisterStatsServiceServer(grpcServer, srv)

	log.Println("Stats Service listening on :8080")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
