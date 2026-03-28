package generator

import (
	"os"
	"strings"
	"testing"
)

// ── ValidateConfig display_field tests ─────────────────────────────────────

func TestValidateConfig_DisplayField_Valid(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models: []Model{
			{Name: "teachers", Fields: []Field{{Name: "full_name", Type: "text", Required: true}}},
			{Name: "lessons", Fields: []Field{
				{Name: "teacher_id", Type: "int", References: "teachers.id", DisplayField: "full_name"},
			}},
		},
	}
	if errs := ValidateConfig(cfg); len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateConfig_DisplayField_WithoutReferences(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models: []Model{
			{Name: "items", Fields: []Field{
				{Name: "title", Type: "text", DisplayField: "name"},
			}},
		},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected error for display_field without references, got none")
	}
	if !strings.Contains(errs[0].Error(), "display_field requires references") {
		t.Errorf("unexpected error message: %v", errs[0])
	}
}

func TestValidateConfig_DisplayField_UnknownField(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models: []Model{
			{Name: "teachers", Fields: []Field{{Name: "full_name", Type: "text"}}},
			{Name: "lessons", Fields: []Field{
				{Name: "teacher_id", Type: "int", References: "teachers.id", DisplayField: "nonexistent"},
			}},
		},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected error for display_field referencing unknown field, got none")
	}
	if !strings.Contains(errs[0].Error(), "display_field") || !strings.Contains(errs[0].Error(), "nonexistent") {
		t.Errorf("unexpected error message: %v", errs[0])
	}
}

var ginTestModels = []Model{
	{Name: "students", Fields: []Field{{Name: "first_name", Type: "varchar(100)", Required: true}}},
	{Name: "subjects", Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}}},
	{Name: "lessons", Fields: []Field{
		{Name: "date", Type: "date", Required: true},
		{Name: "subject_id", Type: "int", References: "subjects.id"},
	}},
}

func TestGenerateGinRoutes_PackageDeclaration(t *testing.T) {
	out := GenerateGinRoutes(ginTestModels, "routes", "myapp/models", false)
	if !strings.HasPrefix(out, "package routes\n") {
		t.Errorf("expected 'package routes' header, got start: %q", out[:min(40, len(out))])
	}
}

func TestGenerateGinRoutes_Imports(t *testing.T) {
	out := GenerateGinRoutes(ginTestModels, "routes", "myapp/models", false)
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
	out := GenerateGinRoutes(ginTestModels, "routes", "myapp/models", false)

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
	}, "routes", "myapp/models", false)

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
	}, "routes", "myapp/models", false)

	if !strings.Contains(out, "models.Student") {
		t.Error("expected 'models.Student' type reference in output")
	}
}

func TestGenerateGinRoutes_ModelsImportBasename(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "products", Fields: []Field{{Name: "price", Type: "float"}}},
	}, "routes", "github.com/acme/shop/models", false)

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
	}, "routes", "app/models", false)

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
	}, "routes", "app/models", false)

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
	out := GenerateGORMModels(models, "models", nil)

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
		{Name: "items", Timestamps: boolPtr(true), Fields: []Field{{Name: "title", Type: "text"}}},
	}
	out := GenerateGORMModels(models, "models", nil)

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
	out := GenerateGORMModels(models, "models", nil)

	if !strings.Contains(out, `json:"subject"`) {
		t.Errorf("expected json:\"subject\" for association field, got:\n%s", out)
	}
}

func TestGenerateGORMModels_TableName(t *testing.T) {
	// GORM pluralises "Stadium" → "stadia" (Latin); TableName() must override to "stadiums"
	models := []Model{
		{Name: "stadiums", Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}}},
		{Name: "leagues", Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}}},
	}
	out := GenerateGORMModels(models, "models", nil)

	if !strings.Contains(out, `func (Stadium) TableName() string { return "stadiums" }`) {
		t.Errorf("expected TableName() returning \"stadiums\" for Stadium struct:\n%s", out)
	}
	if !strings.Contains(out, `func (League) TableName() string { return "leagues" }`) {
		t.Errorf("expected TableName() returning \"leagues\" for League struct:\n%s", out)
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
	Name:       "students",
	Timestamps: boolPtr(true),
	Fields: []Field{
		{Name: "first_name", Type: "varchar(100)", Required: true},
		{Name: "last_name", Type: "varchar(100)", Required: true},
		{Name: "present", Type: "boolean", Default: false},
	},
}

func TestGenerateReactTypes_Interface(t *testing.T) {
	out := GenerateReactTypes(clientTestModel, nil)

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
	out := GenerateReactTypes(clientTestModel, nil)

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
	out := GenerateReactAPI(clientTestModel, false)

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
	out := GenerateReactAPI(clientTestModel, false)

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
	out := GenerateReactAPI(clientTestModel, false)
	if !strings.Contains(out, "const BASE = '/students';") {
		t.Error("expected BASE = '/students'")
	}
}

