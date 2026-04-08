package models

import "time"

type Role string

const (
	RoleTeacher Role = "teacher"
	RoleStudent Role = "student"
	RoleParent  Role = "parent"
)

type User struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	Password  string    `json:"-"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	Class     string    `json:"class,omitempty"`
	Subject   string    `json:"subject,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ParentChild struct {
	ParentID  int64 `json:"parent_id"`
	StudentID int64 `json:"student_id"`
}

type Grade struct {
	ID        int64     `json:"id"`
	StudentID int64     `json:"student_id"`
	TeacherID int64     `json:"teacher_id"`
	Date      string    `json:"date"` // YYYY-MM-DD
	Value     int       `json:"value"`
	Comment   string    `json:"comment"`
	Subject   string    `json:"subject"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GradeWithStudent struct {
	Grade
	StudentName  string `json:"student_name"`
	StudentLogin string `json:"student_login"`
	Class        string `json:"class"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     Role   `json:"role"`
	Class    string `json:"class,omitempty"`
	Subject  string `json:"subject,omitempty"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type SetGradeRequest struct {
	StudentID int64  `json:"student_id"`
	Date      string `json:"date"`
	Value     int    `json:"value"`
	Comment   string `json:"comment"`
	Subject   string `json:"subject"`
}

type ScheduleDay struct {
	Date    string `json:"date"`
	Weekday string `json:"weekday"`
	Label   string `json:"label"`
}
