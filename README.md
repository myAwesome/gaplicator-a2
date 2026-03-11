# vibe-gen

Generate a full-stack web application (database + server + client) from a single YAML config file.

## Stack

| Layer    | Technology        |
|----------|-------------------|
| Database | PostgreSQL        |
| Server   | Go + Gin + GORM   |
| Client   | React             |
| Auth     | JWT               |

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
  name: my_db

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

## What gets generated

Running `gapp build app.yaml` produces:

```
dist/
├── main.go                        # Gin server + GORM auto-migrate
├── go.mod                         # module with gin/gorm/postgres deps
├── docker-compose.yml             # app + postgres services
├── .env                           # DB credentials
├── dev.sh                         # one-command dev startup (see below)
├── schema.sql                     # CREATE TABLE statements
├── migrations/
│   ├── 001_initial.up.sql
│   └── 001_initial.down.sql
├── models/
│   └── models.go                  # GORM structs
└── routes/
    └── routes.go                  # Gin CRUD handlers
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

No local PostgreSQL client required — migrations run inside the container.

## Config reference

See [`sandbox/example.yaml`](sandbox/example.yaml) for a full working example.

## License

MIT
