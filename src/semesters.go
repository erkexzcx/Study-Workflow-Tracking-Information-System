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

// Load semesters ajax page
func semestersPageGet(c *gin.Context) {
	session := sessions.Default(c)
	admin, _ := strconv.ParseBool(fmt.Sprintf("%v", session.Get("admin")))

	query := `
		SELECT
			sem.id,
			sem.title,
			sem.active,
			(
				SELECT COUNT(*) FROM semesters
				INNER JOIN subjects ON semesters.id = subjects.semester
				INNER JOIN teachers ON subjects.teacher = teachers.id
				WHERE semesters.id = sem.id
			) AS teachers_assigned,
			(
				SELECT COUNT(*) FROM semesters
				INNER JOIN subjects ON semesters.id = subjects.semester
				WHERE semesters.id = sem.id
			) AS subjects_assigned,
			(
				SELECT COUNT(*) FROM semesters
				INNER JOIN subjects ON semesters.id = subjects.semester
				INNER JOIN assignments ON assignments.subject = subjects.id
				WHERE semesters.id = sem.id
			) AS assignments_assigned
		FROM semesters AS sem
		ORDER BY title ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type row struct {
		ID                  int
		Title               string
		Active              bool
		TeachersAssigned    int
		SubjectsAssigned    int
		AssignmentsAssigned int
	}

	var data []row

	for rows.Next() {
		var r row
		err := rows.Scan(&r.ID, &r.Title, &r.Active, &r.TeachersAssigned, &r.SubjectsAssigned, &r.AssignmentsAssigned)
		if err != nil {
			log.Println("Failed to read links rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		data = append(data, r)
	}

	c.HTML(http.StatusOK, "semesters.gohtml", gin.H{
		"admin":     admin,
		"semesters": data,
	})
}

// Get semester
func semestersGet(c *gin.Context) {
	semesterID := c.Param("id")

	query := `
		SELECT
			s.title,
			s.active,
			s.created_on,
			IF(uc.name IS NULL, 'Deleted user', CONCAT(uc.name, ' ', uc.surname)) AS created_by,
			s.updated_on,
			IF(uu.name IS NULL, 'Deleted user', CONCAT(uu.name, ' ', uu.surname)) AS updated_by
		FROM semesters AS s
		LEFT JOIN users AS uc ON s.created_by = uc.id
		LEFT JOIN users AS uu ON s.updated_by = uu.id
		WHERE s.id=?
		LIMIT 1
	`

	type row struct {
		Title     string `json:"title"`
		Active    bool   `json:"active"`
		CreatedOn string `json:"created_on"`
		CreatedBy string `json:"created_by"`
		UpdatedOn string `json:"updated_on"`
		UpdatedBy string `json:"updated_by"`
	}

	var data row

	err := db.QueryRow(query, semesterID).Scan(
		&data.Title,
		&data.Active,
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

// Create semester
func semestersPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	title := strings.TrimSpace(c.PostForm("title"))
	active := strings.TrimSpace(c.PostForm("active"))

	// 'Active' validation
	if active == "" || active == "0" {
		active = "0"
	} else {
		active = "1"
	}

	// Check for empty inputs:
	if title == "" {
		log.Println("One of the params was empty!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Check submitted parameters lengths:
	if utf8.RuneCountInString(title) > 60 {
		log.Println("Lengths do not match requirements!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	if active == "1" {

		tx, err := db.Begin()
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}

		_, err = tx.Exec("UPDATE semesters SET active=0 WHERE active=1 LIMIT 1")
		if err != nil {
			log.Println(err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}

		query := "INSERT INTO semesters (title, active, created_by, updated_by) VALUES (?, 1, ?, ?)"
		if _, err := tx.Exec(query, title, sessionUserID, sessionUserID); err != nil {
			log.Println(err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}

		if err = tx.Commit(); err != nil {
			log.Println(err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}

	} else {

		// No transaction needed - simple insert:
		query := "INSERT INTO semesters (title, created_by, updated_by) VALUES (?, ?, ?)"
		_, err := db.Exec(query, title, sessionUserID, sessionUserID)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Activate semester
func semestersActivatePost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	semesterID := c.Param("id")

	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}
	//--------------------------------
	query := "UPDATE semesters SET active=0, updated_by=? WHERE active=1 LIMIT 1"
	_, err = tx.Exec(query, sessionUserID)
	if err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}
	//--------------------------------
	query = "UPDATE semesters SET active=1, updated_by=? WHERE id=? LIMIT 1"
	res, err := tx.Exec(query, sessionUserID, semesterID)
	if err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}
	if rowsAffected <= 0 {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}
	//--------------------------------
	if err = tx.Commit(); err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Update semester
func semestersPut(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	semesterID := c.Param("id")

	title := strings.TrimSpace(c.PostForm("title"))
	active := strings.TrimSpace(c.PostForm("active"))

	// 'Active' validation
	if active == "" || active == "0" {
		active = "0"
	} else {
		active = "1"
	}

	// Check for empty inputs:
	if title == "" {
		log.Println("One of the params was empty!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Check submitted parameters lengths:
	if utf8.RuneCountInString(title) > 60 {
		log.Println("Lengths do not match requirements!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	if active == "1" {

		tx, err := db.Begin()
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		//--------------------------------
		query := "UPDATE semesters SET active=0, updated_by=? WHERE active=1 LIMIT 1"
		_, err = tx.Exec(query, sessionUserID)
		if err != nil {
			log.Println(err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		//--------------------------------
		query = "UPDATE semesters SET title=?, active=1, updated_by=? WHERE id=? LIMIT 1"
		res, err := tx.Exec(query, title, sessionUserID, semesterID)
		if err != nil {
			log.Println(err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			log.Println(err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		if rowsAffected <= 0 {
			log.Println("No rows affected while settings active=1")
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		//--------------------------------
		if err = tx.Commit(); err != nil {
			log.Println(err)
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}

	} else {

		query := "UPDATE semesters SET title=?, updated_by=? WHERE active=0 AND id=? LIMIT 1"
		_, err := db.Exec(query, title, sessionUserID, semesterID)
		if err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}

	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Delete semester
func semestersDelete(c *gin.Context) {
	rowID := c.Param("id")
	msg, err := deleteFromDatabase(rowID, "semesters", "This semester is in-use and cannot be deleted. Please unassign assigned subjects and try again!")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}
