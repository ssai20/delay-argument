package api

import (
	"database/sql"
	_ "github.com/lib/pq"
	"delay-argument-go/cmd/db"
	"delay-argument-go/internal/calculator"
	"delay-argument-go/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type CalculationResponse struct {
	JobID    string     `json:"job_id"`
	Status   string     `json:"status"`
	PDFURL   string     `json:"pdf_url,omitempty"`
	Error    string     `json:"error,omitempty"`
	Classic  [][]string `json:"classic,omitempty"`
	Modified [][]string `json:"modified,omitempty"`
}

// Хранилище для отслеживания заданий
var jobs = make(map[string]*calculator.Job)

// var counter int64
var (
	counter int64
	mu      sync.RWMutex
)

func (s *Server) calculateHandler(resultsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req models.CalculationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Валидация
		if req.EpsilonStart <= 0 {
			req.EpsilonStart = 1.0
		}
		if req.EpsilonMin <= 0 {
			req.EpsilonMin = 1e-8
		}
		if req.NStart < 128 {
			req.NStart = 128
		}
		if req.NMax < req.NStart {
			req.NMax = req.NStart * 2
		}
		if req.MeshType == "" {
			req.MeshType = "uniform"
		}

		// Создаем задание
		job := calculator.NewJob(req, resultsDir)
		jobs[job.ID] = job

		// Запускаем вычисления в фоне
		go calculator.RunJob(job)

		// Возвращаем ID задания
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(CalculationResponse{
			JobID:  job.ID,
			Status: "processing",
		})
		//atomic.AddInt64(&counter, 1)
		//mu.Lock()
		//counter++
		//mu.Unlock()

		err := s.saveCounter()
		if err != nil {
			fmt.Errorf("%w", err)
		}

	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Path[len("/api/status/"):]
	job, exists := jobs[jobID]

	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	resp := CalculationResponse{
		JobID:  job.ID,
		Status: job.Status,
	}

	if job.Status == "completed" && job.PDFPath != "" {
		resp.PDFURL = "/results/" + jobID + ".pdf"
		resp.Classic = job.Classic
		resp.Modified = job.Modified
	} else if job.Status == "failed" {
		resp.Error = job.Error
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func meshTypesHandler(w http.ResponseWriter, r *http.Request) {
	types := []map[string]string{
		{"id": "uniform", "name": "Равномерная сетка", "description": "Обычная равномерная сетка"},
		{"id": "shishkin", "name": "Сетка Шишкина", "description": "Адаптивная сетка для пограничного слоя"},
		{"id": "bakhvalov", "name": "Сетка Бахвалова", "description": "Специальная сетка Бахвалова"},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(types)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy"}`))
}

func (s *Server) SetupRoutes(resultsDir string) *http.ServeMux {
	router := http.NewServeMux()

	// 👇 СПЕЦИАЛЬНЫЙ ОБРАБОТЧИК для статических файлов с правильными MIME-типами
	router.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		// Убираем префикс /static/
		filePath := "." + r.URL.Path

		// Определяем MIME-тип по расширению
		if strings.HasSuffix(filePath, ".css") {
			w.Header().Set("Content-Type", "text/css")
		} else if strings.HasSuffix(filePath, ".js") {
			w.Header().Set("Content-Type", "application/javascript")
		} else if strings.HasSuffix(filePath, ".html") {
			w.Header().Set("Content-Type", "text/html")
		}

		http.ServeFile(w, r, filePath)
	})

	router.HandleFunc("/", basicAuth(homeHandler))

	// Статические файлы (PDF результаты)
	router.Handle("/results/", http.StripPrefix("/results/",
		http.FileServer(http.Dir(resultsDir))))

	// Добавить прямой download URL
	router.HandleFunc("/download/", downloadHandler(resultsDir))

	// API endpoints
	router.HandleFunc("/api/calculate", s.calculateHandler(resultsDir))
	router.HandleFunc("/api/status/", statusHandler)
	router.HandleFunc("/api/mesh-types", meshTypesHandler)
	router.HandleFunc("/health", healthHandler)
	router.HandleFunc("/metrics", s.calculateCount)

	return router
}

// Middleware для Basic Auth
func basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем логин и пароль из заголовка
		username, password, ok := r.BasicAuth()

		// Проверяем (можно вынести в БД или конфиг)
		if !ok || username != "zadorin" || password != "ofim10soran" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Если всё ок - идём дальше
		next(w, r)
	}
}

func downloadHandler(resultsDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		filename := strings.TrimPrefix(r.URL.Path, "/download/")
		if !strings.HasSuffix(filename, ".pdf") {
			filename += ".pdf"
		}

		filepath := filepath.Join(resultsDir, filename)

		// Убеждаемся, что файл существует
		if _, err := os.Stat(filepath); os.IsNotExist(err) {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}

		// Отдаем файл
		w.Header().Set("Content-Disposition", "attachment; filename="+filename)
		w.Header().Set("Content-Type", "application/pdf")
		http.ServeFile(w, r, filepath)
	}
}

type Server struct {
	db     *sql.DB
	router *http.ServeMux
}

func NewServer(config *db.Config) *Server {

	dB, err := initDB(config)
	if err != nil {
		fmt.Errorf("new database initiation failed: %w", err)
	}
	return &Server{
		db:     dB,
		router: http.NewServeMux(),
	}
}

func (s *Server) calculateCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	//json.NewEncoder(w).Encode(atomic.LoadInt64(&counter))
	//mu.RLock()
	//c := counter
	//mu.RUnlock()
	c, err := s.getCounter()
	if err != nil {
		fmt.Errorf("%w", err)
	}
	json.NewEncoder(w).Encode(c)
}

func initDB(config *db.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", config.DBUser, config.DBPassword, config.DBHost, config.DBPort, config.DBName, config.DBSSLMode)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to opern database: %w", err)
	}

	if err := createTable(db); err != nil {
		return nil, err
	}
	return db, nil
}

func createTable(db *sql.DB) error {
	query := `
CREATE TABLE IF NOT EXISTS counter_ids (
    id SERIAL PRIMARY KEY,
    counter VARCHAR(100) UNIQUE NOT NULL
);
`
	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}
	return nil
}

func (s *Server) saveCounter() error {
	query := `
UPDATE counter_ids
SET counter = counter + 1
WHERE id =1
`
	_, err := s.db.Exec(query)
	if err != nil {
		return fmt.Errorf("counter can't save: %w", err)
	}
	return nil
}

func (s *Server) getCounter() (int64, error) {
	query := `
SELECT counter
FROM counter_ids 
WHERE id = 1
`
	err := s.db.QueryRow(query).Scan(&counter)
	if err != nil {
		return 0, fmt.Errorf("counter can't getting: %w", err)
	}
	return counter, nil
}

// Обновите homeHandler для правильного MIME-типа HTML
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	http.ServeFile(w, r, "./static/index.html")
}
