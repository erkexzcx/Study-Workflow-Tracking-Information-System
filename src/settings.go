package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func settingsPageGet(c *gin.Context) {
	session := sessions.Default(c)
	currentUserThemeString := session.Get("theme_id")
	currentUserTheme, _ := strconv.Atoi(fmt.Sprintf("%v", currentUserThemeString))

	type theme struct {
		ID    int
		Code  string
		Title string
	}

	query := `SELECT id, code, title FROM themes ORDER BY title LIKE "%dark%" DESC, title`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Failed to fetch themes from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	var themes []theme

	for rows.Next() {
		var t theme
		err := rows.Scan(&t.ID, &t.Code, &t.Title)
		if err != nil {
			log.Println("Failed to read themes rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		themes = append(themes, t)
	}

	// It's `session.Get("email")` encoded into MD5 hash:
	var hardcoreEncodedEmail string = fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%v", session.Get("email")))))

	c.HTML(http.StatusOK, "settings.gohtml", gin.H{
		"name":        session.Get("name"),
		"surname":     session.Get("surname"),
		"picture":     session.Get("picture"),
		"emailMD5":    hardcoreEncodedEmail,
		"activeTheme": currentUserTheme,
		"themes":      themes,
	})
}

func settingsThemesJSONPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	// Get submitted parameters:
	themeID := c.Param("id")

	// Validate parameters
	if nmb, err := strconv.Atoi(themeID); err != nil || nmb < 0 {
		log.Println("Provided theme ID is not a valid number!:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "UPDATE users SET theme=? WHERE id=? LIMIT 1"
	_, err := db.Exec(query, themeID, sessionUserID)
	if err != nil {
		log.Println("Failed to perform prepared SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Now we need to update theme code in session, but we only know theme ID, so let's query database
	var themeCode string

	query = `SELECT code FROM themes WHERE id=? LIMIT 1`
	err = db.QueryRow(query, themeID).Scan(&themeCode)
	if err != nil {
		log.Println("Failed to perform prepared SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}
	session.Set("theme_code", themeCode)
	session.Set("theme_id", themeID)
	if err := session.Save(); err != nil {
		log.Println("Failed to save (reload) session:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

func settingsPictureJSONPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	// Get submitted picture URL:
	pictureURL := strings.TrimSpace(c.PostForm("picture"))
	if pictureURL == "" {
		pictureURL = "/images/default_picture.png" // empty URL means default picture
	}

	// Validate parameters
	if !validPictureURL(pictureURL) {
		log.Println("Provided invalid URL for picture!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "UPDATE users SET picture=? WHERE id=? LIMIT 1"
	_, err := db.Exec(query, pictureURL, sessionUserID)
	if err != nil {
		log.Println("Failed to perform prepared SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Now we need to update picture URL in session:
	session.Set("picture", pictureURL)
	if err := session.Save(); err != nil {
		log.Println("Failed to save (reload) session:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

func validPictureURL(pictureURL string) bool {
	if pictureURL == "/images/default_picture.png" {
		return true
	}
	u, err := url.Parse(pictureURL)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func settingsPasswordJSONPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	oldPassword := c.PostForm("old_password")
	newPassword := c.PostForm("new_password")
	reNewPassword := c.PostForm("re_new_password")

	// Client-side protection should prevent non matching new passwords, so 500 on such failure:
	if newPassword != reNewPassword {
		log.Println("New paswords do not match")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Client-side protection should prevent empty passwords, so 500 on such failure:
	if strings.TrimSpace(oldPassword) == "" || strings.TrimSpace(newPassword) == "" || strings.TrimSpace(reNewPassword) == "" {
		log.Println("User provided (one of) passwords empty!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Retrieve user hashed password from DB:
	query := "SELECT password FROM users WHERE id=? LIMIT 1"
	stmtOut, err := db.Prepare(query)
	if err != nil {
		log.Println("Unable to prepare DB query:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	var hashedOldPassword string

	err = stmtOut.QueryRow(sessionUserID).Scan(&hashedOldPassword)

	stmtOut.Close()

	if err != nil {
		// Error - check if it's no rows returned or something else:
		log.Println("Error occurrect while quering data for user password. Either such user ID does not exist or something else:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Now let's check if user provided correct old password:
	err = bcrypt.CompareHashAndPassword([]byte(hashedOldPassword), []byte(oldPassword))
	if err != nil {
		log.Println("User provided incorrect existing password:", err)
		c.JSON(http.StatusConflict, gin.H{"message": "Old password is incorrect"})
		return
	}

	// All checks done, so let's change password. Encrypt it to bcrypt hash:
	newHashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		log.Println("Failed to hash new password:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	// Submit new password hash to DB:
	newHashedPassword := string(newHashedPasswordBytes)

	query = "UPDATE users SET password=? WHERE id=? LIMIT 1"
	_, err = db.Exec(query, newHashedPassword, sessionUserID)
	if err != nil {
		log.Println("Failed to perform prepared SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}
