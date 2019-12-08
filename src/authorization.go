package main

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Login handler for GET requests:
func loginGet(c *gin.Context) {
	session := sessions.Default(c)

	// If user is already logged in - redirect to homepage:
	if session.Get("id") != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	c.HTML(http.StatusOK, "login.gohtml", gin.H{"title": config["title"]})
}

// Login handler for POST requests:
func loginPost(c *gin.Context) {
	session := sessions.Default(c)

	// Fail if user is already logged in:
	if session.Get("id") != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Get submitted parameters:
	username := strings.TrimSpace(c.PostForm("username"))
	plainPassword := c.PostForm("password")

	// Check for username and password match, usually from a database
	if username == "" || strings.TrimSpace(plainPassword) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication failed. Check your credentials and try again"})
		return
	}

	//-------------------------------------------

	// Retrieve user details from DB
	query := `
		SELECT
			users.id,
			name,
			surname,
			email,
			password,
			admin,
			picture,
			themes.id AS theme_id,
			themes.code AS theme_code
		FROM users
		INNER JOIN themes ON users.theme = themes.id
		WHERE username = ?
		LIMIT 1
	`

	var id int
	var name string
	var surname string
	var email string
	var hashedPassword string
	var admin bool
	var picture string
	var themeID int
	var themeCode string

	err := db.QueryRow(query, username).Scan(
		&id,
		&name,
		&surname,
		&email,
		&hashedPassword,
		&admin,
		&picture,
		&themeID,
		&themeCode,
	)

	if err != nil {
		// Error - check if it's no rows returned or something else:
		if err != sql.ErrNoRows {
			log.Println("Failed to perform prepared SQL query against database:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication failed. Check your credentials and try again"})
		return
	}

	// Now let's check if user provided correct password:
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication failed. Check your credentials and try again"})
		return
	}

	//-------------------------------------------

	// Save the user data in the session
	session.Set("id", id)
	session.Set("name", name)
	session.Set("surname", surname)
	session.Set("email", email)
	session.Set("username", username)
	session.Set("admin", admin)
	session.Set("picture", picture)
	session.Set("theme_id", themeID)
	session.Set("theme_code", themeCode)
	if err := session.Save(); err != nil {
		log.Println("Failed to save session:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged in"})
}

// login handler for GET requests:
func logout(c *gin.Context) {
	session := sessions.Default(c)

	// If user is already logged out:
	if session.Get("id") == nil {
		c.Redirect(http.StatusTemporaryRedirect, "/")
		return
	}

	// Delete existing session:
	session.Clear()
	if err := session.Save(); err != nil {
		log.Println("Failed to delete session:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, "/")
}
