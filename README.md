# vibe-gen

Generate a full-stack web application (database + server + client) from a single YAML config file.

## Concept

Describe your app in YAML — `vibe-gen` scaffolds the entire stack and keeps it in sync as your config evolves.

```yaml
app:
  name: todo-app
  database:
    type: postgres
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
    framework: express
    auth: jwt
  client:
    framework: react
    pages:
      - name: Home
        route: /
        components: [TaskList, AddTask]
```

Running `vibe-gen build app.yaml` produces:

```
generated/
├── db/
│   ├── migrations/
│   └── schema.sql
├── server/
│   ├── routes/
│   ├── models/
│   └── index.ts
└── client/
    ├── src/
    │   ├── pages/
    │   └── components/
    └── index.html
```

## Features

- **Database** — migrations, schema, ORM models
- **Server** — REST API routes, auth, middleware
- **Client** — pages, components, API bindings
- **Sync** — re-run `build` after config changes; only diffs are regenerated
- **Extensible** — bring your own templates via the `templates/` directory

## Supported Stack Options

| Layer    | Options                          |
|----------|----------------------------------|
| Database | PostgreSQL, MySQL, SQLite        |
| Server   | Express, Fastify, Hono           |
| Client   | React, Vue, Svelte               |
| Auth     | JWT, Session, OAuth2             |
| ORM      | Drizzle, Prisma, Kysely          |

## Getting Started

```bash
# install
npm install -g vibe-gen

# scaffold from config
vibe-gen build app.yaml

# start generated app
cd generated && docker-compose up
```

## Config Reference

Full schema documentation: [docs/config.md](docs/config.md)

## License

MIT
