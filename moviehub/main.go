
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
	// 1. โหลดค่าจากไฟล์ .env ก่อนใช้งานตัวแปร
	if err := godotenv.Load(); err != nil {
		log.Fatal("โหลดไฟล์ .env ไม่สำเร็จ: ", err)
	}

	// 2. อ่านค่าตัวแปรจาก .env
	secret := os.Getenv("JWT_SECRET")
	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	allowedOrigins := strings.TrimSpace(os.Getenv("ALLOWED_ORIGINS"))

	// 3. ตรวจสอบ DATABASE_URL
	if dbURL == "" {
		log.Fatal("DATABASE_URL ว่าง กรุณาตรวจสอบไฟล์ .env")
	}

	if !strings.HasPrefix(dbURL, "postgres://") &&
		!strings.HasPrefix(dbURL, "postgresql://") {
		log.Fatal("DATABASE_URL ต้องเป็น PostgreSQL connection string")
	}

	// 4. ตรวจสอบ JWT_SECRET
	if len(secret) < 32 {
		log.Fatal("JWT_SECRET ต้องยาวอย่างน้อย 32 ตัวอักษร")
	}

	// 5. สร้างการเชื่อมต่อฐานข้อมูล Supabase
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatal("รูปแบบ DATABASE_URL ไม่ถูกต้อง: ", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatal("สร้างการเชื่อมต่อฐานข้อมูลไม่สำเร็จ: ", err)
	}
	defer pool.Close()

	// 6. ทดสอบการเชื่อมต่อฐานข้อมูล
	if err := pool.Ping(ctx); err != nil {
		log.Fatal("ต่อฐานข้อมูลไม่ติด กรุณาตรวจสอบ DATABASE_URL และการตั้งค่า Supabase: ", err)
	}

	log.Println("เชื่อมต่อฐานข้อมูลสำเร็จ")

	// 7. ตั้งค่า Handler
	tickets := &auth.Issuer{
		Secret: []byte(secret),
		TTL:    time.Hour,
	}

	h := &handler.Handler{
		Store:   &store.Store{Pool: pool},
		Tickets: tickets,
	}

	// 8. ตั้งค่า API Routes
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/movies", h.SearchMovies)
	mux.HandleFunc("GET /api/movies/{id}/reviews", h.ListReviews)
    mux.HandleFunc("GET /api/movies/{id}", h.GetMovie)

	// 9. ตั้งค่า CORS
	origins := []string{}

	if allowedOrigins != "" {
		for _, origin := range strings.Split(allowedOrigins, ",") {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				origins = append(origins, origin)
			}
		}
	}

	// 10. เริ่มทำงาน Backend
	log.Println("MovieHub API ฟังอยู่ที่ :8080")
	log.Println("Allowed origins:", origins)

	if err := http.ListenAndServe(
		":8080",
		mw.Logging(mw.CORS(origins, mux)),
	); err != nil {
		log.Fatal("เริ่มเซิร์ฟเวอร์ไม่สำเร็จ: ", err)
	}
}
