
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"moviehub/auth"
	"moviehub/handler"
	"moviehub/mw"
	"moviehub/store"
)

func main() {
	// โหลดค่าจากไฟล์ .env
	if err := godotenv.Load(); err != nil {
		log.Println("ไม่พบไฟล์ .env จะใช้ Environment Variables ของระบบแทน")
	}

	// ตรวจสอบ JWT_SECRET
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET ต้องยาวอย่างน้อย 32 ตัวอักษร กรุณาตรวจสอบไฟล์ .env")
	}

	// ตรวจสอบ DATABASE_URL
	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		log.Fatal("ไม่พบ DATABASE_URL กรุณาตรวจสอบไฟล์ .env")
	}

	// เชื่อมต่อฐานข้อมูล
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatal("สร้างการเชื่อมต่อฐานข้อมูลไม่สำเร็จ: ", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("ต่อฐานข้อมูลไม่ติด: ", err)
	}

	// สร้างระบบ Token
	tickets := &auth.Issuer{
		Secret: []byte(secret),
		TTL:    time.Hour,
	}

	h := &handler.Handler{
		Store:   &store.Store{Pool: pool},
		Tickets: tickets,
	}

	// กำหนด API Routes
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/movies", h.SearchMovies)
	mux.HandleFunc("GET /api/movies/{id}", h.GetMovie)
	mux.HandleFunc("GET /api/movies/{id}/reviews", h.ListReviews)

	// ตั้งค่า CORS
	origins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	// ตั้งค่าพอร์ต
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("MovieHub API ฟังอยู่ที่ :" + port)
	log.Fatal(http.ListenAndServe(
		":"+port,
		mw.Logging(mw.CORS(origins, mux)),
	))
}
