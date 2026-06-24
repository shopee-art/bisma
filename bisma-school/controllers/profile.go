package controllers

import (
	"bisma-school/models"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// ShowProfileHandler menampilkan halaman pengisian profil mandiri
func ShowProfileHandler(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id").(int)
	userRole := session.Get("user_role").(string)

	profile, err := models.GetUserProfileByID(userID, userRole)
	if err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat profil: %v", err)
		return
	}

	c.HTML(http.StatusOK, "base", gin.H{
		"user":         session.Get("user_name"),
		"role":         userRole,
		"page_content": "profile",
		"profile":      profile,
		"error":        c.Query("error"),
		"success":      c.Query("success"),
		// ⚠️ SUNTIKAN DATA UNTUK PROFIL SIDEBAR
		"user_avatar":      session.Get("user_avatar"), // Pastikan ini diset saat login/update profile
		"user_nip":         session.Get("user_nip"),    // ID / NIP guru atau admin
		"user_class":       session.Get("user_class"),  // Nama kelas jika dia siswa
		
	})
}

// UpdateProfileHandler memproses pembaruan data teks profil
func UpdateProfileHandler(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id").(int)
	userRole := session.Get("user_role").(string)

	birthPlace := c.PostForm("birth_place")
	birthDate := c.PostForm("birth_date")
	hobby := c.PostForm("hobby")

	err := models.UpdateUserProfile(userID, userRole, birthPlace, birthDate, hobby)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/profile?error=Gagal memperbarui profil")
		return
	}

	c.Redirect(http.StatusSeeOther, "/profile?success=Profil berhasil diperbarui!")
}

// UploadAvatarHandler menangani upload file foto profil
func UploadAvatarHandler(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id").(int)
	userRole := session.Get("user_role").(string)

	file, err := c.FormFile("avatar")
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/profile?error=Berkas foto tidak ditemukan")
		return
	}

	// Validasi Ekstensi Gambar
	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		c.Redirect(http.StatusSeeOther, "/profile?error=Format foto harus JPG, JPEG, atau PNG")
		return
	}

	// Buat folder penyimpanan jika belum ada
	uploadDir := "./uploads/avatars"
	_ = os.MkdirAll(uploadDir, os.ModePerm)

	// Beri nama unik agar tidak bentrok
	filename := fmt.Sprintf("%s_%d_%d%s", userRole, userID, time.Now().Unix(), ext)
	targetPath := filepath.Join(uploadDir, filename)

	// Simpan file ke server
	if err := c.SaveUploadedFile(file, targetPath); err != nil {
		c.Redirect(http.StatusSeeOther, "/profile?error=Gagal menyimpan file foto")
		return
	}

	// Simpan path relatif ke database
	dbPath := "/uploads/avatars/" + filename
	err = models.UpdateProfilePicture(userID, userRole, dbPath)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/profile?error=Gagal memperbarui database foto")
		return
	}

	c.Redirect(http.StatusSeeOther, "/profile?success=Foto profil berhasil diubah!")
}

// ChangePasswordHandler memproses penggantian password akun
func ChangePasswordHandler(c *gin.Context) {
	session := sessions.Default(c)
	userID := session.Get("user_id").(int)
	userRole := session.Get("user_role").(string)

	oldPassword := c.PostForm("old_password")
	newPassword := c.PostForm("new_password")
	confirmPassword := c.PostForm("confirm_password")

	if newPassword != confirmPassword {
		c.Redirect(http.StatusSeeOther, "/profile?error=Konfirmasi password baru tidak cocok")
		return
	}

	err := models.UpdateUserPassword(userID, userRole, oldPassword, newPassword)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/profile?error="+err.Error())
		return
	}

	c.Redirect(http.StatusSeeOther, "/profile?success=Password berhasil diganti!")
}