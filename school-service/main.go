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