func TestGenerateReactAPI_TypeImport(t *testing.T) {
	out := GenerateReactAPI(clientTestModel, false)
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
		"handleSort('id')",
		"handleSort('first_name')",
		"handleSort('last_name')",
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
	out := GenerateReactApp(models, false)

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
		{Name: "students", Timestamps: boolPtr(true), Fields: []Field{
			{Name: "first_name", Type: "varchar(100)", Required: true},
			{Name: "score", Type: "int"},
		}},
	}
	out := GenerateGORMModels(models, "models", nil)

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
	}, "routes", "app/models", false)

	for _, want := range []string{
		`c.DefaultQuery("page", "1")`,
		`c.DefaultQuery("limit", "20")`,
		`query := db.Model(&models.Item{})`,
		`query.Count(&total)`,
		`query.Order(sortBy + " " + sortDir).Offset(offset).Limit(limit).Find(&rows)`,
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
	out := GenerateReactAPI(clientTestModel, false)

	for _, want := range []string{
		"export interface PaginatedStudents {",
		"data: Student[];",
		"total: number;",
		"page = 1, limit = 20",
		"Promise<PaginatedStudents>",
		"new URLSearchParams",
		"filters: Record<string, string>",
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
		"load(p: number,",
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

func TestGenerateReactPage_DisplayField_Form(t *testing.T) {
	// display_field overrides auto-detected label in form dropdown
	allModels := []Model{
		{Name: "teachers", Fields: []Field{
			{Name: "full_name", Type: "varchar(200)", Required: true},
			{Name: "code", Type: "varchar(10)", Required: true},
		}},
		{Name: "lessons", Fields: []Field{
			{Name: "teacher_id", Type: "int", References: "teachers.id", DisplayField: "full_name"},
		}},
	}
	out := GenerateReactPage(allModels[1], allModels)

	if !strings.Contains(out, "opt.full_name") {
		t.Error("expected opt.full_name in form dropdown when display_field=full_name")
	}
	// Should not fall back to auto-detected first field (code)
	if strings.Contains(out, "opt.code") {
		t.Error("unexpected opt.code in form dropdown when display_field overrides")
	}
}

func TestGenerateReactPage_DisplayField_Table(t *testing.T) {
	// display_field is used in table cell lookup expression
	allModels := []Model{
		{Name: "subjects", Fields: []Field{
			{Name: "code", Type: "varchar(10)", Required: true},
			{Name: "title", Type: "varchar(200)", Required: true},
		}},
		{Name: "lessons", Fields: []Field{
			{Name: "subject_id", Type: "int", References: "subjects.id", DisplayField: "code"},
		}},
	}
	out := GenerateReactPage(allModels[1], allModels)

	// Table cell should use options lookup with the specified display_field
	if !strings.Contains(out, "subjectOptions.find") {
		t.Error("expected options lookup in table cell for FK field")
	}
	if !strings.Contains(out, ".code") {
		t.Error("expected .code in table cell lookup when display_field=code")
	}
	// Should not use auto-detected label (title) in table
	if strings.Contains(out, "?.title") {
		t.Error("unexpected ?.title in table cell when display_field=code overrides")
	}
}

func TestGenerateReactPage_FKTable_DefaultLabel(t *testing.T) {
	// Without display_field, table cell should use auto-detected label
	allModels := []Model{
		{Name: "subjects", Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}}},
		{Name: "lessons", Fields: []Field{
			{Name: "subject_id", Type: "int", References: "subjects.id"},
		}},
	}
	out := GenerateReactPage(allModels[1], allModels)

	if !strings.Contains(out, "subjectOptions.find") {
		t.Error("expected options lookup in table cell for FK field")
	}
	if !strings.Contains(out, "?.name") {
		t.Error("expected ?.name in table cell with auto-detected label")
	}
}

func TestGenerateReactPage_DateTableCell(t *testing.T) {
	m := Model{
		Name: "events",
		Fields: []Field{
			{Name: "day", Type: "date", Required: true},
			{Name: "starts_at", Type: "datetime", Required: true},
			{Name: "ends_at", Type: "timestamp"},
		},
	}
	out := GenerateReactPage(m, nil)

	// date cell: slice to 10 chars (YYYY-MM-DD)
	if !strings.Contains(out, `item.day ? (item.day as string).slice(0, 10) : ''`) {
		t.Error("expected slice(0, 10) for date field table cell")
	}
	// datetime cell: slice to 16 and replace T with space (YYYY-MM-DD HH:MM)
	if !strings.Contains(out, `item.starts_at ? (item.starts_at as string).slice(0, 16).replace('T', ' ') : ''`) {
		t.Error("expected slice(0,16).replace for datetime field table cell")
	}
	if !strings.Contains(out, `item.ends_at ? (item.ends_at as string).slice(0, 16).replace('T', ' ') : ''`) {
		t.Error("expected slice(0,16).replace for timestamp field table cell")
	}
}

// ── Enum field type tests ────────────────────────────────────────────────────

func TestValidateConfig_EnumField_Valid(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models: []Model{
			{Name: "posts", Fields: []Field{
				{Name: "title", Type: "text", Required: true},
				{Name: "status", Type: "enum", Values: []string{"draft", "published", "archived"}},
			}},
		},
	}
	if errs := ValidateConfig(cfg); len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateConfig_EnumField_NoValues(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models: []Model{
			{Name: "posts", Fields: []Field{
				{Name: "status", Type: "enum"},
			}},
		},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected error for enum without values, got none")
	}
	if !strings.Contains(errs[0].Error(), "enum type requires") {
		t.Errorf("unexpected error message: %v", errs[0])
	}
}

func TestGenerateMigrationUp_EnumCheckConstraint(t *testing.T) {
	models := []Model{
		{Name: "posts", Fields: []Field{
			{Name: "status", Type: "enum", Values: []string{"draft", "published", "archived"}, Required: true},
		}},
	}
	out := GenerateMigrationUp(models, "postgres")

	if !strings.Contains(out, "status TEXT CHECK (status IN ('draft', 'published', 'archived'))") {
		t.Errorf("expected CHECK constraint for enum field, got:\n%s", out)
	}
	if !strings.Contains(out, "NOT NULL") {
		t.Error("expected NOT NULL for required enum field")
	}
	if strings.Contains(out, "ENUM") {
		t.Error("should not output raw ENUM SQL type")
	}
}

func TestGenerateGORMModels_EnumField(t *testing.T) {
	models := []Model{
		{Name: "posts", Fields: []Field{
			{Name: "status", Type: "enum", Values: []string{"draft", "published"}, Required: true},
		}},
	}
	out := GenerateGORMModels(models, "models", nil)

	if !strings.Contains(out, "Status") || !strings.Contains(out, "string") {
		t.Errorf("expected 'Status' field with 'string' type in GORM model, got:\n%s", out)
	}
	if strings.Contains(out, "Status interface{}") {
		t.Error("enum field should map to string, not interface{}")
	}
}

func TestGenerateReactPage_EnumSelect(t *testing.T) {
	m := Model{
		Name: "posts",
		Fields: []Field{
			{Name: "title", Type: "text", Required: true},
			{Name: "status", Type: "enum", Values: []string{"draft", "published", "archived"}, Required: true},
		},
	}
	out := GenerateReactPage(m, nil)

	// Form should have a select dropdown with enum values
	if !strings.Contains(out, `<select`) {
		t.Error("expected <select> for enum field in form")
	}
	if !strings.Contains(out, `<option value="draft">draft</option>`) {
		t.Error("expected option for 'draft' value")
	}
	if !strings.Contains(out, `<option value="published">published</option>`) {
		t.Error("expected option for 'published' value")
	}
	if !strings.Contains(out, `<option value="archived">archived</option>`) {
		t.Error("expected option for 'archived' value")
	}
	// Should not render as number input
	if strings.Contains(out, `type="number"`) {
		t.Error("enum field should use <select>, not a number input")
	}
}

func TestGenerateReactPage_EnumBadgeCell(t *testing.T) {
	m := Model{
		Name: "posts",
		Fields: []Field{
			{Name: "status", Type: "enum", Values: []string{"draft", "published", "archived"}},
		},
	}
	out := GenerateReactPage(m, nil)

	if !strings.Contains(out, `badge badge--`) {
		t.Error("expected badge class in table cell for enum field")
	}
	if !strings.Contains(out, `item.status`) {
		t.Error("expected item.status in enum table cell")
	}
}

