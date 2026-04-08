package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"gradebook/internal/auth"
	"gradebook/internal/db"
	"gradebook/internal/models"
)

type Handler struct {
	db *db.DB
}

func New(d *db.DB) *Handler {
	return &Handler{db: d}
}

func respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func ok(w http.ResponseWriter, data interface{}) {
	respond(w, http.StatusOK, models.APIResponse{Success: true, Data: data})
}

func fail(w http.ResponseWriter, status int, msg string) {
	respond(w, status, models.APIResponse{Success: false, Error: msg})
}

// POST /api/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, 400, "invalid request"); return
	}
	user, err := h.db.GetUserByLogin(req.Login)
	if err != nil || user == nil || !h.db.CheckPassword(user, req.Password) {
		fail(w, 401, "неверный логин или пароль"); return
	}
	token, err := auth.GenerateToken(user.ID, user.Login, string(user.Role))
	if err != nil {
		fail(w, 500, "token error"); return
	}
	user.Password = ""
	ok(w, &models.AuthResponse{Token: token, User: user})
}

// POST /api/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, 400, "invalid request"); return
	}
	if req.Login == "" || req.Password == "" || req.Name == "" {
		fail(w, 400, "поля login, password, name обязательны"); return
	}
	if len(req.Password) < 4 {
		fail(w, 400, "пароль минимум 4 символа"); return
	}
	if req.Role == "" {
		fail(w, 400, "роль обязательна"); return
	}
	if req.Role == models.RoleTeacher {
		fail(w, 403, "регистрация учителей запрещена"); return
	}
	if h.db.LoginExists(req.Login) {
		fail(w, 409, "логин уже занят"); return
	}
	user, err := h.db.CreateUser(&req)
	if err != nil {
		fail(w, 500, "ошибка создания пользователя"); return
	}
	token, _ := auth.GenerateToken(user.ID, user.Login, string(user.Role))
	user.Password = ""
	ok(w, &models.AuthResponse{Token: token, User: user})
}

// GET /api/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	user, err := h.db.GetUserByID(claims.UserID)
	if err != nil || user == nil {
		fail(w, 404, "user not found"); return
	}
	user.Password = ""
	ok(w, user)
}

// GET /api/students  [teacher only]
func (h *Handler) GetStudents(w http.ResponseWriter, r *http.Request) {
	students, err := h.db.GetStudents()
	if err != nil {
		fail(w, 500, err.Error()); return
	}
	for _, s := range students {
		s.Password = ""
	}
	ok(w, students)
}

// GET /api/grades?date=YYYY-MM-DD  [teacher]
func (h *Handler) GetGradesByDate(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		fail(w, 400, "date required"); return
	}
	grades, err := h.db.GetGradesByDate(date)
	if err != nil {
		fail(w, 500, err.Error()); return
	}
	ok(w, grades)
}

// GET /api/grades/all  [teacher]
func (h *Handler) GetAllGrades(w http.ResponseWriter, r *http.Request) {
	grades, err := h.db.GetAllGrades()
	if err != nil {
		fail(w, 500, err.Error()); return
	}
	ok(w, grades)
}

// POST /api/grades  [teacher only]
func (h *Handler) SetGrade(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	var req models.SetGradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, 400, "invalid request"); return
	}
	if req.Value < 2 || req.Value > 5 {
		fail(w, 400, "оценка должна быть от 2 до 5"); return
	}
	if req.Subject == "" {
		req.Subject = "Математика"
	}
	grade, err := h.db.SetGrade(claims.UserID, &req)
	if err != nil {
		fail(w, 500, err.Error()); return
	}
	ok(w, grade)
}

// DELETE /api/grades/{id}  [teacher only]
func (h *Handler) DeleteGrade(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	vars := mux.Vars(r)
	id, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		fail(w, 400, "invalid id"); return
	}
	if err := h.db.DeleteGrade(claims.UserID, id); err != nil {
		fail(w, 500, err.Error()); return
	}
	ok(w, map[string]bool{"deleted": true})
}

// GET /api/my-grades  [student]
func (h *Handler) MyGrades(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	grades, err := h.db.GetGradesByStudent(claims.UserID)
	if err != nil {
		fail(w, 500, err.Error()); return
	}
	ok(w, grades)
}

// GET /api/children  [parent]
func (h *Handler) GetChildren(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	children, err := h.db.GetStudentsForParent(claims.UserID)
	if err != nil {
		fail(w, 500, err.Error()); return
	}
	for _, c := range children {
		c.Password = ""
	}
	ok(w, children)
}

// GET /api/child-grades/{studentId}  [parent]
func (h *Handler) ChildGrades(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	vars := mux.Vars(r)
	studentID, _ := strconv.ParseInt(vars["studentId"], 10, 64)

	// verify parent-child relationship
	children, err := h.db.GetStudentsForParent(claims.UserID)
	if err != nil {
		fail(w, 500, err.Error()); return
	}
	allowed := false
	for _, c := range children {
		if c.ID == studentID {
			allowed = true; break
		}
	}
	if !allowed {
		fail(w, 403, "forbidden"); return
	}
	grades, err := h.db.GetGradesByStudent(studentID)
	if err != nil {
		fail(w, 500, err.Error()); return
	}
	ok(w, grades)
}

// POST /api/link-child  [parent - link to child by login]
func (h *Handler) LinkChild(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r)
	var req struct {
		StudentLogin string `json:"student_login"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	student, err := h.db.GetUserByLogin(req.StudentLogin)
	if err != nil || student == nil || student.Role != "student" {
		fail(w, 404, "ученик не найден"); return
	}
	if err := h.db.LinkParentChild(claims.UserID, student.ID); err != nil {
		fail(w, 500, err.Error()); return
	}
	student.Password = ""
	ok(w, student)
}

// GET /api/schedule
func (h *Handler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	days := generateSchedule()
	ok(w, days)
}

func generateSchedule() []models.ScheduleDay {
	start := time.Date(2025, 4, 7, 0, 0, 0, 0, time.UTC)
	weekdays := map[time.Weekday]bool{
		time.Tuesday:  true,
		time.Thursday: true,
		time.Saturday: true,
	}
	dayNames := map[time.Weekday]string{
		time.Tuesday:  "Вт",
		time.Thursday: "Чт",
		time.Saturday: "Сб",
	}
	months := []string{"янв", "фев", "мар", "апр", "май", "июн", "июл", "авг", "сен", "окт", "ноя", "дек"}

	var schedule []models.ScheduleDay
	d := start
	for len(schedule) < 20 {
		if weekdays[d.Weekday()] {
			schedule = append(schedule, models.ScheduleDay{
				Date:    d.Format("2006-01-02"),
				Weekday: dayNames[d.Weekday()],
				Label:   strconv.Itoa(d.Day()) + " " + months[d.Month()-1],
			})
		}
		d = d.AddDate(0, 0, 1)
	}
	return schedule
}

func init() {
	// make sure strconv is used
	_ = strconv.Itoa(0)
}
