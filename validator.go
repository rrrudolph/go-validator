package validation

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// ── Constants & Types ──────────────────────────────────────────────────────────

type OneOf string
type ScalarType string

type KindMask uint8 // bitmask for "OneOf"
const (
	// top level types
	Array  OneOf = "array"  // []string, []int, []year, ...
	Object OneOf = "object" // {gte: ..., lte: ...}
	Scalar OneOf = "scalar" //  bool, ...

	// this replaces OneOf with bitmask in CompiledField for speed
	KindScalar KindMask = 1 << iota
	KindArray
	KindObject

	// item types
	TypeBoolean         ScalarType = "boolean"
	TypeString          ScalarType = "string"
	TypeFloat           ScalarType = "float"           // accepts string version
	TypeInteger         ScalarType = "integer"         // accepts string version
	TypeFullDate        ScalarType = "dayDate"         // YYYY-MM-DD
	TypeMonthDate       ScalarType = "monthDate"       // YYYY-MM
	TypeYearDate        ScalarType = "yearDate"        // YYYY
	TypeDayOrMonthDate  ScalarType = "dayOrMonthDate"  // YYYY-MM-DD or YYYY-MM
	TypeMonthOrYearDate ScalarType = "monthOrYearDate" // YYYY-MM or YYYY

)

type ValidatorFunc func(ctx *ValidationCtx, value any, config *CompiledField) error

type NestedFields map[string]ScalarType

type RangeKeys struct {
	Lower string // "gte", "lower_bound", "start", etc.
	Upper string // "lte", "upper_bound", "end", etc.
}
type FieldConfig struct {
	OneOf                []OneOf                 // [Array, Object, Scalar]
	ScalarType           ScalarType              // bool, string, custom date types, etc.
	Items                *FieldConfig            // for arrays: config for each item
	Properties           map[string]*FieldConfig // for objects: config for each property
	AdditionalValidators []ValidatorFunc         // custom validation functions like gte/lte checks, min/max, etc.
	StrEnumValues        []string                // only allow these string values
	IntEnumValues        []int                   // only allow these integer values
	Required             bool                    // default false
	AllowEmpty           bool                    // for top level request filtering like {"filter": {}}, default false
	Default              any                     // optional default value if field is missing
}

// ── Additional Validators ──────────────────────────────────────────────────────────

func ValidateRangeOrder(keys RangeKeys) ValidatorFunc {
	return func(ctx *ValidationCtx, value any, _ *CompiledField) error {
		// good data is required here so skip if there are already errors
		if ctx.HasErrors() {
			return nil
		}

		obj, ok := value.(map[string]any)
		if !ok {
			ctx.AddErrorf("range must be an object, got %T", value)
			return nil
		}

		lowerVal, hasLower := obj[keys.Lower]
		upperVal, hasUpper := obj[keys.Upper]

		if !hasLower || !hasUpper {
			return nil
		}

		if lowerVal == nil || upperVal == nil {
			// Already should have error from scalar validation; skip
			return nil
		}

		switch lower := lowerVal.(type) {
		case int64:
			upper, _ := upperVal.(int64)
			if lower > upper {
				ctx.AddErrorf("%s (%d) must not be greater than %s (%d)", keys.Lower, lower, keys.Upper, upper)
			}

		case float64:
			upper, _ := upperVal.(float64)
			if lower > upper {
				ctx.AddErrorf("%s (%.f) must not be greater than %s (%.f)", keys.Lower, lower, keys.Upper, upper)
			}

		case time.Time:
			upper, _ := upperVal.(time.Time)
			if lower.After(upper) {
				ctx.AddErrorf("%s (%s) must not be after %s (%s)",
					keys.Lower, lower.Format("2006-01-02"),
					keys.Upper, upper.Format("2006-01-02"))
			}

		default:
			return nil
		}

		return nil
	}
}

func ValidateMinMax(params map[string]int64) ValidatorFunc {
	return func(ctx *ValidationCtx, value any, _ *CompiledField) error {
		num, ok := value.(int64)
		if !ok {
			ctx.AddErrorf("internal type error: expected int64, got %T", value)
			return nil // or return some error if you want to propagate
		}

		if min, has := params["min"]; has {
			if num < int64(min) {
				ctx.AddErrorf("must be >= %d", min)
			}
		}

		if max, has := params["max"]; has {
			if num > int64(max) {
				ctx.AddErrorf("must be <= %d", max)
			}
		}

		return nil
	}
}