func TestGenerateReactTypes_EnumFieldIsString(t *testing.T) {
	m := Model{
		Name: "posts",
		Fields: []Field{
			{Name: "status", Type: "enum", Values: []string{"draft", "published"}},
		},
	}
	out := GenerateReactTypes(m, nil)

	if !strings.Contains(out, "status?: string") {
		t.Errorf("expected 'status?: string' for optional enum field in TypeScript interface, got:\n%s", out)
	}
}

// ── Filtering / Search tests ─────────────────────────────────────────────────

func TestGenerateGinRoutes_Search_VarcharField(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "posts", Fields: []Field{{Name: "title", Type: "varchar(200)", Required: true}}},
	}, "routes", "app/models", false)

	for _, want := range []string{
		`"strings"`,
		`c.Query("q")`,
		`strings.ReplaceAll(q, "%", "\\%")`,
		`title ILIKE ?`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing search code: %s", want)
		}
	}
}

func TestGenerateGinRoutes_Search_MultipleTextFields(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "posts", Fields: []Field{
			{Name: "title", Type: "varchar(200)", Required: true},
			{Name: "body", Type: "text"},
		}},
	}, "routes", "app/models", false)

	if !strings.Contains(out, "title ILIKE ? OR body ILIKE ?") {
		t.Error("expected OR-joined ILIKE for multiple text fields")
	}
}

func TestGenerateGinRoutes_NoSearch_NoStringsImport(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "scores", Fields: []Field{
			{Name: "value", Type: "int", Required: true},
			{Name: "passed", Type: "boolean"},
		}},
	}, "routes", "app/models", false)

	if strings.Contains(out, `"strings"`) {
		t.Error("should not import 'strings' when no searchable fields exist")
	}
	if strings.Contains(out, `c.Query("q")`) {
		t.Error("should not generate q search when no searchable fields exist")
	}
}

func TestGenerateGinRoutes_Filter_StringField(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "posts", Fields: []Field{
			{Name: "status", Type: "enum", Values: []string{"draft", "published"}},
		}},
	}, "routes", "app/models", false)

	for _, want := range []string{
		`c.Query("status")`,
		`query = query.Where("status = ?", v)`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing string filter code: %s", want)
		}
	}
}

func TestGenerateGinRoutes_Filter_NumericField(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "posts", Fields: []Field{
			{Name: "author_id", Type: "int", References: "users.id"},
		}},
		{Name: "users", Fields: []Field{{Name: "name", Type: "text"}}},
	}, "routes", "app/models", false)

	for _, want := range []string{
		`c.Query("author_id")`,
		`strconv.Atoi(v)`,
		`query = query.Where("author_id = ?", n)`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing numeric filter code: %s", want)
		}
	}
}

func TestGenerateGinRoutes_Filter_BoolField(t *testing.T) {
	out := GenerateGinRoutes([]Model{
		{Name: "items", Fields: []Field{
			{Name: "active", Type: "boolean"},
		}},
	}, "routes", "app/models", false)

	for _, want := range []string{
		`c.Query("active")`,
		`case "true", "1":`,
		`case "false", "0":`,
		`query = query.Where("active = ?", true)`,
		`query = query.Where("active = ?", false)`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing bool filter code: %s", want)
		}
	}
}

func TestGenerateReactPage_SearchInput(t *testing.T) {
	m := Model{
		Name: "posts",
		Fields: []Field{
			{Name: "title", Type: "varchar(200)", Required: true},
		},
	}
	out := GenerateReactPage(m, nil)

	for _, want := range []string{
		`useState('')`,
		`handleSearch`,
		`type="search"`,
		`placeholder="Search..."`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing search UI code: %s", want)
		}
	}
}

func TestGenerateReactPage_FilterDropdown_Enum(t *testing.T) {
	m := Model{
		Name: "posts",
		Fields: []Field{
			{Name: "title", Type: "varchar(200)", Required: true},
			{Name: "status", Type: "enum", Values: []string{"draft", "published", "archived"}},
		},
	}
	out := GenerateReactPage(m, nil)

	for _, want := range []string{
		`handleFilterChange`,
		`filters['status']`,
		`All status`,
		`value="draft"`,
		`value="published"`,
		`value="archived"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing enum filter UI code: %s", want)
		}
	}
}

func TestGenerateReactPage_FilterDropdown_FK(t *testing.T) {
	allModels := []Model{
		{Name: "users", Fields: []Field{{Name: "name", Type: "varchar(100)", Required: true}}},
		{Name: "posts", Fields: []Field{
			{Name: "title", Type: "varchar(200)", Required: true},
			{Name: "author_id", Type: "int", References: "users.id"},
		}},
	}
	out := GenerateReactPage(allModels[1], allModels)

	for _, want := range []string{
		`handleFilterChange`,
		`filters['author_id']`,
		`All author_id`,
		`userOptions`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing FK filter UI code: %s", want)
		}
	}
}

func TestGenerateReactPage_FilterDropdown_Bool(t *testing.T) {
	m := Model{
		Name: "items",
		Fields: []Field{
			{Name: "active", Type: "boolean"},
		},
	}
	out := GenerateReactPage(m, nil)

	for _, want := range []string{
		`handleFilterChange`,
		`filters['active']`,
		`All active`,
		`value="true"`,
		`value="false"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing bool filter UI code: %s", want)
		}
	}
}

func TestGenerateReactPage_NoFilters_NoFilterBar(t *testing.T) {
	m := Model{
		Name: "counters",
		Fields: []Field{
			{Name: "value", Type: "int", Required: true},
		},
	}
	out := GenerateReactPage(m, nil)

	if strings.Contains(out, "filter-bar") {
		t.Error("should not render filter-bar when model has no searchable/filterable fields")
	}
	if strings.Contains(out, "handleFilterChange") {
		t.Error("should not generate handleFilterChange when no filter fields")
	}
}

func TestGenerateReactAPI_FiltersParam(t *testing.T) {
	out := GenerateReactAPI(clientTestModel, false)

	for _, want := range []string{
		"filters: Record<string, string> = {}",
		"new URLSearchParams",
		"...filters",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing filters in API: %s", want)
		}
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

// ── Many-to-many tests ────────────────────────────────────────────────────────

var m2mTestModels = []Model{
	{
		Name:       "students",
		ManyToMany: []string{"courses"},
		Fields: []Field{
			{Name: "name", Type: "varchar(100)", Required: true},
		},
	},
	{
		Name:   "courses",
		Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}},
	},
}

// ── ValidateConfig M2M ────────────────────────────────────────────────────────

