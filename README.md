# Gaplicator

Generate a full-stack web application (database + server + client) from a single YAML config file.

## Stack

| Layer    | Technology        |
|----------|-------------------|
| Database | PostgreSQL or MySQL |
| Server   | Go + Gin + GORM   |
| Client   | React + TypeScript + Vite |

## Usage

```bash
gapp build <config.yaml> [-o <output-dir>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-o`, `--output` | `dist` | Output directory for generated files |

## Config format

```yaml
app:
  name: my-app   # lowercase letters, digits, hyphens, underscores; used as Go module name
  port: 8080     # 1–65535

database:
  driver: postgres # optional: "postgres" (default) or "mysql"
  host: localhost
  port: 5432       # optional, default: 5432 (postgres) or 3306 (mysql); must be 1–65535
  name: my_db
  user: postgres   # optional, default: postgres (postgres) or root (mysql)
  password: secret # optional, default: secret

auth:                  # optional: enables JWT authentication
  model: users         # model used for login/register (auto-created if not in models list)

models:
  - name: posts        # plural snake_case → table name; must be unique
    timestamps: true   # optional: adds created_at, updated_at, deleted_at (soft delete). Default: false
    fields:
      - name: title
        type: varchar(200)
        required: true
        label: "Post Title"    # optional: shown in React form labels and table headers
      - name: status
        type: enum
        values: [draft, published, archived]
        default: draft
      - name: published
        type: boolean
        default: false
      - name: author_id
        type: int
        references: users.id   # FK → must reference an existing model and field
        display_field: name    # optional: field from referenced model shown in UI dropdowns

  - name: tags
    many_to_many: [posts]      # auto-creates posts_tags join table
    fields:
      - name: name
        type: varchar(100)
        required: true
        unique: true
```

Supported field types: `int`, `bigint`, `smallint`, `text`, `boolean`, `bool`, `date`, `datetime`, `timestamp`, `uuid`, `float`, `double`, `varchar(N)`, `char(N)`, `decimal(P,S)`, `enum`

Field attributes:

| Attribute | Description |
|-----------|-------------|
| `required` | Adds `NOT NULL` constraint |
| `unique` | Adds `UNIQUE` constraint |
| `default` | Column default value |
| `index` | Creates a non-unique index (`CREATE INDEX`) |
| `references` | Foreign key in `model.field` format (e.g. `users.id`) |
| `display_field` | Field from the referenced model used as the display label in UI dropdowns and table cells. Only valid with `references`. |
| `label` | Human-readable label for React form inputs and table headers |
| `values` | Allowed values list — required when `type: enum` |

Model attributes:

| Attribute | Description |
|-----------|-------------|
| `timestamps` | `true` to add `created_at`, `updated_at`, and `deleted_at` (soft delete) columns. Default: `false`. |
| `many_to_many` | List of other model names to relate through an auto-named join table (names sorted alphabetically, e.g. `posts` + `tags` → `posts_tags`) |

All models include an auto-generated `id` primary key. The field names `id`, `created_at`, `updated_at`, and `deleted_at` are reserved and cannot be declared manually.

## What gets generated

Running `gapp build app.yaml` produces:

```
dist/
├── main.go                        # Gin server + GORM auto-migrate
├── auth.go                        # JWT handlers — only with auth: config
├── go.mod                         # module with gin/gorm/postgres deps
├── docker-compose.yml             # app + postgres services
├── .env                           # DB credentials (+ JWT_SECRET with auth:)
├── dev.sh                         # one-command dev startup (see below)
├── shutdown.sh                    # stops docker containers
├── migrations/
│   ├── 001_initial.up.sql
│   └── 001_initial.down.sql
├── models/
│   └── models.go                  # GORM structs (password field hidden with json:"-")
├── routes/
│   └── routes.go                  # Gin CRUD handlers
└── client/                        # React + TypeScript frontend
    ├── package.json               # react, react-router-dom, vite
    ├── index.html
    ├── vite.config.ts             # dev proxy → Go backend
    ├── tsconfig.json
    └── src/
        ├── main.tsx
        ├── App.tsx                # nav + routes (ProtectedRoute wrapping with auth:)
        ├── context/
        │   └── AuthContext.tsx   # AuthProvider, useAuth, getToken — only with auth:
        ├── types/
        │   └── {model}.ts        # TypeScript interfaces
        ├── api/
        │   ├── auth.ts           # login/register fetch functions — only with auth:
        │   └── {model}.ts        # fetch wrappers (Authorization header with auth:)
        └── pages/
            ├── LoginPage.tsx     # only with auth:
            ├── RegisterPage.tsx  # only with auth:
            └── {Model}Page.tsx   # CRUD table + inline form
```

## Authentication

Add `auth:` to your config to enable JWT-based authentication. All model CRUD routes are automatically protected — unauthenticated requests receive `401 Unauthorized`.

```yaml
auth:
  model: users   # auto-created with email + password + name if not declared
```

**Auth endpoints** (always public):

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/auth/register` | Register — body: `{"email": "...", "password": "..."}` |
| `POST` | `/auth/login` | Login — returns `{"token": "<jwt>"}` |

The JWT token is valid for 24 hours and must be sent as `Authorization: Bearer <token>` on all model routes. The secret is read from the `JWT_SECRET` environment variable (set in the generated `.env`).

The React client stores the token in `localStorage`, wraps all model pages in a `ProtectedRoute` (redirects to `/login`), and sends the token automatically on every API call.

**Identity field** is auto-detected from the auth model: `email` takes priority over `username`, then the first `varchar`/`text` field.

---

## Generated API

Every model gets the following routes:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/{model}` | List with pagination, filtering, search, and sorting |
| `GET` | `/{model}/:id` | Get single item |
| `POST` | `/{model}` | Create item |
| `PUT` | `/{model}/:id` | Update item |
| `DELETE` | `/{model}/:id` | Delete single item |
| `DELETE` | `/{model}/batch` | Batch delete — request body: `{"ids": [1, 2, 3]}` |

### Filtering, search & sorting

Every list endpoint supports filtering, full-text search, sorting, and pagination out of the box.

| Parameter | Description |
|-----------|-------------|
| `q` | Full-text search across all text-type fields (case-insensitive) |
| `<field_name>` | Filter by exact value (numeric, enum, boolean, or foreign key) |
| `sort_by` | Field to sort by, or `id`. When `timestamps: true`, also accepts `created_at` and `updated_at`. Default: `id` |
| `sort_dir` | `asc` or `desc`. Default: `desc` |
| `page` | Page number (1-based). Default: `1` |
| `limit` | Results per page. Default: `20`, max: `100` |

**Example:** `GET /posts?q=hello&status=draft&author_id=5&sort_by=title&sort_dir=desc&page=2&limit=20`

The React frontend generates corresponding UI controls: a search input, filter dropdowns for enum/boolean/FK fields, sortable column headers, pagination, and checkbox-based batch delete.

## Getting Started

```bash
# install
go install github.com/myAwesome/vibe-gen@latest

# scaffold from config
gapp build app.yaml

# start generated app (DB + migrations + server in one command)
cd dist && ./dev.sh
```

`dev.sh` does three things in order:
1. Starts the database container (`postgres` or `mysql` depending on `database.driver`)
2. Waits for the database to be healthy, then applies `migrations/001_initial.up.sql`
3. Starts the Go server with `go run .`

To stop: `./shutdown.sh`

No local database client required — migrations run inside the container.

## Config reference

See [`docs/config.md`](docs/config.md) for the full reference, [`docs/example.yaml`](docs/example.yaml) for a multi-model attendance-journal example, or [`docs/examples/beer-tracker.yaml`](docs/examples/beer-tracker.yaml) for a more complex example with enums and many-to-many relationships.

## License

MIT