func ValidateRequired(required []string) ValidatorFunc {
	return func(ctx *ValidationCtx, value any, _ *CompiledField) error {
		obj, ok := value.(map[string]any)
		if !ok {
			return nil // wrong type → skip (or add error if you want strictness)
		}

		var missing []string
		for _, f := range required {
			if _, exists := obj[f]; !exists {
				missing = append(missing, f)
			}
		}

		if len(missing) == 0 {
			return nil
		}

		if len(missing) == 1 {
			ctx.AddErrorf("%s is required", missing[0])
		} else {
			ctx.AddErrorf("required fields missing: %s", strings.Join(missing, ", "))
		}

		return nil
	}
}

// RequireAtLeastOneOf adds an error if the object does not have at least one of
// the given keys with a non-nil, non-empty value. Use for "at least one of X or Y" validation.
func RequireAtLeastOneOf(fieldNames ...string) ValidatorFunc {
	return func(ctx *ValidationCtx, value any, _ *CompiledField) error {
		if ctx.HasErrors() {
			return nil
		}
		obj, ok := value.(map[string]any)
		if !ok {
			ctx.AddErrorf("expected object, got %T", value)
			return nil
		}
		for _, name := range fieldNames {
			v, exists := obj[name]
			if !exists || v == nil {
				continue
			}
			if m, ok := v.(map[string]any); ok && len(m) == 0 {
				continue
			}
			return nil // at least one present and non-empty
		}
		ctx.AddErrorf("at least one of %s is required", strings.Join(fieldNames, ", "))
		return nil
	}
}

// RequireAtLeastOneRange validates that at least one of the given fields exists and is
// an object with both lower and upper keys (e.g. gte/lte). Use for timeseries requests
// where multiple date fields are available.
func RequireAtLeastOneRange(fieldNames []string, keys RangeKeys) ValidatorFunc {
	return func(ctx *ValidationCtx, value any, _ *CompiledField) error {
		if ctx.HasErrors() {
			return nil
		}
		obj, ok := value.(map[string]any)
		if !ok {
			ctx.AddErrorf("expected object, got %T", value)
			return nil
		}
		for _, name := range fieldNames {
			v, exists := obj[name]
			if !exists || v == nil {
				continue
			}
			m, ok := v.(map[string]any)
			if !ok {
				continue // not a range object; per-field validation will error
			}
			_, hasLower := m[keys.Lower]
			_, hasUpper := m[keys.Upper]
			if hasLower && hasUpper {
				return nil // at least one valid range object
			}
		}
		ctx.AddErrorf("at least one of %s is required and must be an object with %s and %s",
			strings.Join(fieldNames, ", "), keys.Lower, keys.Upper)
		return nil
	}
}

// ── Core Context ───────────────────────────────────────────────────────────────

// this is helpful to track path and errors during recursive validation
type ValidationError struct {
	Path string `json:"path"`
	Msg  string `json:"msg"`
}

type ValidationErrors []ValidationError

func (ve *ValidationErrors) Append(path string, err error) {
	*ve = append(*ve, ValidationError{Path: path, Msg: err.Error()})
}

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.Grow(120 * len(e)) // rough estimation

	for i, err := range e {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Path)
		sb.WriteString(" ")
		sb.WriteString(strings.TrimRight(err.Msg, "."))
	}
	sb.WriteByte('.')

	return sb.String()
}

// Sorted returns a copy of the errors sorted by Path for stable, deterministic API responses.
func (e ValidationErrors) Sorted() ValidationErrors {
	if len(e) <= 1 {
		return e
	}
	out := make(ValidationErrors, len(e))
	copy(out, e)
	slices.SortFunc(out, func(a, b ValidationError) int { return strings.Compare(a.Path, b.Path) })
	return out
}

// APIErrorEntry is a single error in an API error response.
type APIErrorEntry struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// APIErrorResponse is a stable JSON shape for validation errors in API responses.
type APIErrorResponse struct {
	Errors []APIErrorEntry `json:"errors"`
}

// ToAPIResponse returns a sorted, machine-friendly shape for JSON responses.
func (e ValidationErrors) ToAPIResponse() APIErrorResponse {
	sorted := e.Sorted()
	out := make([]APIErrorEntry, len(sorted))
	for i, err := range sorted {
		out[i] = APIErrorEntry{Path: err.Path, Message: err.Msg}
	}
	return APIErrorResponse{Errors: out}
}