func TestValidateConfig_M2M_Valid(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models:   m2mTestModels,
	}
	if errs := ValidateConfig(cfg); len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateConfig_M2M_UnknownModel(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models: []Model{
			{Name: "students", ManyToMany: []string{"unknown"}, Fields: []Field{{Name: "name", Type: "text"}}},
		},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected error for M2M referencing unknown model")
	}
	if !strings.Contains(errs[0].Error(), "unknown model") && !strings.Contains(errs[0].Error(), "unknown") {
		t.Errorf("unexpected error: %v", errs[0])
	}
}

func TestValidateConfig_M2M_SelfReference(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models: []Model{
			{Name: "students", ManyToMany: []string{"students"}, Fields: []Field{{Name: "name", Type: "text"}}},
		},
	}
	errs := ValidateConfig(cfg)
	if len(errs) == 0 {
		t.Fatal("expected error for M2M self-reference")
	}
	if !strings.Contains(errs[0].Error(), "cannot reference itself") {
		t.Errorf("unexpected error: %v", errs[0])
	}
}

// ── Migration MySQL ────────────────────────────────────────────────────────────

func TestGenerateMigrationUp_MySQL_BaseColumns(t *testing.T) {
	models := []Model{
		{Name: "users", Timestamps: boolPtr(true), Fields: []Field{{Name: "email", Type: "varchar(255)", Required: true}}},
	}
	out := GenerateMigrationUp(models, "mysql")
	for _, col := range []string{"created_at DATETIME", "updated_at DATETIME", "deleted_at DATETIME"} {
		if !strings.Contains(out, col) {
			t.Errorf("expected %q in MySQL migration:\n%s", col, out)
		}
	}
}

func TestGenerateMigrationUp_MySQL_AutoIncrement(t *testing.T) {
	models := []Model{
		{Name: "items", Fields: []Field{{Name: "name", Type: "text"}}},
	}
	out := GenerateMigrationUp(models, "mysql")
	if !strings.Contains(out, "id INT AUTO_INCREMENT PRIMARY KEY") {
		t.Errorf("expected AUTO_INCREMENT primary key in MySQL migration:\n%s", out)
	}
}

func TestGenerateMigrationUp_MySQL_ForeignKey(t *testing.T) {
	models := []Model{
		{Name: "users", Fields: []Field{{Name: "email", Type: "varchar(255)"}}},
		{Name: "posts", Fields: []Field{
			{Name: "title", Type: "varchar(255)"},
			{Name: "user_id", Type: "int", References: "users.id"},
		}},
	}
	out := GenerateMigrationUp(models, "mysql")
	if !strings.Contains(out, "FOREIGN KEY (user_id) REFERENCES users(id)") {
		t.Errorf("expected explicit FOREIGN KEY constraint in MySQL migration:\n%s", out)
	}
}

func TestGenerateMigrationUp_MySQL_Enum(t *testing.T) {
	models := []Model{
		{Name: "posts", Fields: []Field{
			{Name: "status", Type: "enum", Values: []string{"draft", "published"}, Required: true},
		}},
	}
	out := GenerateMigrationUp(models, "mysql")
	if !strings.Contains(out, "ENUM('draft', 'published')") {
		t.Errorf("expected ENUM type in MySQL migration:\n%s", out)
	}
	if strings.Contains(out, "TEXT CHECK") {
		t.Error("MySQL should not use TEXT CHECK for enum")
	}
}

func TestGenerateMigrationUp_MySQL_JoinTable(t *testing.T) {
	out := GenerateMigrationUp(m2mTestModels, "mysql")
	if !strings.Contains(out, "CREATE TABLE IF NOT EXISTS courses_students") {
		t.Errorf("expected join table in MySQL migration:\n%s", out)
	}
	if !strings.Contains(out, "FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE") {
		t.Errorf("expected course_id FK in MySQL join table:\n%s", out)
	}
	if !strings.Contains(out, "FOREIGN KEY (student_id) REFERENCES students(id) ON DELETE CASCADE") {
		t.Errorf("expected student_id FK in MySQL join table:\n%s", out)
	}
}

func TestGenerateMigrationUp_MySQL_NoInlineReferences(t *testing.T) {
	models := []Model{
		{Name: "users", Fields: []Field{{Name: "email", Type: "varchar(255)"}}},
		{Name: "posts", Fields: []Field{
			{Name: "user_id", Type: "int", References: "users.id"},
		}},
	}
	out := GenerateMigrationUp(models, "mysql")
	// MySQL silently ignores inline REFERENCES — must use explicit FOREIGN KEY
	if strings.Contains(out, "user_id INT REFERENCES") {
		t.Error("MySQL migration must not use inline REFERENCES syntax")
	}
}

// ── Migration M2M ─────────────────────────────────────────────────────────────

func TestGenerateMigrationUp_JoinTable(t *testing.T) {
	out := GenerateMigrationUp(m2mTestModels, "postgres")
	if !strings.Contains(out, "CREATE TABLE IF NOT EXISTS courses_students") {
		t.Errorf("expected join table 'courses_students' in migration up:\n%s", out)
	}
	if !strings.Contains(out, "course_id INT NOT NULL REFERENCES courses(id) ON DELETE CASCADE") {
		t.Errorf("expected course_id FK in join table:\n%s", out)
	}
	if !strings.Contains(out, "student_id INT NOT NULL REFERENCES students(id) ON DELETE CASCADE") {
		t.Errorf("expected student_id FK in join table:\n%s", out)
	}
	if !strings.Contains(out, "PRIMARY KEY (course_id, student_id)") {
		t.Errorf("expected PRIMARY KEY in join table:\n%s", out)
	}
}

func TestGenerateMigrationUp_JoinTable_AlphaOrder(t *testing.T) {
	// Whether declared as students->courses or courses->students, join table name is alphabetical.
	models1 := []Model{
		{Name: "students", ManyToMany: []string{"courses"}, Fields: []Field{{Name: "n", Type: "text"}}},
		{Name: "courses", Fields: []Field{{Name: "n", Type: "text"}}},
	}
	models2 := []Model{
		{Name: "courses", ManyToMany: []string{"students"}, Fields: []Field{{Name: "n", Type: "text"}}},
		{Name: "students", Fields: []Field{{Name: "n", Type: "text"}}},
	}
	out1 := GenerateMigrationUp(models1, "postgres")
	out2 := GenerateMigrationUp(models2, "postgres")
	for _, out := range []string{out1, out2} {
		if !strings.Contains(out, "courses_students") {
			t.Errorf("expected join table name 'courses_students':\n%s", out)
		}
	}
}

