package controllers

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// AuthRequired memastikan user sudah login sebelum mengakses halaman
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userRole := session.Get("user_role")

		if userRole == nil {
			// Jika belum login, tendang ke halaman login awal
			c.Redirect(http.StatusSeeOther, "/?error=Silakan login terlebih dahulu!")
			c.Abort()
			return
		}
				// 💡 TRIK JITU: Masukkan data session ke dalam Context Gin secara global
		c.Set("global_user_name", session.Get("user_name"))
		c.Set("global_user_role", userRole)
		c.Set("global_user_avatar", session.Get("user_avatar"))
		c.Set("global_user_nip", session.Get("user_nip"))
		c.Set("global_user_class", session.Get("user_class"))
		c.Next()
	}
}

// AdminOnly memastikan hanya user ber-role admin yang bisa masuk
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userRole := session.Get("user_role")

		if userRole != "admin" {
			// Jika bukan admin, blokir akses
			c.HTML(http.StatusForbidden, "login.html", gin.H{
				"error": "Akses Ditolak! Halaman ini hanya untuk Admin.",
			})
			c.Abort()
			return
		}
				// 💡 TRIK JITU: Masukkan data session ke dalam Context Gin secara global
		c.Set("global_user_name", session.Get("user_name"))
		c.Set("global_user_role", userRole)
		c.Set("global_user_avatar", session.Get("user_avatar"))
		c.Set("global_user_nip", session.Get("user_nip"))
		c.Set("global_user_class", session.Get("user_class"))
		c.Next()
	}
}

