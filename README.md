# gaplicator

Generate a full-stack web application (database + server + client) from a single YAML config file.

## Prerequisites

- **Go 1.21+** — to install and run gaplicator
- **Docker** — to run the database via the generated `dev.sh`
- **Node.js 18+** — to run the generated React frontend (and the Node.js backend when `server: node`)

## Quick start

```bash
# install
go install github.com/myAwesome/gaplicator@latest

# write a config
cat > app.yaml <<EOF
app:
  name: my-app
  port: 8080

database:
  host: localhost
  name: my_db

models:
  - name: posts
    fields:
      - name: title
        type: varchar(200)
        required: true
      - name: body
        type: text
EOF

# generate
gaplicator build app.yaml

# start (DB + migrations + server)
cd dist && ./dev.sh
```

Open `http://localhost:8080` — a working CRUD app with a React UI is ready.

To stop: `./shutdown.sh`

## Stack

| Layer    | Technology                               |
|----------|------------------------------------------|
| Database | PostgreSQL or MySQL                      |
| Server   | Go + Gin + GORM *(default)*              |
|          | Node.js + Express + Prisma *(optional)*  |
| Client   | React + TypeScript + Vite                |

## Usage

```bash
gaplicator build <config.yaml> [-o <output-dir>]
gaplicator serve [--host <host>] [--port <port>]
```

### `build` flags

| Flag | Default | Description |
|------|---------|-------------|
| `-o`, `--output` | `dist` | Output directory for generated files |

### `serve` flags

| Flag | Default | Description |
|------|---------|-------------|
| `--host` | `127.0.0.1` | Bridge bind host |
| `--port` | `8787` | Bridge bind port |

### Web Bridge (schema generator -> CLI)

Run both the bridge and web schema generator with one command:

```bash
./dev.sh
```

This starts:
- bridge at `http://127.0.0.1:8787`
- web app at `http://127.0.0.1:5173`

Manual alternative (two terminals):

```bash
# terminal 1
gaplicator serve --host 127.0.0.1 --port 8787

# terminal 2
cd web
npm install
npm run dev
```

Then click **Send to CLI** in the YAML panel.
The web app will `POST` the active YAML tab to `http://127.0.0.1:8787/build` and generate into `dist/<project-name>`.

Bridge endpoints:
- `GET /health` — health check
- `POST /build` — JSON body: `{"yaml":"<schema>","output":"dist/my-app"}`

Optional: set `VITE_GAPLICATOR_BRIDGE_URL` in the web app environment to point to a different bridge URL.

## Config format

```yaml
app:
  name: my-app   # lowercase letters, digits, hyphens, underscores; used as module name
  port: 8080     # 1–65535
  server: go     # optional: "go" (default) or "node"

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

Running `gaplicator build app.yaml` produces different backend files depending on `app.server`.

### `server: go` (default)

```
dist/
├── main.go                        # Gin server + GORM auto-migrate
├── auth.go                        # JWT handlers — only with auth: config
├── go.mod                         # module with gin/gorm/postgres deps
├── docker-compose.yml             # app + postgres/mysql services
├── .env                           # DB credentials (+ JWT_SECRET with auth:)
├── dev.sh                         # one-command dev startup (see below)
├── shutdown.sh                    # stops docker containers
├── migrations/
│   └── 001_initial.up.sql
├── models/
│   └── models.go                  # GORM structs (password hidden with json:"-")
├── routes/
│   └── routes.go                  # Gin CRUD handlers
└── client/                        # React + TypeScript frontend (same for both backends)
```

`dev.sh` order: start DB container → wait healthy → apply `migrations/001_initial.up.sql` → `go run .`

### `server: node`

```
dist/
├── index.js                       # Express server entry point
├── routes.js                      # CRUD handlers for all models
├── auth.js                        # bcrypt + JWT — only with auth: config
├── package.json                   # express, @prisma/client, bcryptjs, jsonwebtoken
├── prisma/
│   └── schema.prisma              # Prisma schema (models, relations, @@map)
├── docker-compose.yml             # app + postgres/mysql services
├── .env                           # DB credentials (DATABASE_URL + JWT_SECRET)
├── dev.sh                         # one-command dev startup (see below)
├── shutdown.sh                    # stops docker containers
└── client/                        # React + TypeScript frontend (same for both backends)
```

`dev.sh` order: start DB container → wait healthy → `npx prisma migrate deploy` → `npm run dev`

Both backends produce an identical React client under `client/`:

```
client/
├── package.json               # react, react-router-dom, vite
├── index.html
├── vite.config.ts             # dev proxy → backend
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

No local database client required — migrations run inside the container.

## Authentication

Add `auth:` to your config to enable JWT-based authentication. All model CRUD routes are automatically protected — unauthenticated requests receive `401 Unauthorized`.

```yaml
auth:
  model: users   # auto-created with email + password + name if not declared
```

**Auth endpoints** (always public):

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/auth/register` | Register — body: `{"email": "...", "password": "..."}` |
| `POST` | `/api/auth/login` | Login — returns `{"token": "<jwt>"}` |

The JWT token is valid for 24 hours and must be sent as `Authorization: Bearer <token>` on all model routes. The secret is read from the `JWT_SECRET` environment variable (set in the generated `.env`).

The React client stores the token in `localStorage`, wraps all model pages in a `ProtectedRoute` (redirects to `/login`), and sends the token automatically on every API call.

**Identity field** is auto-detected from the auth model: `email` takes priority over `username`, then the first `varchar`/`text` field.

---

## Generated API

Every model gets the following routes:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/{model}` | List with pagination, filtering, search, and sorting |
| `GET` | `/api/{model}/:id` | Get single item |
| `POST` | `/api/{model}` | Create item |
| `PUT` | `/api/{model}/:id` | Update item |
| `DELETE` | `/api/{model}/:id` | Delete single item |
| `DELETE` | `/api/{model}/batch` | Batch delete — request body: `{"ids": [1, 2, 3]}` |

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

**Example:** `GET /api/posts?q=hello&status=draft&author_id=5&sort_by=title&sort_dir=desc&page=2&limit=20`

The React frontend generates corresponding UI controls: a search input, filter dropdowns for enum/boolean/FK fields, sortable column headers, pagination, and checkbox-based batch delete.

## Config reference

See [`docs/config.md`](docs/config.md) for the full reference, [`docs/example.yaml`](docs/example.yaml) for a multi-model attendance-journal example, or [`docs/examples/beer-tracker.yaml`](docs/examples/beer-tracker.yaml) for a more complex example with enums and many-to-many relationships.

## License

MIT
