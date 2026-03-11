package main

import (
	"strings"
	"testing"
)

var ginTestModels = []Model{
	{Name: "students", Fields: []Field{{Name: "first_name", Type: "varchar(100)", Required: true}}},
	{Name: "subjects", Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}}},
	{Name: "lessons", Fields: []Field{
		{Name: "date", Type: "date", Required: true},
		{Name: "subject_id", Type: "int", References: "subjects.id"},
	}},
}

func TestGenerateGinRoutes_PackageDeclaration(t *testing.T) {
	out := GenerateGinRoutes(ginTestModels, "routes", "myapp/models")
	if !strings.HasPrefix(out, "package routes\n") {
		t.Errorf("expected 'package routes' header, got start: %q", out[:min(40, len(out))])
	}
}

func TestGenerateGinRoutes_Imports(t *testing.T) {
	out := GenerateGinRoutes(ginTestModels, "routes", "myapp/models")
	for _, want := range []string{
		`"net/http"`,
		`"strconv"`,
		`"github.com/gin-gonic/gin"`,
		`"gorm.io/gorm"`,
		`"myapp/models"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing import %s", want)
		}
	}
}

func TestGenerateGinRoutes_RegisterRoutes(t *testing.T) {
	out := GenerateGinRoutes(ginTestModels, "routes", "myapp/models")

	wants := []string{
		`r.GET("/students", listStudent(db))`,
		`r.GET("/students/:id", getStudent(db))`,
		`r.POST("/students", createStudent(db))`,
		`r.PUT("/students/:id", updateStudent(db))`,
		`r.DELETE("/students/:id", deleteStudent(db))`,
		`r.GET("/subjects", listSubject(db))`,
		`r.GET("/subjects/:id", getSubject(db))`,
		`r.POST("/subjects", createSubject(db))`,
		`r.PUT("/subjects/:id", updateSubject(db))`,
		`r.DELETE("/subjects/:id", deleteSubject(db))`,
		`r.GET("/lessons", listLesson(db))`,
		`r.GET("/lessons/:id", getLesson(db))`,
		`r.POST("/lessons", createLesson(db))`,
		`r.PUT("/lessons/:id", updateLesson(db))`,
		`r.DELETE("/lessons/:id", deleteLesson(db))`,
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("missing route registration: %s", want)
		}
	}
}

func TestGenerateGinRoutes_HandlerSignatures(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "students", Fields: []Field{{Name: "name", Type: "text"}}},
	}, "routes", "myapp/models")

	for _, want := range []string{
		"func listStudent(db *gorm.DB) gin.HandlerFunc",
		"func getStudent(db *gorm.DB) gin.HandlerFunc",
		"func createStudent(db *gorm.DB) gin.HandlerFunc",
		"func updateStudent(db *gorm.DB) gin.HandlerFunc",
		"func deleteStudent(db *gorm.DB) gin.HandlerFunc",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing handler signature: %s", want)
		}
	}
}

func TestGenerateGinRoutes_ModelTypeReference(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "students", Fields: []Field{{Name: "name", Type: "text"}}},
	}, "routes", "myapp/models")

	if !strings.Contains(out, "models.Student") {
		t.Error("expected 'models.Student' type reference in output")
	}
}

func TestGenerateGinRoutes_ModelsImportBasename(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "products", Fields: []Field{{Name: "price", Type: "float"}}},
	}, "routes", "github.com/acme/shop/models")

	if !strings.Contains(out, `"github.com/acme/shop/models"`) {
		t.Error("expected full import path in output")
	}
	if !strings.Contains(out, "models.Product") {
		t.Error("expected 'models.Product' type reference derived from import basename")
	}
}

func TestGenerateGinRoutes_AllHTTPVerbs(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "items", Fields: []Field{{Name: "title", Type: "text"}}},
	}, "routes", "app/models")

	for _, want := range []string{
		"http.StatusOK",
		"http.StatusCreated",
		"http.StatusNoContent",
		"http.StatusBadRequest",
		"http.StatusNotFound",
		"http.StatusInternalServerError",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing HTTP status constant: %s", want)
		}
	}
}

func TestGenerateGinRoutes_SingularRouteNames(t *testing.T) {
	// Handler function names should use singular struct names, not plural table names.
	out := GenerateGinRoutes([]Model{
		{Name: "categories", Fields: []Field{{Name: "name", Type: "text"}}},
	}, "routes", "app/models")

	// "categories" → singular "category" → PascalCase "Category"
	if !strings.Contains(out, "listCategory") {
		t.Error("expected 'listCategory' (singular) handler name for 'categories' model")
	}
	if strings.Contains(out, "listCategories") {
		t.Error("handler name should be singular 'listCategory', not plural 'listCategories'")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
