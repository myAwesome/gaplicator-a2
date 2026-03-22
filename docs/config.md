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
| `many_to_many` | list of strings | no | Names of other models to relate through a join table. The join table is auto-created and named by sorting both model names alphabetically (e.g. `courses` + `students` → `courses_students`). The referenced model must exist in the same config. |

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
| `enum` | Requires `values` list. Generates a `TEXT CHECK (... IN (...))` column and dropdown filters in the UI. |

---

## Filtering, search & sorting

Every generated list endpoint supports filtering, full-text search, sorting, and pagination out of the box — no extra config required. The behaviour is determined by field types:

| Field type | Generated filter |
|------------|-----------------|
| `text`, `varchar(N)`, `char(N)`, `uuid` | Included in full-text search via `q` parameter (case-insensitive `ILIKE`) |
| `int`, `bigint`, `smallint`, `float`, `double`, `decimal` | Exact-match filter via `?field_name=value` |
| `references` (foreign key) | Exact-match filter via `?field_name=value`, with a dropdown in the UI |
| `enum` | Exact-match filter via `?field_name=value`, with a dropdown showing `values` |
| `boolean` / `bool` | Boolean filter via `?field_name=true\|false\|1\|0`, with a Yes/No dropdown |

### Query parameters

| Parameter | Description |
|-----------|-------------|
| `q` | Full-text search across all text-type fields. |
| `<field_name>` | Filter by the exact value of that field (numeric, enum, boolean, or FK). |
| `sort_by` | Sort by any field name, or `id`, `created_at`, `updated_at`. Default: `id`. |
| `sort_dir` | Sort direction: `asc` or `desc`. Default: `desc`. |
| `page` | Page number (1-based). Default: `1`. |
| `limit` | Results per page. Default: `20`, max: `100`. |

**Example:** `GET /posts?q=hello&status=draft&author_id=5&sort_by=created_at&sort_dir=desc&page=2&limit=20`

---

## Full example

See [`docs/example.yaml`](example.yaml) for a working multi-model config.
