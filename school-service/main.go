package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"

	schoolpb "github.com/noxturnedev/lms-monorepo/proto/school"
	studentpb "github.com/noxturnedev/lms-monorepo/proto/student"
	teacherpb "github.com/noxturnedev/lms-monorepo/proto/teacher"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type server struct {
	schoolpb.UnimplementedSchoolServiceServer
	db            *sql.DB
	teacherClient teacherpb.TeacherServiceClient
	studentClient studentpb.StudentServiceClient
}

// ============================================
// AUTHENTICATION
// ============================================

func (s *server) LoginAdmin(ctx context.Context, req *schoolpb.LoginAdminRequest) (*schoolpb.LoginAdminResponse, error) {
	log.Printf("Login attempt for admin: %v", req.Email)

	query := `SELECT a.id, a.email, a.full_name, a.password_hash, a.school_id, s.name
			FROM admins a
			JOIN schools s ON a.school_id = s.id
			WHERE a.email = $1`
	var id, email, fullName, passwordHash, schoolID, schoolName string
	err := s.db.QueryRow(query, req.Email).Scan(&id, &email, &fullName, &passwordHash, &schoolID, &schoolName)
	if err != nil {
		if err == sql.ErrNoRows {
			return &schoolpb.LoginAdminResponse{
				Success: false,
				Message: "Invalid email or password",
			}, nil
		}
		return nil, err
	}

	if passwordHash != req.Password {
		return &schoolpb.LoginAdminResponse{
			Success: false,
			Message: "Invalid email or password",
		}, nil
	}

	return &schoolpb.LoginAdminResponse{
		Success:    true,
		Message:    "Login successful",
		AdminId:    id,
		Email:      email,
		FullName:   fullName,
		SchoolId:   schoolID,
		SchoolName: schoolName,
	}, nil
}

// ============================================
// ADMIN CRUD
// ============================================

func (s *server) CreateAdmin(ctx context.Context, req *schoolpb.CreateAdminRequest) (*schoolpb.AdminResponse, error) {
	log.Printf("Creating Admin: %v", req.Email)

	// Verify school exists
	var schoolName string
	err := s.db.QueryRow("SELECT name FROM schools WHERE id = $1", req.SchoolId).Scan(&schoolName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("school not found")
		}
		return nil, err
	}

	query := `INSERT INTO admins (email, password_hash, full_name, school_id) VALUES ($1, $2, $3, $4) RETURNING id`
	var id string
	err = s.db.QueryRow(query, req.Email, req.Password, req.FullName, req.SchoolId).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin: %v", err)
	}
	return &schoolpb.AdminResponse{Id: id, Email: req.Email, FullName: req.FullName, SchoolId: req.SchoolId, SchoolName: schoolName}, nil
}

func (s *server) GetAdmin(ctx context.Context, req *schoolpb.GetAdminRequest) (*schoolpb.AdminResponse, error) {
	log.Printf("Getting Admin: %v", req.Id)
	query := `SELECT a.id, a.email, a.full_name, a.school_id, s.name
			FROM admins a
			JOIN schools s ON a.school_id = s.id
			WHERE a.id = $1`
	var admin schoolpb.AdminResponse
	err := s.db.QueryRow(query, req.Id).Scan(&admin.Id, &admin.Email, &admin.FullName, &admin.SchoolId, &admin.SchoolName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("admin not found")
		}
		return nil, err
	}
	return &admin, nil
}

