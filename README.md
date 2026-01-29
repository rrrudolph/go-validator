# go-validator

Pre-compiled, flat schemas for validating API request bodies. Define your request shape once, compile at startup, then validate raw JSON and get typed structs + stable error responses.

---

## Quick start

**1. Define a schema and compile it (once at startup):**

```go
var TotalsSchema *validation.CompiledSchema

func init() {
	config := map[string]*validation.FieldConfig{
		"filter":  &FilterConfig,  // your shared filter object config
		"metrics": validation.ScalarArray(validation.TypeString),
	}
	var err error
	TotalsSchema, err = validation.CompileConfig(config)
	if err != nil {
		panic(err)
	}
}
```

**2. In middleware: validate, decode into your request struct, and save it to context:**

Use a generic middleware so the struct type is known. It validates, calls `DecodeValidated`, then stores the **typed struct** in context so handlers don’t decode again.

```go
// ValidateRequest validates the body, decodes into T, and sets the result on the context.
// Register with the request type: ValidateRequest[TotalsRequest](TotalsSchema)
func ValidateRequest[T any](schema *validation.CompiledSchema) gin.HandlerFunc {
	return func(c *gin.Context) {
		var raw map[string]any
		if err := c.ShouldBindJSON(&raw); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		validated, errs := schema.Validate(raw)
		if len(errs) > 0 {
			c.JSON(http.StatusBadRequest, errs.ToAPIResponse())
			c.Abort()
			return
		}

		var req T
		if err := validation.DecodeValidated(validated, &req); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Set("validatedBody", &req)
		c.Next()
	}
}
```

**3. In the handler: get the struct from context (no DecodeValidated):**

```go
// Register route with the typed middleware:
// router.POST("/totals", ValidateRequest[validation.TotalsRequest](TotalsSchema), handler.GetTotals)

func (h *Handler) GetTotals(c *gin.Context) {
	req := c.MustGet("validatedBody").(*validation.TotalsRequest)

	// req.Filter and req.Metrics are already typed
	query, err := h.buildQuery(req.Filter, req.Metrics)
	// ...
}
```

---

## Validation errors and ToAPIResponse

`Validate` returns `(map[string]any, ValidationErrors)`. When there are errors, return a **stable JSON body** so clients can parse it reliably.

### Use ToAPIResponse for the response body

`ValidationErrors` implements `error`, but for HTTP you want a structured body. Call **`ToAPIResponse()`**:

```go
if len(errs) > 0 {
	c.JSON(http.StatusBadRequest, errs.ToAPIResponse())
	c.Abort()
	return
}
```

That sends a body like:

```json
{
  "errors": [
    { "path": "filter.layoff_date", "message": "cannot be empty object" },
    { "path": "metrics[2]", "message": "must be one of: total_events, ..." }
  ]
}
```

- Errors are **sorted by path** so the response is deterministic.
- `APIErrorResponse` is the struct; you can also use it with your own error envelope (e.g. add `status`, `title`) by mapping from `errs.ToAPIResponse().Errors`.

### Optional: Sorted() only

If you build your own response shape but want stable ordering:

```go
for _, e := range errs.Sorted() {
	// e.Path, e.Msg — in path order
}
```

---

## Typed request structs

Use request structs so handlers get typed fields instead of `map[string]any` and manual casting. When you use the generic **`ValidateRequest[T]`** middleware (see Quick start), the middleware calls **`DecodeValidated`** and stores the struct in context; handlers then do a single type-assert: `req := c.MustGet("validatedBody").(*TotalsRequest)`.

### Request structs (in this package or copy into yours)

The package provides optional structs that match common request shapes; use `json` tags that match your schema keys:

```go
// From this package (see request_structs.go)
type TotalsRequest struct {
	Filter  Filter   `json:"filter"`
	Metrics []string `json:"metrics"`
}

type RecordsRequest struct {
	Filter Filter  `json:"filter"`
	Fields []string `json:"fields"`
	Order  string  `json:"order"`
	Limit  int64   `json:"limit"`
}
```

Define your own the same way: struct fields with `json` tags matching the validated keys.

### Where to call DecodeValidated

- **Recommended:** Use the generic **`ValidateRequest[T]`** middleware so it validates, decodes into `T`, and sets the struct on context. Handlers then do `req := c.MustGet("validatedBody").(*YourRequest)` and use typed fields.
- **Alternatively:** If you don’t use the generic middleware, store the raw validated map and in each handler call `validation.DecodeValidated(validated, &req)` before using the struct.

---

## Schema configuration overview

### FieldConfig

Each field in your request is described by a `FieldConfig`:

| Field | Meaning |
|-------|--------|
| `OneOf` | Allowed shapes: `[]OneOf{Array}`, `[]OneOf{Object}`, `[]OneOf{Scalar}`, or combined (e.g. `[]OneOf{Array, Object}`). |
| `ScalarType` | For scalars: `TypeString`, `TypeInteger`, `TypeBoolean`, `TypeFloat`, or a date type (see below). |
| `Items` | For arrays: config for each element. |
| `Properties` | For objects: map of field name → `*FieldConfig`. |
| `StrEnumValues` | Allowed string values (e.g. `[]string{"year","month"}`). For arrays, set on **Items**. |
| `IntEnumValues` | Allowed integer values. |
| `Required` | If true, field must be present. |
| `AllowEmpty` | If true, empty string / empty array / empty object is allowed. |
| `Default` | Value used when field is missing (optional fields). |
| `AdditionalValidators` | Extra checks (e.g. `ValidateRangeOrder`, `RequireAtLeastOneRange`). |

