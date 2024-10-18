package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type UlasanWhatsapp struct {
	IDUlasan string `json:"reviewID"`
	Isi      string `json:"content"`
	Skor     string `json:"score"`
}

type DBConfig struct {
	Username string
	Password string
	Host     string
	Port     string
	Database string
}

func getDBConfig() DBConfig {
	return DBConfig{
		Username: "root",          
		Password: "",              
		Host:     "127.0.0.1",     
		Port:     "3306",          
		Database: "app-review",    
	}
}

func buildConnectionString(config DBConfig) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
	)
}

func connectDB() (*sql.DB, error) {
	config := getDBConfig()
	connectionString := buildConnectionString(config)
	return sql.Open("mysql", connectionString)
}

// Fungsi untuk Basic Authentication
func checkBasicAuth(r *http.Request) bool {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Basic" {
		return false
	}

	payload, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}

	authPair := strings.SplitN(string(payload), ":", 2)
	if len(authPair) != 2 {
		return false
	}

	expectedUsername := "admin" 
	expectedPassword := "password123" 

	return authPair[0] == expectedUsername && authPair[1] == expectedPassword
}

// Fungsi untuk menghandle request HTTP
func handleRequest(w http.ResponseWriter, r *http.Request) {
	// Cek Basic Auth sebelum melanjutkan
	if !checkBasicAuth(r) {
		w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "GET":
		if r.URL.Path == "/" {
			handleRoot(w)
		} else if r.URL.Path == "/whatsapp" {
			handleWhatsappReviews(w, r)
		} else {
			handleNotFound(w)
		}
	case "POST":
		if r.URL.Path == "/whatsapp" {
			handleCreateReview(w, r)
		} else {
			handleNotFound(w)
		}
	case "PUT":
		if strings.HasPrefix(r.URL.Path, "/whatsapp/") {
			handleUpdateReview(w, r)
		} else {
			handleNotFound(w)
		}
	case "DELETE":
		if strings.HasPrefix(r.URL.Path, "/whatsapp/") {
			handleDeleteReview(w, r)
		} else {
			handleNotFound(w)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// Fungsi untuk menampilkan halaman utama
func handleRoot(w http.ResponseWriter) {
	fmt.Fprintf(w, "======= Selamat datang di API Ulasan WhatsApp ======\n")
	fmt.Fprintf(w, "Berikut adalah daftar API yang tersedia:\n")
	fmt.Fprintf(w, "/whatsapp (GET) - Menampilkan semua ulasan\n")
	fmt.Fprintf(w, "/whatsapp (POST) - Menambahkan ulasan baru\n")
	fmt.Fprintf(w, "/whatsapp/{id} (PUT) - Memperbarui ulasan berdasarkan ID\n")
	fmt.Fprintf(w, "/whatsapp/{id} (DELETE) - Menghapus ulasan berdasarkan ID\n")
}

func handleWhatsappReviews(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
	if err != nil {
		http.Error(w, "Koneksi database gagal", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	reviews, err := fetchWhatsappReviews(db)
	if err != nil {
		http.Error(w, "Gagal mengambil data ulasan", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, reviews)
}

func fetchWhatsappReviews(db *sql.DB) ([]UlasanWhatsapp, error) {
	rows, err := db.Query("SELECT * FROM whatsapp")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []UlasanWhatsapp
	for rows.Next() {
		var review UlasanWhatsapp
		if err := rows.Scan(&review.IDUlasan, &review.Isi, &review.Skor); err != nil {
			return nil, err
		}
		reviews = append(reviews, review)
	}
	return reviews, nil
}

// Fungsi untuk menambahkan ulasan (POST)
func handleCreateReview(w http.ResponseWriter, r *http.Request) {
	var newReview UlasanWhatsapp
	if err := json.NewDecoder(r.Body).Decode(&newReview); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	db, err := connectDB()
	if err != nil {
		http.Error(w, "Koneksi database gagal", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO whatsapp (reviewID, content, score) VALUES (?, ?, ?)", newReview.IDUlasan, newReview.Isi, newReview.Skor)
	if err != nil {
		http.Error(w, "Gagal menambah ulasan", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, map[string]string{"message": "Ulasan berhasil ditambahkan"})
}

// Fungsi untuk memperbarui ulasan (PUT)
func handleUpdateReview(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/whatsapp/")

	var updatedReview UlasanWhatsapp
	if err := json.NewDecoder(r.Body).Decode(&updatedReview); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	db, err := connectDB()
	if err != nil {
		http.Error(w, "Koneksi database gagal", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("UPDATE whatsapp SET content = ?, score = ? WHERE reviewID = ?", updatedReview.Isi, updatedReview.Skor, id)
	if err != nil {
		http.Error(w, "Gagal memperbarui ulasan", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, map[string]string{"message": "Ulasan berhasil diperbarui"})
}

// Fungsi untuk menghapus ulasan (DELETE)
func handleDeleteReview(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/whatsapp/")

	db, err := connectDB()
	if err != nil {
		http.Error(w, "Koneksi database gagal", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM whatsapp WHERE reviewID = ?", id)
	if err != nil {
		http.Error(w, "Gagal menghapus ulasan", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, map[string]string{"message": "Ulasan berhasil dihapus"})
}

func handleNotFound(w http.ResponseWriter) {
	http.Error(w, "Not found", http.StatusNotFound)
}

func respondWithJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func main() {
	http.HandleFunc("/", handleRequest)

	fmt.Println("Server Berjalan di http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