func (s *server) UpdateAdmin(ctx context.Context, req *schoolpb.UpdateAdminRequest) (*schoolpb.AdminResponse, error) {
	log.Printf("Updating Admin: %v", req.Id)

	if req.Password != "" {
		query := `UPDATE admins SET email = $1, full_name = $2, password_hash = $3, school_id = $4 WHERE id = $5 RETURNING id, email, full_name, school_id`
		var admin schoolpb.AdminResponse
		err := s.db.QueryRow(query, req.Email, req.FullName, req.Password, req.SchoolId, req.Id).Scan(&admin.Id, &admin.Email, &admin.FullName, &admin.SchoolId)
		if err != nil {
			return nil, fmt.Errorf("failed to update admin: %v", err)
		}
		s.db.QueryRow("SELECT name FROM schools WHERE id = $1", admin.SchoolId).Scan(&admin.SchoolName)
		return &admin, nil
	}

	query := `UPDATE admins SET email = $1, full_name = $2, school_id = $3 WHERE id = $4 RETURNING id, email, full_name, school_id`
	var admin schoolpb.AdminResponse
	err := s.db.QueryRow(query, req.Email, req.FullName, req.SchoolId, req.Id).Scan(&admin.Id, &admin.Email, &admin.FullName, &admin.SchoolId)
	if err != nil {
		return nil, fmt.Errorf("failed to update admin: %v", err)
	}
	s.db.QueryRow("SELECT name FROM schools WHERE id = $1", admin.SchoolId).Scan(&admin.SchoolName)
	return &admin, nil
}

func (s *server) DeleteAdmin(ctx context.Context, req *schoolpb.DeleteAdminRequest) (*schoolpb.DeleteAdminResponse, error) {
	log.Printf("Deleting Admin: %v", req.Id)
	_, err := s.db.Exec("DELETE FROM admins WHERE id = $1", req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete admin: %v", err)
	}
	return &schoolpb.DeleteAdminResponse{Success: true}, nil
}

func (s *server) ListAdmins(ctx context.Context, req *schoolpb.ListAdminsRequest) (*schoolpb.ListAdminsResponse, error) {
	log.Println("Listing all admins")
	query := `SELECT a.id, a.email, a.full_name, a.school_id, s.name
			FROM admins a
			JOIN schools s ON a.school_id = s.id`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []*schoolpb.AdminResponse
	for rows.Next() {
		var admin schoolpb.AdminResponse
		if err := rows.Scan(&admin.Id, &admin.Email, &admin.FullName, &admin.SchoolId, &admin.SchoolName); err != nil {
			continue
		}
		admins = append(admins, &admin)
	}
	return &schoolpb.ListAdminsResponse{Admins: admins}, nil
}

// ============================================
// SCHOOL CRUD
// ============================================

func (s *server) CreateSchool(ctx context.Context, req *schoolpb.CreateSchoolRequest) (*schoolpb.SchoolResponse, error) {
	log.Printf("Creating School: %v", req.Name)
	query := `INSERT INTO schools (name, address) VALUES ($1, $2) RETURNING id`
	var id string
	err := s.db.QueryRow(query, req.Name, req.Address).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create school: %v", err)
	}
	return &schoolpb.SchoolResponse{Id: id, Name: req.Name, Address: req.Address}, nil
}

func (s *server) GetSchool(ctx context.Context, req *schoolpb.GetSchoolRequest) (*schoolpb.SchoolResponse, error) {
	log.Printf("Getting School: %v", req.Id)
	query := `SELECT id, name, address FROM schools WHERE id = $1`
	var school schoolpb.SchoolResponse
	err := s.db.QueryRow(query, req.Id).Scan(&school.Id, &school.Name, &school.Address)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("school not found")
		}
		return nil, err
	}
	return &school, nil
}

func (s *server) UpdateSchool(ctx context.Context, req *schoolpb.UpdateSchoolRequest) (*schoolpb.SchoolResponse, error) {
	log.Printf("Updating School: %v", req.Id)
	query := `UPDATE schools SET name = $1, address = $2 WHERE id = $3 RETURNING id, name, address`
	var school schoolpb.SchoolResponse
	err := s.db.QueryRow(query, req.Name, req.Address, req.Id).Scan(&school.Id, &school.Name, &school.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to update school: %v", err)
	}
	return &school, nil
}

