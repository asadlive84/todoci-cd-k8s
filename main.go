package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	// _ "docs/docs.go" // Change this to your docs path
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// @title Todo API
// @version 1.0
// @description This is a simple Todo API server.
// @host localhost:8080
// @BasePath /api

type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type App struct {
	DB *sql.DB
}

func seedSampleData(db *sql.DB) {
	// Check if there are already records
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM todos").Scan(&count)
	if err != nil {
		log.Fatal(err)
	}

	// Insert sample data only if table is empty
	if count == 0 {
		_, err := db.Exec(`
			INSERT INTO todos (title, completed) VALUES
			('Learn Go', false),
			('Build REST API', false),
			('Write Documentation', true)
		`)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Sample data inserted into todos table")
	}
}

// @Summary Get all todos
// @Description Get all todos from the database
// @Tags todos
// @Produce json
// @Success 200 {array} Todo
// @Failure 500 {object} map[string]string
// @Router /api/todos [get]
func (app *App) getTodos(w http.ResponseWriter, r *http.Request) {
	rows, err := app.DB.Query("SELECT id, title, completed, created_at FROM todos ORDER BY created_at DESC")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		if err := rows.Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
		todos = append(todos, t)
	}

	respondWithJSON(w, http.StatusOK, todos)
}

// @Summary Get a todo
// @Description Get a todo by ID
// @Tags todos
// @Param id path int true "Todo ID"
// @Produce json
// @Success 200 {object} Todo
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/todos/{id} [get]
func (app *App) getTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var t Todo

	err := app.DB.QueryRow("SELECT id, title, completed, created_at FROM todos WHERE id = $1",
		vars["id"]).Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Todo not found")
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, t)
}

// @Summary Create a todo
// @Description Create a new todo
// @Tags todos
// @Accept json
// @Produce json
// @Param todo body Todo true "Todo to create"
// @Success 201 {object} Todo
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/todos [post]
func (app *App) createTodo(w http.ResponseWriter, r *http.Request) {
	var t Todo
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&t); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	err := app.DB.QueryRow(
		"INSERT INTO todos (title) VALUES ($1) RETURNING id, title, completed, created_at",
		t.Title,
	).Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, t)
}

// @Summary Update a todo
// @Description Update a todo by ID
// @Tags todos
// @Accept json
// @Produce json
// @Param id path int true "Todo ID"
// @Param todo body Todo true "Updated todo"
// @Success 200 {object} Todo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/todos/{id} [put]
func (app *App) updateTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var t Todo
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&t); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	err := app.DB.QueryRow(
		"UPDATE todos SET title=$1, completed=$2 WHERE id=$3 RETURNING id, title, completed, created_at",
		t.Title, t.Completed, vars["id"],
	).Scan(&t.ID, &t.Title, &t.Completed, &t.CreatedAt)

	if err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Todo not found")
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, t)
}

// @Summary Delete a todo
// @Description Delete a todo by ID
// @Tags todos
// @Param id path int true "Todo ID"
// @Success 204
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/todos/{id} [delete]
func (app *App) deleteTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	result, err := app.DB.Exec("DELETE FROM todos WHERE id = $1", vars["id"])
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if rowsAffected == 0 {
		respondWithError(w, http.StatusNotFound, "Todo not found")
		return
	}

	respondWithJSON(w, http.StatusNoContent, nil)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	db, err := sql.Open("postgres", "postgres://postgres:password@postgres/todo?sslmode=disable")

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create todos table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS todos (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			completed BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Insert sample data if the table is empty
	seedSampleData(db)

	app := &App{DB: db}
	router := mux.NewRouter()

	router.HandleFunc("/api/todos", app.getTodos).Methods("GET")
	router.HandleFunc("/api/todos/{id}", app.getTodo).Methods("GET")
	router.HandleFunc("/api/todos", app.createTodo).Methods("POST")
	router.HandleFunc("/api/todos/{id}", app.updateTodo).Methods("PUT")
	router.HandleFunc("/api/todos/{id}", app.deleteTodo).Methods("DELETE")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
