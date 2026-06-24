package config

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB adalah variabel global untuk mengakses pool koneksi database
var DB *pgxpool.Pool

// ConnectDB digunakan untuk menginisialisasi koneksi ke PostgreSQL
func ConnectDB() {
	// Ganti username, password, localhost, dan nama_database sesuai dengan di laptop Anda
	dsn := "postgres://postgres:Cidahu1*@localhost:5432/bisma_db?sslmode=disable"
	
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("Gagal memproses DSN Database: %v\n", err)
	}

	// Membuat pool koneksi
	DB, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatalf("Gagal membuat koneksi ke database: %v\n", err)
	}

	// Tes koneksi ke database
	err = DB.Ping(context.Background())
	if err != nil {
		log.Fatalf("Database tidak merespon (Ping Gagal): %v\n", err)
	}

	fmt.Println("Sukses terhubung ke database PostgreSQL!")
}