func (s *server) DeleteSchool(ctx context.Context, req *schoolpb.DeleteSchoolRequest) (*schoolpb.DeleteSchoolResponse, error) {
	log.Printf("Deleting School: %v", req.Id)

	// Check for admins belonging to this school
	var adminCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM admins WHERE school_id = $1", req.Id).Scan(&adminCount)
	if err != nil {
		return nil, err
	}
	if adminCount > 0 {
		return &schoolpb.DeleteSchoolResponse{
			Success: false,
			Message: fmt.Sprintf("Cannot delete school with %d admins", adminCount),
		}, nil
	}

	// Check for classes belonging to this school
	var classCount int
	err = s.db.QueryRow("SELECT COUNT(*) FROM classes WHERE school_id = $1", req.Id).Scan(&classCount)
	if err != nil {
		return nil, err
	}
	if classCount > 0 {
		return &schoolpb.DeleteSchoolResponse{
			Success: false,
			Message: fmt.Sprintf("Cannot delete school with %d classes", classCount),
		}, nil
	}

	_, err = s.db.Exec("DELETE FROM schools WHERE id = $1", req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete school: %v", err)
	}
	return &schoolpb.DeleteSchoolResponse{Success: true, Message: "School deleted successfully"}, nil
}

func (s *server) ListSchools(ctx context.Context, req *schoolpb.ListSchoolsRequest) (*schoolpb.ListSchoolsResponse, error) {
	log.Println("Listing all schools")
	query := `SELECT id, name, address FROM schools`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schools []*schoolpb.SchoolResponse
	for rows.Next() {
		var school schoolpb.SchoolResponse
		if err := rows.Scan(&school.Id, &school.Name, &school.Address); err != nil {
			continue
		}
		schools = append(schools, &school)
	}
	return &schoolpb.ListSchoolsResponse{Schools: schools}, nil
}

// ============================================
// CLASS CRUD
// ============================================

func (s *server) CreateClass(ctx context.Context, req *schoolpb.CreateClassRequest) (*schoolpb.ClassResponse, error) {
	log.Printf("Creating Class: %v", req.Name)

	// Verify school exists
	var schoolName string
	err := s.db.QueryRow("SELECT name FROM schools WHERE id = $1", req.SchoolId).Scan(&schoolName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("school not found")
		}
		return nil, err
	}

	query := `INSERT INTO classes (school_id, name, grade_level) VALUES ($1, $2, $3) RETURNING id`
	var id string
	err = s.db.QueryRow(query, req.SchoolId, req.Name, req.GradeLevel).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create class: %v", err)
	}
	return &schoolpb.ClassResponse{Id: id, SchoolId: req.SchoolId, SchoolName: schoolName, Name: req.Name, GradeLevel: req.GradeLevel}, nil
}

func (s *server) GetClass(ctx context.Context, req *schoolpb.GetClassRequest) (*schoolpb.ClassResponse, error) {
	log.Printf("Getting Class: %v", req.Id)
	query := `SELECT c.id, c.school_id, s.name, c.name, c.grade_level
			FROM classes c
			JOIN schools s ON c.school_id = s.id
			WHERE c.id = $1`
	var class schoolpb.ClassResponse
	err := s.db.QueryRow(query, req.Id).Scan(&class.Id, &class.SchoolId, &class.SchoolName, &class.Name, &class.GradeLevel)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("class not found")
		}
		return nil, err
	}
	return &class, nil
}

func (s *server) UpdateClass(ctx context.Context, req *schoolpb.UpdateClassRequest) (*schoolpb.ClassResponse, error) {
	log.Printf("Updating Class: %v", req.Id)
	query := `UPDATE classes SET name = $1, grade_level = $2 WHERE id = $3 RETURNING id, school_id`
	var id, schoolId string
	err := s.db.QueryRow(query, req.Name, req.GradeLevel, req.Id).Scan(&id, &schoolId)
	if err != nil {
		return nil, fmt.Errorf("failed to update class: %v", err)
	}

	var schoolName string
	s.db.QueryRow("SELECT name FROM schools WHERE id = $1", schoolId).Scan(&schoolName)

	return &schoolpb.ClassResponse{Id: id, SchoolId: schoolId, SchoolName: schoolName, Name: req.Name, GradeLevel: req.GradeLevel}, nil
}

func (s *server) DeleteClass(ctx context.Context, req *schoolpb.DeleteClassRequest) (*schoolpb.DeleteClassResponse, error) {
	log.Printf("Deleting Class: %v", req.Id)
	_, err := s.db.Exec("DELETE FROM classes WHERE id = $1", req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete class: %v", err)
	}
	return &schoolpb.DeleteClassResponse{Success: true}, nil
}

