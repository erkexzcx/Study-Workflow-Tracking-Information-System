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

func teachersPageGet(c *gin.Context) {
	session := sessions.Default(c)
	admin, _ := strconv.ParseBool(fmt.Sprintf("%v", session.Get("admin")))

	query := `
		SELECT
			t.id,
			t.name,
			t.surname,
			t.email,
			(
				SELECT COUNT(*) FROM subjects
				INNER JOIN semesters ON subjects.semester = semesters.id AND semesters.active = 1
				WHERE subjects.teacher = t.id
			) AS semester_subjects,
			(
				SELECT COUNT(*) FROM subjects
				INNER JOIN semesters ON subjects.semester = semesters.id
				WHERE subjects.teacher = t.id
			) AS total_subjects,
			(
				SELECT COUNT(*) FROM assignments
				INNER JOIN subjects ON assignments.subject = subjects.id
				INNER JOIN semesters ON subjects.semester = semesters.id AND semesters.active = 1
				WHERE subjects.teacher = t.id
			) AS semester_assignments,
			(
				SELECT COUNT(*) FROM assignments
				INNER JOIN subjects ON assignments.subject = subjects.id
				INNER JOIN semesters ON subjects.semester = semesters.id
				WHERE subjects.teacher = t.id
			) AS total_assignments
		FROM teachers AS t
		ORDER BY name ASC, surname ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type row struct {
		ID                  int
		Name                string
		Surname             string
		Email               string
		SemesterSubjects    int
		TotalSubjects       int
		SemesterAssignments int
		TotalAssignments    int
	}

	var data []row

	for rows.Next() {
		var r row
		err := rows.Scan(&r.ID, &r.Name, &r.Surname, &r.Email, &r.SemesterSubjects, &r.TotalSubjects, &r.SemesterAssignments, &r.TotalAssignments)
		if err != nil {
			log.Println("Failed to read links rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		data = append(data, r)
	}

	c.HTML(http.StatusOK, "teachers.gohtml", gin.H{
		"admin":    admin,
		"teachers": data,
	})
}

// Get teacher
func teachersGet(c *gin.Context) {
	teacherID := c.Param("id")

	query := `
		SELECT
			t.name,
			t.surname,
			t.email,
			t.created_on,
			IF(uc.name IS NULL, 'Deleted user', CONCAT(uc.name, ' ', uc.surname)) AS created_by,
			t.updated_on,
			IF(uu.name IS NULL, 'Deleted user', CONCAT(uu.name, ' ', uu.surname)) AS updated_by
		FROM teachers AS t
		LEFT JOIN users AS uc ON t.created_by = uc.id
		LEFT JOIN users AS uu ON t.updated_by = uu.id
		WHERE t.id=?
		LIMIT 1
	`

	type row struct {
		Name      string `json:"name"`
		Surname   string `json:"surname"`
		Email     string `json:"email"`
		CreatedOn string `json:"created_on"`
		CreatedBy string `json:"created_by"`
		UpdatedOn string `json:"updated_on"`
		UpdatedBy string `json:"updated_by"`
	}

	var data row
	err := db.QueryRow(query, teacherID).Scan(
		&data.Name,
		&data.Surname,
		&data.Email,
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

// Create teacher
func teachersPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	name := strings.TrimSpace(c.PostForm("name"))
	surname := strings.TrimSpace(c.PostForm("surname"))
	email := strings.TrimSpace(c.PostForm("email"))

	nameLength := utf8.RuneCountInString(name)
	surnameLength := utf8.RuneCountInString(surname)
	emailLength := utf8.RuneCountInString(email)

	// Check inputs
	if (nameLength == 0 || nameLength > 60) || (surnameLength == 0 || surnameLength > 60) || (emailLength == 0 || emailLength > 254) {
		log.Println("Invalid inputs!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "INSERT INTO teachers (name, surname, email, created_by, updated_by) VALUES (?,?,?,?,?)"
	_, err := db.Exec(query, name, surname, email, sessionUserID, sessionUserID)
	if err != nil {
		log.Println("Failed to perform SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Update teacher
func teachersPut(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	teacherID := c.Param("id")

	name := strings.TrimSpace(c.PostForm("name"))
	surname := strings.TrimSpace(c.PostForm("surname"))
	email := strings.TrimSpace(c.PostForm("email"))

	nameLength := utf8.RuneCountInString(name)
	surnameLength := utf8.RuneCountInString(surname)
	emailLength := utf8.RuneCountInString(email)

	// Check inputs
	if (nameLength == 0 || nameLength > 60) || (surnameLength == 0 || surnameLength > 60) || (emailLength == 0 || emailLength > 254) {
		log.Println("Invalid inputs!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "UPDATE teachers SET name=?, surname=?, email=?, updated_by=? WHERE id=? LIMIT 1"
	_, err := db.Exec(query, name, surname, email, sessionUserID, teacherID)
	if err != nil {
		log.Println("Failed to perform SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Delete teacher
func teachersDelete(c *gin.Context) {
	rowID := c.Param("id")
	msg, err := deleteFromDatabase(rowID, "teachers", "This teacher is in-use and cannot be deleted. Please unassign assigned subjects and try again!")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}
