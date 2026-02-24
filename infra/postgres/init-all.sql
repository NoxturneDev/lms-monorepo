-- Consolidated PostgreSQL initialization script for all 4 LMS databases
-- This script creates all databases, users, and schemas in a single Postgres container

-- ================================================================
-- DATABASE CREATION
-- ================================================================

CREATE DATABASE teacher_db;
CREATE DATABASE student_db;
CREATE DATABASE school_db;
CREATE DATABASE stats_db;

-- ================================================================
-- USER CREATION (Per-service users with scoped permissions)
-- ================================================================

CREATE USER teacher_admin WITH PASSWORD 'teacher_password';
CREATE USER student_admin WITH PASSWORD 'student_password';
CREATE USER school_admin WITH PASSWORD 'school_password';
CREATE USER stats_admin WITH PASSWORD 'stats_password';

-- ================================================================
-- GRANT PERMISSIONS (Least privilege: database-level only)
-- ================================================================

GRANT ALL PRIVILEGES ON DATABASE teacher_db TO teacher_admin;
GRANT ALL PRIVILEGES ON DATABASE student_db TO student_admin;
GRANT ALL PRIVILEGES ON DATABASE school_db TO school_admin;
GRANT ALL PRIVILEGES ON DATABASE stats_db TO stats_admin;

-- ================================================================
-- TEACHER DATABASE SCHEMA
-- ================================================================

\c teacher_db

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. TEACHERS (The "User" for this service)
CREATE TABLE teachers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. ASSIGNMENTS (Course has many assignments)
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    course_id UUID NOT NULL,
    title VARCHAR(150) NOT NULL,
    description TEXT,
    category VARCHAR(100),  -- "Lab", "Exam", "Quiz", "Project", etc.
    max_score INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. GRADES (Per-assignment, per-student)
CREATE TABLE grades (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    assignment_id UUID NOT NULL,
    student_id UUID NOT NULL,
    score INTEGER CHECK (score >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_grade_assignment FOREIGN KEY (assignment_id) REFERENCES assignments(id) ON DELETE CASCADE,
    UNIQUE(assignment_id, student_id)
);

-- Teacher Seed Data
INSERT INTO teachers (id, email, password_hash, full_name) VALUES
('d290f1ee-6c54-4b01-90e6-d701748f0851', 'turing@uni.edu', 'secret', 'Alan Turing'),
('e390f1ee-6c54-4b01-90e6-d701748f0852', 'hopper@uni.edu', 'secret', 'Grace Hopper'),
('f490f1ee-6c54-4b01-90e6-d701748f0853', 'ritchie@uni.edu', 'secret', 'Dennis Ritchie');

INSERT INTO assignments (id, course_id, title, description, category, max_score) VALUES
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'Midterm Exam', 'Covers sorting and graph algorithms', 'Exam', 100),
('a100f1ee-6c54-4b01-90e6-d701748f0002', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'Final Project', 'Implement a novel algorithm', 'Project', 200),
('a200f1ee-6c54-4b01-90e6-d701748f0003', 'c200f1ee-6c54-4b01-90e6-d701748f0852', 'Lab Report 1', 'Process scheduling analysis', 'Lab', 50);

INSERT INTO grades (assignment_id, student_id, score) VALUES
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 95),
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'a888f1ee-6c54-4b01-90e6-d701748f0852', 88),
('a200f1ee-6c54-4b01-90e6-d701748f0003', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 45);

-- ================================================================
-- STUDENT DATABASE SCHEMA
-- ================================================================

\c student_db

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. STUDENTS (The "User" for this service)
CREATE TABLE students (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100),
    student_number VARCHAR(20) UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Student Seed Data
INSERT INTO students (id, email, password_hash, full_name, student_number) VALUES
('a999f1ee-6c54-4b01-90e6-d701748f0851', 'john@student.edu', 'secret', 'John Doe', 'STD-2026-001'),
('a888f1ee-6c54-4b01-90e6-d701748f0852', 'jane@student.edu', 'secret', 'Jane Smith', 'STD-2026-002'),
('a777f1ee-6c54-4b01-90e6-d701748f0853', 'bob@student.edu', 'secret', 'Bob Martin', 'STD-2026-003'),
('a666f1ee-6c54-4b01-90e6-d701748f0854', 'alice@student.edu', 'secret', 'Alice Wonderland', 'STD-2026-004');

-- ================================================================
-- SCHOOL DATABASE SCHEMA
-- ================================================================

\c school_db

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. SCHOOLS
CREATE TABLE schools (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    address TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. ADMINS (School administrators)
CREATE TABLE admins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100),
    school_id UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_school FOREIGN KEY (school_id) REFERENCES schools(id)
);

-- 3. CLASSES
CREATE TABLE classes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    school_id UUID NOT NULL,
    name VARCHAR(150) NOT NULL,
    grade_level VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_class_school FOREIGN KEY (school_id) REFERENCES schools(id)
);

-- 4. COURSES (School-owned academic courses)
CREATE TABLE courses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    school_id UUID NOT NULL,
    title VARCHAR(150) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_course_school FOREIGN KEY (school_id) REFERENCES schools(id) ON DELETE CASCADE
);

-- 5. COURSE_TEACHER_ASSIGNMENTS (Many-to-many: courses ↔ teachers)
CREATE TABLE course_teacher_assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    course_id UUID NOT NULL,
    teacher_id UUID NOT NULL,
    assigned_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_cta_course FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
    UNIQUE(course_id, teacher_id)
);

-- 6. ENROLLMENTS (Student-Course relationship)
CREATE TABLE enrollments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    course_id UUID NOT NULL,
    student_id UUID NOT NULL,
    enrolled_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_enrollment_course FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
    UNIQUE(course_id, student_id)
);

