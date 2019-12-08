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

// Load users ajax page
func tutorsPageGet(c *gin.Context) {
	session := sessions.Default(c)
	admin, _ := strconv.ParseBool(fmt.Sprintf("%v", session.Get("admin")))

	query := `
		SELECT id, title, number, email, address, url, note
		FROM tutors
		ORDER BY title ASC, note ASC

	`
	rows, err := db.Query(query)
	if err != nil {
		log.Println("Failed to fetch users from DB:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	type row struct {
		ID      int
		Title   string
		Number  string
		Email   string
		Address string
		URL     string
		Note    string
	}

	var data []row

	for rows.Next() {
		var r row
		err := rows.Scan(&r.ID, &r.Title, &r.Number, &r.Email, &r.Address, &r.URL, &r.Note)
		if err != nil {
			log.Println("Failed to read links rows from fetched data:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
			return
		}
		data = append(data, r)
	}

	c.HTML(http.StatusOK, "tutors.gohtml", gin.H{
		"admin":  admin,
		"tutors": data,
	})
}

// Get tutor
func tutorsGet(c *gin.Context) {
	tutorID := c.Param("id")

	query := `
		SELECT
			t.title,
			t.number,
			t.email,
			t.address,
			t.url,
			t.note,
			t.created_on,
			IF(uc.name IS NULL, 'Deleted user', CONCAT(uc.name, ' ', uc.surname)) AS created_by,
			t.updated_on,
			IF(uu.name IS NULL, 'Deleted user', CONCAT(uu.name, ' ', uu.surname)) AS updated_by
		FROM tutors AS t
		LEFT JOIN users AS uc ON t.created_by = uc.id
		LEFT JOIN users AS uu ON t.updated_by = uu.id
		WHERE t.id=?
		LIMIT 1
	`

	type row struct {
		Title     string `json:"title"`
		Number    string `json:"number"`
		Email     string `json:"email"`
		Address   string `json:"address"`
		URL       string `json:"url"`
		Note      string `json:"note"`
		CreatedOn string `json:"created_on"`
		CreatedBy string `json:"created_by"`
		UpdatedOn string `json:"updated_on"`
		UpdatedBy string `json:"updated_by"`
	}

	var data row
	err := db.QueryRow(query, tutorID).Scan(
		&data.Title,
		&data.Number,
		&data.Email,
		&data.Address,
		&data.URL,
		&data.Note,
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

// Create tutor
func tutorsPost(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	title := strings.TrimSpace(c.PostForm("title"))
	number := strings.ReplaceAll(c.PostForm("number"), " ", "")
	email := strings.TrimSpace(c.PostForm("email"))
	address := strings.TrimSpace(c.PostForm("address"))
	url := strings.TrimSpace(c.PostForm("url"))
	note := strings.TrimSpace(c.PostForm("note"))

	titleLength := utf8.RuneCountInString(title)
	numberLength := utf8.RuneCountInString(number)
	emailLength := utf8.RuneCountInString(email)
	addressLength := utf8.RuneCountInString(address)
	urlLength := utf8.RuneCountInString(url)
	noteLength := utf8.RuneCountInString(note)

	// Check inputs
	if (titleLength == 0 || titleLength > 60) || (numberLength != 12 && numberLength != 0) || emailLength > 254 ||
		addressLength > 100 || urlLength > 100 || noteLength > 1000 {
		log.Println("Invalid inputs!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "INSERT INTO tutors (title, number, email, address, url, note, created_by, updated_by) VALUES (?,?,?,?,?,?,?,?)"
	_, err := db.Exec(query, title, number, email, address, url, note, sessionUserID, sessionUserID)
	if err != nil {
		log.Println("Failed to perform SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Update tutor
func tutorsPut(c *gin.Context) {
	session := sessions.Default(c)
	sessionUserID := session.Get("id")

	tutorID := c.Param("id")

	title := strings.TrimSpace(c.PostForm("title"))
	number := strings.ReplaceAll(c.PostForm("number"), " ", "")
	email := strings.TrimSpace(c.PostForm("email"))
	address := strings.TrimSpace(c.PostForm("address"))
	url := strings.TrimSpace(c.PostForm("url"))
	note := strings.TrimSpace(c.PostForm("note"))

	titleLength := utf8.RuneCountInString(title)
	numberLength := utf8.RuneCountInString(number)
	emailLength := utf8.RuneCountInString(email)
	addressLength := utf8.RuneCountInString(address)
	urlLength := utf8.RuneCountInString(url)
	noteLength := utf8.RuneCountInString(note)

	// Check inputs
	if (titleLength == 0 || titleLength > 60) || (numberLength != 12 && numberLength != 0) || emailLength > 254 ||
		addressLength > 100 || urlLength > 100 || noteLength > 1000 {
		log.Println("Invalid inputs!")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	query := "UPDATE tutors SET title=?, number=?, email=?, address=?, url=?, note=?, updated_by=? WHERE id=? LIMIT 1"
	_, err := db.Exec(query, title, number, email, address, url, note, sessionUserID, tutorID)
	if err != nil {
		log.Println("Failed to perform SQL query against database:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}

// Delete tutor
func tutorsDelete(c *gin.Context) {
	rowID := c.Param("id")
	msg, err := deleteFromDatabase(rowID, "tutors", "")
	if err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Success"})
}