type ValidationCtx struct {
	PathStack    []string
	Errors       ValidationErrors
	CurrentDepth int
	MaxDepth     int  // prevent stack bomb / DoS
	StopOnFirst  bool // fast-fail mode
}

// NewValidationCtx returns a context used during Validate. MaxDepth defaults to 20
// (nesting beyond that returns "maximum nesting depth exceeded") to prevent stack
// bomb / DoS from deeply nested input; 20 is sufficient for typical API request bodies.
func NewValidationCtx() *ValidationCtx {
	return &ValidationCtx{
		MaxDepth: 20,
	}
}

func (c *ValidationCtx) Push(pathPart string) {
	c.PathStack = append(c.PathStack, pathPart)
	c.CurrentDepth++
}

func (c *ValidationCtx) Pop() {
	if len(c.PathStack) > 0 {
		c.PathStack = c.PathStack[:len(c.PathStack)-1]
	}
	c.CurrentDepth--
}

func (c *ValidationCtx) CurrentPath() string {
	if len(c.PathStack) == 0 {
		return ""
	}

	var sb strings.Builder

	for i, seg := range c.PathStack {
		if i == 0 {
			// First segment: never gets a leading dot or bracket
			sb.WriteString(seg)
			continue
		}

		prev := c.PathStack[i-1]

		// Decide whether to put a dot before this segment
		isCurrentIndex := isNumeric(seg)
		isPrevIndex := isNumeric(prev)

		if isCurrentIndex {
			// Current segment is array index → use [n] notation, no dot
			sb.WriteString("[")
			sb.WriteString(seg)
			sb.WriteString("]")
		} else {
			// Current segment is object key
			if isPrevIndex {
				// Coming from array index → no dot
				sb.WriteString(seg)
			} else {
				// Coming from object key → use dot
				sb.WriteString(".")
				sb.WriteString(seg)
			}
		}
	}

	return sb.String()
}

// Helper — more explicit than inline strconv every time
func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func (c *ValidationCtx) AddError(msg string) {
	c.Errors = append(c.Errors, ValidationError{
		Path: c.CurrentPath(),
		Msg:  msg,
	})
}

func (c *ValidationCtx) AddErrorf(format string, args ...any) {
	c.AddError(fmt.Sprintf(format, args...))
}

func (c *ValidationCtx) HasErrors() bool {
	return len(c.Errors) > 0
}

// ── Main recursive validation entry point ─────────────────────────────────────

func validateNode(
	ctx *ValidationCtx,
	node gjson.Result,
	cf *CompiledField,
) any {
	if ctx.CurrentDepth > ctx.MaxDepth {
		ctx.AddErrorf("maximum nesting depth exceeded")
		return nil
	}

	if !node.Exists() {
		if cf.Required {
			ctx.AddError("field is required")
			return nil
		}
		if cf.Default != nil {
			return cf.Default
		}
		return nil // optional + no default → absent is ok
	}

	// ── Type dispatch ───────────────────────────────────────────────────────
	switch {
	case node.Type == gjson.True || node.Type == gjson.False || node.Type == gjson.Number || node.Type == gjson.String:
		if cf.Kind&KindScalar == 0 {
			allowed := strings.Join(cf.AllowedTypes, ", ")
			ctx.AddErrorf("scalar not allowed, must be one of: %s", allowed)
			return nil
		}
		return validateScalar(ctx, node, cf)

	case node.IsArray():
		if cf.Kind&KindArray == 0 {
			allowed := strings.Join(cf.AllowedTypes, ", ")
			ctx.AddErrorf("array not allowed, must be one of: %s", allowed)
			return nil
		}
		return validateArray(ctx, node, cf)

	case node.IsObject():
		if cf.Kind&KindObject == 0 {
			allowed := strings.Join(cf.AllowedTypes, ", ")
			ctx.AddErrorf("object not allowed, must be one of: %s", allowed)
			return nil
		}
		return validateObject(ctx, node, cf)
	default:
		ctx.AddErrorf("unsupported value type: %v", node.Type)
		return nil
	}
}

// ── Scalar ─────────────────────────────────────────────────────────────────────