func (s *server) ListClasses(ctx context.Context, req *schoolpb.ListClassesRequest) (*schoolpb.ListClassesResponse, error) {
	log.Printf("Listing classes, filter school_id: %v", req.SchoolId)

	var query string
	var rows *sql.Rows
	var err error

	if req.SchoolId != "" {
		query = `SELECT c.id, c.school_id, s.name, c.name, c.grade_level
				FROM classes c
				JOIN schools s ON c.school_id = s.id
				WHERE c.school_id = $1`
		rows, err = s.db.Query(query, req.SchoolId)
	} else {
		query = `SELECT c.id, c.school_id, s.name, c.name, c.grade_level
				FROM classes c
				JOIN schools s ON c.school_id = s.id`
		rows, err = s.db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var classes []*schoolpb.ClassResponse
	for rows.Next() {
		var class schoolpb.ClassResponse
		if err := rows.Scan(&class.Id, &class.SchoolId, &class.SchoolName, &class.Name, &class.GradeLevel); err != nil {
			continue
		}
		classes = append(classes, &class)
	}
	return &schoolpb.ListClassesResponse{Classes: classes}, nil
}

// ============================================
// COURSE MANAGEMENT
// ============================================

func (s *server) CreateCourse(ctx context.Context, req *schoolpb.CreateCourseRequest) (*schoolpb.CourseResponse, error) {
	log.Printf("Creating Course: %v for school: %v", req.Title, req.SchoolId)

	// Verify school exists
	var schoolName string
	err := s.db.QueryRow("SELECT name FROM schools WHERE id = $1", req.SchoolId).Scan(&schoolName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("school not found")
		}
		return nil, err
	}

	query := `INSERT INTO courses (school_id, title, description) VALUES ($1, $2, $3) RETURNING id`
	var id string
	err = s.db.QueryRow(query, req.SchoolId, req.Title, req.Description).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to create course: %v", err)
	}

	return &schoolpb.CourseResponse{
		Id:       id,
		Title:    req.Title,
		SchoolId: req.SchoolId,
	}, nil
}

func (s *server) GetCourse(ctx context.Context, req *schoolpb.GetCourseRequest) (*schoolpb.CourseDetailResponse, error) {
	log.Printf("Getting Course: %v", req.Id)

	// Get course details with school info
	query := `SELECT c.id, c.school_id, s.name, c.title, c.description
			FROM courses c
			JOIN schools s ON c.school_id = s.id
			WHERE c.id = $1`
	var course schoolpb.CourseDetailResponse
	err := s.db.QueryRow(query, req.Id).Scan(
		&course.Id,
		&course.SchoolId,
		&course.SchoolName,
		&course.Title,
		&course.Description,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("course not found")
		}
		return nil, err
	}

	// Get assigned teachers
	teacherQuery := `SELECT teacher_id FROM course_teacher_assignments WHERE course_id = $1`
	rows, err := s.db.Query(teacherQuery, req.Id)
	if err != nil {
		log.Printf("Failed to query teacher assignments: %v", err)
	} else {
		defer rows.Close()
		var teacherIds []string
		var teacherNames []string

		for rows.Next() {
			var teacherId string
			if err := rows.Scan(&teacherId); err != nil {
				continue
			}
			teacherIds = append(teacherIds, teacherId)

			// Fetch teacher name from Teacher Service
			if s.teacherClient != nil {
				teacherResp, err := s.teacherClient.GetTeacher(ctx, &teacherpb.GetTeacherRequest{Id: teacherId})
				if err != nil {
					log.Printf("Failed to get teacher %s: %v", teacherId, err)
					teacherNames = append(teacherNames, "Unknown")
				} else {
					teacherNames = append(teacherNames, teacherResp.FullName)
				}
			}
		}

		course.TeacherIds = teacherIds
		course.TeacherNames = teacherNames
	}

	return &course, nil
}

