# Config Reference

A Gaplicator config file is a YAML document with up to four top-level sections: `app`, `database`, `models`, and the optional `auth`.

```yaml
app:
  name: my-app
  port: 8080

database:
  driver: postgres  # optional: "postgres" (default) or "mysql"
  host: localhost
  name: my_db

auth:               # optional — enables JWT authentication
  model: users

models:
  - name: posts
    fields:
      - name: title
        type: varchar(200)
        required: true
```

---

## `app`

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `name` | string | yes | Application name. Used as the Go module name and React app title. Must match `^[a-z][a-z0-9_-]*$` (lowercase letters, digits, hyphens, underscores). |
| `port` | int | yes | HTTP port the generated server listens on. Must be between 1 and 65535. |

---

## `database`

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `driver` | string | no | `postgres` | Database driver. Accepted values: `postgres`, `mysql`. |
| `host` | string | yes | — | Database hostname. |
| `name` | string | yes | — | Database name. |
| `port` | int | no | `5432` / `3306` | Port — defaults to `5432` for PostgreSQL and `3306` for MySQL. Must be between 1 and 65535. |
| `user` | string | no | `postgres` / `root` | Database user — defaults to `postgres` for PostgreSQL and `root` for MySQL. |
| `password` | string | no | `secret` | Database password. |

### Driver differences

| Feature | PostgreSQL | MySQL |
|---------|-----------|-------|
| Primary key | `SERIAL PRIMARY KEY` | `INT AUTO_INCREMENT PRIMARY KEY` |
| Enum type | `TEXT CHECK (col IN (...))` | Native `ENUM(...)` |
| UUID type | `UUID` | `VARCHAR(36)` |
| Text search | `ILIKE` (case-insensitive) | `LIKE` (case-insensitive with utf8mb4) |
| Docker image | `postgres:16-alpine` | `mysql:8` |

---

## `auth`

Optional. When present, enables JWT-based authentication for the generated app.

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `model` | string | yes | Name of the model used for login and registration. Must match one of the names in `models`. If the model does not exist in the `models` list, a default one is auto-created with `email` (varchar 255, unique), `password` (varchar 255), and `name` (varchar 100) fields. |

**What gets generated when `auth` is set:**

- **`auth.go`** — `POST /auth/register` and `POST /auth/login` handlers (bcrypt + JWT HS256), plus `JWTMiddleware` that validates the `Authorization: Bearer <token>` header
- All model CRUD routes are mounted behind the JWT middleware — unauthenticated requests get `401`
- **`go.mod`** — adds `golang-jwt/jwt/v5` and `golang.org/x/crypto`
- **`.env`** — adds `JWT_SECRET=change-me-in-production`
- **`models/models.go`** — the `password` field gets `json:"-"` so it is never included in API responses
- **React client** — `AuthContext`, `LoginPage`, `RegisterPage`, `ProtectedRoute`, and `Authorization` headers on all fetch calls

**Identity field** — the field used as the login identifier is auto-detected from the auth model: `email` takes priority over `username`, then the first `varchar`/`text` field.

**Example:**

```yaml
auth:
  model: users

models:
  - name: users          # declare explicitly to customise fields…
    fields:
      - name: email
        type: varchar(255)
        required: true
        unique: true
      - name: password
        type: varchar(255)
        required: true
      - name: name
        type: varchar(100)

  # …or omit the model entirely and let Gaplicator create a default one:
  # auth:
  #   model: users
  # (no users entry in models needed)
```

**Auth API:**

| Method | Path | Body | Response |
|--------|------|------|----------|
| `POST` | `/auth/register` | `{"email": "...", "password": "..."}` | `{"id": 1}` |
| `POST` | `/auth/login` | `{"email": "...", "password": "..."}` | `{"token": "<jwt>"}` |

Tokens expire after 24 hours. Pass them as `Authorization: Bearer <token>`.

---

## `models`

A list of data models. Each model maps to a database table and gets full CRUD routes and a React page.