-- School Seed Data
INSERT INTO schools (id, name, address) VALUES
('b100f1ee-6c54-4b01-90e6-d701748f0001', 'Greenwood Academy', '123 Elm Street, Springfield'),
('b200f1ee-6c54-4b01-90e6-d701748f0002', 'Riverside High School', '456 Oak Avenue, Shelbyville');

INSERT INTO admins (id, email, password_hash, full_name, school_id) VALUES
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'admin@greenwood.edu', 'secret', 'Alice Greenwood', 'b100f1ee-6c54-4b01-90e6-d701748f0001'),
('a200f1ee-6c54-4b01-90e6-d701748f0002', 'admin@riverside.edu', 'secret', 'Bob Riverside', 'b200f1ee-6c54-4b01-90e6-d701748f0002');

INSERT INTO classes (id, school_id, name, grade_level) VALUES
('c000f1ee-6c54-4b01-90e6-d701748f0001', 'b100f1ee-6c54-4b01-90e6-d701748f0001', 'Mathematics A', '10th Grade'),
('c000f1ee-6c54-4b01-90e6-d701748f0002', 'b100f1ee-6c54-4b01-90e6-d701748f0001', 'Science B', '11th Grade'),
('c000f1ee-6c54-4b01-90e6-d701748f0003', 'b200f1ee-6c54-4b01-90e6-d701748f0002', 'English Literature', '12th Grade');

INSERT INTO courses (id, school_id, title, description) VALUES
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'b100f1ee-6c54-4b01-90e6-d701748f0001', 'Advanced Algorithms', 'P vs NP and beyond'),
('c101f1ee-6c54-4b01-90e6-d701748f0851', 'b100f1ee-6c54-4b01-90e6-d701748f0001', 'Cryptography 101', 'Breaking Enigma'),
('c200f1ee-6c54-4b01-90e6-d701748f0852', 'b200f1ee-6c54-4b01-90e6-d701748f0002', 'Operating Systems', 'Compilers and Cobol');

INSERT INTO course_teacher_assignments (course_id, teacher_id) VALUES
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'd290f1ee-6c54-4b01-90e6-d701748f0851'),
('c101f1ee-6c54-4b01-90e6-d701748f0851', 'd290f1ee-6c54-4b01-90e6-d701748f0851'),
('c200f1ee-6c54-4b01-90e6-d701748f0852', 'e390f1ee-6c54-4b01-90e6-d701748f0852');

INSERT INTO enrollments (course_id, student_id) VALUES
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'a999f1ee-6c54-4b01-90e6-d701748f0851'),
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'a888f1ee-6c54-4b01-90e6-d701748f0852');

-- ================================================================
-- STATS DATABASE SCHEMA
-- ================================================================

\c stats_db

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

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
    category VARCHAR(100),
    max_score INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. GRADES (from GradeAssigned events - fully denormalized for fast queries)
CREATE TABLE grades (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL,
    assignment_id UUID NOT NULL,
    student_id UUID NOT NULL,
    score INTEGER NOT NULL,
    max_score INTEGER NOT NULL,
    category VARCHAR(100),
    percentage DECIMAL(5,2) GENERATED ALWAYS AS ((score::DECIMAL / max_score) * 100) STORED,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(id)
);

-- 4. STUDENT_ENROLLMENTS (track which students are in which courses)
CREATE TABLE student_enrollments (
    course_id UUID NOT NULL,
    student_id UUID NOT NULL,
    enrolled_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (course_id, student_id)
);

-- 5. DELETED_STUDENTS (tombstone pattern for StudentDeleted events)
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

-- Stats Seed Data
INSERT INTO courses (id, title) VALUES
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'Advanced Algorithms'),
('c101f1ee-6c54-4b01-90e6-d701748f0851', 'Cryptography 101'),
('c200f1ee-6c54-4b01-90e6-d701748f0852', 'Operating Systems');

INSERT INTO assignments (id, course_id, title, category, max_score) VALUES
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'Midterm Exam', 'Exam', 100),
('a100f1ee-6c54-4b01-90e6-d701748f0002', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'Final Project', 'Project', 200),
('a200f1ee-6c54-4b01-90e6-d701748f0003', 'c200f1ee-6c54-4b01-90e6-d701748f0852', 'Lab Report 1', 'Lab', 50);

INSERT INTO grades (id, course_id, assignment_id, student_id, score, max_score, category) VALUES
('d100f1ee-6c54-4b01-90e6-d701748f0851', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'a100f1ee-6c54-4b01-90e6-d701748f0001', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 95, 100, 'Exam'),
('d100f1ee-6c54-4b01-90e6-d701748f0852', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'a100f1ee-6c54-4b01-90e6-d701748f0002', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 95, 200, 'Project'),
('d200f1ee-6c54-4b01-90e6-d701748f0853', 'c200f1ee-6c54-4b01-90e6-d701748f0852', 'a200f1ee-6c54-4b01-90e6-d701748f0003', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 45, 50, 'Lab'),
('d100f1ee-6c54-4b01-90e6-d701748f0854', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'a100f1ee-6c54-4b01-90e6-d701748f0001', 'a888f1ee-6c54-4b01-90e6-d701748f0852', 88, 100, 'Exam');

INSERT INTO student_enrollments (course_id, student_id) VALUES
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'a999f1ee-6c54-4b01-90e6-d701748f0851'),
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'a888f1ee-6c54-4b01-90e6-d701748f0852'),
('c200f1ee-6c54-4b01-90e6-d701748f0852', 'a999f1ee-6c54-4b01-90e6-d701748f0851');
