CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. STUDENTS (The "User" for this service)
CREATE TABLE students (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(100),
    student_number VARCHAR(20) UNIQUE, -- e.g., "STD-2026-001"
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Seed dummy data
INSERT INTO students (id, email, password_hash, full_name, student_number) VALUES 
('a999f1ee-6c54-4b01-90e6-d701748f0851', 'john@student.edu', 'secret', 'John Doe', 'STD-2026-001'),
('a888f1ee-6c54-4b01-90e6-d701748f0852', 'jane@student.edu', 'secret', 'Jane Smith', 'STD-2026-002'),
('a777f1ee-6c54-4b01-90e6-d701748f0853', 'bob@student.edu', 'secret', 'Bob Martin', 'STD-2026-003'),
('a666f1ee-6c54-4b01-90e6-d701748f0854', 'alice@student.edu', 'secret', 'Alice Wonderland', 'STD-2026-004');
