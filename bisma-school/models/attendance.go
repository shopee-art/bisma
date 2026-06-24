package models

import (
	"context"
	"math"
	"bisma-school/config"
	"time"
	"log"
	
)

// Struct bawaan absensi sebelumnya
type Attendance struct {
	ID        int
	StudentID int
	Latitude  float64
	Longitude float64
	Photo     string
	Status    string
}

// Struct bawaan riwayat sebelumnya
type AttendanceHistory struct {
	ID        int
	Date      string
	Time      string
	Latitude  float64
	Longitude float64
	Photo     string
	Status    string
}

// ==========================================
// KUNCI UTAMANYA: STRUCT UNTUK DASHBOARD ADMIN
// ==========================================

// AdminDashboardStats menampung data ringkasan untuk dashboard admin
type AdminDashboardStats struct {
	TotalClasses  int
	TotalTeachers int
	TotalStudents int
}


// ==========================================
// FUNGSI-FUNGSI UTAMA ABSENSI & STATISTIK
// ==========================================

// CheckAlreadyAbsent memeriksa apakah siswa sudah absen hari ini
func CheckAlreadyAbsent(studentID int) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM attendances WHERE student_id = $1 AND attendance_date = CURRENT_DATE)"
	err := config.DB.QueryRow(context.Background(), query, studentID).Scan(&exists)
	return exists, err
}

// SaveAttendance menyimpan data absensi ke database
func SaveAttendance(studentID int, lat, lng float64, photo, status string) error {
	query := `INSERT INTO attendances (student_id, latitude, longitude, photo, status) 
			  VALUES ($1, $2, $3, $4, $5)`
	_, err := config.DB.Exec(context.Background(), query, studentID, lat, lng, photo, status)
	return err
}

// CalculateDistance menghitung jarak radius menggunakan formula Haversine
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeter = 6371000.0
	radLat1 := lat1 * math.Pi / 180
	radLng1 := lon1 * math.Pi / 180
	radLat2 := lat2 * math.Pi / 180
	radLng2 := lon2 * math.Pi / 180

	dLat := radLat2 - radLat1
	dLng := radLng2 - radLng1

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(radLat1)*math.Cos(radLat2)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusMeter * c
}

// GetStudentAttendanceHistory mengambil seluruh riwayat absensi berdasarkan ID siswa
func GetStudentAttendanceHistory(studentID int) ([]AttendanceHistory, error) {
	ctx := context.Background()
	var history []AttendanceHistory

	query := `SELECT id, 
				     TO_CHAR(attendance_date, 'DD-MM-YYYY'), 
				     TO_CHAR(check_in_time, 'HH24:MI:SS'), 
				     latitude, longitude, photo, status 
			  FROM attendances 
			  WHERE student_id = $1 
			  ORDER BY attendance_date DESC, check_in_time DESC`

	rows, err := config.DB.Query(ctx, query, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var ah AttendanceHistory
		err := rows.Scan(&ah.ID, &ah.Date, &ah.Time, &ah.Latitude, &ah.Longitude, &ah.Photo, &ah.Status)
		if err != nil {
			return nil, err
		}
		history = append(history, ah)
	}
	return history, nil
}

// GetAdminDashboardStats menghitung total data master di database untuk Admin
func GetAdminDashboardStats() (AdminDashboardStats, error) {
	ctx := context.Background()
	var stats AdminDashboardStats

	_ = config.DB.QueryRow(ctx, "SELECT COUNT(*) FROM classes").Scan(&stats.TotalClasses)
	_ = config.DB.QueryRow(ctx, "SELECT COUNT(*) FROM teachers").Scan(&stats.TotalTeachers)
	_ = config.DB.QueryRow(ctx, "SELECT COUNT(*) FROM students").Scan(&stats.TotalStudents)

	return stats, nil
}

// GetTodayAttendanceCount menghitung jumlah siswa yang sudah absen HARI INI
func GetTodayAttendanceCount() (int, error) {
	ctx := context.Background()
	var count int

	query := "SELECT COUNT(*) FROM attendances WHERE attendance_date = CURRENT_DATE"
	err := config.DB.QueryRow(ctx, query).Scan(&count)
	
	return count, err
}

// IsHoliday mengecek apakah tanggal tertentu adalah hari libur (Sabtu, Minggu, atau Tanggal Merah)
func IsHoliday(t time.Time) bool {
	ctx := context.Background()

	// 1. Cek jika hari Sabtu (6) atau Minggu (0)
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return true
	}

	// 2. Cek apakah tanggal ini terdaftar di tabel hari libur / libur semester di DB
	dateStr := t.Format("2006-01-02")
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM holidays WHERE holiday_date = $1)"
	
	err := config.DB.QueryRow(ctx, query, dateStr).Scan(&exists)
	if err != nil {
		log.Printf("[ERROR] Gagal mengecek tabel holidays: %v", err)
		return false
	}

	return exists
}

// AutoFillAlpa mencari siswa yang tidak absen hari ini dan menandainya sebagai Alpa
func AutoFillAlpa() {
	ctx := context.Background()
	wib, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(wib)
	today := now.Format("2006-01-02")

	// 💡 PERBAIKAN: Jika hari ini hari libur, hentikan proses cron (jangan isi Alpa)
	if IsHoliday(now) {
		log.Printf("[CRON] Hari ini (%s) adalah hari libur sekolah. Proses Auto-Alpa dibatalkan.", today)
		return
	}

	log.Printf("[CRON] Menjalankan pengecekan Alpa otomatis untuk tanggal: %s", today)

	// Query mengambil siswa yang hari ini belum absen (Tetap seperti kode sebelumnya)
	queryGetAlpaSiswa := `
		SELECT id FROM students 
		WHERE id NOT IN (
			SELECT student_id FROM attendances WHERE date = $1
		)
	`

	rows, err := config.DB.Query(ctx, queryGetAlpaSiswa, today)
	if err != nil {
		log.Printf("[CRON ERROR] Gagal mengambil data siswa: %v", err)
		return
	}
	defer rows.Close()

	var studentIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			studentIDs = append(studentIDs, id)
		}
	}

	if len(studentIDs) == 0 {
		log.Println("[CRON] Semua siswa sudah melakukan absensi hari ini. Tidak ada Alpa.")
		return
	}

	queryInsertAlpa := `
		INSERT INTO attendances (student_id, date, status, info) 
		VALUES ($1, $2, 'Alpa', 'Sistem: Tidak melakukan absen hingga batas waktu')
	`

	count := 0
	for _, sID := range studentIDs {
		_, err := config.DB.Exec(ctx, queryInsertAlpa, sID, today)
		if err == nil {
			count++
		}
	}

	log.Printf("[CRON SUCCESS] Berhasil mencatat otomatis %d siswa sebagai Alpa.", count)
}