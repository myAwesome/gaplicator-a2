package models

import (
	"time"

	"gorm.io/gorm"
)

type Student struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	FirstName            string       `gorm:"column:first_name;size:100;not null" json:"first_name"`
	LastName             string       `gorm:"column:last_name;size:100;not null" json:"last_name"`
	Email                string       `gorm:"column:email;size:255;uniqueIndex" json:"email"`
}

type Subject struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Name                 string       `gorm:"column:name;size:200;not null" json:"name"`
}

type Lesson struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	Date                 time.Time    `gorm:"column:date;not null" json:"date"`
	SubjectID            int          `gorm:"column:subject_id" json:"subject_id"`
	Subject              Subject      `gorm:"foreignKey:SubjectID" json:"-"`
}

type Attendance struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	LessonID             int          `gorm:"column:lesson_id" json:"lesson_id"`
	Lesson               Lesson       `gorm:"foreignKey:LessonID" json:"-"`
	StudentID            int          `gorm:"column:student_id" json:"student_id"`
	Student              Student      `gorm:"foreignKey:StudentID" json:"-"`
	Present              bool         `gorm:"column:present;default:FALSE" json:"present"`
}
