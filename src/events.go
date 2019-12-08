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

func eventsPageGet(c *gin.Context) {
	session := sessions.Default(c)
	admin, _ := strconv.ParseBool(fmt.Sprintf("%v", session.Get("admin")))

	query := `
		SELECT
			id,
			title,
			description,
			date,
			datediff(date, CURDATE()) AS days_remaining,
			mandatory
		FROM events
		ORDER BY date DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type row struct {
		ID            int
		Title         string
		Description   string
		Date          string
		DaysRemaining int
		Mandatory     bool
	}

	var data []row

	for rows.Next() {
		var r row
		err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Date, &r.DaysRemaining, &r.Mandatory)
		if err != nil {
			log.Println("Failed to read links rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		data = append(data, r)
	}

	c.HTML(http.StatusOK, "events.gohtml", gin.H{
		"admin":  admin,
		"events": data,
	})
}

// Get event
func eventsGet(c *gin.Context) {
	eventID := c.Param("id")

	query := `
		SELECT
			e.title,
			e.description,
			e.date,
			e.mandatory,
			e.created_on,
			IF(uc.name IS NULL, 'Deleted user', CONCAT(uc.name, ' ', uc.surname)) AS created_by,
			e.updated_on,
			IF(uu.name IS NULL, 'Deleted user', CONCAT(uu.name, ' ', uu.surname)) AS updated_by
		FROM events AS e
		LEFT JOIN users AS uc ON e.created_by = uc.id
		LEFT JOIN users AS uu ON e.updated_by = uu.id
		WHERE e.id=?
		LIMIT 1
	`

	type row struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Date        string `json:"date"`
		Mandatory   bool   `json:"mandatory"`
		CreatedOn   string `json:"created_on"`
		CreatedBy   string `json:"created_by"`
		UpdatedOn   string `json:"updated_on"`
		UpdatedBy   string `json:"updated_by"`
	}

	var data row

	err := db.QueryRow(query, eventID).Scan(
		&data.Title,
		&data.Description,
		&data.Date,
		&data.Mandatory,
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

// Create event
func eventsPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	title := strings.TrimSpace(c.PostForm("title"))
	description := strings.TrimSpace(c.PostForm("description"))
	date := strings.TrimSpace(c.PostForm("date"))
	mandatory := strings.TrimSpace(c.PostForm("mandatory"))

	// 'Active' validation
	if mandatory == "" || mandatory == "0" {
		mandatory = "0"
	} else {
		mandatory = "1"
	}

	// Check for empty inputs:
	if title == "" || date == "" {
		log.Println("One of the params was empty!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Check submitted parameters lengths:
	if utf8.RuneCountInString(title) > 60 || utf8.RuneCountInString(description) > 1000 {
		log.Println("Lengths do not match requirements!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "INSERT INTO events (title, description, mandatory, date, created_by, updated_by) VALUES (?,?,?,?,?,?)"
	_, err := db.Exec(query, title, description, mandatory, date, sessionUserID, sessionUserID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Update event
func eventsPut(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	eventID := c.Param("id")

	title := strings.TrimSpace(c.PostForm("title"))
	description := strings.TrimSpace(c.PostForm("description"))
	date := strings.TrimSpace(c.PostForm("date"))
	mandatory := strings.TrimSpace(c.PostForm("mandatory"))

	// 'Active' validation
	if mandatory == "" || mandatory == "0" {
		mandatory = "0"
	} else {
		mandatory = "1"
	}

	// Check for empty inputs:
	if title == "" || date == "" {
		log.Println("One of the params was empty!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Check submitted parameters lengths:
	if utf8.RuneCountInString(title) > 60 || utf8.RuneCountInString(description) > 1000 {
		log.Println("Lengths do not match requirements!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "UPDATE events SET title=?, description=?, mandatory=?, date=?, updated_by=? WHERE id=? LIMIT 1"
	_, err := db.Exec(query, title, description, mandatory, date, sessionUserID, eventID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Delete event
func eventsDelete(c *gin.Context) {
	rowID := c.Param("id")
	msg, err := deleteFromDatabase(rowID, "events", "")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}
