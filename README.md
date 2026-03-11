# vibe-gen

Generate a full-stack web application (database + server + client) from a single YAML config file.

## Stack

| Layer    | Technology        |
|----------|-------------------|
| Database | PostgreSQL        |
| Server   | Go + Gin + GORM   |
| Client   | React             |
| Auth     | JWT               |

## Concept

Describe your app in YAML — `vibe-gen` scaffolds the entire stack and keeps it in sync as your config evolves.

```yaml
app:
  name: todo-app
  database:
    models:
      - name: Task
        fields:
          - name: title
            type: string
            required: true
          - name: done
            type: boolean
            default: false
  server:
    auth: jwt
  client:
    pages:
      - name: Home
        route: /
        components: [TaskList, AddTask]
```

Running `vibe-gen build app.yaml` produces:

```
generated/
├── db/
│   └── migrations/
├── server/
│   ├── handlers/
│   ├── models/
│   ├── middleware/
│   └── main.go
└── client/
    └── src/
        ├── pages/
        └── components/
```

## Features

- **Database** — PostgreSQL migrations and GORM models
- **Server** — Gin REST API routes, JWT auth middleware
- **Client** — React pages, components, API bindings
- **Sync** — re-run `build` after config changes; only diffs are regenerated

## Getting Started

```bash
# install
go install github.com/myAwesome/vibe-gen@latest

# scaffold from config
vibe-gen build app.yaml

# start generated app
cd generated && docker-compose up
```

## Config Reference

Full schema documentation: [docs/config.md](docs/config.md)

## License

MIT
