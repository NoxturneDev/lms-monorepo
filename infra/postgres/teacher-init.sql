CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. TEACHERS (The "User" for this service)
CREATE TABLE teachers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL, -- In real life, bcrypt hash
    full_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. COURSES (e.g., "Intro to Go", "Advanced Algorithms")
CREATE TABLE courses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    teacher_id UUID NOT NULL, -- FK to local teachers table
    title VARCHAR(150) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_teacher FOREIGN KEY (teacher_id) REFERENCES teachers(id)
);

-- 3. ENROLLMENTS (Student-Course relationship)
CREATE TABLE enrollments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    course_id UUID NOT NULL,
    student_id UUID NOT NULL, -- From Student Service
    enrolled_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_enrollment_course FOREIGN KEY (course_id) REFERENCES courses(id),
    UNIQUE(course_id, student_id) -- Prevent duplicate enrollments
);

-- 4. ASSIGNMENTS (Course has many assignments)
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    course_id UUID NOT NULL,
    title VARCHAR(150) NOT NULL,
    description TEXT,
    max_score INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_assignment_course FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE
);

-- 5. GRADES (Per-assignment, per-student)
CREATE TABLE grades (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    assignment_id UUID NOT NULL,

    -- THE MICROSERVICES LINK:
    -- This ID belongs to the Student Service. We just store it as text/uuid here.
    -- No "REFERENCES students(id)" allowed!
    student_id UUID NOT NULL,

    score INTEGER CHECK (score >= 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_grade_assignment FOREIGN KEY (assignment_id) REFERENCES assignments(id) ON DELETE CASCADE,
    UNIQUE(assignment_id, student_id) -- One grade per student per assignment
);

-- === SEED DATA ===

-- Teachers
INSERT INTO teachers (id, email, password_hash, full_name) VALUES
('d290f1ee-6c54-4b01-90e6-d701748f0851', 'turing@uni.edu', 'secret', 'Alan Turing'),
('e390f1ee-6c54-4b01-90e6-d701748f0852', 'hopper@uni.edu', 'secret', 'Grace Hopper'),
('f490f1ee-6c54-4b01-90e6-d701748f0853', 'ritchie@uni.edu', 'secret', 'Dennis Ritchie');

-- Courses
-- Alan Turing teaches Algorithms
INSERT INTO courses (id, teacher_id, title, description) VALUES
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'd290f1ee-6c54-4b01-90e6-d701748f0851', 'Advanced Algorithms', 'P vs NP and beyond'),
('c101f1ee-6c54-4b01-90e6-d701748f0851', 'd290f1ee-6c54-4b01-90e6-d701748f0851', 'Cryptography 101', 'Breaking Enigma');

-- Grace Hopper teaches Systems
INSERT INTO courses (id, teacher_id, title, description) VALUES
('c200f1ee-6c54-4b01-90e6-d701748f0852', 'e390f1ee-6c54-4b01-90e6-d701748f0852', 'Operating Systems', 'Compilers and Cobol'),
('c201f1ee-6c54-4b01-90e6-d701748f0852', 'e390f1ee-6c54-4b01-90e6-d701748f0852', 'Legacy Systems', 'Why banks still use mainframe');

-- Assignments
INSERT INTO assignments (id, course_id, title, description, max_score) VALUES
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'Midterm Exam', 'Covers sorting and graph algorithms', 100),
('a100f1ee-6c54-4b01-90e6-d701748f0002', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'Final Project', 'Implement a novel algorithm', 200),
('a200f1ee-6c54-4b01-90e6-d701748f0003', 'c200f1ee-6c54-4b01-90e6-d701748f0852', 'Lab Report 1', 'Process scheduling analysis', 50);

-- Grades (Pre-assigning some grades to Students)
-- Student 1 (John) got 95 on Algorithms Midterm
INSERT INTO grades (assignment_id, student_id, score) VALUES
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 95),
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'a888f1ee-6c54-4b01-90e6-d701748f0852', 88),
('a200f1ee-6c54-4b01-90e6-d701748f0003', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 45);
