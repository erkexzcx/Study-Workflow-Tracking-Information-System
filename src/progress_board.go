package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func progressBoardPageGet(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	query := `
		SELECT
			assignments.id AS assignment_id,
			assignments.title AS assignment_title,
			assignments.description,
			assignments.until,
			subjects.title AS subject_title,
			subjects.url AS url,
			teachers.name AS teacher_name,
			teachers.surname AS teacher_surname,
			semesters.active AS active,
			semesters.title AS semester_title,
			datediff(assignments.until, CURDATE()) AS days_remaining,
			IF(assignment_status.status IS NULL, 0, assignment_status.status) AS assignment_status,
			IF(assignment_status.note IS NULL, "", assignment_status.note) AS assignment_status_note,
			(SELECT COUNT(*) FROM assignment_status WHERE assignment_status.assignment = assignments.id AND assignment_status.status = 3) AS people_done,
			(SELECT COUNT(*) FROM assignment_status WHERE assignment_status.assignment = assignments.id AND assignment_status.status = 2) AS people_pending,
			(SELECT COUNT(*) FROM users) AS people_count
		FROM assignments
		LEFT JOIN subjects ON assignments.subject = subjects.id
		LEFT JOIN semesters ON subjects.semester = semesters.id
		LEFT JOIN teachers ON subjects.teacher = teachers.id
		LEFT JOIN assignment_status ON assignments.id = assignment_status.assignment AND assignment_status.user=?
		WHERE (assignment_status.status <> 3 OR assignment_status.status IS NULL) OR (semesters.active=1 AND assignment_status.status = 3)
		ORDER BY assignments.until DESC, assignments.title ASC
	`

	rows, err := db.Query(query, sessionUserID)
	if err != nil {
		log.Println("Failed to fetch progress_board data from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type row struct {
		AssignmentID         int
		AssignmentTitle      string
		Description          string
		Until                string
		SubjectTitle         string
		SubjectURL           string
		TeacherName          string
		TeacherSurname       string
		Active               bool
		SemesterTitle        string
		DaysRemaining        int
		AssignmentStatus     int
		AssignmentStatusNote string
		PeopleDone           int
		PeoplePending        int
		PeopleCount          int
	}

	data := []row{}

	for rows.Next() {
		var r row
		err := rows.Scan(
			&r.AssignmentID,
			&r.AssignmentTitle,
			&r.Description,
			&r.Until,
			&r.SubjectTitle,
			&r.SubjectURL,
			&r.TeacherName,
			&r.TeacherSurname,
			&r.Active,
			&r.SemesterTitle,
			&r.DaysRemaining,
			&r.AssignmentStatus,
			&r.AssignmentStatusNote,
			&r.PeopleDone,
			&r.PeoplePending,
			&r.PeopleCount,
		)
		if err != nil {
			log.Println("Failed to read progress_board rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		data = append(data, r)
	}

	c.HTML(http.StatusOK, "progress_board.gohtml", gin.H{
		"assignments": data,
	})
}

func assignmentStatusGet(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	assignmentID := c.Param("assignmenID")

	query := `
		SELECT
			assignments.title,
			teachers.name AS name,
			teachers.surname AS surname,
			IF(assignment_status.status IS NULL, 0, assignment_status.status) AS status,
			IF(assignment_status.note IS NULL, '', assignment_status.note) AS note,
			IF(assignment_status.updated_on IS NULL, '', assignment_status.updated_on) AS updated_on
		FROM assignments
		LEFT JOIN subjects ON assignments.subject = subjects.id
		LEFT JOIN teachers ON subjects.teacher = teachers.id
		LEFT JOIN assignment_status ON assignments.id = assignment_status.assignment AND assignment_status.user=?
		WHERE assignments.id=?
		LIMIT 1
	`
	type row struct {
		AssignmentTitle      string `json:"assignment_title"`
		TeacherName          string `json:"teacher_name"`
		TeacherSurname       string `json:"teacher_surname"`
		AssignmentStatus     int    `json:"assignment_status"`
		AssignmentStatusNote string `json:"assignment_status_note"`
		UpdatedOn            string `json:"updated_on"`
	}

	var data row

	err := db.QueryRow(query, sessionUserID, assignmentID).Scan(
		&data.AssignmentTitle,
		&data.TeacherName,
		&data.TeacherSurname,
		&data.AssignmentStatus,
		&data.AssignmentStatusNote,
		&data.UpdatedOn,
	)
	if err != nil {
		// Error - check if it's no rows returned or something else:
		log.Println("Failed to perform prepared SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, data)
}

func assignmentStatusPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	// Get submitted parameters:
	assignmentID := c.Param("assignmenID")
	status := c.PostForm("status")
	note := c.PostForm("note")

	// Validate parameters
	if nmb, err := strconv.Atoi(assignmentID); err != nil || nmb < 0 {
		log.Println("Provided assignment ID is not a valid number!:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}
	if len(note) > 1000 {
		log.Println("Provided note is longer than 1000 characters!:")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}
	if status != "0" && status != "1" && status != "2" && status != "3" {
		log.Println("Provided status is invalid!:")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "INSERT INTO assignment_status (user, assignment, status, note) VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE status=?, note=?"

	_, err := db.Exec(query, sessionUserID, assignmentID, status, note, status, note)
	if err != nil {
		log.Println("Failed to perform prepared SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}
