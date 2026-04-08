package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"gradebook/internal/models"
)

type DB struct {
	conn *sql.DB
}

func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite3", path+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1)
	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return d, nil
}

func (d *DB) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS users (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	login      TEXT UNIQUE NOT NULL,
	password   TEXT NOT NULL,
	name       TEXT NOT NULL,
	role       TEXT NOT NULL CHECK(role IN ('teacher','student','parent')),
	class      TEXT DEFAULT '',
	subject    TEXT DEFAULT '',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS parent_children (
	parent_id  INTEGER NOT NULL REFERENCES users(id),
	student_id INTEGER NOT NULL REFERENCES users(id),
	PRIMARY KEY (parent_id, student_id)
);

CREATE TABLE IF NOT EXISTS grades (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	student_id INTEGER NOT NULL REFERENCES users(id),
	teacher_id INTEGER NOT NULL REFERENCES users(id),
	date       TEXT NOT NULL,
	value      INTEGER NOT NULL CHECK(value BETWEEN 2 AND 5),
	comment    TEXT DEFAULT '',
	subject    TEXT DEFAULT 'Математика',
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(student_id, date, subject)
);

CREATE INDEX IF NOT EXISTS idx_grades_student ON grades(student_id);
CREATE INDEX IF NOT EXISTS idx_grades_date ON grades(date);
`
	_, err := d.conn.Exec(schema)
	return err
}

func (d *DB) SeedTeacher() error {
	var count int
	d.conn.QueryRow(`SELECT COUNT(*) FROM users WHERE role='teacher'`).Scan(&count)
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("БагиЛада"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(
		`INSERT INTO users (login, password, name, role, subject) VALUES (?,?,?,?,?)`,
		"Лада", string(hash), "Лада", "teacher", "Математика",
	)
	if err != nil {
		log.Printf("seed teacher: %v", err)
	}
	return err
}

func (d *DB) GetUserByLogin(login string) (*models.User, error) {
	u := &models.User{}
	err := d.conn.QueryRow(
		`SELECT id, login, password, name, role, class, subject, created_at FROM users WHERE login=?`, login,
	).Scan(&u.ID, &u.Login, &u.Password, &u.Name, &u.Role, &u.Class, &u.Subject, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (d *DB) GetUserByID(id int64) (*models.User, error) {
	u := &models.User{}
	err := d.conn.QueryRow(
		`SELECT id, login, password, name, role, class, subject, created_at FROM users WHERE id=?`, id,
	).Scan(&u.ID, &u.Login, &u.Password, &u.Name, &u.Role, &u.Class, &u.Subject, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return u, err
}

func (d *DB) CreateUser(req *models.RegisterRequest) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	res, err := d.conn.Exec(
		`INSERT INTO users (login, password, name, role, class, subject) VALUES (?,?,?,?,?,?)`,
		req.Login, string(hash), req.Name, req.Role, req.Class, req.Subject,
	)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return d.GetUserByID(id)
}

func (d *DB) CheckPassword(user *models.User, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

func (d *DB) GetStudents() ([]*models.User, error) {
	rows, err := d.conn.Query(
		`SELECT id, login, password, name, role, class, subject, created_at FROM users WHERE role='student' ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		rows.Scan(&u.ID, &u.Login, &u.Password, &u.Name, &u.Role, &u.Class, &u.Subject, &u.CreatedAt)
		users = append(users, u)
	}
	return users, nil
}