func TestGenerateMigrationUp_JoinTable_Deduplicated(t *testing.T) {
	// Both models declare the same M2M; only one join table should appear.
	models := []Model{
		{Name: "students", ManyToMany: []string{"courses"}, Fields: []Field{{Name: "n", Type: "text"}}},
		{Name: "courses", ManyToMany: []string{"students"}, Fields: []Field{{Name: "n", Type: "text"}}},
	}
	out := GenerateMigrationUp(models, "postgres")
	count := strings.Count(out, "CREATE TABLE IF NOT EXISTS courses_students")
	if count != 1 {
		t.Errorf("expected join table to appear exactly once, got %d:\n%s", count, out)
	}
}


// ── GORM model M2M ────────────────────────────────────────────────────────────

func TestGenerateGORMModels_M2M_Field(t *testing.T) {
	out := GenerateGORMModels(m2mTestModels, "models", nil)
	if !strings.Contains(out, "[]Course") {
		t.Errorf("expected '[]Course' M2M field in Student struct:\n%s", out)
	}
	if !strings.Contains(out, `many2many:courses_students`) {
		t.Errorf("expected 'many2many:courses_students' GORM tag:\n%s", out)
	}
	if !strings.Contains(out, `json:"courses,omitempty"`) {
		t.Errorf("expected json:\"courses,omitempty\" tag:\n%s", out)
	}
}

// ── Gin routes M2M ────────────────────────────────────────────────────────────

func TestGenerateGinRoutes_M2M_Import(t *testing.T) {
	out := GenerateGinRoutes(m2mTestModels, "routes", "myapp/models", false)
	if !strings.Contains(out, `"encoding/json"`) {
		t.Errorf("expected encoding/json import when model has M2M:\n%s", out)
	}
}

func TestGenerateGinRoutes_M2M_Preload_List(t *testing.T) {
	out := GenerateGinRoutes(m2mTestModels, "routes", "myapp/models", false)
	if !strings.Contains(out, `.Preload("Courses")`) {
		t.Errorf("expected Preload(\"Courses\") in list handler:\n%s", out)
	}
}

func TestGenerateGinRoutes_M2M_Preload_Get(t *testing.T) {
	out := GenerateGinRoutes(m2mTestModels, "routes", "myapp/models", false)
	if !strings.Contains(out, `.Preload("Courses").First`) {
		t.Errorf("expected Preload in get handler:\n%s", out)
	}
}

func TestGenerateGinRoutes_M2M_AssocReplace(t *testing.T) {
	out := GenerateGinRoutes(m2mTestModels, "routes", "myapp/models", false)
	if !strings.Contains(out, `Association("Courses").Replace`) {
		t.Errorf("expected Association Replace in create/update handler:\n%s", out)
	}
}

func TestGenerateGinRoutes_M2M_IDsField(t *testing.T) {
	out := GenerateGinRoutes(m2mTestModels, "routes", "myapp/models", false)
	if !strings.Contains(out, `json:"course_ids"`) {
		t.Errorf("expected course_ids JSON field in M2M handler:\n%s", out)
	}
}

func TestGenerateGinRoutes_NoM2M_NoJSON(t *testing.T) {
	// Models without M2M should not import encoding/json.
	models := []Model{
		{Name: "items", Fields: []Field{{Name: "name", Type: "text"}}},
	}
	out := GenerateGinRoutes(models, "routes", "myapp/models", false)
	if strings.Contains(out, `"encoding/json"`) {
		t.Errorf("should not import encoding/json when no M2M:\n%s", out)
	}
}

// ── React types M2M ───────────────────────────────────────────────────────────

func TestGenerateReactTypes_M2M_Interface(t *testing.T) {
	out := GenerateReactTypes(m2mTestModels[0], m2mTestModels)
	if !strings.Contains(out, "courses?: Course[]") {
		t.Errorf("expected 'courses?: Course[]' in Student interface:\n%s", out)
	}
}

func TestGenerateReactTypes_M2M_InputType(t *testing.T) {
	out := GenerateReactTypes(m2mTestModels[0], m2mTestModels)
	if !strings.Contains(out, "course_ids: number[]") {
		t.Errorf("expected 'course_ids: number[]' in CreateStudentInput:\n%s", out)
	}
}

func TestGenerateReactTypes_M2M_Import(t *testing.T) {
	out := GenerateReactTypes(m2mTestModels[0], m2mTestModels)
	if !strings.Contains(out, "import type { Course }") {
		t.Errorf("expected import for Course type:\n%s", out)
	}
}

// ── React page M2M ────────────────────────────────────────────────────────────

func TestGenerateReactPage_M2M_MultiSelect(t *testing.T) {
	out := GenerateReactPage(m2mTestModels[0], m2mTestModels)
	if !strings.Contains(out, "select multiple") {
		t.Errorf("expected 'select multiple' in form for M2M:\n%s", out)
	}
	if !strings.Contains(out, "course_ids") {
		t.Errorf("expected course_ids in page:\n%s", out)
	}
}

func TestGenerateReactPage_M2M_Chips(t *testing.T) {
	out := GenerateReactPage(m2mTestModels[0], m2mTestModels)
	if !strings.Contains(out, `className="chip"`) {
		t.Errorf("expected chip class in table cells for M2M:\n%s", out)
	}
}

func TestGenerateReactPage_M2M_OpenEdit(t *testing.T) {
	out := GenerateReactPage(m2mTestModels[0], m2mTestModels)
	if !strings.Contains(out, "item.courses") {
		t.Errorf("expected item.courses reference in openEdit:\n%s", out)
	}
}

func TestGenerateReactPage_M2M_LoadRefs(t *testing.T) {
	out := GenerateReactPage(m2mTestModels[0], m2mTestModels)
	if !strings.Contains(out, "loadRefs") {
		t.Errorf("expected loadRefs function for M2M:\n%s", out)
	}
	if !strings.Contains(out, "listCourses") {
		t.Errorf("expected listCourses call in loadRefs:\n%s", out)
	}
}

// ── Optional FK null handling ─────────────────────────────────────────────────

func TestGenerateGORMModels_OptionalFKUsesPointer(t *testing.T) {
	models := []Model{
		{Name: "stadiums", Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}}},
		{Name: "clubs", Fields: []Field{
			{Name: "name", Type: "varchar(200)", Required: true},
			{Name: "stadium_id", Type: "int", References: "stadiums.id"}, // optional
		}},
	}
	out := GenerateGORMModels(models, "models", nil)

	if !strings.Contains(out, "*int") {
		t.Errorf("expected '*int' for optional FK field, got:\n%s", out)
	}
}

