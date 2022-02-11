package main

import (
	"database/sql"
	"encoding/base32"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"text/template"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/mattn/go-sqlite3"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type UserModel struct {
	Username string `form:"name"`
	Password string `form:"password"`
	Next     string `form:"next"`
}

var mut sync.Mutex

func main() {
	db, err := sql.Open("sqlite3", "./4.db")
	// db.SetMaxOpenConns(1)
	if err != nil {
		log.Fatal("Database connection failed")
	}
	defer db.Close()
	e := echo.New()

	t := &Template{
		templates: template.Must(template.ParseGlob("./views/*.html")),
	}

	e.Renderer = t
	e.Use(middleware.Logger())

	e.GET("/", func(c echo.Context) error {
		type Items struct {
			Items []string
			Name  string
		}

		u, _ := c.Cookie("user")
		if u == nil {
			return c.Redirect(http.StatusTemporaryRedirect, "/login")
		}

		cookieString, err := base32.StdEncoding.DecodeString(u.Value)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}

		items := strings.Split(string(cookieString), "==")
		mut.Lock()
		result, err := db.Query(fmt.Sprintf("SELECT note FROM todos where uid = %s;", items[1]))
		mut.Unlock()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}

		results := Items{Items: []string{}, Name: items[0]}

		for result.Next() {
			note := ""
			result.Scan(&note)

			results.Items = append(results.Items, note)
		}

		return c.Render(http.StatusOK, "index", results)
	})

	e.GET("/register", func(c echo.Context) error {

		return c.Render(http.StatusOK, "register", "/login")
	})

	e.POST("/register", func(c echo.Context) error {
		values := new(UserModel)
		if err := c.Bind(values); err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}
		tx, _ := db.Begin()
		defer tx.Commit()
		mut.Lock()
		_, err := tx.ExecContext(c.Request().Context(), fmt.Sprintf("INSERT INTO userinfo ('username','password','role') VALUES ('%s', '%s', '1');", values.Username, values.Password))
		mut.Unlock()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}

		return c.Render(http.StatusCreated, "redirect", values.Next)
	})

	e.GET("/login", func(c echo.Context) error {
		return c.Render(http.StatusOK, "register", "/")
	})

	e.POST("/login", func(c echo.Context) error {
		values := new(UserModel)
		if err := c.Bind(values); err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}
		mut.Lock()
		result, err := db.Query(fmt.Sprintf("SELECT username, uid,role FROM userinfo where username = '%s' and password = '%s';", values.Username, values.Password))
		mut.Unlock()

		if err != nil {
			return c.JSON(http.StatusInternalServerError, err)
		}

		if result.Next() {
			username := ""
			uid := 0
			role := 0
			result.Scan(&username, &uid, &role)
			cookie := base32.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s==%d==%d", username, uid, role)))
			log.Print(username, uid, role, cookie)

			c.SetCookie(&http.Cookie{Name: "user", Value: cookie})

			return c.Render(http.StatusCreated, "redirect", values.Next)
		}

		return c.Render(http.StatusCreated, "redirect", "/login")
	})

	e.Logger.Fatal(e.Start(":1323"))
}