func (s *server) UpdateCourse(ctx context.Context, req *schoolpb.UpdateCourseRequest) (*schoolpb.CourseResponse, error) {
	log.Printf("Updating Course: %v", req.Id)

	query := `UPDATE courses SET title = $1, description = $2 WHERE id = $3 RETURNING id, title, school_id`
	var course schoolpb.CourseResponse
	err := s.db.QueryRow(query, req.Title, req.Description, req.Id).Scan(&course.Id, &course.Title, &course.SchoolId)
	if err != nil {
		return nil, fmt.Errorf("failed to update course: %v", err)
	}

	return &course, nil
}

func (s *server) DeleteCourse(ctx context.Context, req *schoolpb.DeleteCourseRequest) (*schoolpb.DeleteCourseResponse, error) {
	log.Printf("Deleting Course: %v", req.Id)

	// Check for enrollments
	var enrollmentCount int
	err := s.db.QueryRow("SELECT COUNT(*) FROM enrollments WHERE course_id = $1", req.Id).Scan(&enrollmentCount)
	if err != nil {
		return nil, err
	}
	if enrollmentCount > 0 {
		return &schoolpb.DeleteCourseResponse{
			Success: false,
			Message: fmt.Sprintf("Cannot delete course with %d enrollments", enrollmentCount),
		}, nil
	}

	// Check for teacher assignments via teacher service
	if s.teacherClient != nil {
		validationResp, err := s.teacherClient.ValidateCourseHasAssignments(ctx, &teacherpb.ValidateCourseAssignmentsRequest{CourseId: req.Id})
		if err != nil {
			log.Printf("Failed to validate course assignments: %v", err)
		} else if validationResp.HasAssignments {
			return &schoolpb.DeleteCourseResponse{
				Success: false,
				Message: "Cannot delete course with existing assignments",
			}, nil
		}
	}

	// Delete course (will cascade delete teacher assignments due to FK)
	_, err = s.db.Exec("DELETE FROM courses WHERE id = $1", req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to delete course: %v", err)
	}

	return &schoolpb.DeleteCourseResponse{
		Success: true,
		Message: "Course deleted successfully",
	}, nil
}

