package controllers

import (
	"bisma-school/models"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// ====== HELPER FUNCTIONS UNTUK REFACTOR ADMIN.GO ======

// parsePageNumber mengekstrak dan validate nomor halaman dari query parameter
func parsePageNumber(pageStr string) int {
	page, err := ParseIntFromString(pageStr)
	if err != nil || page < 1 {
		return 1
	}
	return page
}

// parseClassID mengekstrak class ID dari form
func parseClassID(classIDStr string) *int {
	if !ValidateNonEmpty(classIDStr) {
		return nil
	}
	cid, err := ParseIntFromString(classIDStr)
	if err != nil {
		return nil
	}
	return &cid
}

// getSessionData mengambil data user dari session dengan aman
func getSessionData(c *gin.Context) (name string, role string, avatar interface{}, nip interface{}, className interface{}) {
	session := sessions.Default(c)

	// Type assertion yang aman untuk string fields
	if nameVal := session.Get("user_name"); nameVal != nil {
		if nameStr, ok := nameVal.(string); ok {
			name = nameStr
		}
	}

	if roleVal := session.Get("user_role"); roleVal != nil {
		if roleStr, ok := roleVal.(string); ok {
			role = roleStr
		}
	}

	// Interface fields biarkan as-is (tidak perlu type assertion)
	avatar = session.Get("user_avatar")
	nip = session.Get("user_nip")
	className = session.Get("user_class")

	return
}

// renderAdminPage adalah helper untuk render halaman admin dengan data sidebar
func renderAdminPage(c *gin.Context, status int, pageContent string, data gin.H) {
	name, role, avatar, nip, className := getSessionData(c)
	data["user"] = name
	data["role"] = role
	data["user_avatar"] = avatar
	data["user_nip"] = nip
	data["user_class"] = className
	data["page_content"] = pageContent
	c.HTML(status, "base", data)
}

// ====== REFACTORED ADMIN HANDLERS ======

// ManageStudentsHandler - versi dioptimasi dengan helper functions
func ManageStudentsHandlerOptimized(c *gin.Context) {
	search := strings.TrimSpace(c.Query("search"))
	pageStr := c.Query("page")
	classIDStr := c.Query("class_id")

	page := parsePageNumber(pageStr)
	classID := 0
	if classIDStr != "" {
		if cid, err := ParseIntFromString(classIDStr); err == nil {
			classID = cid
		}
	}

	const limit = 35
	offset := (page - 1) * limit

	students, totalData, err := models.GetFilteredStudents(limit, offset, search, classID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat data siswa: %v", err)
		return
	}

	classes, err := models.GetAllClasses()
	if err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat data kelas: %v", err)
		return
	}

	totalPage := (totalData + limit - 1) / limit
	if totalPage < 1 {
		totalPage = 1
	}

	data := gin.H{
		"students":      students,
		"classes":       classes,
		"search":        search,
		"selectedClass": classID,
		"currentPage":   page,
		"totalPage":     totalPage,
		"prevPage":      page - 1,
		"nextPage":      page + 1,
		"error":         c.Query("error"),
		"success":       c.Query("success"),
	}

	renderAdminPage(c, http.StatusOK, "students", data)
}

// StoreStudentHandler - versi dioptimasi dengan validasi lebih ketat
func StoreStudentHandlerOptimized(c *gin.Context) {
	nis := strings.TrimSpace(c.PostForm("nis"))
	name := strings.TrimSpace(c.PostForm("name"))
	password := strings.TrimSpace(c.PostForm("password"))
	classIDStr := c.PostForm("class_id")

	// Validasi input
	if !ValidateNonEmpty(nis) || !ValidateMinLength(nis, 3) {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=NIS harus minimal 3 karakter")
		return
	}
	if !ValidateNonEmpty(name) || !ValidateMinLength(name, 3) {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Nama harus minimal 3 karakter")
		return
	}
	if !ValidateNonEmpty(password) || !ValidateMinLength(password, 6) {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Password harus minimal 6 karakter")
		return
	}

	classID, err := ParseIntFromString(classIDStr)
	if err != nil || classID < 1 {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Kelas wajib dipilih!")
		return
	}

	err = models.CreateStudent(nis, name, password, classID)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=NIS sudah terdaftar atau data tidak valid!")
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/students?success=Siswa baru berhasil ditambahkan!")
}

// ImportStudentsExcelHandlerOptimized - validasi lebih baik dan batch insert lebih cepat
func ImportStudentsExcelHandlerOptimized(c *gin.Context) {
	file, err := c.FormFile("excel_file")
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Gagal mengunggah file!")
		return
	}

	openedFile, err := file.Open()
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Gagal membaca file!")
		return
	}
	defer openedFile.Close()

	f, err := excelize.OpenReader(openedFile)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Format file harus Excel (.xlsx)!")
		return
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Sheet Excel kosong!")
		return
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil || len(rows) <= 1 {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Tidak ada data siswa di dalam file!")
		return
	}

	var studentsToImport []models.Student

	// Parse dengan lebih ketat
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 4 {
			continue
		}

		// Validasi setiap field
		nis := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		password := strings.TrimSpace(row[2])
		classIDStr := strings.TrimSpace(row[3])

		if !ValidateNonEmpty(nis) || !ValidateNonEmpty(name) || !ValidateNonEmpty(password) {
			continue
		}

		classID, err := ParseIntFromString(classIDStr)
		if err != nil || classID < 1 {
			continue
		}

		studentsToImport = append(studentsToImport, models.Student{
			NIS:      nis,
			Name:     name,
			Password: password,
			ClassID:  classID,
		})
	}

	if len(studentsToImport) == 0 {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Tidak ada data valid di file!")
		return
	}

	err = models.ImportStudentsBulk(studentsToImport)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Gagal simpan massal! Periksa apakah ada NIS yang duplikat.")
		return
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/admin/students?success=Berhasil mengimpor %d siswa!", len(studentsToImport)))
}