func (d *DB) GetStudentsForParent(parentID int64) ([]*models.User, error) {
	rows, err := d.conn.Query(
		`SELECT u.id, u.login, u.password, u.name, u.role, u.class, u.subject, u.created_at
		 FROM users u JOIN parent_children pc ON u.id = pc.student_id
		 WHERE pc.parent_id = ?`, parentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*models.User
	for rows.Next() {
		u := &models.User{}
		rows.Scan(&u.ID, &u.Login, &u.Password, &u.Name, &u.Role, &u.Class, &u.Subject, &u.CreatedAt)
		users = append(users, u)
	}
	return users, nil
}

func (d *DB) LinkParentChild(parentID, studentID int64) error {
	_, err := d.conn.Exec(
		`INSERT OR IGNORE INTO parent_children (parent_id, student_id) VALUES (?,?)`,
		parentID, studentID,
	)
	return err
}

func (d *DB) SetGrade(teacherID int64, req *models.SetGradeRequest) (*models.Grade, error) {
	now := time.Now()
	_, err := d.conn.Exec(`
		INSERT INTO grades (student_id, teacher_id, date, value, comment, subject, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)
		ON CONFLICT(student_id, date, subject)
		DO UPDATE SET value=excluded.value, comment=excluded.comment,
		              teacher_id=excluded.teacher_id, updated_at=excluded.updated_at`,
		req.StudentID, teacherID, req.Date, req.Value, req.Comment,
		req.Subject, now, now,
	)
	if err != nil {
		return nil, err
	}
	g := &models.Grade{}
	d.conn.QueryRow(
		`SELECT id, student_id, teacher_id, date, value, comment, subject, created_at, updated_at
		 FROM grades WHERE student_id=? AND date=? AND subject=?`,
		req.StudentID, req.Date, req.Subject,
	).Scan(&g.ID, &g.StudentID, &g.TeacherID, &g.Date, &g.Value, &g.Comment, &g.Subject, &g.CreatedAt, &g.UpdatedAt)
	return g, nil
}

func (d *DB) DeleteGrade(teacherID, gradeID int64) error {
	_, err := d.conn.Exec(`DELETE FROM grades WHERE id=? AND teacher_id=?`, gradeID, teacherID)
	return err
}

func (d *DB) GetGradesByStudent(studentID int64) ([]*models.Grade, error) {
	rows, err := d.conn.Query(
		`SELECT id, student_id, teacher_id, date, value, comment, subject, created_at, updated_at
		 FROM grades WHERE student_id=? ORDER BY date DESC`, studentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var grades []*models.Grade
	for rows.Next() {
		g := &models.Grade{}
		rows.Scan(&g.ID, &g.StudentID, &g.TeacherID, &g.Date, &g.Value, &g.Comment, &g.Subject, &g.CreatedAt, &g.UpdatedAt)
		grades = append(grades, g)
	}
	return grades, nil
}

func (d *DB) GetGradesByDate(date string) ([]*models.GradeWithStudent, error) {
	rows, err := d.conn.Query(
		`SELECT g.id, g.student_id, g.teacher_id, g.date, g.value, g.comment, g.subject,
		        g.created_at, g.updated_at, u.name, u.login, u.class
		 FROM grades g JOIN users u ON g.student_id = u.id
		 WHERE g.date=? ORDER BY u.name`, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var grades []*models.GradeWithStudent
	for rows.Next() {
		g := &models.GradeWithStudent{}
		rows.Scan(&g.ID, &g.StudentID, &g.TeacherID, &g.Date, &g.Value, &g.Comment, &g.Subject,
			&g.CreatedAt, &g.UpdatedAt, &g.StudentName, &g.StudentLogin, &g.Class)
		grades = append(grades, g)
	}
	return grades, nil
}

func (d *DB) GetAllGrades() ([]*models.GradeWithStudent, error) {
	rows, err := d.conn.Query(
		`SELECT g.id, g.student_id, g.teacher_id, g.date, g.value, g.comment, g.subject,
		        g.created_at, g.updated_at, u.name, u.login, u.class
		 FROM grades g JOIN users u ON g.student_id = u.id
		 ORDER BY g.date DESC, u.name`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var grades []*models.GradeWithStudent
	for rows.Next() {
		g := &models.GradeWithStudent{}
		rows.Scan(&g.ID, &g.StudentID, &g.TeacherID, &g.Date, &g.Value, &g.Comment, &g.Subject,
			&g.CreatedAt, &g.UpdatedAt, &g.StudentName, &g.StudentLogin, &g.Class)
		grades = append(grades, g)
	}
	return grades, nil
}

func (d *DB) LoginExists(login string) bool {
	var c int
	d.conn.QueryRow(`SELECT COUNT(*) FROM users WHERE login=?`, login).Scan(&c)
	return c > 0
}
