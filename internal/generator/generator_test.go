package generator

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

func TestGenerateMain_PackageAndImports(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb"},
		Models:   []Model{{Name: "users", Fields: []Field{{Name: "name", Type: "text"}}}},
	}
	out, err := GenerateMain(cfg, "myapp")
	if err != nil {
		t.Fatalf("GenerateMain: %v", err)
	}

	for _, want := range []string{
		"package main",
		`"github.com/gin-gonic/gin"`,
		`"gorm.io/driver/postgres"`,
		`"gorm.io/gorm"`,
		`"myapp/models"`,
		`"myapp/routes"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing: %s", want)
		}
	}
}

func TestGenerateMain_DBConnection(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "db.example.com", Name: "prod_db"},
		Models:   []Model{{Name: "items", Fields: []Field{{Name: "title", Type: "text"}}}},
	}
	out, err := GenerateMain(cfg, "myapp")
	if err != nil {
		t.Fatalf("GenerateMain: %v", err)
	}

	for _, want := range []string{
		`"db.example.com"`,
		`"prod_db"`,
		`os.Getenv("DB_USER")`,
		`os.Getenv("DB_PASSWORD")`,
		`os.Getenv("DB_PORT")`,
		`gorm.Open(postgres.Open(dsn)`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing: %s", want)
		}
	}
}

func TestGenerateMain_DBPortDefault(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb"},
		Models:   []Model{{Name: "items", Fields: []Field{{Name: "title", Type: "text"}}}},
	}
	out, err := GenerateMain(cfg, "myapp")
	if err != nil {
		t.Fatalf("GenerateMain: %v", err)
	}

	if !strings.Contains(out, `dbPort = "5432"`) {
		t.Error("expected default DB_PORT fallback to 5432")
	}
}

func TestGenerateMain_AutoMigrate(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb"},
		Models: []Model{
			{Name: "users", Fields: []Field{{Name: "name", Type: "text"}}},
			{Name: "posts", Fields: []Field{{Name: "title", Type: "text"}}},
		},
	}
	out, err := GenerateMain(cfg, "myapp")
	if err != nil {
		t.Fatalf("GenerateMain: %v", err)
	}

	for _, want := range []string{
		"db.AutoMigrate(",
		"&models.User{}",
		"&models.Post{}",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing: %s", want)
		}
	}
}

func TestGenerateMain_RouterAndServer(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 3000},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb"},
		Models:   []Model{{Name: "items", Fields: []Field{{Name: "name", Type: "text"}}}},
	}
	out, err := GenerateMain(cfg, "myapp")
	if err != nil {
		t.Fatalf("GenerateMain: %v", err)
	}

	for _, want := range []string{
		"gin.Default()",
		"routes.RegisterRoutes(r, db)",
		`r.Run(":3000")`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing: %s", want)
		}
	}
}

func TestGenerateDockerCompose_Services(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb"},
		Models:   []Model{{Name: "users", Fields: []Field{{Name: "name", Type: "text"}}}},
	}
	out, err := GenerateDockerCompose(cfg)
	if err != nil {
		t.Fatalf("GenerateDockerCompose: %v", err)
	}

	for _, want := range []string{
		"services:",
		"app:",
		"postgres:",
		"image: postgres:16-alpine",
		`"8080:8080"`,
		"DB_HOST: postgres",
		"POSTGRES_DB: mydb",
		"postgres_data:",
		"service_healthy",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing: %s", want)
		}
	}
}

func TestGenerateDockerCompose_PortAndDBName(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "journal", Port: 3000},
		Database: DatabaseConfig{Host: "localhost", Name: "attendance_db"},
		Models:   []Model{{Name: "items", Fields: []Field{{Name: "title", Type: "text"}}}},
	}
	out, err := GenerateDockerCompose(cfg)
	if err != nil {
		t.Fatalf("GenerateDockerCompose: %v", err)
	}

	if !strings.Contains(out, `"3000:3000"`) {
		t.Error("expected port 3000 in app service")
	}
	if !strings.Contains(out, "POSTGRES_DB: attendance_db") {
		t.Error("expected POSTGRES_DB: attendance_db")
	}
	if !strings.Contains(out, "pg_isready") {
		t.Error("expected pg_isready healthcheck")
	}
}

func TestGenerateMain_DBHostEnv(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb"},
		Models:   []Model{{Name: "items", Fields: []Field{{Name: "title", Type: "text"}}}},
	}
	out, err := GenerateMain(cfg, "myapp")
	if err != nil {
		t.Fatalf("GenerateMain: %v", err)
	}

	if !strings.Contains(out, `os.Getenv("DB_HOST")`) {
		t.Error("expected DB_HOST env var reading in generated main.go")
	}
}

func TestGenerateGoMod_ModuleAndGo(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb"},
	}
	out, err := GenerateGoMod(cfg)
	if err != nil {
		t.Fatalf("GenerateGoMod: %v", err)
	}

	if !strings.HasPrefix(out, "module myapp\n") {
		t.Errorf("expected 'module myapp' header, got: %q", out[:min(40, len(out))])
	}
	if !strings.Contains(out, "go 1.21") {
		t.Error("expected go version directive")
	}
}

func TestGenerateGoMod_Dependencies(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "attendance-journal", Port: 3000},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb"},
	}
	out, err := GenerateGoMod(cfg)
	if err != nil {
		t.Fatalf("GenerateGoMod: %v", err)
	}

	for _, want := range []string{
		"github.com/gin-gonic/gin",
		"gorm.io/driver/postgres",
		"gorm.io/gorm",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing dependency: %s", want)
		}
	}
}

func TestGenerateGoMod_ModuleName(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "shop-api", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "shopdb"},
	}
	out, err := GenerateGoMod(cfg)
	if err != nil {
		t.Fatalf("GenerateGoMod: %v", err)
	}

	if !strings.Contains(out, "module shop-api") {
		t.Errorf("expected module name 'shop-api', got: %q", out)
	}
}

func TestGenerateEnv_AllVars(t *testing.T) {
	cfg := &Config{
		App: AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{
			Host:     "db.example.com",
			Port:     5432,
			Name:     "mydb",
			User:     "admin",
			Password: "s3cr3t",
		},
	}
	out, err := GenerateEnv(cfg)
	if err != nil {
		t.Fatalf("GenerateEnv: %v", err)
	}

	for _, want := range []string{
		"DB_HOST=db.example.com",
		"DB_PORT=5432",
		"DB_USER=admin",
		"DB_PASSWORD=s3cr3t",
		"DB_NAME=mydb",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing: %s", want)
		}
	}
}

func TestGenerateEnv_Defaults(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Port: 5432, Name: "mydb", User: "postgres", Password: "secret"},
	}
	out, err := GenerateEnv(cfg)
	if err != nil {
		t.Fatalf("GenerateEnv: %v", err)
	}

	if !strings.Contains(out, "DB_PORT=5432") {
		t.Error("expected DB_PORT=5432")
	}
	if !strings.Contains(out, "DB_USER=postgres") {
		t.Error("expected DB_USER=postgres")
	}
}

func TestGenerateGORMModels_SnakeCaseJSONTags(t *testing.T) {
	models := []Model{
		{Name: "students", Fields: []Field{
			{Name: "first_name", Type: "varchar(100)", Required: true},
			{Name: "email", Type: "varchar(255)", Unique: true},
		}},
	}
	out := GenerateGORMModels(models, "models")

	for _, want := range []string{
		`json:"first_name"`,
		`json:"email"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing json tag %s in output:\n%s", want, out)
		}
	}
}

