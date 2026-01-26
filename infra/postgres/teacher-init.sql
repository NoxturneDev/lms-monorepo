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

-- 3. GRADES (The "Link" to the outside world)
CREATE TABLE grades (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    course_id UUID NOT NULL,
    
    -- THE MICROSERVICES LINK:
    -- This ID belongs to the Student Service. We just store it as text/uuid here.
    -- No "REFERENCES students(id)" allowed!
    student_id UUID NOT NULL, 
    
    score INTEGER CHECK (score >= 0 AND score <= 100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_course FOREIGN KEY (course_id) REFERENCES courses(id)
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

-- Grades (Pre-assigning some grades to Students we are about to create)
-- Student 1 (John) got 95 in Algorithms
INSERT INTO grades (course_id, student_id, score) VALUES
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 95),
('c100f1ee-6c54-4b01-90e6-d701748f0851', 'a888f1ee-6c54-4b01-90e6-d701748f0852', 88),
('c200f1ee-6c54-4b01-90e6-d701748f0852', 'a999f1ee-6c54-4b01-90e6-d701748f0851', 100);
