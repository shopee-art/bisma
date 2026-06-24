package controllers

import (
	"bisma-school/models"
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// RenderWithSidebar otomatis menyuntikkan data profil sidebar ke halaman mana pun
func RenderWithSidebar(c *gin.Context, status int, pageContent string, data gin.H) {
	// Ambil data global dari middleware AuthRequired aman tanpa crash
	data["user"] = c.MustGet("global_user_name")
	data["role"] = c.MustGet("global_user_role")
	data["user_avatar"] = c.MustGet("global_user_avatar")
	data["user_nip"] = c.MustGet("global_user_nip")
	data["user_class"] = c.MustGet("global_user_class")
	
	// Set konten halaman
	data["page_content"] = pageContent

	// Render langsung ke base layout induk
	c.HTML(status, "base", data)
}

// LoginHandler memproses data dari form login
func LoginHandler(c *gin.Context) {
	session := sessions.Default(c) // Inisialisasi session
	roleType := c.PostForm("role_type")
	identityNumber := c.PostForm("identity_number")
	password := c.PostForm("password")

	if roleType == "guru" {
		teacher, err := models.FindTeacherByNIP(identityNumber)
		if err != nil || teacher.Password != password {
			c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "NIP tidak terdaftar atau password salah!"})
			return
		}

		// SIMPAN KE SESSION
		session.Set("user_id", teacher.ID)
		session.Set("user_name", teacher.Name)
		session.Set("user_role", teacher.Role) // Mengambil role dari DB (bisa "admin" atau "teacher"/"guru")
		session.Set("user_nip", teacher.NIP)  // Menyimpan NIP sebagai identitas unik di sidebar
		
		// Mengambil foto profil jika ada, gunakan field dari struct model Anda (asumsi bernama ProfilePicture)
		// Jika nama field di struct teacher Anda berbeda, sesuaikan namanya di bawah ini.
		session.Set("user_avatar", teacher.ProfilePicture) 
		session.Set("user_class", "") // Guru/Admin tidak memiliki kelas tetap di sidebar
		session.Save()

		c.Redirect(http.StatusSeeOther, "/dashboard")
		return

	} else if roleType == "siswa" {
		student, err := models.FindStudentByNIS(identityNumber)
		if err != nil || student.Password != password {
			c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "NIS tidak terdaftar atau password salah!"})
			return
		}

		// SIMPAN KE SESSION
		session.Set("user_id", student.ID)
		session.Set("user_name", student.Name)
		session.Set("user_role", "siswa")
		session.Set("user_nip", student.NIS) // Siswa menggunakan NIS sebagai ID unik di sidebar
		session.Set("user_avatar", student.ProfilePicture) // Menyimpan foto profil siswa dari DB

		// Mengambil nama kelas siswa (asumsi struct model student Anda memiliki field ClassName hasil JOIN)
		// Jika tidak ada, ia akan default terisi string kosong terlebih dahulu.
		session.Set("user_class", student.ClassName) 
		session.Save()

		c.Redirect(http.StatusSeeOther, "/dashboard")
		return
	}

	c.HTML(http.StatusBadRequest, "login.html", gin.H{"error": "Role tidak valid!"})
}

// DashboardHandler menampilkan halaman utama berdasarkan session dan menyuntikkan statistik admin
func DashboardHandler(c *gin.Context) {
	session := sessions.Default(c)
	userName := session.Get("user_name")
	userRole := session.Get("user_role")
	userID := session.Get("user_id")

	if userRole == nil {
		c.Redirect(http.StatusSeeOther, "/?error=Silakan login terlebih dahulu!")
		return
	}

	// 1. Deklarasikan semua variabel di scope teratas fungsi
	alreadyAbsent := false
	var adminStats models.AdminDashboardStats
	todayAbsentCount := 0 // Menggunakan '=' biasa nanti di dalam blok, bukan ':='

	// 2. Logika pencarian data adaptif berdasarkan Role
	if userRole == "siswa" {
		if id, ok := userID.(int); ok {
			alreadyAbsent, _ = models.CheckAlreadyAbsent(id)
		}
	} else if userRole == "admin" {
		// PERBAIKAN: Gunakan '=' untuk assignment agar tidak membuat variabel baru yang terisolasi
		adminStats, _ = models.GetAdminDashboardStats()
		todayAbsentCount, _ = models.GetTodayAttendanceCount()
	}

	// 3. Kirim paket data lengkap ke layout induk "base"
	c.HTML(http.StatusOK, "base", gin.H{
		"user":             userName,
		"role":             userRole,
		"alreadyAbsent":    alreadyAbsent,
		"page_content":     "dashboard",
		"stats":            adminStats,
		"todayAbsentCount": todayAbsentCount, // Sekarang variabel ini dijamin terdefinisi dan aman!
		
		// SUNTIKAN DATA UNTUK PROFIL SIDEBAR
		"user_avatar":      session.Get("user_avatar"), 
		"user_nip":         session.Get("user_nip"),    
		"user_class":       session.Get("user_class"),  
	})
}

// LogoutHandler menghapus session saat keluar
func LogoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusSeeOther, "/")
}