### Scalar types

- **Basic:** `TypeString`, `TypeInteger`, `TypeBoolean`, `TypeFloat`
- **Dates:** `TypeFullDate` (YYYY-MM-DD), `TypeMonthDate` (YYYY-MM), `TypeYearDate` (YYYY), `TypeDayOrMonthDate`, `TypeMonthOrYearDate`

### Helper constructors

| Helper | Use case |
|--------|----------|
| `ScalarField(scalarType, required)` | Single value (string, int, date, etc.). |
| `ScalarArray(scalarType)` | Array of scalars. |
| `RangeOrArray(scalarType, lowerKey, upperKey)` | Either an array of scalars or an object with `lowerKey`/`upperKey` (e.g. `gte`/`lte`). Validates lower ≤ upper. |
| `GteZeroRangeOrArray(...)` | Same as above with lower bound ≥ 0. |
| `DefaultMinMaxInteger(defaultVal, min, max)` | Integer with default and min/max (e.g. `limit`). |
| `RequireAtLeastOneRange(fieldNames, RangeKeys{Lower:"gte", Upper:"lte"})` | At least one of the named fields must be present and be a range object (for range queries that can target various fields). |
| `ValidateRangeOrder(RangeKeys{Lower:"gte", Upper:"lte"})` | Add to a field’s `AdditionalValidators` to enforce gte ≤ lte. |

### Example: filter and totals config

```go
var FilterConfig = validation.FieldConfig{
	OneOf:      []validation.OneOf{validation.Object},
	Required:   true,
	AllowEmpty: true,
	Properties: map[string]*validation.FieldConfig{
		"number_employees_affected": validation.GteZeroRangeOrArray(validation.TypeInteger, "gte", "lte"),
		"layoff_date":               validation.RangeOrArray(validation.TypeDayOrMonthDate, "gte", "lte"),
		"state":                     validation.ScalarArray(validation.TypeString),
		// ...
	},
}

totalsConfig := map[string]*validation.FieldConfig{
	"filter":  &FilterConfig,
	"metrics": validation.ScalarArray(validation.TypeString), // or with StrEnumValues on Items
}
```

Compile with `validation.CompileConfig(totalsConfig)` and use the result in `ValidateRequest` as in Quick start.

---

## Compilation and init

Compile once at startup (e.g. in `init()` or an explicit setup function). Invalid configs (missing `Items` for arrays, missing `ScalarType` for scalars, etc.) are reported by `CompileConfig`.

```go
func mustCompile(name string, config map[string]*validation.FieldConfig) *validation.CompiledSchema {
	schema, err := validation.CompileConfig(config)
	if err != nil {
		panic(fmt.Sprintf("invalid config for %q: %v", name, err))
	}
	return schema
}

func init() {
	CompiledSchemas.Totals = mustCompile("totals", TotalsRequestConfig)
	CompiledSchemas.Ranking = mustCompile("ranking", RankingRequestConfig)
	// ...
}
```

When you need a variant (e.g. timeseries requires a date range), clone the base config and override:

```go
timeseriesFilter := maps.Clone(FilterConfig.Properties)
timeseriesFilter["submission_date"] = &validation.FieldConfig{
	OneOf:      []validation.OneOf{validation.Object},
	Required:   true,
	Properties: map[string]*validation.FieldConfig{
		"gte": validation.ScalarField(validation.TypeMonthOrYearDate, true),
		"lte": validation.ScalarField(validation.TypeMonthOrYearDate, true),
	},
	AdditionalValidators: []validation.ValidatorFunc{
		validation.ValidateRangeOrder(validation.RangeKeys{Lower: "gte", Upper: "lte"}),
	},
}
```

---

## API quick reference

**Entry points**

- `CompileConfig(config map[string]*FieldConfig) (*CompiledSchema, error)` — compile schema (once at startup).
- `(*CompiledSchema) Validate(value any) (map[string]any, ValidationErrors)` — validate raw JSON (e.g. `map[string]any` from Gin).
- `DecodeValidated(validated map[string]any, out any) error` — decode validated map into a struct (`json` tags).

**Error response**

- `errs.ToAPIResponse()` — `APIErrorResponse{ Errors: []APIErrorEntry{Path, Message} }` for `c.JSON(400, errs.ToAPIResponse())`.
- `errs.Sorted()` — copy of errors sorted by path.

**Types**

- `FieldConfig`, `CompiledSchema`, `ValidationErrors`, `ValidationError` (Path, Msg), `APIErrorResponse`, `APIErrorEntry`.
- `TotalsRequest`, `RecordsRequest`, `RankingRequest`, etc. in `request_structs.go` (optional; copy or adapt).

**Scalar types:** `TypeString`, `TypeInteger`, `TypeBoolean`, `TypeFloat`, `TypeFullDate`, `TypeMonthDate`, `TypeYearDate`, `TypeDayOrMonthDate`, `TypeMonthOrYearDate`.

**OneOf:** `Array`, `Object`, `Scalar`.

**Helpers:** `ScalarField`, `ScalarArray`, `RangeOrArray`, `GteZeroRangeOrArray`, `DefaultMinMaxInteger`, `RequireAtLeastOneRange`, `ValidateRangeOrder`, `RangeKeys`.