func TestGenerateGORMModels_BaseStructWithSnakeCaseTags(t *testing.T) {
	models := []Model{
		{Name: "items", Fields: []Field{{Name: "title", Type: "text"}}},
	}
	out := GenerateGORMModels(models, "models")

	for _, want := range []string{
		`json:"id"`,
		`json:"created_at"`,
		`json:"updated_at"`,
		`json:"deleted_at,omitempty"`,
		"type Base struct",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output:\n%s", want, out)
		}
	}

	if strings.Contains(out, "gorm.Model") {
		t.Error("output should not embed gorm.Model directly; expected custom Base struct")
	}
}

func TestGenerateGORMModels_AssociationJSONTag(t *testing.T) {
	models := []Model{
		{Name: "subjects", Fields: []Field{{Name: "name", Type: "text"}}},
		{Name: "lessons", Fields: []Field{
			{Name: "subject_id", Type: "int", References: "subjects.id"},
		}},
	}
	out := GenerateGORMModels(models, "models")

	if !strings.Contains(out, `json:"subject"`) {
		t.Errorf("expected json:\"subject\" for association field, got:\n%s", out)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ── React client generation tests ──────────────────────────────────────────

var clientTestModel = Model{
	Name: "students",
	Fields: []Field{
		{Name: "first_name", Type: "varchar(100)", Required: true},
		{Name: "last_name", Type: "varchar(100)", Required: true},
		{Name: "present", Type: "boolean", Default: false},
	},
}

func TestGenerateReactTypes_Interface(t *testing.T) {
	out := GenerateReactTypes(clientTestModel)

	for _, want := range []string{
		"export interface Student {",
		"id: number;",
		"first_name: string;",
		"last_name: string;",
		"present?: boolean;",
		"created_at: string;",
		"updated_at: string;",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in types output", want)
		}
	}
}

func TestGenerateReactTypes_InputType(t *testing.T) {
	out := GenerateReactTypes(clientTestModel)

	if !strings.Contains(out, "export type CreateStudentInput = {") {
		t.Error("missing CreateStudentInput type")
	}
	// Required fields are non-optional in CreateInput
	if !strings.Contains(out, "first_name: string;") {
		t.Error("expected first_name as required string in CreateStudentInput")
	}
	// ID and timestamps are NOT in CreateInput
	if strings.Contains(out, "id: number") && strings.Count(out, "id: number") > 1 {
		t.Error("id should only appear in interface, not in CreateInput")
	}
}

func TestGenerateReactAPI_Functions(t *testing.T) {
	out := GenerateReactAPI(clientTestModel)

	for _, want := range []string{
		"export async function listStudents(",
		"export async function getStudent(",
		"export async function createStudent(",
		"export async function updateStudent(",
		"export async function deleteStudent(",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in API output", want)
		}
	}
}

