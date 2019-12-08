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

func assignmentsPageGet(c *gin.Context) {
	session := sessions.Default(c)
	admin, _ := strconv.ParseBool(fmt.Sprintf("%v", session.Get("admin")))
	sessionUserID := session.Get("id")

	query := `
		SELECT
			assignments.id AS assignment_id,
			assignments.title AS assignment_title,
			assignments.until,
			datediff(assignments.until, CURDATE()) AS days_remaining,
			semesters.title AS semester_title,
			semesters.active AS semester_active,
			subjects.title AS subject_title,
			subjects.url AS subject_url,
			teachers.name AS teacher_name,
			teachers.surname AS teacher_surname,
			assignments.description,
			IF(assignment_status.status IS NULL, 0, assignment_status.status) AS assignment_status,
			IF(datediff(assignments.created_on, CURDATE()) > -3, 1, 0) AS new
		FROM assignments
		LEFT JOIN subjects ON assignments.subject = subjects.id
		LEFT JOIN semesters ON subjects.semester = semesters.id
		LEFT JOIN teachers ON subjects.teacher = teachers.id
		LEFT JOIN assignment_status ON assignments.id = assignment_status.assignment AND assignment_status.user=?
		ORDER BY assignments.until DESC, assignments.title ASC
	`
	rows, err := db.Query(query, sessionUserID)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type row struct {
		AssignmentID     int
		AssignmentTitle  string
		Until            string
		DaysRemaining    int
		SemesterTitle    string
		SemesterActive   bool
		SubjectTitle     string
		SubjectURL       string
		TeacherName      string
		TeacherSurname   string
		Description      string
		AssignmentStatus int
		New              bool
	}

	var data []row

	for rows.Next() {
		var r row
		err := rows.Scan(
			&r.AssignmentID,
			&r.AssignmentTitle,
			&r.Until,
			&r.DaysRemaining,
			&r.SemesterTitle,
			&r.SemesterActive,
			&r.SubjectTitle,
			&r.SubjectURL,
			&r.TeacherName,
			&r.TeacherSurname,
			&r.Description,
			&r.AssignmentStatus,
			&r.New,
		)
		if err != nil {
			log.Println("Failed to read links rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		data = append(data, r)
	}

	query = `
	SELECT
		subjects.id,
		subjects.title,
		semesters.active
	FROM subjects
	LEFT JOIN semesters ON subjects.semester = semesters.id
`
	rows, err = db.Query(query)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type subjectsRow struct {
		ID     int
		Title  string
		Active bool
	}

	var subjectsData []subjectsRow

	for rows.Next() {
		var sr subjectsRow
		err := rows.Scan(
			&sr.ID,
			&sr.Title,
			&sr.Active,
		)
		if err != nil {
			log.Println("Failed to read links rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		subjectsData = append(subjectsData, sr)
	}

	query = `
	SELECT IF(COUNT(*) > 0, 1, 0) AS count
	FROM subjects
	INNER JOIN semesters ON subjects.semester = semesters.id
	WHERE semesters.active = 1
	`
	var activeSubjectsExist bool
	err = db.QueryRow(query).Scan(&activeSubjectsExist)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.HTML(http.StatusOK, "assignments.gohtml", gin.H{
		"admin":               admin,
		"assignments":         data,
		"subjects":            subjectsData,
		"activeSubjectsExist": activeSubjectsExist,
	})
}

// Get assignment
func assignmentsGet(c *gin.Context) {
	assignmentID := c.Param("id")

	query := `
		SELECT
			a.title,
			a.subject AS subject_id,
			a.until,
			a.description,
			a.created_on,
			IF(uc.name IS NULL, 'Deleted user', CONCAT(uc.name, ' ', uc.surname)) AS created_by,
			a.updated_on,
			IF(uu.name IS NULL, 'Deleted user', CONCAT(uu.name, ' ', uu.surname)) AS updated_by
		FROM assignments AS a
		LEFT JOIN users AS uc ON a.created_by = uc.id
		LEFT JOIN users AS uu ON a.updated_by = uu.id
		WHERE a.id=?
		LIMIT 1
	`

	type row struct {
		Title       string `json:"title"`
		SubjectID   int    `json:"subject_id"`
		Until       string `json:"until"`
		Description string `json:"description"`
		CreatedOn   string `json:"created_on"`
		CreatedBy   string `json:"created_by"`
		UpdatedOn   string `json:"updated_on"`
		UpdatedBy   string `json:"updated_by"`
	}

	var data row
	err := db.QueryRow(query, assignmentID).Scan(
		&data.Title,
		&data.SubjectID,
		&data.Until,
		&data.Description,
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

// Create assignment
func assignmentsPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	title := strings.TrimSpace(c.PostForm("title"))
	until := strings.TrimSpace(c.PostForm("until"))
	subject := strings.TrimSpace(c.PostForm("subject"))
	description := strings.TrimSpace(c.PostForm("description"))

	titleLength := utf8.RuneCountInString(title)
	descriptionLength := utf8.RuneCountInString(description)

	// Check inputs
	if (titleLength == 0 || titleLength > 60) || descriptionLength > 1000 {
		log.Println("Invalid inputs!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "INSERT INTO assignments (title, until, subject, description, created_by, updated_by) VALUES (?,?,?,?,?,?)"
	_, err := db.Exec(query, title, until, subject, description, sessionUserID, sessionUserID)
	if err != nil {
		log.Println("Failed to perform SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Update assignment
func assignmentsPut(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	assignmentID := c.Param("id")

	title := strings.TrimSpace(c.PostForm("title"))
	until := strings.TrimSpace(c.PostForm("until"))
	subject := strings.TrimSpace(c.PostForm("subject"))
	description := strings.TrimSpace(c.PostForm("description"))

	titleLength := utf8.RuneCountInString(title)
	descriptionLength := utf8.RuneCountInString(description)

	// Check inputs
	if (titleLength == 0 || titleLength > 60) || descriptionLength > 1000 {
		log.Println("Invalid inputs!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "UPDATE assignments SET title=?, until=?, subject=?, description=?, updated_by=? WHERE id=? LIMIT 1"
	_, err := db.Exec(query, title, until, subject, description, sessionUserID, assignmentID)
	if err != nil {
		log.Println("Failed to perform SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Delete assignment
func assignmentsDelete(c *gin.Context) {
	rowID := c.Param("id")
	msg, err := deleteFromDatabase(rowID, "assignments", "")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}