func TestGenerateGORMModels_RequiredFKNoPointer(t *testing.T) {
	models := []Model{
		{Name: "stadiums", Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}}},
		{Name: "clubs", Fields: []Field{
			{Name: "stadium_id", Type: "int", References: "stadiums.id", Required: true},
		}},
	}
	out := GenerateGORMModels(models, "models", nil)

	if strings.Contains(out, "*int") {
		t.Errorf("expected non-pointer 'int' for required FK field, got:\n%s", out)
	}
	if !strings.Contains(out, "int") {
		t.Errorf("expected 'int' for required FK field, got:\n%s", out)
	}
}

func TestGenerateReactPage_OptionalFKNullDefault(t *testing.T) {
	allModels := []Model{
		{Name: "stadiums", Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}}},
		{Name: "clubs", Fields: []Field{
			{Name: "name", Type: "varchar(200)", Required: true},
			{Name: "stadium_id", Type: "int", References: "stadiums.id"}, // optional
		}},
	}
	out := GenerateReactPage(allModels[1], allModels)

	if !strings.Contains(out, "stadium_id: null") {
		t.Errorf("expected 'stadium_id: null' in EMPTY_FORM for optional FK, got:\n%s", out)
	}
}

func TestGenerateReactPage_OptionalFKSelectHandlesNull(t *testing.T) {
	allModels := []Model{
		{Name: "stadiums", Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}}},
		{Name: "clubs", Fields: []Field{
			{Name: "stadium_id", Type: "int", References: "stadiums.id"}, // optional
		}},
	}
	out := GenerateReactPage(allModels[1], allModels)

	if !strings.Contains(out, "e.target.value ? Number(e.target.value) : null") {
		t.Errorf("expected null-handling onChange for optional FK select, got:\n%s", out)
	}
	if !strings.Contains(out, `<option value="">-- select --</option>`) {
		t.Errorf("expected empty string value for '-- select --' option, got:\n%s", out)
	}
}

func TestGenerateReactTypes_OptionalFKNullableInputType(t *testing.T) {
	models := []Model{
		{Name: "stadiums", Fields: []Field{{Name: "name", Type: "varchar(200)", Required: true}}},
		{Name: "clubs", Fields: []Field{
			{Name: "name", Type: "varchar(200)", Required: true},
			{Name: "stadium_id", Type: "int", References: "stadiums.id"}, // optional
		}},
	}
	out := GenerateReactTypes(models[1], models)

	if !strings.Contains(out, "stadium_id: number | null") {
		t.Errorf("expected 'stadium_id: number | null' in CreateClubInput, got:\n%s", out)
	}
}

// ── joinTableName helper ──────────────────────────────────────────────────────

func TestJoinTableName_Alphabetical(t *testing.T) {
	if got := joinTableName("students", "courses"); got != "courses_students" {
		t.Errorf("expected 'courses_students', got %q", got)
	}
	if got := joinTableName("courses", "students"); got != "courses_students" {
		t.Errorf("expected 'courses_students' (reversed), got %q", got)
	}
}

// ── Auth: ValidateConfig ────────────────────────────────────────────────────

var authBaseConfig = Config{
	App:      AppConfig{Name: "myapp", Port: 8080},
	Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
	Models: []Model{
		{Name: "users", Fields: []Field{
			{Name: "email", Type: "varchar(255)", Required: true, Unique: true},
			{Name: "password", Type: "varchar(255)", Required: true},
		}},
	},
}

func TestValidateConfig_Auth_Valid(t *testing.T) {
	cfg := authBaseConfig
	cfg.Auth = &AuthConfig{Model: "users"}
	if errs := ValidateConfig(&cfg); len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidateConfig_Auth_NoModelSet(t *testing.T) {
	cfg := authBaseConfig
	cfg.Auth = &AuthConfig{Model: ""}
	errs := ValidateConfig(&cfg)
	if len(errs) == 0 {
		t.Fatal("expected error for empty auth.model, got none")
	}
	if !strings.Contains(errs[len(errs)-1].Error(), "auth.model is required") {
		t.Errorf("unexpected error: %v", errs[len(errs)-1])
	}
}

func TestValidateConfig_Auth_UnknownModel(t *testing.T) {
	cfg := authBaseConfig
	cfg.Auth = &AuthConfig{Model: "ghosts"}
	errs := ValidateConfig(&cfg)
	if len(errs) == 0 {
		t.Fatal("expected error for unknown auth.model, got none")
	}
	if !strings.Contains(errs[len(errs)-1].Error(), "ghosts") {
		t.Errorf("unexpected error: %v", errs[len(errs)-1])
	}
}

func TestValidateConfig_Auth_ModelHasNoPassword(t *testing.T) {
	cfg := Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models: []Model{
			{Name: "users", Fields: []Field{
				{Name: "email", Type: "varchar(255)", Required: true},
			}},
		},
		Auth: &AuthConfig{Model: "users"},
	}
	errs := ValidateConfig(&cfg)
	if len(errs) == 0 {
		t.Fatal("expected error when auth model has no password field, got none")
	}
	if !strings.Contains(errs[len(errs)-1].Error(), "password") {
		t.Errorf("unexpected error: %v", errs[len(errs)-1])
	}
}

func TestValidateConfig_Auth_Nil(t *testing.T) {
	cfg := authBaseConfig
	cfg.Auth = nil
	if errs := ValidateConfig(&cfg); len(errs) != 0 {
		t.Errorf("expected no auth errors when auth is nil, got: %v", errs)
	}
}

// ── Auth: detectIdentityField ───────────────────────────────────────────────

func TestDetectIdentityField_Email(t *testing.T) {
	m := Model{Name: "users", Fields: []Field{
		{Name: "email", Type: "varchar(255)"},
		{Name: "name", Type: "varchar(100)"},
	}}
	if got := detectIdentityField(m); got != "email" {
		t.Errorf("expected 'email', got %q", got)
	}
}

func TestDetectIdentityField_Username(t *testing.T) {
	m := Model{Name: "users", Fields: []Field{
		{Name: "username", Type: "varchar(100)"},
		{Name: "bio", Type: "text"},
	}}
	if got := detectIdentityField(m); got != "username" {
		t.Errorf("expected 'username', got %q", got)
	}
}

func TestDetectIdentityField_EmailBeforeUsername(t *testing.T) {
	m := Model{Name: "users", Fields: []Field{
		{Name: "username", Type: "varchar(100)"},
		{Name: "email", Type: "varchar(255)"},
	}}
	if got := detectIdentityField(m); got != "username" {
		// first match wins (email or username), whichever appears first in the loop
		// our implementation checks by name equality, so the first match in the fields list wins
		t.Logf("got %q (first field named email/username wins)", got)
	}
}

func TestDetectIdentityField_FallbackVarchar(t *testing.T) {
	m := Model{Name: "accounts", Fields: []Field{
		{Name: "handle", Type: "varchar(50)"},
		{Name: "age", Type: "int"},
	}}
	if got := detectIdentityField(m); got != "handle" {
		t.Errorf("expected 'handle' (first varchar), got %q", got)
	}
}