func (s *server) ListCourses(ctx context.Context, req *schoolpb.ListCoursesRequest) (*schoolpb.ListCoursesResponse, error) {
	log.Printf("Listing courses, filter school_id: %v", req.SchoolId)

	var query string
	var rows *sql.Rows
	var err error

	if req.SchoolId != "" {
		query = `SELECT c.id, c.school_id, s.name, c.title, c.description
				FROM courses c
				JOIN schools s ON c.school_id = s.id
				WHERE c.school_id = $1`
		rows, err = s.db.Query(query, req.SchoolId)
	} else {
		query = `SELECT c.id, c.school_id, s.name, c.title, c.description
				FROM courses c
				JOIN schools s ON c.school_id = s.id`
		rows, err = s.db.Query(query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []*schoolpb.CourseDetailResponse
	for rows.Next() {
		var course schoolpb.CourseDetailResponse
		if err := rows.Scan(&course.Id, &course.SchoolId, &course.SchoolName, &course.Title, &course.Description); err != nil {
			continue
		}

		// Get assigned teachers for each course
		teacherQuery := `SELECT teacher_id FROM course_teacher_assignments WHERE course_id = $1`
		teacherRows, err := s.db.Query(teacherQuery, course.Id)
		if err == nil {
			var teacherIds []string
			var teacherNames []string

			for teacherRows.Next() {
				var teacherId string
				if err := teacherRows.Scan(&teacherId); err != nil {
					continue
				}
				teacherIds = append(teacherIds, teacherId)

				// Fetch teacher name from Teacher Service
				if s.teacherClient != nil {
					teacherResp, err := s.teacherClient.GetTeacher(ctx, &teacherpb.GetTeacherRequest{Id: teacherId})
					if err != nil {
						teacherNames = append(teacherNames, "Unknown")
					} else {
						teacherNames = append(teacherNames, teacherResp.FullName)
					}
				}
			}
			teacherRows.Close()

			course.TeacherIds = teacherIds
			course.TeacherNames = teacherNames
		}

		courses = append(courses, &course)
	}

	return &schoolpb.ListCoursesResponse{Courses: courses}, nil
}

// ============================================
// COURSE-TEACHER ASSIGNMENT
// ============================================

func (s *server) AssignTeacherToCourse(ctx context.Context, req *schoolpb.AssignTeacherToCourseRequest) (*schoolpb.AssignmentResponse, error) {
	log.Printf("Assigning teacher %v to course %v", req.TeacherId, req.CourseId)

	// Verify course exists
	var courseId string
	err := s.db.QueryRow("SELECT id FROM courses WHERE id = $1", req.CourseId).Scan(&courseId)
	if err != nil {
		if err == sql.ErrNoRows {
			return &schoolpb.AssignmentResponse{
				Success: false,
				Message: "Course not found",
			}, nil
		}
		return nil, err
	}

	// Verify teacher exists via Teacher Service
	if s.teacherClient != nil {
		_, err := s.teacherClient.GetTeacher(ctx, &teacherpb.GetTeacherRequest{Id: req.TeacherId})
		if err != nil {
			return &schoolpb.AssignmentResponse{
				Success: false,
				Message: "Teacher not found",
			}, nil
		}
	}

	// Insert assignment
	query := `INSERT INTO course_teacher_assignments (course_id, teacher_id) VALUES ($1, $2) RETURNING id`
	var id string
	err = s.db.QueryRow(query, req.CourseId, req.TeacherId).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to assign teacher: %v", err)
	}

	return &schoolpb.AssignmentResponse{
		Id:      id,
		Success: true,
		Message: "Teacher assigned successfully",
	}, nil
}

func (s *server) UnassignTeacherFromCourse(ctx context.Context, req *schoolpb.UnassignTeacherRequest) (*schoolpb.UnassignmentResponse, error) {
	log.Printf("Unassigning teacher %v from course %v", req.TeacherId, req.CourseId)

	query := `DELETE FROM course_teacher_assignments WHERE course_id = $1 AND teacher_id = $2`
	result, err := s.db.Exec(query, req.CourseId, req.TeacherId)
	if err != nil {
		return nil, fmt.Errorf("failed to unassign teacher: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return &schoolpb.UnassignmentResponse{
			Success: false,
			Message: "Assignment not found",
		}, nil
	}

	return &schoolpb.UnassignmentResponse{
		Success: true,
		Message: "Teacher unassigned successfully",
	}, nil
}

func (s *server) GetCourseTeachers(ctx context.Context, req *schoolpb.GetCourseTeachersRequest) (*schoolpb.CourseTeachersResponse, error) {
	log.Printf("Getting teachers for course: %v", req.CourseId)

	query := `SELECT teacher_id FROM course_teacher_assignments WHERE course_id = $1`
	rows, err := s.db.Query(query, req.CourseId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teachers []*schoolpb.TeacherInfo
	for rows.Next() {
		var teacherId string
		if err := rows.Scan(&teacherId); err != nil {
			continue
		}

		// Fetch teacher details from Teacher Service
		teacherInfo := &schoolpb.TeacherInfo{TeacherId: teacherId}
		if s.teacherClient != nil {
			teacherResp, err := s.teacherClient.GetTeacher(ctx, &teacherpb.GetTeacherRequest{Id: teacherId})
			if err != nil {
				log.Printf("Failed to get teacher %s: %v", teacherId, err)
				teacherInfo.TeacherName = "Unknown"
				teacherInfo.TeacherEmail = "unknown@email.com"
			} else {
				teacherInfo.TeacherName = teacherResp.FullName
				teacherInfo.TeacherEmail = teacherResp.Email
			}
		}

		teachers = append(teachers, teacherInfo)
	}

	return &schoolpb.CourseTeachersResponse{Teachers: teachers}, nil
}

// ============================================
// ENROLLMENT MANAGEMENT
// ============================================

func (s *server) EnrollStudent(ctx context.Context, req *schoolpb.EnrollStudentRequest) (*schoolpb.EnrollmentResponse, error) {
	log.Printf("Enrolling student %v in course %v", req.StudentId, req.CourseId)

	// Verify course exists
	var courseId string
	err := s.db.QueryRow("SELECT id FROM courses WHERE id = $1", req.CourseId).Scan(&courseId)
	if err != nil {
		if err == sql.ErrNoRows {
			return &schoolpb.EnrollmentResponse{
				Success: false,
				Message: "Course not found",
			}, nil
		}
		return nil, err
	}

	// Verify student exists via Student Service
	if s.studentClient != nil {
		_, err := s.studentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{Id: req.StudentId})
		if err != nil {
			return &schoolpb.EnrollmentResponse{
				Success: false,
				Message: "Student not found",
			}, nil
		}
	}

	// Insert enrollment
	query := `INSERT INTO enrollments (course_id, student_id) VALUES ($1, $2) RETURNING id`
	var id string
	err = s.db.QueryRow(query, req.CourseId, req.StudentId).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to enroll student: %v", err)
	}

	return &schoolpb.EnrollmentResponse{
		Id:      id,
		Success: true,
		Message: "Student enrolled successfully",
	}, nil
}

