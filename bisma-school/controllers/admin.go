package controllers

import (
	"bisma-school/models"
	"fmt"
	"net/http"
	"github.com/gin-contrib/sessions" // <-- Pastikan baris ini ada
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// ManageClassesHandler menampilkan halaman daftar kelas menggunakan layout base
func ManageClassesHandler(c *gin.Context) {
	classes, err := models.GetAllClasses()
	if err != nil {
		c.String(http.StatusInternalServerError, "Gagal mengambil data kelas: %v", err)
		return
	}

	// UPDATE: Dialihkan memanggil template master "base"
	c.HTML(http.StatusOK, "base", gin.H{
		"user":    "Admin Utama",
		"role":    "admin",
		"classes": classes,
		"error":   c.Query("error"),
		"success": c.Query("success"),
		"page_content": "classes",
		
	})
}

// StoreClassHandler memproses input tambah kelas baru
func StoreClassHandler(c *gin.Context) {
	className := c.PostForm("class_name")

	if className == "" {
		c.Redirect(http.StatusSeeOther, "/admin/classes?error=Nama kelas tidak boleh kosong")
		return
	}

	err := models.CreateClass(className)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/classes?error=Gagal menyimpan atau kelas sudah ada")
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/classes?success=Kelas berhasil ditambahkan!")
}

// ManageTeachersHandler mendukung Pencarian, Pagination (35 data), dan layout base
func ManageTeachersHandler(c *gin.Context) {
	session := sessions.Default(c)
	search := c.Query("search")
	pageStr := c.Query("page")

	page := 1
	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if page < 1 {
		page = 1
	}

	limit := 35
	offset := (page - 1) * limit

	teachers, totalData, err := models.GetFilteredTeachers(limit, offset, search)
	if err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat data guru: %v", err)
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

	// UPDATE: Dialihkan memanggil template master "base"
	c.HTML(http.StatusOK, "base", gin.H{
		"user":        "Admin Utama",
		"role":        "admin",
		"teachers":    teachers,
		"classes":     classes,
		"search":      search,
		"currentPage": page,
		"totalPage":   totalPage,
		"prevPage":    page - 1,
		"nextPage":    page + 1,
		"error":       c.Query("error"),
		"success":     c.Query("success"),
		"page_content": "teachers",
		// ⚠️ SUNTIKAN DATA UNTUK PROFIL SIDEBAR
		"user_avatar":      session.Get("user_avatar"), // Pastikan ini diset saat login/update profile
		"user_nip":         session.Get("user_nip"),    // ID / NIP guru atau admin
		"user_class":       session.Get("user_class"),  // Nama kelas jika dia siswa
	})
}

// UpdateTeacherHandler memproses perubahan data guru
func UpdateTeacherHandler(c *gin.Context) {
	var id int
	fmt.Sscanf(c.Param("id"), "%d", &id)

	nip := c.PostForm("nip")
	name := c.PostForm("name")
	role := c.PostForm("role")
	additionalRole := c.PostForm("additional_role")
	classIDStr := c.PostForm("managed_class_id")

	var managedClassID *int
	if additionalRole == "wali_kelas" && classIDStr != "" {
		var cid int
		if _, err := fmt.Sscanf(classIDStr, "%d", &cid); err == nil {
			managedClassID = &cid
		}
	}

	err := models.UpdateTeacher(id, nip, name, role, additionalRole, managedClassID)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/teachers?error=Gagal mengubah data guru!")
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/teachers?success=Data guru berhasil diubah!")
}

// DeleteTeacherHandler menghapus guru berdasarkan ID
func DeleteTeacherHandler(c *gin.Context) {
	var id int
	fmt.Sscanf(c.Param("id"), "%d", &id)

	err := models.DeleteTeacher(id)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/teachers?error=Gagal menghapus data guru!")
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/teachers?success=Guru berhasil dihapus!")
}

// ImportTeachersExcelHandler memproses berkas Excel guru massal
func ImportTeachersExcelHandler(c *gin.Context) {
	file, err := c.FormFile("excel_file")
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/teachers?error=Gagal mengunggah berkas!")
		return
	}

	openedFile, err := file.Open()
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/teachers?error=Gagal membaca berkas!")
		return
	}
	defer openedFile.Close()

	f, err := excelize.OpenReader(openedFile)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/teachers?error=Format harus berkas Excel (.xlsx)!")
		return
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		c.Redirect(http.StatusSeeOther, "/admin/teachers?error=Berkas Excel kosong!")
		return
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil || len(rows) <= 1 {
		c.Redirect(http.StatusSeeOther, "/admin/teachers?error=Tidak ada data guru!")
		return
	}

	var teachersToImport []models.Teacher

	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 4 {
			continue
		}

		var addRole *string
		if len(row) >= 5 && row[4] != "" {
			val := row[4]
			addRole = &val
		}

		var managedClassID *int
		if len(row) >= 6 && row[5] != "" {
			var cid int
			if _, err := fmt.Sscanf(row[5], "%d", &cid); err == nil {
				managedClassID = &cid
			}
		}

		teachersToImport = append(teachersToImport, models.Teacher{
			NIP:            row[0],
			Name:           row[1],
			Password:       row[2],
			Role:           row[3],
			AdditionalRole: addRole,
			ManagedClassID: managedClassID,
		})
	}

	err = models.ImportTeachersBulk(teachersToImport)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/teachers?error=Gagal simpan massal! Periksa NIP duplikat.")
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/teachers?success=Berhasil mengimpor data guru massal!")
}

