CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Read Model for Stats Service
-- This is a FLATTENED projection built from events

-- 1. COURSES (from CourseCreated events)
CREATE TABLE courses (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. ASSIGNMENTS (from AssignmentCreated events - if you add this later)
CREATE TABLE assignments (
    id UUID PRIMARY KEY,
    course_id UUID NOT NULL,
    title VARCHAR(255) NOT NULL,
    category VARCHAR(100),  -- "Lab", "Exam", "Quiz", "Project", etc.
    max_score INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 3. GRADES (from GradeAssigned events)
-- This is the MAIN table - fully denormalized for fast queries
CREATE TABLE grades (
    id UUID PRIMARY KEY,                -- grade_id from event (for idempotency)
    course_id UUID NOT NULL,
    assignment_id UUID NOT NULL,
    student_id UUID NOT NULL,
    score INTEGER NOT NULL,
    max_score INTEGER NOT NULL,         -- Denormalized from assignment
    category VARCHAR(100),              -- Denormalized from assignment
    percentage DECIMAL(5,2) GENERATED ALWAYS AS ((score::DECIMAL / max_score) * 100) STORED,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Idempotency: Prevent duplicate event processing
    UNIQUE(id)
);

-- 4. STUDENT_ENROLLMENTS (track which students are in which courses)
-- Built from first GradeAssigned event or explicit enrollment events
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

-- === SEED DATA (for testing) ===

-- Courses (matching school service data)
INSERT INTO courses (id, title) VALUES
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'Advanced Algorithms'),
('c101f1ee-6c54-4b01-90e6-d701748f0851', 'Cryptography 101'),
('c200f1ee-6c54-4b01-90e6-d701748f0852', 'Operating Systems');

-- Assignments (matching teacher service data)
INSERT INTO assignments (id, course_id, title, category, max_score) VALUES
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'Midterm Exam', 'Exam', 100),
('a100f1ee-6c54-4b01-90e6-d701748f0002', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'Final Project', 'Project', 200),
('a200f1ee-6c54-4b01-90e6-d701748f0003', 'c200f1ee-6c54-4b01-90e6-d701748f0852', 'Lab Report 1', 'Lab', 50);

-- Grades (matching teacher service data)
INSERT INTO grades (id, course_id, assignment_id, student_id, score, max_score, category) VALUES
-- John Doe grades
('d100f1ee-6c54-4b01-90e6-d701748f0851', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'a100f1ee-6c54-4b01-90e6-d701748f0001', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 95, 100, 'Exam'),
('d100f1ee-6c54-4b01-90e6-d701748f0852', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'a100f1ee-6c54-4b01-90e6-d701748f0002', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 95, 200, 'Project'),
('d200f1ee-6c54-4b01-90e6-d701748f0853', 'c200f1ee-6c54-4b01-90e6-d701748f0852', 'a200f1ee-6c54-4b01-90e6-d701748f0003', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 45, 50, 'Lab'),
-- Jane Smith grades
('d100f1ee-6c54-4b01-90e6-d701748f0854', 'c100f1ee-6c54-4b01-90e6-d701748f0851', 'a100f1ee-6c54-4b01-90e6-d701748f0001', 'a888f1ee-6c54-4b01-90e6-d701748f0852', 88, 100, 'Exam');

-- Student Enrollments
INSERT INTO student_enrollments (course_id, student_id) VALUES
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'a999f1ee-6c54-4b01-90e6-d701748f0851'), -- John
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'a888f1ee-6c54-4b01-90e6-d701748f0852'), -- Jane
('c200f1ee-6c54-4b01-90e6-d701748f0852', 'a999f1ee-6c54-4b01-90e6-d701748f0851'); -- John in OS