func validateScalar(ctx *ValidationCtx, node gjson.Result, cf *CompiledField) any {
	// Safety net - should be handled by caller
	if !node.Exists() {
		if cf.Required {
			ctx.AddError("field is required")
		}
		return cf.Default
	}

	var parsed any

	switch cf.ScalarType {
	case TypeString:
		if node.Type != gjson.String {
			ctx.AddError("must be a string")
			return nil
		}
		parsed = node.String()

		// enforce non-empty strings unless AllowEmpty is true
		if parsed == "" && !cf.AllowEmpty {
			ctx.AddError("cannot be empty")
			return nil
		}

	case TypeBoolean:
		if node.Type != gjson.True && node.Type != gjson.False {
			ctx.AddError("must be a boolean")
			return nil
		}
		parsed = node.Bool()

	case TypeInteger:
		if node.Type == gjson.Number {
			f := node.Float()
			i := int64(f)
			if f != float64(i) {
				ctx.AddErrorf("must be an integer, got %v", f)
				return nil
			}
			parsed = i
		} else if node.Type == gjson.String {
			s := node.String()
			i, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				ctx.AddErrorf("invalid integer: %q", s)
				return nil
			}
			parsed = i
		} else {
			ctx.AddError("must be a number or integer string")
			return nil
		}

	case TypeFloat:
		if node.Type != gjson.Number {
			ctx.AddError("must be a number")
			return nil
		}
		parsed = node.Float()

	case TypeFullDate: // YYYY-MM-DD
		if node.Type != gjson.String {
			ctx.AddError("must be a string in YYYY-MM-DD format")
			return nil
		}
		s := node.String()
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			ctx.AddErrorf("invalid format (expected YYYY-MM-DD): %q", s)
			return nil
		}
		parsed = t

	case TypeMonthDate: // YYYY-MM → normalize to YYYY-MM-01
		if node.Type != gjson.String {
			ctx.AddError("must be a string")
			return nil
		}
		s := node.String()
		t, err := time.Parse("2006-01", s)
		if err != nil {
			ctx.AddErrorf("invalid format (expected YYYY-MM): %q", s)
			return nil
		}
		// Normalize to first day of month
		t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		parsed = t

	case TypeYearDate: // YYYY → normalize to YYYY-01-01
		if node.Type != gjson.String {
			ctx.AddError("year must be a string (YYYY)")
			return nil
		}
		s := node.String()
		if len(s) != 4 {
			ctx.AddErrorf("invalid format (expected YYYY): %q", s)
			return nil
		}
		y, err := strconv.Atoi(s)
		if err != nil {
			ctx.AddErrorf("invalid year: %q", s)
			return nil
		}
		parsed = time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)

	case TypeDayOrMonthDate: // YYYY-MM-DD or YYYY-MM → normalize month to YYYY-MM-01
		if node.Type != gjson.String {
			ctx.AddError("must be a string")
			return nil
		}
		s := node.String()
		var t time.Time
		var err error

		// Try full date first
		t, err = time.Parse("2006-01-02", s)
		if err == nil {
			parsed = t
			break
		}

		// Then month → normalize
		t, err = time.Parse("2006-01", s)
		if err != nil {
			ctx.AddErrorf("invalid format (expected YYYY-MM-DD or YYYY-MM): %q", s)
			return nil
		}
		t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		parsed = t

	case TypeMonthOrYearDate: // YYYY-MM or YYYY → normalize to YYYY-MM-01 or YYYY-01-01
		if node.Type != gjson.String {
			ctx.AddError("must be a string")
			return nil
		}
		s := node.String()
		var t time.Time
		var err error

		// Try month first
		t, err = time.Parse("2006-01", s)
		if err == nil {
			t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
			parsed = t
			break
		}

		// Then year
		if len(s) != 4 {
			ctx.AddErrorf("invalid format (expected YYYY-MM or YYYY): %q", s)
			return nil
		}
		y, err := strconv.Atoi(s)
		if err != nil {
			ctx.AddErrorf("invalid year: %q", s)
			return nil
		}
		parsed = time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)

	default:
		ctx.AddErrorf("unsupported scalar type: %q", cf.ScalarType)
		return nil
	}

	// Enum validation: enum values are set on the field config (e.g. cfg.Items for array items).
	if len(cf.StrEnumValues) > 0 || len(cf.IntEnumValues) > 0 {
		if !isAllowedEnumValue(parsed, cf) {
			var allowedStr string

			if len(cf.StrEnumValues) > 0 {
				allowedStr = strings.Join(cf.StrEnumValues, ", ")
			} else if len(cf.IntEnumValues) > 0 {
				var strs []string
				for _, v := range cf.IntEnumValues {
					strs = append(strs, strconv.Itoa(v))
				}
				allowedStr = strings.Join(strs, ", ")
			}

			if allowedStr == "" {
				allowedStr = "(no values defined)"
			}

			ctx.AddErrorf("must be one of: %s", allowedStr)
			return nil
		}
	}

	// Run additional validators
	for _, validator := range cf.AdditionalValidators {
		if err := validator(ctx, parsed, cf); err != nil {
			ctx.AddError(err.Error())
			if ctx.StopOnFirst {
				return nil
			}
		}
	}

	return parsed
}