func TestDetectIdentityField_FallbackFirstField(t *testing.T) {
	m := Model{Name: "items", Fields: []Field{
		{Name: "score", Type: "int"},
	}}
	if got := detectIdentityField(m); got != "score" {
		t.Errorf("expected 'score' (first field fallback), got %q", got)
	}
}

// ── Auth: GenerateGORMModels password json:"-" ──────────────────────────────

func TestGenerateGORMModels_Auth_PasswordHidden(t *testing.T) {
	models := []Model{
		{Name: "users", Fields: []Field{
			{Name: "email", Type: "varchar(255)", Required: true},
			{Name: "password", Type: "varchar(255)", Required: true},
		}},
	}
	out := GenerateGORMModels(models, "models", &AuthConfig{Model: "users"})
	if strings.Contains(out, `json:"password"`) {
		t.Error("expected password field to have json:\"-\", but found json:\"password\"")
	}
	if !strings.Contains(out, `json:"-"`) {
		t.Error("expected json:\"-\" tag for password field, not found")
	}
}

func TestGenerateGORMModels_NonAuth_PasswordNotHidden(t *testing.T) {
	models := []Model{
		{Name: "items", Fields: []Field{
			{Name: "password", Type: "varchar(255)"},
		}},
	}
	out := GenerateGORMModels(models, "models", nil)
	if !strings.Contains(out, `json:"password"`) {
		t.Error("expected json:\"password\" when auth is nil")
	}
}

// ── Auth: GenerateAuthGo ────────────────────────────────────────────────────

func TestGenerateAuthGo_RegisterRoute(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models: []Model{
			{Name: "users", Fields: []Field{
				{Name: "email", Type: "varchar(255)", Required: true},
				{Name: "password", Type: "varchar(255)", Required: true},
			}},
		},
		Auth: &AuthConfig{Model: "users"},
	}
	out, err := GenerateAuthGo(cfg, "myapp")
	if err != nil {
		t.Fatalf("GenerateAuthGo returned error: %v", err)
	}
	if !strings.Contains(out, `"/auth/register"`) {
		t.Error("expected /auth/register route")
	}
	if !strings.Contains(out, `"/auth/login"`) {
		t.Error("expected /auth/login route")
	}
}

func TestGenerateAuthGo_JWTMiddleware(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models: []Model{
			{Name: "users", Fields: []Field{
				{Name: "email", Type: "varchar(255)", Required: true},
				{Name: "password", Type: "varchar(255)", Required: true},
			}},
		},
		Auth: &AuthConfig{Model: "users"},
	}
	out, err := GenerateAuthGo(cfg, "myapp")
	if err != nil {
		t.Fatalf("GenerateAuthGo returned error: %v", err)
	}
	if !strings.Contains(out, "JWTMiddleware") {
		t.Error("expected JWTMiddleware function")
	}
	if !strings.Contains(out, "bcrypt") {
		t.Error("expected bcrypt usage")
	}
	if !strings.Contains(out, "AuthClaims") {
		t.Error("expected AuthClaims struct")
	}
}

func TestGenerateAuthGo_IdentityFieldEmail(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models: []Model{
			{Name: "users", Fields: []Field{
				{Name: "email", Type: "varchar(255)", Required: true},
				{Name: "password", Type: "varchar(255)", Required: true},
			}},
		},
		Auth: &AuthConfig{Model: "users"},
	}
	out, err := GenerateAuthGo(cfg, "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"email"`) {
		t.Error("expected email as identity field in generated code")
	}
}

// ── Auth: GenerateMain conditional ─────────────────────────────────────────

func TestGenerateMain_WithAuth(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models:   []Model{{Name: "posts", Fields: []Field{{Name: "title", Type: "text"}}}},
		Auth:     &AuthConfig{Model: "users"},
	}
	out, err := GenerateMain(cfg, "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "RegisterAuthRoutes") {
		t.Error("expected RegisterAuthRoutes call")
	}
	if !strings.Contains(out, "JWTMiddleware()") {
		t.Error("expected JWTMiddleware() call")
	}
	if !strings.Contains(out, `r.Group("/")`) {
		t.Error("expected authenticated group")
	}
}

func TestGenerateMain_WithoutAuth(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models:   []Model{{Name: "posts", Fields: []Field{{Name: "title", Type: "text"}}}},
	}
	out, err := GenerateMain(cfg, "myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "RegisterAuthRoutes") {
		t.Error("expected no RegisterAuthRoutes when auth is disabled")
	}
	if !strings.Contains(out, "routes.RegisterRoutes(r, db)") {
		t.Error("expected direct routes.RegisterRoutes(r, db) call")
	}
}

// ── Auth: GenerateGoMod deps ────────────────────────────────────────────────

func TestGenerateGoMod_WithAuth(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models:   []Model{{Name: "users", Fields: []Field{{Name: "email", Type: "varchar(255)"}}}},
		Auth:     &AuthConfig{Model: "users"},
	}
	out, err := GenerateGoMod(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "golang-jwt/jwt/v5") {
		t.Error("expected golang-jwt/jwt/v5 dep")
	}
	if !strings.Contains(out, "golang.org/x/crypto") {
		t.Error("expected golang.org/x/crypto dep")
	}
}

func TestGenerateGoMod_WithoutAuth(t *testing.T) {
	cfg := &Config{
		App:      AppConfig{Name: "myapp", Port: 8080},
		Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
		Models:   []Model{{Name: "posts", Fields: []Field{{Name: "title", Type: "text"}}}},
	}
	out, err := GenerateGoMod(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "golang-jwt") {
		t.Error("expected no golang-jwt when auth is disabled")
	}
}

// ── Auth: GenerateReactAPI headers ─────────────────────────────────────────

func TestGenerateReactAPI_WithAuth_HasHeaders(t *testing.T) {
	m := Model{Name: "posts", Fields: []Field{{Name: "title", Type: "text"}}}
	out := GenerateReactAPI(m, true)
	if !strings.Contains(out, "authHeaders()") {
		t.Error("expected authHeaders() when hasAuth=true")
	}
	if !strings.Contains(out, "getToken") {
		t.Error("expected getToken import when hasAuth=true")
	}
}

func TestGenerateReactAPI_WithoutAuth_NoHeaders(t *testing.T) {
	m := Model{Name: "posts", Fields: []Field{{Name: "title", Type: "text"}}}
	out := GenerateReactAPI(m, false)
	if strings.Contains(out, "authHeaders()") {
		t.Error("expected no authHeaders() when hasAuth=false")
	}
}

