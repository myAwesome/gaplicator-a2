# Config Reference

A Gaplicator config file is a YAML document with three top-level sections: `app`, `database`, and `models`.

```yaml
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
| `host` | string | yes | — | PostgreSQL hostname. |
| `name` | string | yes | — | Database name. |
| `port` | int | no | `5432` | PostgreSQL port. Must be between 1 and 65535. |
| `user` | string | no | `postgres` | Database user. |
| `password` | string | no | `secret` | Database password. |

---

## `models`

A list of data models. Each model maps to a database table and gets full CRUD routes and a React page.

| Key | Type | Required | Description |
|-----|------|----------|-------------|
| `name` | string | yes | Table name in plural snake_case (e.g. `blog_posts`). Must be unique across all models. |
| `fields` | list | yes | At least one field required. |

All models automatically include `id`, `created_at`, `updated_at`, and `deleted_at` (soft delete) — do not declare these manually.

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
| `label` | string | no | Human-readable display name used in React form `<label>` elements and table column headers. Defaults to the field `name`. |

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

---

## Full example

See [`sandbox/example.yaml`](../sandbox/example.yaml) for a working multi-model config.
