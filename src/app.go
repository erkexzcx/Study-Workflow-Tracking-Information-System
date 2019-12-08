package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func appGet(c *gin.Context) {
	session := sessions.Default(c)

	c.HTML(http.StatusOK, "app.gohtml", gin.H{
		"title":       config["title"],
		"author":      config["author"],
		"authorurl":   config["authorurl"],
		"currentyear": time.Now().Year(),
		"name":        session.Get("name"),
		"surname":     session.Get("surname"),
		"picture":     session.Get("picture"),
		"email":       session.Get("email"),
		"admin":       session.Get("admin"),
		"theme_code":  session.Get("theme_code"),
	})
}

func activeSubjectsContainerGet(c *gin.Context) {
	query := `
		SELECT subjects.title, subjects.url FROM subjects
		INNER JOIN semesters ON subjects.semester = semesters.id AND semesters.active = 1
		ORDER BY subjects.title DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Failed to fetch active subcjets from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	var data []map[string]string

	for rows.Next() {
		var title, url string
		err := rows.Scan(&title, &url)
		if err != nil {
			log.Println("Failed to read active subject rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		data = append(data, map[string]string{
			"title": title,
			"url":   url,
		})
	}

	c.JSON(http.StatusOK, data)
}

func linksContainerGet(c *gin.Context) {
	query := `SELECT title, url FROM links ORDER BY title DESC;`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Failed to fetch links from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	var data []map[string]string

	for rows.Next() {
		var title, url string
		err := rows.Scan(&title, &url)
		if err != nil {
			log.Println("Failed to read links rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		data = append(data, map[string]string{
			"title": title,
			"url":   url,
		})
	}

	c.JSON(http.StatusOK, data)
}

func menuLabelsGet(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	query := `
		SELECT

		(SELECT COUNT(*) FROM events WHERE date >= now() && date <= now() + INTERVAL 3 DAY) AS events_red,

		(SELECT COUNT(*) FROM events WHERE date >= now() + INTERVAL 3 DAY && date <= now() + INTERVAL 7 DAY) AS events_yellow,

		(SELECT COUNT(*) FROM assignments AS a
		LEFT JOIN assignment_status AS ast ON ast.assignment = a.id AND ast.user = ?
		WHERE a.until >= now() AND a.until <= now() + INTERVAL 7 DAY AND (ast.status != 3 OR ast.status IS NULL)) AS assignments_red,

		(SELECT COUNT(*) FROM assignments AS a
		LEFT JOIN assignment_status AS ast ON ast.assignment = a.id AND ast.user = ?
		WHERE a.until >= now() + INTERVAL 7 DAY AND a.until <= now() + INTERVAL 30 DAY AND (ast.status != 3 OR ast.status IS NULL)) AS assignments_yellow
	`

	var eventsRed, eventsYellow, assignmentsRed, assignmentsYellow int
	err := db.QueryRow(query, sessionUserID, sessionUserID).Scan(&eventsRed, &eventsYellow, &assignmentsRed, &assignmentsYellow)
	if err != nil {
		// Error - check if it's no rows returned or something else:
		log.Println("Failed to perform prepared SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	data := map[string]int{
		"events_red":         eventsRed,
		"events_yellow":      eventsYellow,
		"assignments_red":    assignmentsRed,
		"assignments_yellow": assignmentsYellow,
	}

	c.JSON(http.StatusOK, data)
}
