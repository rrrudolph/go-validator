# go-validator

A schema-based, pre-compiled request validator for Go APIs. Define schemas in code, compile once at startup, then validate JSON request bodies and get typed structs with stable, structured error output.

---

## Why This Exists

Most Go validation libraries rely on struct tags and reflection at request time. While flexible, this can:

- **Scatter validation logic** across many structs
- **Make rules harder to reuse** (e.g. shared filter shapes)
- **Produce inconsistent error formatting** across endpoints
- **Repeat reflection work** on every request

`go-validator` takes a different approach:

1. **Define schemas explicitly** in code (nested objects, arrays, enums, ranges, defaults)
2. **Compile them once** at startup into an optimized form
3. **Reuse the compiled schema** for every request on that endpoint
4. **Return stable, structured errors** with paths (e.g. `filter.submission_date.lte: invalid format`)

That makes validation **explicit**, **centralized**, **reusable**, and **predictable**.


## Installation

```bash
go get github.com/rrrudolph/go-validator
```


## How It Works

### 1. Define a schema

You describe each request body as a tree of `Field` configs: scalars, arrays, objects, enums, ranges, required/default, and custom validators. Helpers cut down boilerplate.

```go
package myapp

import v "github.com/rrrudolph/go-validator"

var (
	filterConfig = v.Field{
		OneOf:      []v.OneOf{v.Object},
		Required:   true,
		AllowEmpty: true,
		Properties: map[string]*v.Field{
			"submission_date": v.RangeOrArray(v.TypeDayOrMonthDate, "gte", "lte"),
			"state":           v.ScalarArray(v.TypeString),
		},
	}

	RecordsRequestConfig = map[string]*v.Field{
		"filter": &filterConfig,
		"fields": {
			OneOf: []v.OneOf{v.Array},
			Items: &v.Field{
				OneOf:         []v.OneOf{v.Scalar},
				ScalarType:    v.TypeString,
				StrEnumValues: getAllowedFields(), // your app’s list
			},
			Default: []any{"id", "state", "date"},
		},
		"limit": v.DefaultMinMaxInteger(10, 1, 100),
	}

	RecordsSchema = v.MustCompile("records", RecordsRequestConfig)
)
```

Invalid configs (e.g. array without `Items`) fail at compile time via `MustCompile`.

### 2. Use in a handler

Validate the raw body, then decode into a typed struct. Return structured errors when validation fails.

```go
// ValidateRequest is middleware: parse JSON, validate with schema, decode into T, set on context.
func ValidateRequest[T any](schema *v.CompiledSchema) gin.HandlerFunc {
	return func(c *gin.Context) {
		var raw map[string]any
		if err := c.ShouldBindJSON(&raw); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		validated, errs := schema.Validate(raw)
		if len(errs) > 0 {
			c.JSON(http.StatusBadRequest, errs.ToStandardErrors())
			c.Abort()
			return
		}

		var req T
		if err := v.DecodeValidated(validated, &req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Set("validatedBody", &req)
		c.Next()
	}
}

// Register route with the schema and request type.
r.POST("/records",
	ValidateRequest[RecordsRequest](RecordsSchema),
	handler.Records,
)
```

In the handler, the body is already validated and typed:

```go
func (h *Handler) Records(c *gin.Context) {
	req := c.MustGet("validatedBody").(*RecordsRequest)
	// req.Filter, req.Fields, req.Limit are set; empty fields got defaults from schema
}
```

### 3. Error response

Validation errors are path + message. `ToStandardErrors()` turns them into a consistent API shape (e.g. status/title/detail with path in detail):

```json
{
  "errors": [
    {
      "status": 400,
      "title": "Invalid Request",
      "detail": "filter.submission_date.lte: invalid format (expected YYYY-MM-DD or YYYY-MM): \"2024\""
    },
    {
      "status": 400,
      "title": "Invalid Request",
      "detail": "fields[1]: must be one of: id, state, date, ..."
    }
  ]
}
```

For machine-friendly path/message only, use `errs.ToAPIResponse()`. Date fields in validated output are strings (e.g. `"2024-01-15"`), so struct fields can be `string`.


## Design principles

- **Compile once, validate many** — Schemas are compiled at startup into a form that avoids repeated reflection and config parsing on each request.
- **Explicit over implicit** — Rules live in schema definitions, not struct tags, so they’re easy to review and reuse (e.g. shared filter config).
- **Stable error format** — Errors are structured and path-based, so API clients get a predictable shape.
- **Type-safe output** — Validated data is decoded into your structs via `DecodeValidated`; generics let the middleware stay type-safe per route.


## Tradeoffs

- **More upfront code** — You define schemas explicitly instead of relying on tags.
- **Not a full JSON Schema engine** — Focused on API request bodies (nested objects/arrays/scalars, enums, ranges, defaults).
- **Best for request validation** — Optimized for validating and decoding JSON request payloads, not general-purpose document validation.


## When to use

`go-validator` is a good fit when you:

- Build JSON APIs and want **centralized** validation
- Need **stable, structured error responses** with paths
- Want to **reuse** rules (e.g. shared filter or pagination shape) across endpoints
- Prefer **explicit schemas** over tag-based validation


## Testing

Schemas can be tested without HTTP: compile with `CompileConfig`, call `schema.Validate(map[string]any{...})`, and assert on the returned data and `ValidationErrors`. Invalid configs are caught at compile time or via `CompileConfig` in tests.