func isAllowedEnumValue(v any, cf *CompiledField) bool {
	switch x := v.(type) {
	case string:
		return slices.Contains(cf.StrEnumValues, x)
	case int64:
		for _, allowed := range cf.IntEnumValues {
			if int64(allowed) == x {
				return true
			}
		}
	case float64: // rare, but if someone passes float for integer enum
		i := int64(x)
		if float64(i) == x {
			for _, allowed := range cf.IntEnumValues {
				if int64(allowed) == i {
					return true
				}
			}
		}
	}

	return false
}

func ValidateNonEmptyString() ValidatorFunc {
	return func(ctx *ValidationCtx, value any, _ *CompiledField) error {
		s, ok := value.(string)
		if !ok {
			return nil // let type validator handle this
		}
		if strings.TrimSpace(s) == "" {
			ctx.AddError("cannot be empty")
		}
		return nil
	}
}

// ── Array ──────────────────────────────────────────────────────────────────────

func validateArray(ctx *ValidationCtx, node gjson.Result, cf *CompiledField) []any {
	if cf.Items == nil {
		ctx.AddError("array items configuration missing")
		return nil
	}

	result := make([]any, 0, len(node.Array()))

	node.ForEach(func(key, value gjson.Result) bool {
		ctx.Push(key.String())
		parsed := validateNode(ctx, value, cf.Items)
		ctx.Pop()

		if !ctx.HasErrors() {
			result = append(result, parsed)
		}

		// Only stop early if configured to fast-fail
		return !ctx.StopOnFirst || !ctx.HasErrors()
	})

	if !cf.AllowEmpty && len(node.Array()) == 0 {
		ctx.AddError("cannot be empty array")
	}

	return result
}

// ── Object ─────────────────────────────────────────────────────────────────────

func validateObject(ctx *ValidationCtx, node gjson.Result, cf *CompiledField) map[string]any {
	result := make(map[string]any)

	// Process known fields
	var parsed any
	for propName, propCf := range cf.Properties {
		propValue := node.Get(propName)
		ctx.Push(propName)
		parsed = validateNode(ctx, propValue, propCf)
		ctx.Pop()

		if parsed != nil || propValue.Exists() || propCf.Default != nil {
			if parsed == nil && propCf.Default != nil {
				result[propName] = propCf.Default
			} else {
				result[propName] = parsed
			}
		}
	}

	// Inside validateObject, the unknown fields rejection loop:
	// Reject unknown fields
	node.ForEach(func(key, _ gjson.Result) bool {
		keyStr := key.String()
		propCfg, known := cf.Properties[keyStr]
		if !known {
			ctx.Push(keyStr)

			// Get the ALLOWED PROPERTIES FOR THIS NESTED FIELD
			var allowed []string
			if propCfg != nil && propCfg.Properties != nil {
				for name := range propCfg.Properties {
					allowed = append(allowed, name)
				}
				slices.Sort(allowed)
			}

			ctx.AddErrorf("unknown field %q", keyStr)
			ctx.Pop()
		}
		return true
	})

	// Run custom validators (range order, etc)
	for _, validator := range cf.AdditionalValidators {
		if err := validator(ctx, result, cf); err != nil {
			ctx.AddError(err.Error())
			if ctx.StopOnFirst {
				return nil
			}
		}
	}

	if !cf.AllowEmpty && len(result) == 0 {
		ctx.AddError("cannot be empty object")
	}

	return result
}

// ── Root entry point ───────────────────────────────────────────────────────────

