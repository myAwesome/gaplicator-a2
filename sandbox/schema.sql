CREATE TABLE students (
    id SERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    email VARCHAR(255) UNIQUE
);

CREATE TABLE subjects (
    id SERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL
);

CREATE TABLE lessons (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    subject_id INT REFERENCES subjects(id)
);

CREATE TABLE attendance (
    id SERIAL PRIMARY KEY,
    lesson_id INT REFERENCES lessons(id),
    student_id INT REFERENCES students(id),
    present BOOLEAN DEFAULT FALSE
);
