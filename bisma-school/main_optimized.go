package main

import (
	"bisma-school/config"
	"bisma-school/controllers"
	"bisma-school/models"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

func main() {
	// ==================== KONEKSI DATABASE ====================
	config.ConnectDB()
	defer config.DB.Close()

	// ==================== SETUP CRON JOB ====================
	wib, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Fatalf("❌ Gagal load timezone WIB: %v", err)
	}

	jakartaCron := cron.New(cron.WithLocation(wib))

	// Schedule auto-fill absence setiap Senin-Jumat jam 08:00 WIB
	_, err = jakartaCron.AddFunc("0 8 * * 1-5", func() {
		log.Println("[CRON] Menjalankan auto-fill absensi...")
		models.AutoFillAlpa()
	})
	if err != nil {
		log.Printf("❌ Gagal membuat jadwal otomatis: %v\n", err)
	}

	jakartaCron.Start()
	defer jakartaCron.Stop()
	log.Println("✅ Cron job berhasil diinisialisasi")

	// ==================== GIN SETUP ====================
	// Production mode jika environment = production
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()

	// Middleware recovery untuk handle panic
	r.Use(gin.Recovery())

	// 1. Static files
	r.Static("/uploads", "./uploads")

	// 2. Session middleware
	store := cookie.NewStore([]byte("secret-key-bisma-2026"))
	r.Use(sessions.Sessions("bisma_session", store))

	// 3. Custom template functions
	r.SetFuncMap(template.FuncMap{
		"md_add": func(a, b int) int { return a + b },
	})

	// 4. Load templates
	r.LoadHTMLGlob("views/*/*.html")

	// ==================== ROUTES ====================

	// Public routes (no auth required)
	r.GET("/", func(c *gin.Context) {
		errMessage := c.Query("error")
		c.HTML(http.StatusOK, "login.html", gin.H{"error": errMessage})
	})
	r.POST("/login", controllers.LoginHandler)
	r.GET("/logout", controllers.LogoutHandler)

	// Protected routes (auth required)
	protected := r.Group("/")
	protected.Use(controllers.AuthRequired())
	{
		protected.GET("/dashboard", controllers.DashboardHandler)
		protected.GET("/profile", controllers.ShowProfileHandler)
		protected.POST("/profile/update", controllers.UpdateProfileHandler)
		protected.POST("/profile/avatar", controllers.UploadAvatarHandler)
		protected.POST("/profile/password", controllers.ChangePasswordHandler)

		protected.GET("/student/attendance", controllers.AttendancePageHandler)
		protected.POST("/student/attendance/submit", controllers.SubmitAttendanceHandler)
		protected.GET("/student/attendance/history", controllers.AttendanceHistoryHandler)

		// Admin-only routes
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

	// ==================== GRACEFUL SHUTDOWN ====================
	srvPort := ":8080"
	srv := &http.Server{
		Addr:         srvPort,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Run server di goroutine
	go func() {
		log.Printf("🚀 Server BiSMA berjalan di http://localhost%s\n", srvPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Graceful shutdown handler
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\n🛑 Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server stopped gracefully")
}
