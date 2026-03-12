# Gaplicator

Generate a full-stack web application (database + server + client) from a single YAML config file.

## Stack

| Layer    | Technology        |
|----------|-------------------|
| Database | PostgreSQL        |
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
  name: my-app   # used as Go module name
  port: 8080

database:
  host: localhost
  port: 5432       # optional, default: 5432
  name: my_db
  user: postgres   # optional, default: postgres
  password: secret # optional, default: secret

models:
  - name: posts        # plural snake_case → table name
    fields:
      - name: title
        type: varchar(200)
        required: true
      - name: published
        type: boolean
        default: false
      - name: author_id
        type: int
        references: users.id   # FK → users table
```

Supported field types: `int`, `bigint`, `smallint`, `text`, `boolean`, `bool`, `date`, `datetime`, `timestamp`, `uuid`, `float`, `double`, `varchar(N)`, `char(N)`, `decimal(P,S)`

Field attributes: `required` (NOT NULL), `unique` (UNIQUE constraint), `default`, `references` (foreign key, e.g. `users.id`)

All models include auto-managed `id`, `created_at`, `updated_at`, and `deleted_at` (soft delete) fields via GORM.

## What gets generated

Running `gapp build app.yaml` produces:

```
dist/
├── main.go                        # Gin server + GORM auto-migrate
├── go.mod                         # module with gin/gorm/postgres deps
├── docker-compose.yml             # app + postgres services
├── .env                           # DB credentials
├── dev.sh                         # one-command dev startup (see below)
├── shutdown.sh                    # stops docker containers
├── schema.sql                     # CREATE TABLE statements
├── migrations/
│   ├── 001_initial.up.sql
│   └── 001_initial.down.sql
├── models/
│   └── models.go                  # GORM structs (snake_case JSON tags)
├── routes/
│   └── routes.go                  # Gin CRUD handlers
└── client/                        # React + TypeScript frontend
    ├── package.json               # react, react-router-dom, vite
    ├── index.html
    ├── vite.config.ts             # dev proxy → Go backend
    ├── tsconfig.json
    └── src/
        ├── main.tsx
        ├── App.tsx                # nav + routes per model
        ├── types/
        │   └── {model}.ts        # TypeScript interfaces
        ├── api/
        │   └── {model}.ts        # fetch wrappers (list/get/create/update/delete)
        └── pages/
            └── {Model}Page.tsx   # CRUD table + inline form
```

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
1. Starts the PostgreSQL container via `docker compose up -d postgres`
2. Waits for the database to be healthy, then applies `migrations/001_initial.up.sql`
3. Starts the Go server with `go run .`

To stop: `./shutdown.sh`

No local PostgreSQL client required — migrations run inside the container.

## Config reference

See [`sandbox/example.yaml`](sandbox/example.yaml) for a full working example.

## License

MIT
