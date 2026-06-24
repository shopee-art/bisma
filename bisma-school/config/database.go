package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB adalah variabel global untuk mengakses pool koneksi database
var DB *pgxpool.Pool

// ConnectDB menginisialisasi koneksi ke PostgreSQL dengan konfigurasi optimal
func ConnectDB() {
	// Baca dari environment variable atau gunakan default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:Cidahu1*@localhost:5432/bisma_db?sslmode=disable"
	}

	// Parse konfigurasi dari DSN
	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("❌ Gagal memproses DSN Database: %v\n", err)
	}

	// ==================== OPTIMASI CONNECTION POOL ====================
	// Sesuaikan berdasarkan jumlah concurrent users
	config.MaxConns = 25           // Max concurrent connections (default 4, terlalu rendah)
	config.MinConns = 5            // Min idle connections (warm pool)
	config.MaxConnLifetime = 15 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute
	config.HealthCheckInterval = 1 * time.Minute
	config.ConnectTimeout = 10 * time.Second

	// Membuat pool koneksi
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("❌ Gagal membuat pool koneksi: %v\n", err)
	}

	DB = pool

	// Tes koneksi ke database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = DB.Ping(ctx)
	if err != nil {
		log.Fatalf("❌ Database tidak merespon (Ping Gagal): %v\n", err)
	}

	fmt.Println("✅ Sukses terhubung ke database PostgreSQL!")
	fmt.Printf("📊 Pool Config: Max=%d, Min=%d\n", config.MaxConns, config.MinConns)
}
