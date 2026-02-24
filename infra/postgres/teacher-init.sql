CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. TEACHERS (The "User" for this service)
CREATE TABLE teachers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL, -- In real life, bcrypt hash
    full_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. ASSIGNMENTS (Course has many assignments)
-- NOTE: course_id is an external reference to School Service (no FK constraint)
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    course_id UUID NOT NULL,  -- External ID from School Service
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

-- Assignments (course_id references courses in School Service)
INSERT INTO assignments (id, course_id, title, description, category, max_score) VALUES
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'Midterm Exam', 'Covers sorting and graph algorithms', 'Exam', 100),
('a100f1ee-6c54-4b01-90e6-d701748f0002', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'Final Project', 'Implement a novel algorithm', 'Project', 200),
('a200f1ee-6c54-4b01-90e6-d701748f0003', 'c200f1ee-6c54-4b01-90e6-d701748f0852', 'Lab Report 1', 'Process scheduling analysis', 'Lab', 50);

-- Grades (Pre-assigning some grades to Students)
-- Student 1 (John) got 95 on Algorithms Midterm
INSERT INTO grades (assignment_id, student_id, score) VALUES
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 95),
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'a888f1ee-6c54-4b01-90e6-d701748f0852', 88),
('a200f1ee-6c54-4b01-90e6-d701748f0003', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 45);