| Key | Type | Required | Default | Description |
|-----|------|----------|---------|-------------|
| `name` | string | yes | — | Table name in plural snake_case (e.g. `blog_posts`). Must be unique across all models. |
| `timestamps` | bool | no | `false` | When `true`, adds `created_at`, `updated_at`, and `deleted_at` (soft-delete) columns to the table, a `Base` struct embedding in the GORM model, timestamp fields in the TypeScript interface, and `created_at`/`updated_at` as valid `sort_by` options. |
| `fields` | list | yes | — | At least one field required. |
| `many_to_many` | list of strings | no | — | Names of other models to relate through a join table. The join table is auto-created and named by sorting both model names alphabetically (e.g. `courses` + `students` → `courses_students`). The referenced model must exist in the same config. |

All models automatically include an `id` primary key. The field names `id`, `created_at`, `updated_at`, and `deleted_at` are reserved — declaring them manually in `fields` is a validation error.

### `models[].fields`

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `name` | string | yes | Column name in snake_case. |
| `type` | string | yes | SQL type. See [Field types](#field-types) below. |
| `required` | bool | no | Adds `NOT NULL` constraint. Default: `false`. |
| `unique` | bool | no | Adds `UNIQUE` constraint. Default: `false`. |
| `default` | any | no | Column default value. |
| `index` | bool | no | Creates a non-unique database index (`CREATE INDEX IF NOT EXISTS`). Ignored when `unique: true`. Default: `false`. |
| `references` | string | no | Foreign key in `model.field` format (e.g. `users.id`). The referenced model **and field** must exist in the same config (`id` is always valid as it is auto-generated). |
| `display_field` | string | no | Field from the referenced model to use as the display label in the UI — in form dropdowns and table cells. Only meaningful when `references` is set. Defaults to the first `name` or `title` field of the referenced model, or its first field if neither exists. |
| `label` | string | no | Human-readable display name used in React form `<label>` elements and table column headers. Defaults to the field `name`. |
| `values` | list of strings | no | Allowed values for `enum` type fields. **Required when `type: enum`**. |

#### Field types

| Type | Notes |
|------|-------|
| `int` | |
| `bigint` | |
| `smallint` | |
| `float` | |
| `double` | |
| `decimal(P,S)` | e.g. `decimal(10,2)` |
| `text` | |
| `varchar(N)` | e.g. `varchar(255)` |
| `char(N)` | e.g. `char(2)` |
| `boolean` / `bool` | |
| `date` | |
| `datetime` / `timestamp` | |
| `uuid` | |
| `enum` | Requires `values` list. Generates a `TEXT CHECK (... IN (...))` column (PostgreSQL) or native `ENUM(...)` (MySQL), and dropdown filters in the UI. |

---

## Filtering, search & sorting

Every generated list endpoint supports filtering, full-text search, sorting, and pagination out of the box — no extra config required. The behaviour is determined by field types:

| Field type | Generated filter |
|------------|-----------------|
| `text`, `varchar(N)`, `char(N)`, `uuid` | Included in full-text search via `q` parameter (case-insensitive: `ILIKE` for PostgreSQL, `LIKE` for MySQL) |
| `int`, `bigint`, `smallint`, `float`, `double`, `decimal` | Exact-match filter via `?field_name=value` |
| `references` (foreign key) | Exact-match filter via `?field_name=value`, with a dropdown in the UI |
| `enum` | Exact-match filter via `?field_name=value`, with a dropdown showing `values` |
| `boolean` / `bool` | Boolean filter via `?field_name=true\|false\|1\|0`, with a Yes/No dropdown |

### Query parameters

| Parameter | Description |
|-----------|-------------|
| `q` | Full-text search across all text-type fields. |
| `<field_name>` | Filter by the exact value of that field (numeric, enum, boolean, or FK). |
| `sort_by` | Sort by any field name, or `id`. When the model has `timestamps: true`, also accepts `created_at` and `updated_at`. Default: `id`. |
| `sort_dir` | Sort direction: `asc` or `desc`. Default: `desc`. |
| `page` | Page number (1-based). Default: `1`. |
| `limit` | Results per page. Default: `20`, max: `100`. |

**Example:** `GET /posts?q=hello&status=draft&author_id=5&sort_by=id&sort_dir=desc&page=2&limit=20`

---

## Full example

See [`docs/example.yaml`](example.yaml) for a working multi-model config, or [`docs/examples/beer-tracker.yaml`](examples/beer-tracker.yaml) for a more complex example with enums, foreign keys, and many-to-many relationships.