// Validate validates a pre-decoded JSON value (e.g. map[string]any from Gin).
// This is the preferred entry point when the body is already unmarshaled.
// All unknown root field errors and all validation errors are collected so API
// responses can return a full list of issues in one response.
func (s *CompiledSchema) Validate(value any) (map[string]any, ValidationErrors) {
	root, ok := value.(map[string]any)
	if !ok {
		return nil, ValidationErrors{{
			Path: "",
			Msg:  "root must be a JSON object",
		}}
	}

	ctx := NewValidationCtx()
	validated := make(map[string]any)

	// Collect all unknown root field errors
	for key := range root {
		if _, known := s.fields[key]; !known {
			ctx.Push(key)
			ctx.AddErrorf("unknown field %q", key)
			ctx.Pop()
		}
	}

	// Validate known fields
	for field, cf := range s.fields {
		rawVal, exists := root[field]

		var node gjson.Result
		if exists {
			bytes, err := json.Marshal(rawVal)
			if err != nil {
				ctx.Push(field)
				ctx.AddErrorf("internal marshal error for field %q: %v", field, err)
				ctx.Pop()
				continue
			}
			node = gjson.ParseBytes(bytes)
		}

		ctx.Push(field)
		parsed := validateNode(ctx, node, cf)
		ctx.Pop()

		if parsed != nil || exists {
			validated[field] = parsed
		} else if cf.Default != nil {
			// Apply default for missing optional fields
			validated[field] = cf.Default
		}
	}

	return validated, ctx.Errors
}

// DecodeValidated decodes the validated map into a struct pointer using
// json marshal/unmarshal so struct fields with json tags are filled. Use after
// Validate to get typed request bodies in handlers.
func DecodeValidated(validated map[string]any, requestStruct any) error {
	if validated == nil || requestStruct == nil {
		return fmt.Errorf("validated map and requestStruct must be non-nil")
	}
	bytes, err := json.Marshal(validated)
	if err != nil {
		return err
	}
	return json.Unmarshal(bytes, requestStruct)
}

// ── config validation and compiling ──────────────────────────────────────────────

// This makes the runtime validation table-driven, no runtime type guessing
type CompiledField struct {
	Path                 string
	Kind                 KindMask // replaces []OneOf
	AllowedTypes         []string // friendly names of allowed types for error messages
	ScalarType           ScalarType
	Items                *CompiledField
	Properties           map[string]*CompiledField
	Required             bool
	AllowEmpty           bool
	Default              any
	StrEnumValues        []string
	IntEnumValues        []int
	AdditionalValidators []ValidatorFunc
}

type CompiledSchema struct {
	fields map[string]*CompiledField
}

// Public constructor
func CompileConfig(config map[string]*FieldConfig) (*CompiledSchema, error) {
	fields, err := validateAndCompileConfig(config)
	if err != nil {
		return nil, err
	}

	return &CompiledSchema{
		fields: fields,
	}, nil
}

// ValidateAndCompile compiles a single FieldConfig map into a precompiled schema
func validateAndCompileConfig(config map[string]*FieldConfig) (map[string]*CompiledField, error) {
	compiled := make(map[string]*CompiledField, len(config))

	for field, cfg := range config {
		if cfg == nil {
			return nil, fmt.Errorf("field %q is nil", field)
		}

		// Start recursion with the field name as initial path
		cf, err := compileField(field, *cfg)
		if err != nil {
			return nil, err
		}

		compiled[field] = cf
	}

	return compiled, nil
}

