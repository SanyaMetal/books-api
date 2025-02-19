package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type Book struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
}

var db *sql.DB

func createBookHandler(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	author := r.FormValue("author")
	description := r.FormValue("description")

	if title == "" || author == "" {
		http.Error(w, "title и author обязательны", http.StatusBadRequest)
		return
	}

	var newID int

	err := db.QueryRow(`
		INSERT INTO books (title, author, description)
		VALUES ($1, $2, $3)
		RETURNING id
	`, title, author, description).Scan(&newID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка создания записи: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "Книга создана с ID: %d\n", newID)
}

func getAllBookHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, title, author, description FROM books")
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка при получении книг: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var books []Book

	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.Description); err != nil {
			http.Error(w, fmt.Sprintf("Ошибка при чтении строки: %v", err), http.StatusInternalServerError)
			return
		}
		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка итерирования по строкам: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "[\n")
	for i, book := range books {
		fmt.Fprintf(w, "  {\n")
		fmt.Fprintf(w, "    \"id\": %d, \n", book.ID)
		fmt.Fprintf(w, "    \"title\": \"%s\",\n", book.Title)
		fmt.Fprintf(w, "    \"author\": \"%s\",\n", book.Author)
		fmt.Fprintf(w, "    \"description\": \"%s\"\n", book.Description)
		if i < len(books)-1 {
			fmt.Printf(w, "  },\n")
		} else {
			fmt.Fprintf(w, "  }\n")
		}
	}
	fmt.Fprintf(w, "]\n")
}

func getBookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Wrong ID", http.StatusBadRequest)
		return
	}

	var book Book
	err = db.QueryRow(`
	SELECT id, title, author, description FROM books WHERE id = $1`, id).Scan(&book.ID, &book.Author, &book.Title, &book.Description)

	if err == sql.ErrNoRows {
		http.Error(w, "Книга не найдена ", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка получаения книги: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "  {\n")
	fmt.Fprintf(w, "    \"id\": %d, \n", book.ID)
	fmt.Fprintf(w, "    \"title\": \"%s\",\n", book.Title)
	fmt.Fprintf(w, "    \"author\": \"%s\",\n", book.Author)
	fmt.Fprintf(w, "    \"description\": \"%s\"\n", book.Description)
	fmt.Fprintf(w, "  }\n")
}

func updateBookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Wrong ID", http.StatusBadRequest)
		return
	}

	title := r.FormValue("title")
	author := r.FormValue("author")
	description := r.FormValue("description")

	if title == "" || author == "" {
		http.Error(w, "title и author обязательны", http.StatusBadRequest)
		return
	}

	res, err := db.Exec(`
	UPDATE books
	SET title = $1, author = $2, description = $3
	WHERE id = $4	
	`, title, author, description, id)

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка при проверке изменения строк: %v", err), http.StatusInternalServerError)
		return
	}

	if rowsAffected == 0 {
		http.Error(w, "Книга не найдена!", http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, "Книга с ID %d обновлена \n", id)
}

func deleteBookHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Wrong ID", http.StatusBadRequest)
		return
	}

	res, err := db.Exec(`DELETE FROM books WHERE id = $1`, id)

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка при попытке изменения строк: %v", err), http.StatusInternalServerError)
		return
	}
	if rowsAffected == 0 {
		http.Error(w, "Книга не найдена!", http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "Книга с Id %d удалена", id)
}

func main() {
	dbUser := "user"
	dbPassword := "password"
	dbName := "bookdb"
	dbHost := "localhost"
	dbPort := "5433"

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbPort, dbUser, dbPassword, dbName)

	var errOpen error

	db, errOpen = sql.Open("postgres", psqlInfo)
	if errOpen != nil {
		log.Fatalf("Не удалось подключиться к БД: %v", errOpen)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Не удалось выполнить ping к БД: %v", err)
	}
	log.Println("Успешное подключение к БД!")

	router := mux.NewRouter()

	router.HandleFunc("/books", createBookHandler).Methods("POST")
	router.HandleFunc("/books", getAllBookHandler).Methods("GET")
	router.HandleFunc("/books/{id}", getBookHandler).Methods("GET")
	router.HandleFunc("/books/{id}", updateBookHandler).Methods("PUT")
	router.HandleFunc("/books/{id}", deleteBookHandler).Methods("DELETE")

	port := "8080"
	log.Printf("Сервер запущен на порту %s", port)

	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatalf("Ошибка при запуске сервера: %v", err)
	}

}
