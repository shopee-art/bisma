package controllers

import (
	"bisma-school/models"
	"fmt"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"time"
)

// TITIK KOORDINAT SEKOLAH - CIRAHAYU
const SchoolLat = -7.017201371959697
const SchoolLng = 108.61782892234378
const MaxRadiusMeter = 100.0 // Jarak maksimal toleransi siswa (100 meter)

// AttendancePageHandler menampilkan form halaman absensi siswa menggunakan layout base
func AttendancePageHandler(c *gin.Context) {
	session := sessions.Default(c)
	studentID := session.Get("user_id").(int)

	// Cek apakah sudah absen hari ini
	alreadyAbsent, err := models.CheckAlreadyAbsent(studentID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Gagal memeriksa status absensi")
		return
	}

	c.HTML(http.StatusOK, "base", gin.H{
		"role":          "siswa",
		"user":          session.Get("user_name"),
		"alreadyAbsent": alreadyAbsent,
		"schoolLat":     SchoolLat,
		"schoolLng":     SchoolLng,
		"page_content":  "attendance_form",
	})
}

// SubmitAttendanceHandler memproses kiriman data koordinat & foto selfie (API JSON)
func SubmitAttendanceHandler(c *gin.Context) {
	session := sessions.Default(c)
	studentID := session.Get("user_id").(int)
	
	// 1. Ambil waktu server saat ini berdasarkan zona waktu lokal (WIB/WITA/WIT)
	wib, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Now().In(wib)
	
	// 💡 Blokir akses absen masuk jika hari libur
    if models.IsHoliday(now) {
        c.Redirect(http.StatusSeeOther, "/dashboard?error=Hari ini sekolah libur! Anda tidak perlu melakukan absensi.")
        return
    }
	
	currentHour := now.Hour()
	currentMinute := now.Minute()
	currentTimeInMinutes := (currentHour * 60) + currentMinute

	// 2. Tentukan batas jam dalam satuan menit
	startAbsen := 6 * 60          // Jam 06:00 Pagi
	batasAbsen := (7 * 60) + 15   // Jam 07:15 Pagi (Silakan ganti ke 7 * 60 + 30 jika jam 07:30)

	// 3. Validasi Batas Waktu Mulai Absen
	if currentTimeInMinutes < startAbsen {
		c.Redirect(http.StatusSeeOther, "/student/attendance?error=Absensi belum dibuka! Silakan absen mulai pukul 06:00 WIB.")
		return
	}

	// 4. Validasi Batas Akhir (Siswa Kesiangan / Tidak Bisa Absen Lagi)
	if currentTimeInMinutes > batasAbsen {
		c.Redirect(http.StatusSeeOther, "/student/attendance?error=Batas waktu absensi telah habis! Anda dinyatakan Alpa/Terlambat, silakan hubungi Guru Piket.")
		return
	}

	// Ambil data dari request JSON
	var input struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Photo     string  `json:"photo"` // Base64 string gambar
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Format data tidak valid"})
		return
	}

	// 1. Validasi Jarak Radius GPS
	distance := models.CalculateDistance(input.Latitude, input.Longitude, SchoolLat, SchoolLng)
	if distance > MaxRadiusMeter {
		msg := fmt.Sprintf("Absen ditolak! Anda berada di luar radius sekolah (Jarak: %.2f meter)", distance)
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": msg})
		return
	}

	// 2. Simpan Ke Database
	err := models.SaveAttendance(studentID, input.Latitude, input.Longitude, input.Photo, "Hadir")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Anda sudah melakukan absensi hari ini!"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Absensi berhasil disimpan! Selamat belajar."})
}

// AttendanceHistoryHandler menampilkan halaman riwayat kehadiran siswa menggunakan layout base
func AttendanceHistoryHandler(c *gin.Context) {
	session := sessions.Default(c)
	studentID := session.Get("user_id").(int)

	history, err := models.GetStudentAttendanceHistory(studentID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat riwayat kehadiran: %v", err)
		return
	}

	c.HTML(http.StatusOK, "base", gin.H{
		"role":         "siswa",
		"user":         session.Get("user_name"),
		"history":      history,
		"page_content": "attendance_history",
	})
}