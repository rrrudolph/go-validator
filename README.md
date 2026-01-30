# go-validator

Pre-compiled, flat schemas for validating API request bodies. Define your request shape, compile at startup, then validate raw JSON and get typed structs + stable error responses.

The existing validation stuff out there is not customizable enough for our needs and leaves a lot of validation still needing to be done in handlers, which in our case means lots of copy paste code. Plus it's difficult to get beautiful error messages.

The underlying `Field` config looks more or less like jsonschema.

```go
type Field struct {
	OneOf                []OneOf                 // [Array, Object, Scalar]
	ScalarType           ScalarType              // bool, string, custom date types, etc.
	Items                *Field            // for arrays: config for each item
	Properties           map[string]*Field // for objects: config for each property
	AdditionalValidators []ValidatorFunc         // custom validation functions like gte/lte checks, min/max, etc.
	StrEnumValues        []string                // only allow these string values
	IntEnumValues        []int                   // only allow these integer values
	Required             bool                    // default false
	AllowEmpty           bool                    // empty, default false
	Default              any                     // default value if field is missing or empty
}
```

But there are helper funcs to save a lot of the boilerplate, which makes a fairly clean interface.

```go
filtersConfig = map[string]*v.Field{
	"submission_date": v.RangeOrArray(v.TypeDayOrMonthDate, "gte", "lte"),
	"salary": 		   v.GteZeroRangeOrArray(v.TypeInteger, "gte", "lte"),

	"lot_career_area":      v.ScalarArray(v.TypeInteger),
	"lot_career_area_name": v.ScalarArray(v.TypeString),

	"is_certified":    v.ScalarField(v.TypeBoolean, false),
}

FilterConfig = v.Field{
	OneOf:      []v.OneOf{v.Object},
	Required:   true,
	AllowEmpty: true,
	Properties: filterProperties,
}

RecordsRequestConfig = map[string]*v.Field{
	"filter": &FilterConfig,
	"fields": {
		OneOf: []v.OneOf{v.Array},
		Items: &v.Field{
			OneOf:         []v.OneOf{v.Scalar},
			ScalarType:    v.TypeString,
			StrEnumValues: GetAllRecordFields(),  // client implemented
		},
		Default: DefaultRecordsFields,
	},
	"order": {
		OneOf:         []v.OneOf{v.Scalar},
		ScalarType:    v.TypeString,
		StrEnumValues: []string{"score", "notice_date", "layoff_date"},
		Default:       "score",
	},
	"limit": v.DefaultMinMaxInteger(10, 1, 100),
}
```

Configuration errors are caught at compile/startup. Use `v.MustCompile(name, config)` in `init()` — the name appears in the panic. `v.CompileConfig(config)` returns an error and exists for tests.
```go
func init() {
	var RecordsSchema = v.MustCompile("records", RecordsRequestConfig)
}
```

Define a middleware function to call the validator.

```go
// ValidateRequest validates the body, decodes into T, and sets the result on the context.
func ValidateRequest[T any](schema *v.CompiledSchema) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var raw map[string]any
		if err := gctx.ShouldBindJSON(&raw); err != nil {
			gctx.Error(apimodel.NewInvalidRequestError(err))
			gctx.Abort()
			return
		}

		validated, errs := schema.Validate(raw)
		if len(errs) > 0 {
			// This outputs the standard error format. For machine friendly response use errs.ToAPIResponse().
			c.JSON(http.StatusBadRequest, errs.ToStandardErrors())
			gctx.Abort()
			return
		}

		var req T
		if err := v.DecodeValidated(validated, &req); err != nil {
			gctx.Error(apimodel.NewInternalServerError(err))
			gctx.Abort()
			return
		}

		gctx.Set("validatedBody", &req)
		gctx.Next()
	}
}
```


Then register it in the router. All standard request bodies are defined as structs in the library.

```go
r.POST(
	"/records",
	lib.ValidateRequest[v.RecordsRequest](lib.RecordsSchema),
	handler.Records
)
```

And behold, `errs.ToStandardErrors()`...

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
            "detail": "rank.extra_metrics[1]: must be one of: amended_petition, average_salary, ..."
        }
    ]
}
```

Get the request struct from the context.

```go
func (h *Handler) Records(gctx *gin.Context) {
	req := gctx.MustGet("validatedBody").(*lib.RecordsRequest)
}
```

