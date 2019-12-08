package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func subjectsPageGet(c *gin.Context) {
	session := sessions.Default(c)
	admin, _ := strconv.ParseBool(fmt.Sprintf("%v", session.Get("admin")))

	query := `
		SELECT
			s.id,
			s.title,
			IF(s.url IS NULL, '', s.url) AS url,
			IF(s.access_key IS NULL, '', s.access_key) AS access_key,
			t.name,
			t.surname,
			sem.title AS semester_title,
			sem.active AS semester_active,
			(
				SELECT COUNT(*) FROM assignments
				INNER JOIN subjects ON assignments.subject = subjects.id
				WHERE subjects.id = s.id
			) AS assignments
		FROM subjects AS s
		JOIN semesters AS sem ON s.semester = sem.id
		JOIN teachers AS t ON s.teacher = t.id
		ORDER BY sem.active DESC, sem.title DESC, s.title ASC, s.id DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type row struct {
		ID             int
		Title          string
		URL            string
		AccessKey      string
		Name           string
		Surname        string
		SemesterTitle  string
		SemesterActive bool
		Assignments    int
	}

	var data []row

	for rows.Next() {
		var r row
		err := rows.Scan(&r.ID, &r.Title, &r.URL, &r.AccessKey, &r.Name, &r.Surname, &r.SemesterTitle, &r.SemesterActive, &r.Assignments)
		if err != nil {
			log.Println("Failed to read links rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		data = append(data, r)
	}

	query = `
		SELECT
			id,
			name,
			surname
		FROM teachers
		ORDER BY name, surname
	`
	rows, err = db.Query(query)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type teachersRow struct {
		ID      int
		Name    string
		Surname string
	}

	var dataTeachers []teachersRow

	for rows.Next() {
		var tr teachersRow
		err := rows.Scan(&tr.ID, &tr.Name, &tr.Surname)
		if err != nil {
			log.Println("Failed to read rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		dataTeachers = append(dataTeachers, tr)
	}

	query = `
		SELECT
			id,
			title,
			active
		FROM semesters
		ORDER BY active DESC, title ASC, id DESC
	`
	rows, err = db.Query(query)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type semestersRow struct {
		ID     int
		Title  string
		Active bool
	}

	var dataSemesters []semestersRow

	for rows.Next() {
		var sr semestersRow
		err := rows.Scan(&sr.ID, &sr.Title, &sr.Active)
		if err != nil {
			log.Println("Failed to read rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		dataSemesters = append(dataSemesters, sr)
	}

	c.HTML(http.StatusOK, "subjects.gohtml", gin.H{
		"admin":     admin,
		"subjects":  data,
		"teachers":  dataTeachers,
		"semesters": dataSemesters,
	})
}

// Get subject
func subjectsGet(c *gin.Context) {
	subjectID := c.Param("id")

	query := `
		SELECT
			s.title,
			s.url,
			s.access_key,
			s.teacher AS teacher_id,
			s.semester AS semester_id,
			s.created_on,
			IF(uc.name IS NULL, 'Deleted user', CONCAT(uc.name, ' ', uc.surname)) AS created_by,
			s.updated_on,
			IF(uu.name IS NULL, 'Deleted user', CONCAT(uu.name, ' ', uu.surname)) AS updated_by
		FROM subjects AS s
		LEFT JOIN users AS uc ON s.created_by = uc.id
		LEFT JOIN users AS uu ON s.updated_by = uu.id
		WHERE s.id=?
		LIMIT 1
	`

	type row struct {
		Title      string `json:"title"`
		URL        string `json:"url"`
		AccessKey  string `json:"access_key"`
		TeacherID  int    `json:"teacher_id"`
		SemesterID int    `json:"semester_id"`
		CreatedOn  string `json:"created_on"`
		CreatedBy  string `json:"created_by"`
		UpdatedOn  string `json:"updated_on"`
		UpdatedBy  string `json:"updated_by"`
	}

	var data row
	err := db.QueryRow(query, subjectID).Scan(
		&data.Title,
		&data.URL,
		&data.AccessKey,
		&data.TeacherID,
		&data.SemesterID,
		&data.CreatedOn,
		&data.CreatedBy,
		&data.UpdatedOn,
		&data.UpdatedBy,
	)
	if err != nil {
		// Error - check if it's no rows returned or something else:
		log.Println("Failed to perform prepared SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, data)
}

// Create subject
func subjectsPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	title := strings.TrimSpace(c.PostForm("title"))
	teacher := strings.TrimSpace(c.PostForm("teacher"))
	url := strings.TrimSpace(c.PostForm("url"))
	accessKey := strings.TrimSpace(c.PostForm("access_key"))
	semester := strings.TrimSpace(c.PostForm("semester"))

	titleLength := utf8.RuneCountInString(title)
	urlLength := utf8.RuneCountInString(url)
	accessKeyLength := utf8.RuneCountInString(accessKey)

	// Check inputs
	if (titleLength == 0 || titleLength > 60) || urlLength > 1000 || accessKeyLength > 100 {
		log.Println("Invalid inputs!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "INSERT INTO subjects (title, teacher, url, access_key, semester, created_by, updated_by) VALUES (?,?,?,?,?,?,?)"
	_, err := db.Exec(query, title, teacher, url, accessKey, semester, sessionUserID, sessionUserID)
	if err != nil {
		log.Println("Failed to perform SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Update subject
func subjectsPut(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	subjectID := c.Param("id")

	title := strings.TrimSpace(c.PostForm("title"))
	teacher := strings.TrimSpace(c.PostForm("teacher"))
	url := strings.TrimSpace(c.PostForm("url"))
	accessKey := strings.TrimSpace(c.PostForm("access_key"))
	semester := strings.TrimSpace(c.PostForm("semester"))

	titleLength := utf8.RuneCountInString(title)
	urlLength := utf8.RuneCountInString(url)
	accessKeyLength := utf8.RuneCountInString(accessKey)

	// Check inputs
	if (titleLength == 0 || titleLength > 60) || urlLength > 1000 || accessKeyLength > 100 {
		log.Println("Invalid inputs!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "UPDATE subjects SET title=?, url=?, access_key=?, teacher=?, semester=?, updated_by=? WHERE id=? LIMIT 1"
	_, err := db.Exec(query, title, url, accessKey, teacher, semester, sessionUserID, subjectID)
	if err != nil {
		log.Println("Failed to perform SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Delete subject
func subjectsDelete(c *gin.Context) {
	rowID := c.Param("id")
	msg, err := deleteFromDatabase(rowID, "subjects", "This subject is in-use and cannot be deleted. Please unassign assigned assignments and try again!")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}
