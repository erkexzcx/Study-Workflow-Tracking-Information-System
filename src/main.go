package main

import (
	"bufio"
	"crypto/rand"
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

// Store settings map here:
var config map[string]string

// Store DB object here, so it can be reached from everywhere
var db *sql.DB

// Function used in golang templates
func negative(number int) int {
	return number * -1
}

func main() {

	// Read config file:
	var err error
	config, err = readPropertiesFile("./swtis.conf")
	if err != nil {
		log.Fatal("Failed to read configuration file:", err)
	}

	// Connect to the database:
	db, err = sql.Open("mysql", config["dbuser"]+":"+config["dbpassword"]+"@/"+config["dbname"])
	if err != nil {
		log.Fatal("Unable to connect to database:", err)
	}
	defer db.Close()

	// Open doesn't open a connection to database. Validate DSN data:
	err = db.Ping()
	if err != nil {
		log.Fatal("Unable to reach the database:", err)
	}

	// Initiate gin web framework/server:
	r := gin.Default()

	// Use cookies (sessions):
	token := make([]byte, 64)
	rand.Read(token)
	store := cookie.NewStore(token)
	r.Use(sessions.Sessions("swtis", store))

	r.SetFuncMap(template.FuncMap{
		"negative": negative,
	})

	// Define where to take HTML files from
	r.LoadHTMLGlob("./templates/*.gohtml")

	// Define some paths for files/templates:
	r.Static("/css", "./static/css")
	r.Static("/images", "./static/images")
	r.Static("/js", "./static/js")
	r.Static("/fonts", "./static/fonts")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")

	// Define authorized routes group:
	authorized := r.Group("/")
	authorized.Use(authRequired)

	// Define admin-only authorized routes group:
	adminAuthorized := r.Group("/")
	adminAuthorized.Use(adminAuthRequired)

	// Define routes:
	defineRoutes(r, authorized, adminAuthorized)

	// Start web server:
	if err := r.Run(":8080"); err != nil {
		log.Fatal("Unable to start web server:", err)
	}
}

// Define routes for application:
func defineRoutes(r *gin.Engine, authorized *gin.RouterGroup, adminAuthorized *gin.RouterGroup) {

	// Routes without authorization:
	r.GET("/", rootGet)
	r.GET("/login", loginGet)
	r.POST("/login", loginPost)
	r.GET("/logout", logout)

	// Routes with authorization:
	authorized.GET("/app", appGet)
	authorized.GET("/app/activeSubjectsContainer", activeSubjectsContainerGet)
	authorized.GET("/app/linksContainer", linksContainerGet)
	authorized.GET("/app/menuLabels", menuLabelsGet)

	authorized.GET("/settings", settingsPageGet)
	authorized.POST("/settings/themes/:id", settingsThemesJSONPost)
	authorized.POST("/settings/picture", settingsPictureJSONPost)
	authorized.POST("/settings/password", settingsPasswordJSONPost)

	authorized.GET("/progress_board", progressBoardPageGet)
	authorized.GET("/assignment_status/:assignmenID", assignmentStatusGet)
	authorized.POST("/assignment_status/:assignmenID", assignmentStatusPost)

	authorized.GET("/assignments", assignmentsPageGet)
	adminAuthorized.GET("/assignments/:id", assignmentsGet)
	adminAuthorized.POST("/assignments", assignmentsPost)
	adminAuthorized.PUT("/assignments/:id", assignmentsPut)
	adminAuthorized.DELETE("/assignments/:id", assignmentsDelete)

	authorized.GET("/events", eventsPageGet)
	adminAuthorized.GET("/events/:id", eventsGet)
	adminAuthorized.POST("/events", eventsPost)
	adminAuthorized.PUT("/events/:id", eventsPut)
	adminAuthorized.DELETE("/events/:id", eventsDelete)

	authorized.GET("/subjects", subjectsPageGet)
	adminAuthorized.GET("/subjects/:id", subjectsGet)
	adminAuthorized.POST("/subjects", subjectsPost)
	adminAuthorized.PUT("/subjects/:id", subjectsPut)
	adminAuthorized.DELETE("/subjects/:id", subjectsDelete)

	authorized.GET("/teachers", teachersPageGet)
	adminAuthorized.GET("/teachers/:id", teachersGet)
	adminAuthorized.POST("/teachers", teachersPost)
	adminAuthorized.PUT("/teachers/:id", teachersPut)
	adminAuthorized.DELETE("/teachers/:id", teachersDelete)

	authorized.GET("/tutors", tutorsPageGet)
	adminAuthorized.GET("/tutors/:id", tutorsGet)
	adminAuthorized.POST("/tutors", tutorsPost)
	adminAuthorized.PUT("/tutors/:id", tutorsPut)
	adminAuthorized.DELETE("/tutors/:id", tutorsDelete)

	adminAuthorized.GET("/semesters", semestersPageGet)
	adminAuthorized.GET("/semesters/:id", semestersGet)
	adminAuthorized.POST("/semesters", semestersPost)
	adminAuthorized.POST("/semesters/activate/:id", semestersActivatePost)
	adminAuthorized.PUT("/semesters/:id", semestersPut)
	adminAuthorized.DELETE("/semesters/:id", semestersDelete)

	adminAuthorized.GET("/users", usersPageGet)
	adminAuthorized.GET("/users/:id", usersGet)
	adminAuthorized.POST("/users", usersPost)
	adminAuthorized.PUT("/users/:id", usersPut)
	adminAuthorized.DELETE("/users/:id", usersDelete)

	authorized.GET("/links", linksPageGet)
	adminAuthorized.GET("/links/:id", linksGet)
	adminAuthorized.POST("/links", linksPost)
	adminAuthorized.PUT("/links/:id", linksPut)
	adminAuthorized.DELETE("/links/:id", linksDelete)

}

// AuthRequired is a simple middleware to check if user is logged in
func authRequired(c *gin.Context) {
	session := sessions.Default(c)

	if session.Get("id") == nil && c.FullPath() == "/app" {
		// Redirect user to "/" instead of showing 403
		c.Redirect(http.StatusFound, "/app")
		return
	} else if session.Get("id") == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	c.Next()
}

func adminAuthRequired(c *gin.Context) {
	session := sessions.Default(c)

	if session.Get("id") == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	if session.Get("admin") == "0" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}

	c.Next()
}

// Default root path handler:
func rootGet(c *gin.Context) {
	session := sessions.Default(c)

	// If user is already logged in - redirect to homepage:
	if session.Get("id") != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/app")
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, "/login")
}

func readPropertiesFile(filename string) (map[string]string, error) {
	config := map[string]string{}

	if len(filename) == 0 {
		return config, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if equal := strings.Index(line, "="); equal >= 0 {
			if key := strings.TrimSpace(line[:equal]); len(key) > 0 {
				value := ""
				if len(line) > equal {
					value = strings.TrimSpace(line[equal+1:])
				}
				config[key] = value
			}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
		return nil, err
	}

	return config, nil
}