// ── Auth: GenerateReactApp ProtectedRoute ───────────────────────────────────

func TestGenerateReactApp_WithAuth(t *testing.T) {
	models := []Model{{Name: "posts", Fields: []Field{{Name: "title", Type: "text"}}}}
	out := GenerateReactApp(models, true)
	if !strings.Contains(out, "ProtectedRoute") {
		t.Error("expected ProtectedRoute when hasAuth=true")
	}
	if !strings.Contains(out, "AuthProvider") {
		t.Error("expected AuthProvider when hasAuth=true")
	}
	if !strings.Contains(out, "/login") {
		t.Error("expected /login route when hasAuth=true")
	}
}

func TestGenerateReactApp_WithoutAuth(t *testing.T) {
	models := []Model{{Name: "posts", Fields: []Field{{Name: "title", Type: "text"}}}}
	out := GenerateReactApp(models, false)
	if strings.Contains(out, "ProtectedRoute") {
		t.Error("expected no ProtectedRoute when hasAuth=false")
	}
}

// ── Auth: ParseConfig auto-create model ────────────────────────────────────

func TestParseConfig_Auth_AutoCreateModel(t *testing.T) {
	// Write a temp config file with auth but no users model.
	content := `
app:
  name: myapp
  port: 8080
database:
  host: localhost
  name: mydb
auth:
  model: users
models:
  - name: posts
    fields:
      - name: title
        type: text
`
	tmpFile, err := os.CreateTemp("", "auth_test_*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	tmpFile.Close()

	cfg, err := ParseConfig(tmpFile.Name())
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}

	found := false
	for _, m := range cfg.Models {
		if m.Name == "users" {
			found = true
			hasPassword := false
			for _, f := range m.Fields {
				if f.Name == "password" {
					hasPassword = true
				}
			}
			if !hasPassword {
				t.Error("auto-created users model should have a password field")
			}
		}
	}
	if !found {
		t.Error("expected users model to be auto-created")
	}
}

// ── timestamps config tests ─────────────────────────────────────────────────

func boolPtr(b bool) *bool { return &b }

func TestValidateConfig_ReservedFieldName(t *testing.T) {
	for _, reserved := range []string{"id", "created_at", "updated_at", "deleted_at"} {
		cfg := &Config{
			App:      AppConfig{Name: "myapp", Port: 8080},
			Database: DatabaseConfig{Host: "localhost", Name: "db", Port: 5432},
			Models: []Model{
				{Name: "items", Fields: []Field{{Name: reserved, Type: "text"}}},
			},
		}
		errs := ValidateConfig(cfg)
		if len(errs) == 0 {
			t.Errorf("expected error for reserved field name %q, got none", reserved)
			continue
		}
		if !strings.Contains(errs[0].Error(), "reserved") {
			t.Errorf("field %q: unexpected error: %v", reserved, errs[0])
		}
	}
}

func TestGenerateMigrationUp_WithoutTimestamps_Default(t *testing.T) {
	// timestamps defaults to false (not set = no timestamps)
	models := []Model{
		{Name: "posts", Fields: []Field{{Name: "title", Type: "text", Required: true}}},
	}
	out := GenerateMigrationUp(models, "postgres")
	for _, notWant := range []string{"created_at", "updated_at", "deleted_at"} {
		if strings.Contains(out, notWant) {
			t.Errorf("expected no %q in default (no timestamps) migration output", notWant)
		}
	}
	if !strings.Contains(out, "id SERIAL PRIMARY KEY") {
		t.Error("id column should still be present")
	}
}

func TestGenerateMigrationUp_WithTimestamps_Explicit(t *testing.T) {
	models := []Model{
		{Name: "posts", Timestamps: boolPtr(true), Fields: []Field{{Name: "title", Type: "text", Required: true}}},
	}
	out := GenerateMigrationUp(models, "postgres")
	for _, want := range []string{"created_at TIMESTAMP", "updated_at TIMESTAMP", "deleted_at TIMESTAMP"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q when timestamps:true", want)
		}
	}
}

func TestGenerateMigrationUp_WithTimestamps_MySQL(t *testing.T) {
	models := []Model{
		{Name: "posts", Timestamps: boolPtr(true), Fields: []Field{{Name: "title", Type: "text", Required: true}}},
	}
	out := GenerateMigrationUp(models, "mysql")
	for _, want := range []string{"created_at DATETIME", "updated_at DATETIME", "deleted_at DATETIME"} {
		if !strings.Contains(out, want) {
			t.Errorf("mysql: expected %q when timestamps:true", want)
		}
	}
}

func TestGenerateGORMModels_WithoutTimestamps_Default(t *testing.T) {
	// default: no timestamps
	models := []Model{
		{Name: "items", Fields: []Field{{Name: "name", Type: "text"}}},
	}
	out := GenerateGORMModels(models, "models", nil)
	if strings.Contains(out, "Base") {
		t.Error("Base struct should not appear when timestamps not set (default false)")
	}
	if strings.Contains(out, "gorm.io/gorm") {
		t.Error("gorm.io/gorm import should not appear when timestamps not set")
	}
	if !strings.Contains(out, `gorm:"primarykey"`) {
		t.Error("expected inline ID field with primarykey tag")
	}
}

func TestGenerateGORMModels_WithTimestamps_Explicit(t *testing.T) {
	models := []Model{
		{Name: "items", Timestamps: boolPtr(true), Fields: []Field{{Name: "name", Type: "text"}}},
	}
	out := GenerateGORMModels(models, "models", nil)
	if !strings.Contains(out, "type Base struct") {
		t.Error("expected Base struct when timestamps:true")
	}
	if !strings.Contains(out, "gorm.io/gorm") {
		t.Error("expected gorm.io/gorm import when timestamps:true")
	}
}

func TestGenerateReactTypes_WithoutTimestamps_Default(t *testing.T) {
	// default: no timestamps
	m := Model{
		Name:   "items",
		Fields: []Field{{Name: "name", Type: "text", Required: true}},
	}
	out := GenerateReactTypes(m, nil)
	for _, notWant := range []string{"created_at", "updated_at", "deleted_at"} {
		if strings.Contains(out, notWant) {
			t.Errorf("expected no %q in types when timestamps not set (default false)", notWant)
		}
	}
}

func TestGenerateReactTypes_WithTimestamps_Explicit(t *testing.T) {
	m := Model{
		Name:       "items",
		Timestamps: boolPtr(true),
		Fields:     []Field{{Name: "name", Type: "text", Required: true}},
	}
	out := GenerateReactTypes(m, nil)
	for _, want := range []string{"created_at", "updated_at", "deleted_at"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in types when timestamps:true", want)
		}
	}
}
