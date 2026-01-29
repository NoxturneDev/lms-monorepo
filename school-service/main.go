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
	"google.golang.org/grpc/reflection"

	schoolpb "github.com/noxturnedev/lms-monorepo/proto/school"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

type server struct {
	schoolpb.UnimplementedSchoolServiceServer
	db *sql.DB
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

func main() {
	shutdown := initTracer()
	defer shutdown(context.Background())

	dbHost := os.Getenv("DATABASE_URL")
	db, err := sql.Open("postgres", dbHost)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)

	schoolpb.RegisterSchoolServiceServer(s, &server{db: db})
	reflection.Register(s)

	log.Println("School Service listening on port 8080...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