// StoreTeacherHandler memproses input data guru baru
func StoreTeacherHandler(c *gin.Context) {
	nip := c.PostForm("nip")
	name := c.PostForm("name")
	password := c.PostForm("password")
	role := c.PostForm("role")
	additionalRole := c.PostForm("additional_role")
	classIDStr := c.PostForm("managed_class_id")

	var managedClassID *int
	if classIDStr != "" {
		var id int
		if _, err := fmt.Sscanf(classIDStr, "%d", &id); err == nil {
			managedClassID = &id
		}
	}

	err := models.CreateTeacher(nip, name, password, role, additionalRole, managedClassID)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/teachers?error=Gagal menyimpan guru baru")
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/teachers?success=Guru baru berhasil ditambahkan!")
}

// StoreStudentHandler memproses input data siswa baru
func StoreStudentHandler(c *gin.Context) {
	nis := c.PostForm("nis")
	name := c.PostForm("name")
	password := c.PostForm("password")
	classIDStr := c.PostForm("class_id")

	var classID int
	if _, err := fmt.Sscanf(classIDStr, "%d", &classID); err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Kelas wajib dipilih!")
		return
	}

	err := models.CreateStudent(nis, name, password, classID)
	if err != nil {
		fmt.Printf("ERROR DATABASE POSTGRESQL: %v\n", err)
		c.Redirect(http.StatusSeeOther, "/admin/students?error=NIS sudah terdaftar atau data tidak valid!")
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/students?success=Siswa baru berhasil ditambahkan!")
}

// ImportStudentsExcelHandler memproses unggahan file Excel data siswa
func ImportStudentsExcelHandler(c *gin.Context) {
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

	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 4 {
			continue
		}

		var classID int
		fmt.Sscanf(row[3], "%d", &classID)

		studentsToImport = append(studentsToImport, models.Student{
			NIS:      row[0],
			Name:     row[1],
			Password: row[2],
			ClassID:  classID,
		})
	}

	err = models.ImportStudentsBulk(studentsToImport)
	if err != nil {
		fmt.Printf("ERROR BULK INSERT EXCEL: %v\n", err)
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Gagal simpan massal! Periksa apakah ada NIS yang duplikat.")
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin/students?success=Berhasil mengimpor data siswa secara massal!")
}

// ManageStudentsHandler mendukung limit 35 data, filter kelas, dan layout base
func ManageStudentsHandler(c *gin.Context) {
	session := sessions.Default(c)
	search := c.Query("search")
	pageStr := c.Query("page")
	classIDStr := c.Query("class_id")

	page := 1
	if pageStr != "" {
		fmt.Sscanf(pageStr, "%d", &page)
	}
	if page < 1 {
		page = 1
	}

	classID := 0
	if classIDStr != "" {
		fmt.Sscanf(classIDStr, "%d", &classID)
	}

	limit := 35
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

	// UPDATE: Dialihkan memanggil template master "base"
	c.HTML(http.StatusOK, "base", gin.H{
		"user":          "Admin Utama",
		"role":          "admin",
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
		"page_content":  "students",
		// ⚠️ SUNTIKAN DATA UNTUK PROFIL SIDEBAR
		"user_avatar":      session.Get("user_avatar"), // Pastikan ini diset saat login/update profile
		"user_nip":         session.Get("user_nip"),    // ID / NIP guru atau admin
		"user_class":       session.Get("user_class"),  // Nama kelas jika dia siswa
	})
}

// UpdateStudentHandler memproses perubahan data siswa
func UpdateStudentHandler(c *gin.Context) {
	idStr := c.Param("id")
	nis := c.PostForm("nis")
	name := c.PostForm("name")
	classIDStr := c.PostForm("class_id")

	var id, classID int
	fmt.Sscanf(idStr, "%d", &id)
	fmt.Sscanf(classIDStr, "%d", &classID)

	err := models.UpdateStudent(id, nis, name, classID)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Gagal memperbarui data siswa!")
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/students?success=Data siswa berhasil diperbarui!")
}

// DeleteStudentHandler menghapus siswa berdasarkan ID di URL
func DeleteStudentHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)

	err := models.DeleteStudent(id)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/students?error=Gagal menghapus siswa!")
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/students?success=Siswa berhasil dihapus!")
}

// UpdateClassHandler memproses perubahan nama kelas
func UpdateClassHandler(c *gin.Context) {
	var id int
	fmt.Sscanf(c.Param("id"), "%d", &id)
	className := c.PostForm("class_name")

	err := models.UpdateClass(id, className)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/classes?error=Gagal memperbarui nama kelas!")
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/classes?success=Nama kelas berhasil diperbarui!")
}

// DeleteClassHandler menghapus data kelas
func DeleteClassHandler(c *gin.Context) {
	var id int
	fmt.Sscanf(c.Param("id"), "%d", &id)

	err := models.DeleteClass(id)
	if err != nil {
		c.Redirect(http.StatusSeeOther, "/admin/classes?error=Gagal menghapus kelas! Pastikan tidak ada guru/siswa yang terikat di kelas ini.")
		return
	}
	c.Redirect(http.StatusSeeOther, "/admin/classes?success=Kelas berhasil dihapus!")
}