func TestGenerateReactAPI_FetchMethods(t *testing.T) {
	out := GenerateReactAPI(clientTestModel)

	for _, want := range []string{
		"method: 'POST'",
		"method: 'PUT'",
		"method: 'DELETE'",
		"'Content-Type': 'application/json'",
		"JSON.stringify(data)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in API output", want)
		}
	}
}

func TestGenerateReactAPI_BaseURL(t *testing.T) {
	out := GenerateReactAPI(clientTestModel)
	if !strings.Contains(out, "const BASE = '/students';") {
		t.Error("expected BASE = '/students'")
	}
}

func TestGenerateReactAPI_TypeImport(t *testing.T) {
	out := GenerateReactAPI(clientTestModel)
	if !strings.Contains(out, "import type { Student, CreateStudentInput } from '../types/student';") {
		t.Error("expected type import from '../types/student'")
	}
}

func TestGenerateReactPage_Component(t *testing.T) {
	out := GenerateReactPage(clientTestModel, nil)

	for _, want := range []string{
		"export default function StudentPage()",
		"useState<Student[]>",
		"useState<Student | null>",
		"useState<CreateStudentInput>",
		"useEffect",
		"handleSubmit",
		"handleDelete",
		"openCreate",
		"openEdit",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in page output", want)
		}
	}
}

func TestGenerateReactPage_Table(t *testing.T) {
	out := GenerateReactPage(clientTestModel, nil)

	for _, want := range []string{
		"<th>id</th>",
		"<th>first_name</th>",
		"<th>last_name</th>",
		"item.id",
		"item.first_name",
		"{item.present ? 'yes' : 'no'}",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in page table", want)
		}
	}
}

