package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Load users ajax page
func usersPageGet(c *gin.Context) {
	session := sessions.Default(c)
	admin, _ := strconv.ParseBool(fmt.Sprintf("%v", session.Get("admin")))
	sessionUserID := session.Get("id")

	query := `
		SELECT id, name, surname, email, username, admin
		FROM users
		ORDER BY name ASC, surname ASC
	`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type row struct {
		ID       int
		Name     string
		Surname  string
		Email    string
		Username string
		Admin    bool
	}

	var data []row

	for rows.Next() {
		var r row
		err := rows.Scan(&r.ID, &r.Name, &r.Surname, &r.Email, &r.Username, &r.Admin)
		if err != nil {
			log.Println("Failed to read links rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		data = append(data, r)
	}

	c.HTML(http.StatusOK, "users.gohtml", gin.H{
		"admin":         admin,
		"sessionUserID": sessionUserID,
		"users":         data,
	})
}

// Get user
func usersGet(c *gin.Context) {
	userID := c.Param("id")

	query := `
		SELECT
			u.name,
			u.surname,
			u.email,
			u.username,
			u.admin,
			u.created_on,
			IF(uc.name IS NULL, 'Deleted user', CONCAT(uc.name, ' ', uc.surname)) AS created_by,
			u.updated_on,
			IF(uu.name IS NULL, 'Deleted user', CONCAT(uu.name, ' ', uu.surname)) AS updated_by
		FROM users AS u
		LEFT JOIN users AS uc ON u.created_by = uc.id
		LEFT JOIN users AS uu ON u.updated_by = uu.id
		WHERE u.id=?
		LIMIT 1
	`

	type row struct {
		Name      string `json:"name"`
		Surname   string `json:"surname"`
		Email     string `json:"email"`
		Username  string `json:"username"`
		Admin     bool   `json:"admin"`
		CreatedOn string `json:"created_on"`
		CreatedBy string `json:"created_by"`
		UpdatedOn string `json:"updated_on"`
		UpdatedBy string `json:"updated_by"`
	}

	var data row

	err := db.QueryRow(query, userID).Scan(
		&data.Name,
		&data.Surname,
		&data.Email,
		&data.Username,
		&data.Admin,
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

// Create user
func usersPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	name := strings.TrimSpace(c.PostForm("name"))
	surname := strings.TrimSpace(c.PostForm("surname"))
	email := strings.TrimSpace(c.PostForm("email"))
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")
	admin := strings.TrimSpace(c.PostForm("admin"))

	// 'Admin' validation
	if admin == "" || admin == "0" {
		admin = "0"
	} else {
		admin = "1"
	}

	// Check for empty inputs:
	if name == "" || surname == "" || email == "" || username == "" || strings.TrimSpace(password) == "" {
		log.Println("One of the params was empty!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Check if contains spaces:
	if strings.Contains(name, " ") || strings.Contains(surname, " ") || strings.Contains(email, " ") || strings.Contains(username, " ") {
		log.Println("Submitted parameters contain spaces where it should not be")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Check submitted parameters lengths:
	if utf8.RuneCountInString(name) > 60 || utf8.RuneCountInString(surname) > 60 || utf8.RuneCountInString(username) > 60 || utf8.RuneCountInString(username) < 3 || utf8.RuneCountInString(password) < 6 {
		log.Println("Lengths do not match requirements!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// All looks good, so it's time to hash the password:
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Failed to encrypt the pasword!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}
	hashedPassword := string(hash)

	// Will use transaction for 2 queries:
	tx, err := db.Begin()
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Insert user into the database:
	query := `INSERT INTO users (name, surname, email, username, password, admin, created_by, updated_by)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := tx.Exec(query, name, surname, email, username, hashedPassword, admin, sessionUserID, sessionUserID)
	if err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Get last inserted user ID:
	lastInsertedID, err := res.LastInsertId()
	if err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Mark all existing assignments (except active ones) as completed by system:
	query = `
	INSERT INTO assignment_status (user, assignment, status, note)
	SELECT
		?,
		assignments.id,
		3,
		"Automatically marked as completed during user creation"
	FROM assignments
	LEFT JOIN subjects ON assignments.subject = subjects.id
	LEFT JOIN semesters ON subjects.semester = semesters.id
	WHERE semesters.active <> 1
	`
	_, err = tx.Exec(query, lastInsertedID)
	if err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Commit transaction:
	if err = tx.Commit(); err != nil {
		log.Println(err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Update user
func usersPut(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	userID := c.Param("id")

	name := strings.TrimSpace(c.PostForm("name"))
	surname := strings.TrimSpace(c.PostForm("surname"))
	email := strings.TrimSpace(c.PostForm("email"))
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")
	admin := strings.TrimSpace(c.PostForm("admin"))

	changePassword := false
	if password != "" {
		changePassword = true
	}

	// 'Admin' validation
	if admin == "" || admin == "0" {
		admin = "0"
	} else {
		admin = "1"
	}

	// Check for empty inputs:
	if name == "" || surname == "" || email == "" || username == "" {
		log.Println("One of the params was empty!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Check if contains spaces:
	if strings.Contains(name, " ") || strings.Contains(surname, " ") || strings.Contains(email, " ") || strings.Contains(username, " ") {
		log.Println("Submitted parameters contain spaces where it should not be")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Check submitted parameters lengths:
	if utf8.RuneCountInString(name) > 60 || utf8.RuneCountInString(surname) > 60 || utf8.RuneCountInString(username) > 60 || utf8.RuneCountInString(username) < 3 || (changePassword && utf8.RuneCountInString(password) < 6) {
		log.Println("Lengths do not match requirements!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	if changePassword {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			log.Println("Failed to encrypt the pasword!")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		hashedPassword := string(hash)

		query := "UPDATE users SET name=?, surname=?, email=?, username=?, password=?, admin=?, updated_by=? WHERE id=? LIMIT 1"
		_, err = db.Exec(query, name, surname, email, username, hashedPassword, admin, sessionUserID, userID)
		if err != nil {
			log.Println("Failed to perform prepared SQL query against database:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
	} else {
		query := "UPDATE users SET name=?, surname=?, email=?, username=?, admin=?, updated_by=? WHERE id=? LIMIT 1"
		_, err := db.Exec(query, name, surname, email, username, admin, sessionUserID, userID)
		if err != nil {
			log.Println("Failed to perform prepared SQL query against database:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Delete user
func usersDelete(c *gin.Context) {
	rowID := c.Param("id")
	msg, err := deleteFromDatabase(rowID, "users", "")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}
