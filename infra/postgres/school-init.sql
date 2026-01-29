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

-- === SEED DATA ===

-- Schools
INSERT INTO schools (id, name, address) VALUES
('b100f1ee-6c54-4b01-90e6-d701748f0001', 'Greenwood Academy', '123 Elm Street, Springfield'),
('b200f1ee-6c54-4b01-90e6-d701748f0002', 'Riverside High School', '456 Oak Avenue, Shelbyville');

-- Admins
INSERT INTO admins (id, email, password_hash, full_name, school_id) VALUES
('a100f1ee-6c54-4b01-90e6-d701748f0001', 'admin@greenwood.edu', 'secret', 'Alice Greenwood', 'b100f1ee-6c54-4b01-90e6-d701748f0001'),
('a200f1ee-6c54-4b01-90e6-d701748f0002', 'admin@riverside.edu', 'secret', 'Bob Riverside', 'b200f1ee-6c54-4b01-90e6-d701748f0002');

-- Classes
INSERT INTO classes (id, school_id, name, grade_level) VALUES
('cl00f1ee-6c54-4b01-90e6-d701748f0001', 'b100f1ee-6c54-4b01-90e6-d701748f0001', 'Mathematics A', '10th Grade'),
('cl00f1ee-6c54-4b01-90e6-d701748f0002', 'b100f1ee-6c54-4b01-90e6-d701748f0001', 'Science B', '11th Grade'),
('cl00f1ee-6c54-4b01-90e6-d701748f0003', 'b200f1ee-6c54-4b01-90e6-d701748f0002', 'English Literature', '12th Grade');