func TestGenerateReactPage_Form(t *testing.T) {
	out := GenerateReactPage(clientTestModel, nil)

	// Required text field has required attribute
	if !strings.Contains(out, `type="text"`) {
		t.Error("expected text input for varchar field")
	}
	if !strings.Contains(out, " required") {
		t.Error("expected required attribute on required fields")
	}
	// Boolean field uses checkbox
	if !strings.Contains(out, `type="checkbox"`) {
		t.Error("expected checkbox input for boolean field")
	}
}

func TestGenerateReactApp_Routes(t *testing.T) {
	models := []Model{
		{Name: "students", Fields: []Field{{Name: "name", Type: "text"}}},
		{Name: "subjects", Fields: []Field{{Name: "name", Type: "text"}}},
	}
	out := GenerateReactApp(models)

	for _, want := range []string{
		"import StudentPage from './pages/StudentPage';",
		"import SubjectPage from './pages/SubjectPage';",
		`<Route path="/students" element={<StudentPage />} />`,
		`<Route path="/subjects" element={<SubjectPage />} />`,
		`<NavLink to="/students"`,
		`<NavLink to="/subjects"`,
		"BrowserRouter",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in App.tsx output", want)
		}
	}
}

func TestGenerateReactPackageJSON_Deps(t *testing.T) {
	cfg := &Config{App: AppConfig{Name: "myapp", Port: 8080}}
	out := GenerateReactPackageJSON(cfg)

	for _, want := range []string{
		`"react":`,
		`"react-dom":`,
		`"react-router-dom":`,
		`"vite":`,
		`"typescript":`,
		`"myapp-client"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in package.json", want)
		}
	}
}

func TestGenerateReactViteConfig_Proxy(t *testing.T) {
	cfg := &Config{
		App:    AppConfig{Name: "myapp", Port: 3000},
		Models: []Model{{Name: "students"}, {Name: "subjects"}},
	}
	out := GenerateReactViteConfig(cfg)

	for _, want := range []string{
		"proxy:",
		"'/students': 'http://localhost:3000'",
		"'/subjects': 'http://localhost:3000'",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in vite.config.ts", want)
		}
	}
}

func TestGenerateGORMModels_JSONTags(t *testing.T) {
	models := []Model{
		{Name: "students", Fields: []Field{
			{Name: "first_name", Type: "varchar(100)", Required: true},
			{Name: "score", Type: "int"},
		}},
	}
	out := GenerateGORMModels(models, "models")

	for _, want := range []string{
		`json:"id"`,
		`json:"created_at"`,
		`json:"updated_at"`,
		`json:"first_name"`,
		`json:"score"`,
		"type Base struct",
		"\tBase\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in GORM models output", want)
		}
	}
	// Must NOT embed gorm.Model directly
	if strings.Contains(out, "gorm.Model\n") {
		t.Error("should not embed gorm.Model; use Base struct instead")
	}
}

// ── dev.sh / shutdown.sh generation tests ──────────────────────────────────

func TestGenerateDevScript_ShebangAndSafety(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb", User: "postgres", Password: "secret", Port: 5432},
		Models:   []Model{{Name: "users", Fields: []Field{{Name: "name", Type: "text"}}}},
	}
	out, err := GenerateDevScript(cfg)
	if err != nil {
		t.Fatalf("GenerateDevScript: %v", err)
	}

	for _, want := range []string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in dev.sh output", want)
		}
	}
}

func TestGenerateDevScript_StartsDatabase(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb", User: "dbuser", Password: "secret", Port: 5432},
		Models:   []Model{{Name: "users", Fields: []Field{{Name: "name", Type: "text"}}}},
	}
	out, err := GenerateDevScript(cfg)
	if err != nil {
		t.Fatalf("GenerateDevScript: %v", err)
	}

	for _, want := range []string{
		"docker compose up -d postgres",
		"pg_isready",
		"dbuser",
		"mydb",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in dev.sh output", want)
		}
	}
}

func TestGenerateDevScript_AppliesMigrations(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb", User: "postgres", Password: "secret", Port: 5432},
		Models:   []Model{{Name: "users", Fields: []Field{{Name: "name", Type: "text"}}}},
	}
	out, err := GenerateDevScript(cfg)
	if err != nil {
		t.Fatalf("GenerateDevScript: %v", err)
	}

	if !strings.Contains(out, "migrations/001_initial.up.sql") {
		t.Error("expected migration file reference in dev.sh")
	}
}

func TestGenerateDevScript_StartsServer(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 3000},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb", User: "postgres", Password: "secret", Port: 5432},
		Models:   []Model{{Name: "users", Fields: []Field{{Name: "name", Type: "text"}}}},
	}
	out, err := GenerateDevScript(cfg)
	if err != nil {
		t.Fatalf("GenerateDevScript: %v", err)
	}

	for _, want := range []string{
		"go run .",
		"3000",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in dev.sh output", want)
		}
	}
}

func TestGenerateDevScript_StartsClient(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb", User: "postgres", Password: "secret", Port: 5432},
		Models:   []Model{{Name: "users", Fields: []Field{{Name: "name", Type: "text"}}}},
	}
	out, err := GenerateDevScript(cfg)
	if err != nil {
		t.Fatalf("GenerateDevScript: %v", err)
	}

	for _, want := range []string{
		"npm install",
		"npm run dev",
		"client",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in dev.sh output", want)
		}
	}
}

func TestGenerateDevScript_BackgroundProcesses(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb", User: "postgres", Password: "secret", Port: 5432},
		Models:   []Model{{Name: "users", Fields: []Field{{Name: "name", Type: "text"}}}},
	}
	out, err := GenerateDevScript(cfg)
	if err != nil {
		t.Fatalf("GenerateDevScript: %v", err)
	}

	// Server runs in background so client can also start
	if !strings.Contains(out, "go run . &") {
		t.Error("expected server to run in background ('go run . &')")
	}
	// Trap ensures both processes are cleaned up on exit
	if !strings.Contains(out, "trap") {
		t.Error("expected trap for clean shutdown of background processes")
	}
}

// ── Pagination tests ────────────────────────────────────────────────────────

func TestGenerateGinRoutes_Pagination(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "items", Fields: []Field{{Name: "title", Type: "text"}}},
	}, "routes", "app/models")

	for _, want := range []string{
		`c.DefaultQuery("page", "1")`,
		`c.DefaultQuery("limit", "20")`,
		`db.Offset(offset).Limit(limit).Find(&rows)`,
		`db.Model(&models.Item{}).Count(&total)`,
		`"data": rows`,
		`"total": total`,
		`"page": page`,
		`"limit": limit`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing pagination code: %s", want)
		}
	}
}

func TestGenerateReactAPI_Pagination(t *testing.T) {
	out := GenerateReactAPI(clientTestModel)

	for _, want := range []string{
		"export interface PaginatedStudents {",
		"data: Student[];",
		"total: number;",
		"page = 1, limit = 20",
		"Promise<PaginatedStudents>",
		"?page=${page}&limit=${limit}",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing pagination in API: %s", want)
		}
	}
}

func TestGenerateReactPage_Pagination(t *testing.T) {
	out := GenerateReactPage(clientTestModel, nil)

	for _, want := range []string{
		"useState(1)",
		"setTotal",
		"const limit = 20",
		"load(1)",
		"load(p: number)",
		"res.data",
		"res.total",
		"Prev",
		"Next",
		"Math.ceil(total / limit)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing pagination in page: %s", want)
		}
	}
}

// ── CORS + sslmode tests ────────────────────────────────────────────────────

func TestGenerateMain_CORS(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb"},
		Models:   []Model{{Name: "items", Fields: []Field{{Name: "title", Type: "text"}}}},
	}
	out, err := GenerateMain(cfg, "myapp")
	if err != nil {
		t.Fatalf("GenerateMain: %v", err)
	}

	for _, want := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		`c.Request.Method == "OPTIONS"`,
		"c.AbortWithStatus",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing CORS middleware: %s", want)
		}
	}
}

func TestGenerateMain_SSLMode(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "mydb"},
		Models:   []Model{{Name: "items", Fields: []Field{{Name: "title", Type: "text"}}}},
	}
	out, err := GenerateMain(cfg, "myapp")
	if err != nil {
		t.Fatalf("GenerateMain: %v", err)
	}

	for _, want := range []string{
		`os.Getenv("DB_SSLMODE")`,
		`dbSSLMode = "disable"`,
		`sslmode=%s`,
		`dbSSLMode`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing sslmode in main: %s", want)
		}
	}
}

func TestGenerateEnv_SSLMode(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Port: 5432, Name: "mydb", User: "postgres", Password: "secret"},
	}
	out, err := GenerateEnv(cfg)
	if err != nil {
		t.Fatalf("GenerateEnv: %v", err)
	}

	if !strings.Contains(out, "DB_SSLMODE=disable") {
		t.Error("expected DB_SSLMODE=disable in .env")
	}
}

// ── datetime/timestamp tests ────────────────────────────────────────────────

func TestGenerateReactPage_DatetimeLocalInput(t *testing.T) {
	m := Model{
		Name: "events",
		Fields: []Field{
			{Name: "name", Type: "varchar(100)", Required: true},
			{Name: "starts_at", Type: "datetime", Required: true},
			{Name: "ends_at", Type: "timestamp"},
		},
	}
	out := GenerateReactPage(m, nil)

	if !strings.Contains(out, `type="datetime-local"`) {
		t.Error("expected datetime-local input for datetime field")
	}
	// openEdit should slice to 16 chars (YYYY-MM-DDTHH:MM)
	if !strings.Contains(out, ".slice(0, 16)") {
		t.Error("expected slice(0, 16) for datetime-local value in openEdit")
	}
	// handleSubmit should append :00Z
	if !strings.Contains(out, "':00Z'") {
		t.Error("expected ':00Z' suffix for datetime-local payload")
	}
}

// ── FK dropdown tests ───────────────────────────────────────────────────────

func TestGenerateReactPage_FKDropdown(t *testing.T) {
	allModels := []Model{
		{Name: "subjects", Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}}},
		{Name: "lessons", Fields: []Field{
			{Name: "date", Type: "date", Required: true},
			{Name: "subject_id", Type: "int", References: "subjects.id"},
		}},
	}
	lessonModel := allModels[1]
	out := GenerateReactPage(lessonModel, allModels)

	for _, want := range []string{
		"import type { Subject } from '../types/subject'",
		"import { listSubjects } from '../api/subject'",
		"subjectOptions",
		"listSubjects(1, 1000)",
		`<select`,
		"-- select --",
		"opt.name",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing FK dropdown code: %s", want)
		}
	}
	// The subject_id field should NOT render as a number input
	if strings.Contains(out, `type="number"`) {
		t.Error("FK field should use <select>, not type=number input")
	}
}

func TestGenerateReactPage_FKLabelFallback(t *testing.T) {
	// When referenced model has no "name"/"title" field, use first field
	allModels := []Model{
		{Name: "codes", Fields: []Field{{Name: "value", Type: "varchar(10)", Required: true}}},
		{Name: "items", Fields: []Field{{Name: "code_id", Type: "int", References: "codes.id"}}},
	}
	out := GenerateReactPage(allModels[1], allModels)

	if !strings.Contains(out, "opt.value") {
		t.Error("expected opt.value as label for model with no name/title field")
	}
}

func TestGenerateShutdownScript_DockerDown(t *testing.T) {
	out, err := GenerateShutdownScript()
	if err != nil {
		t.Fatalf("GenerateShutdownScript: %v", err)
	}

	if !strings.HasPrefix(out, "#!/usr/bin/env bash") {
		t.Errorf("expected bash shebang at start, got: %q", out[:min(30, len(out))])
	}
	if !strings.Contains(out, "docker compose down") {
		t.Error("expected 'docker compose down' in shutdown script")
	}
}
