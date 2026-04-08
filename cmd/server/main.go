package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"

	"gradebook/internal/auth"
	"gradebook/internal/db"
	"gradebook/internal/handlers"
)

//go:embed frontend
var frontendFS embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./gradebook.db"
	}

	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("failed to init db: %v", err)
	}

	if err := database.SeedTeacher(); err != nil {
		log.Printf("seed warning: %v", err)
	}

	h := handlers.New(database)
	r := mux.NewRouter()

	// CORS
	r.Use(corsMiddleware)

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Public
	api.HandleFunc("/auth/login", h.Login).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/register", h.Register).Methods("POST", "OPTIONS")
	api.HandleFunc("/schedule", h.GetSchedule).Methods("GET", "OPTIONS")

	// Protected
	protected := api.NewRoute().Subrouter()
	protected.Use(auth.Middleware)

	protected.HandleFunc("/me", h.Me).Methods("GET")

	// Teacher routes
	teacher := protected.NewRoute().Subrouter()
	teacher.Use(func(next http.Handler) http.Handler {
		return auth.RequireRole("teacher", next)
	})
	teacher.HandleFunc("/students", h.GetStudents).Methods("GET")
	teacher.HandleFunc("/grades", h.GetGradesByDate).Methods("GET")
	teacher.HandleFunc("/grades/all", h.GetAllGrades).Methods("GET")
	teacher.HandleFunc("/grades", h.SetGrade).Methods("POST")
	teacher.HandleFunc("/grades/{id}", h.DeleteGrade).Methods("DELETE")

	// Student routes
	student := protected.NewRoute().Subrouter()
	student.Use(func(next http.Handler) http.Handler {
		return auth.RequireRole("student", next)
	})
	student.HandleFunc("/my-grades", h.MyGrades).Methods("GET")

	// Parent routes
	parent := protected.NewRoute().Subrouter()
	parent.Use(func(next http.Handler) http.Handler {
		return auth.RequireRole("parent", next)
	})
	parent.HandleFunc("/children", h.GetChildren).Methods("GET")
	parent.HandleFunc("/child-grades/{studentId}", h.ChildGrades).Methods("GET")
	parent.HandleFunc("/link-child", h.LinkChild).Methods("POST")

	// Frontend static files
	frontendSub, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		log.Fatal(err)
	}
	fileServer := http.FileServer(http.FS(frontendSub))

	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// SPA routing
		path := req.URL.Path
		if !strings.HasPrefix(path, "/api") {
			// Try to serve the file, fallback to index.html for SPA routes
			if path == "/" || path == "" {
				req.URL.Path = "/login.html"
			}
		}
		fileServer.ServeHTTP(w, req)
	})

	log.Printf("🚀 GradeBook сервер запущен на http://localhost:%s", port)
	log.Printf("   Учитель: логин=Лада  пароль=БагиЛада")
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