func (s *server) UnenrollStudent(ctx context.Context, req *schoolpb.UnenrollStudentRequest) (*schoolpb.UnenrollmentResponse, error) {
	log.Printf("Unenrolling student %v from course %v", req.StudentId, req.CourseId)

	query := `DELETE FROM enrollments WHERE course_id = $1 AND student_id = $2`
	result, err := s.db.Exec(query, req.CourseId, req.StudentId)
	if err != nil {
		return nil, fmt.Errorf("failed to unenroll student: %v", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return &schoolpb.UnenrollmentResponse{
			Success: false,
			Message: "Enrollment not found",
		}, nil
	}

	return &schoolpb.UnenrollmentResponse{
		Success: true,
		Message: "Student unenrolled successfully",
	}, nil
}

func (s *server) GetCourseEnrollments(ctx context.Context, req *schoolpb.GetCourseEnrollmentsRequest) (*schoolpb.CourseEnrollmentsResponse, error) {
	log.Printf("Getting enrollments for course: %v", req.CourseId)

	query := `SELECT student_id FROM enrollments WHERE course_id = $1`
	rows, err := s.db.Query(query, req.CourseId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var students []*schoolpb.StudentInfo
	for rows.Next() {
		var studentId string
		if err := rows.Scan(&studentId); err != nil {
			continue
		}

		// Fetch student details from Student Service
		studentInfo := &schoolpb.StudentInfo{StudentId: studentId}
		if s.studentClient != nil {
			studentResp, err := s.studentClient.GetStudentById(ctx, &studentpb.GetStudentByIdRequest{Id: studentId})
			if err != nil {
				log.Printf("Failed to get student %s: %v", studentId, err)
				studentInfo.StudentName = "Unknown"
				studentInfo.StudentNumber = "N/A"
			} else {
				studentInfo.StudentName = studentResp.FullName
				studentInfo.StudentNumber = studentResp.StudentNumber
			}
		}

		students = append(students, studentInfo)
	}

	return &schoolpb.CourseEnrollmentsResponse{Students: students}, nil
}

func (s *server) GetStudentEnrollments(ctx context.Context, req *schoolpb.GetStudentEnrollmentsRequest) (*schoolpb.StudentEnrollmentsResponse, error) {
	log.Printf("Getting enrollments for student: %v", req.StudentId)

	query := `SELECT c.id, c.school_id, s.name, c.title, c.description
			FROM enrollments e
			JOIN courses c ON e.course_id = c.id
			JOIN schools s ON c.school_id = s.id
			WHERE e.student_id = $1`
	rows, err := s.db.Query(query, req.StudentId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []*schoolpb.CourseDetailResponse
	for rows.Next() {
		var course schoolpb.CourseDetailResponse
		if err := rows.Scan(&course.Id, &course.SchoolId, &course.SchoolName, &course.Title, &course.Description); err != nil {
			continue
		}

		// Get assigned teachers for each course
		teacherQuery := `SELECT teacher_id FROM course_teacher_assignments WHERE course_id = $1`
		teacherRows, err := s.db.Query(teacherQuery, course.Id)
		if err == nil {
			var teacherIds []string
			var teacherNames []string

			for teacherRows.Next() {
				var teacherId string
				if err := teacherRows.Scan(&teacherId); err != nil {
					continue
				}
				teacherIds = append(teacherIds, teacherId)

				// Fetch teacher name from Teacher Service
				if s.teacherClient != nil {
					teacherResp, err := s.teacherClient.GetTeacher(ctx, &teacherpb.GetTeacherRequest{Id: teacherId})
					if err != nil {
						teacherNames = append(teacherNames, "Unknown")
					} else {
						teacherNames = append(teacherNames, teacherResp.FullName)
					}
				}
			}
			teacherRows.Close()

			course.TeacherIds = teacherIds
			course.TeacherNames = teacherNames
		}

		courses = append(courses, &course)
	}

	return &schoolpb.StudentEnrollmentsResponse{Courses: courses}, nil
}

func (s *server) ValidateCourseExists(ctx context.Context, req *schoolpb.ValidateCourseRequest) (*schoolpb.ValidateCourseResponse, error) {
	log.Printf("Validating course exists: %v", req.CourseId)

	var schoolId string
	err := s.db.QueryRow("SELECT school_id FROM courses WHERE id = $1", req.CourseId).Scan(&schoolId)
	if err != nil {
		if err == sql.ErrNoRows {
			return &schoolpb.ValidateCourseResponse{
				Exists:   false,
				SchoolId: "",
			}, nil
		}
		return nil, err
	}

	return &schoolpb.ValidateCourseResponse{
		Exists:   true,
		SchoolId: schoolId,
	}, nil
}

// ============================================
// TEACHER COURSE LIST
// ============================================

func (s *server) GetTeacherCourseList(ctx context.Context, req *schoolpb.GetTeacherCourseListRequest) (*schoolpb.GetTeacherCourseListResponse, error) {
	log.Printf("Getting courses for teacher: %v", req.TeacherId)

	query := `
		SELECT c.id, c.title, cta.teacher_id
		FROM course_teacher_assignments cta
		JOIN courses c ON cta.course_id = c.id
		WHERE cta.teacher_id = $1
	`

	rows, err := s.db.Query(query, req.TeacherId)
	if err != nil {
		return nil, fmt.Errorf("failed to get teacher courses: %v", err)
	}
	defer rows.Close()

	var courses []*schoolpb.TeacherCourseListResponse
	for rows.Next() {
		var course schoolpb.TeacherCourseListResponse
		if err := rows.Scan(&course.CourseId, &course.Title, &course.TeacherId); err != nil {
			continue
		}
		courses = append(courses, &course)
	}

	return &schoolpb.GetTeacherCourseListResponse{Courses: courses}, nil
}

func main() {
	shutdown := initTracer()
	defer shutdown(context.Background())

	dbHost := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", dbHost)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	// Connect to Teacher Service
	teacherServiceURL := os.Getenv("TEACHER_SERVICE_URL")
	if teacherServiceURL == "" {
		teacherServiceURL = "teacher-service:8080"
	}
	teacherConn, err := grpc.NewClient(
		teacherServiceURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("failed to connect to teacher service: %v", err)
	}
	defer teacherConn.Close()
	teacherClient := teacherpb.NewTeacherServiceClient(teacherConn)

	// Connect to Student Service
	studentServiceURL := os.Getenv("STUDENT_SERVICE_URL")
	if studentServiceURL == "" {
		studentServiceURL = "student-service:8080"
	}
	studentConn, err := grpc.NewClient(
		studentServiceURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("failed to connect to student service: %v", err)
	}
	defer studentConn.Close()
	studentClient := studentpb.NewStudentServiceClient(studentConn)

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	schoolpb.RegisterSchoolServiceServer(s, &server{
		db:            db,
		teacherClient: teacherClient,
		studentClient: studentClient,
	})
	reflection.Register(s)

	log.Println("School Service listening on port 8080...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
