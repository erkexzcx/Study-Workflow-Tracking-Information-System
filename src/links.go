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

func linksPageGet(c *gin.Context) {
	session := sessions.Default(c)
	admin, _ := strconv.ParseBool(fmt.Sprintf("%v", session.Get("admin")))

	query := `
		SELECT
			id,
			title,
			url
		FROM links
		ORDER BY title, id DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type row struct {
		ID    int
		Title string
		URL   string
	}

	var data []row

	for rows.Next() {
		var r row
		err := rows.Scan(&r.ID, &r.Title, &r.URL)
		if err != nil {
			log.Println("Failed to read links rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		data = append(data, r)
	}

	c.HTML(http.StatusOK, "links.gohtml", gin.H{
		"admin": admin,
		"links": data,
	})
}

// Get link
func linksGet(c *gin.Context) {
	linkID := c.Param("id")

	query := `
		SELECT
			e.title,
			e.url,
			e.created_on,
			IF(uc.name IS NULL, 'Deleted user', CONCAT(uc.name, ' ', uc.surname)) AS created_by,
			e.updated_on,
			IF(uu.name IS NULL, 'Deleted user', CONCAT(uu.name, ' ', uu.surname)) AS updated_by
		FROM links AS e
		LEFT JOIN users AS uc ON e.created_by = uc.id
		LEFT JOIN users AS uu ON e.updated_by = uu.id
		WHERE e.id=?
		LIMIT 1
	`

	type row struct {
		Title     string `json:"title"`
		URL       string `json:"url"`
		CreatedOn string `json:"created_on"`
		CreatedBy string `json:"created_by"`
		UpdatedOn string `json:"updated_on"`
		UpdatedBy string `json:"updated_by"`
	}

	var data row

	err := db.QueryRow(query, linkID).Scan(
		&data.Title,
		&data.URL,
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

// Create link
func linksPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	title := strings.TrimSpace(c.PostForm("title"))
	url := strings.TrimSpace(c.PostForm("url"))

	// Check for empty inputs:
	if title == "" || url == "" {
		log.Println("One of the params was empty!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Check submitted parameters lengths:
	if utf8.RuneCountInString(title) > 60 || utf8.RuneCountInString(url) > 1000 {
		log.Println("Lengths do not match requirements!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "INSERT INTO links (title, url, created_by, updated_by) VALUES (?,?,?,?)"
	_, err := db.Exec(query, title, url, sessionUserID, sessionUserID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Update link
func linksPut(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	linkID := c.Param("id")

	title := strings.TrimSpace(c.PostForm("title"))
	url := strings.TrimSpace(c.PostForm("url"))

	// Check for empty inputs:
	if title == "" || url == "" {
		log.Println("One of the params was empty!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Check submitted parameters lengths:
	if utf8.RuneCountInString(title) > 60 || utf8.RuneCountInString(url) > 1000 {
		log.Println("Lengths do not match requirements!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "UPDATE links SET title=?, url=?, updated_by=? WHERE id=? LIMIT 1"
	_, err := db.Exec(query, title, url, sessionUserID, linkID)
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Delete link
func linksDelete(c *gin.Context) {
	rowID := c.Param("id")
	msg, err := deleteFromDatabase(rowID, "links", "")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}