func compileField(parentPath string, cfg FieldConfig) (*CompiledField, error) {
	currentPath := parentPath
	if parentPath != "" && parentPath != "." {
		currentPath = parentPath
	}

	// First compute kind — very important!
	var kind KindMask
	for _, o := range cfg.OneOf {
		switch o {
		case Scalar:
			kind |= KindScalar
		case Array:
			kind |= KindArray
		case Object:
			kind |= KindObject
		default:
			return nil, fmt.Errorf("%s: invalid OneOf value %q", currentPath, o)
		}
	}

	// Now we can safely populate allowed types
	var types []string
	if kind&KindObject != 0 {
		types = append(types, "object")
	}
	if kind&KindArray != 0 {
		types = append(types, "array")
	}
	if kind&KindScalar != 0 {
		types = append(types, "scalar")
	}

	// Now create the field
	cf := &CompiledField{
		Path:                 currentPath,
		Required:             cfg.Required,
		AllowEmpty:           cfg.AllowEmpty,
		Default:              cfg.Default,
		StrEnumValues:        cfg.StrEnumValues,
		IntEnumValues:        cfg.IntEnumValues,
		AdditionalValidators: cfg.AdditionalValidators,
		Properties:           nil,
		Items:                nil,
		AllowedTypes:         types, // ← now correct!
	}

	// Optional: debug
	// if len(types) > 0 {
	// 	println("DEBUG: compiling field with types:", currentPath, strings.Join(types, ", "))
	// }

	cf.Kind = kind

	// Scalar check
	if cf.Kind&KindScalar != 0 {
		if cfg.ScalarType == "" {
			return nil, fmt.Errorf("%s: ScalarType required for scalar", currentPath)
		}
		cf.ScalarType = cfg.ScalarType
	}

	// Array items
	if cf.Kind&KindArray != 0 {
		if cfg.Items == nil {
			return nil, fmt.Errorf("%s: Items required for array", currentPath)
		}
		itemPath := currentPath
		if itemPath == "" {
			itemPath = "[]"
		} else {
			itemPath += "[]"
		}
		itemField, err := compileField(itemPath, *cfg.Items)
		if err != nil {
			return nil, err
		}
		cf.Items = itemField
	}

	// Object properties
	if cf.Kind&KindObject != 0 {
		if len(cfg.Properties) == 0 {
			return nil, fmt.Errorf("%s: Properties required for object", currentPath)
		}
		cf.Properties = make(map[string]*CompiledField, len(cfg.Properties))

		for propName, propCfg := range cfg.Properties {
			if propCfg == nil {
				return nil, fmt.Errorf("%s.%s: property config is nil", currentPath, propName)
			}
			propPath := currentPath
			if propPath == "" {
				propPath = propName
			} else {
				propPath += "." + propName
			}

			propField, err := compileField(propPath, *propCfg)
			if err != nil {
				return nil, err
			}
			cf.Properties[propName] = propField
		}
	}

	return cf, nil
}

// Helper functions that allow client configs to be more concise
// ─────────────────────────────────────────────────────────────────────────────

// ScalarField returns a basic scalar field config
func ScalarField(scalarType ScalarType, required bool) *FieldConfig {
	return &FieldConfig{
		OneOf:      []OneOf{Scalar},
		ScalarType: scalarType,
		Required:   required,
	}
}

// ScalarArray returns a simple array of scalars (most common pattern in your config)
func ScalarArray(scalarType ScalarType) *FieldConfig {
	return &FieldConfig{
		OneOf: []OneOf{Array},
		Items: ScalarField(scalarType, false),
	}
}

// RangeOrArray creates a filter field that accepts EITHER:
//   - an array of scalars
//   - or an object with range key(s) ({gte: ..., lte: ...})
//
// Range gets validated for proper ordering (lower <= upper).
func RangeOrArray(scalarType ScalarType, lowerField string, upperField string) *FieldConfig {
	return &FieldConfig{
		OneOf: []OneOf{Array, Object},
		Items: ScalarField(scalarType, false),
		Properties: map[string]*FieldConfig{
			lowerField: ScalarField(scalarType, false),
			upperField: ScalarField(scalarType, false),
		},
		AdditionalValidators: []ValidatorFunc{
			ValidateRangeOrder(RangeKeys{Lower: lowerField, Upper: upperField}),
		},
	}
}

// GteZeroRangeOrArray creates a range object with a minimum bound (>= 0)
// on the lower key, and range-order validation between lower and upper if type is object
func GteZeroRangeOrArray(scalarType ScalarType, lowerField string, upperField string) *FieldConfig {
	cfg := RangeOrArray(scalarType, lowerField, upperField)

	// Attach min >= 0 validator **only to the lower bound field** (gte)
	lowerCfg := cfg.Properties[lowerField]
	if lowerCfg == nil {
		// Safety check - should never happen with correct RangeOrArray
		return cfg
	}

	lowerCfg.AdditionalValidators = append(
		lowerCfg.AdditionalValidators,
		ValidateMinMax(map[string]int64{"min": 0}),
	)

	return cfg
}

// DefaultMinMaxInteger creates a configuration for a typical limit/offset/per-page style field:
// - Must be a positive integer
// - Provides a default value if omitted
func DefaultMinMaxInteger(defaultValue int64, min int64, max int64) *FieldConfig {
	return &FieldConfig{
		OneOf:      []OneOf{Scalar},
		ScalarType: TypeInteger,
		Default:    defaultValue,
		AdditionalValidators: []ValidatorFunc{
			ValidateMinMax(map[string]int64{"min": min, "max": max}),
		},
	}
}
