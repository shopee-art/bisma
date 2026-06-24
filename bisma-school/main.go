package main

import (
	"bisma-school/config"
	"bisma-school/controllers"
	"bisma-school/models"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie" // Store session di cookie
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func main() {
	config.ConnectDB()
	defer config.DB.Close()
	
	// =======================================================
	// INI DIA: INISIALISASI AUTOMATIC CRON JOB (JAM 08:00 PAGI)
	// =======================================================
	wib, _ := time.LoadLocation("Asia/Jakarta")
	jakartaCron := cron.New(cron.WithLocation(wib))
	
	// Format: Menit, Jam, Hari, Bulan, Hari dalam Seminggu
	// "0 8 * * 1-6" berarti: Setiap Jam 08:00 Pagi, Hari Senin sampai Sabtu
	_, err := jakartaCron.AddFunc("0 8 * * 1-5", func() {
		models.AutoFillAlpa()
	})
	if err != nil {
		fmt.Println("Gagal membuat jadwal otomatis:", err)
	}
	jakartaCron.Start() // Jalankan engine cron di background worker Go
	defer jakartaCron.Stop()
	

	r := gin.Default()

	// 1. DAFTARKAN FOLDER STATIC UPLOADS
	// Agar foto profil di ./uploads/avatars bisa diakses via URL browser /uploads/avatars/...
	r.Static("/uploads", "./uploads")

	// Inisialisasi Cookie Store untuk Session (Kunci enkripsi rahasia bebas diubah)
	store := cookie.NewStore([]byte("secret-key-bisma-2026"))
	r.Use(sessions.Sessions("bisma_session", store))

	r.SetFuncMap(template.FuncMap{
		"md_add": func(a, b int) int { return a + b },
	})
	
	// Pola pemanggilan HTML template berlapis yang aman untuk subfolder views
	r.LoadHTMLGlob("views/*/*.html")

	// ==================== ROUTE UMUM (TANPA LOGIN) ====================
	r.GET("/", func(c *gin.Context) {
		errMessage := c.Query("error")
		c.HTML(http.StatusOK, "login.html", gin.H{"error": errMessage})
	})
	r.POST("/login", controllers.LoginHandler)
	r.GET("/logout", controllers.LogoutHandler)

	// ==================== ROUTE TERKUNCI (WAJIB LOGIN) ====================
	protected := r.Group("/")
	protected.Use(controllers.AuthRequired()) // Pasang pengunci login global
	{
		protected.GET("/dashboard", controllers.DashboardHandler)
		
		// =======================================================
		// PERBAIKAN: DAFTARKAN RUTE PROFIL DI SINI (TERKUNCI AUTH)
		// =======================================================
		protected.GET("/profile", controllers.ShowProfileHandler)
		protected.POST("/profile/update", controllers.UpdateProfileHandler)
		protected.POST("/profile/avatar", controllers.UploadAvatarHandler)
		protected.POST("/profile/password", controllers.ChangePasswordHandler)
		
		// ROUTE ABSENSI SISWA
		protected.GET("/student/attendance", controllers.AttendancePageHandler)
		protected.POST("/student/attendance/submit", controllers.SubmitAttendanceHandler)
		protected.GET("/student/attendance/history", controllers.AttendanceHistoryHandler)

		// Group Khusus Admin (Harus Login DAN memiliki role Admin)
		adminGroup := protected.Group("/admin")
		adminGroup.Use(controllers.AdminOnly())
		{
			adminGroup.GET("/classes", controllers.ManageClassesHandler)
			adminGroup.POST("/classes", controllers.StoreClassHandler)
			adminGroup.GET("/teachers", controllers.ManageTeachersHandler)
			adminGroup.POST("/teachers", controllers.StoreTeacherHandler)
			adminGroup.GET("/students", controllers.ManageStudentsHandler)
			adminGroup.POST("/students", controllers.StoreStudentHandler)
			adminGroup.POST("/students/import", controllers.ImportStudentsExcelHandler)
			adminGroup.POST("/students/update/:id", controllers.UpdateStudentHandler)
			adminGroup.GET("/students/delete/:id", controllers.DeleteStudentHandler)
			adminGroup.POST("/teachers/update/:id", controllers.UpdateTeacherHandler)
			adminGroup.GET("/teachers/delete/:id", controllers.DeleteTeacherHandler)
			adminGroup.POST("/teachers/import", controllers.ImportTeachersExcelHandler)
			adminGroup.POST("/classes/update/:id", controllers.UpdateClassHandler)
			adminGroup.GET("/classes/delete/:id", controllers.DeleteClassHandler)
		}
	}

	fmt.Println("Server BiSMA berjalan di http://localhost:8080")
	r.Run(":8080")
}