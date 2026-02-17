package validation

import (
	"encoding/json"
	"testing"
)

// ── Test Helpers ────────────────────────────────────────────────────────────────

func assertNoErrors(t *testing.T, errs ValidationErrors) {
	t.Helper()
	if len(errs) > 0 {
		t.Errorf("expected no errors, got %d: %v", len(errs), errs)
	}
}

func assertHasErrors(t *testing.T, errs ValidationErrors, expectedCount int) {
	t.Helper()
	if len(errs) != expectedCount {
		t.Errorf("expected %d errors, got %d: %v", expectedCount, len(errs), errs)
	}
}

func assertErrorContains(t *testing.T, errs ValidationErrors, path, msgSubstring string) {
	t.Helper()
	for _, err := range errs {
		if err.Path == path && contains(err.Msg, msgSubstring) {
			return
		}
	}
	t.Errorf("expected error at path %q containing %q, got errors: %v", path, msgSubstring, errs)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ── Scalar Type Tests ──────────────────────────────────────────────────────────

func TestScalarString(t *testing.T) {
	config := map[string]*Field{
		"name": ScalarField(TypeString, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid string", func(t *testing.T) {
		input := map[string]any{"name": "test"}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["name"] != "test" {
			t.Errorf("expected 'test', got %v", result["name"])
		}
	})

	t.Run("missing optional field", func(t *testing.T) {
		input := map[string]any{}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["name"] != nil {
			t.Errorf("expected nil, got %v", result["name"])
		}
	})

	t.Run("empty string not allowed", func(t *testing.T) {
		input := map[string]any{"name": ""}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "name", "cannot be empty")
	})

	t.Run("wrong type", func(t *testing.T) {
		input := map[string]any{"name": 123}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "name", "must be a string")
	})
}

func TestScalarStringAllowEmpty(t *testing.T) {
	config := map[string]*Field{
		"name": {
			OneOf:      []OneOf{Scalar},
			ScalarType: TypeString,
			AllowEmpty: true,
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{"name": ""}
	result, errs := schema.Validate(input)
	assertNoErrors(t, errs)
	if result["name"] != "" {
		t.Errorf("expected empty string, got %v", result["name"])
	}
}

func TestScalarInteger(t *testing.T) {
	config := map[string]*Field{
		"count": ScalarField(TypeInteger, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid integer", func(t *testing.T) {
		input := map[string]any{"count": 42}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["count"] != int64(42) {
			t.Errorf("expected 42, got %v", result["count"])
		}
	})

	t.Run("integer from string", func(t *testing.T) {
		input := map[string]any{"count": "42"}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["count"] != int64(42) {
			t.Errorf("expected 42, got %v", result["count"])
		}
	})

	t.Run("invalid string", func(t *testing.T) {
		input := map[string]any{"count": "not-a-number"}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "count", "invalid integer")
	})

	t.Run("float not allowed", func(t *testing.T) {
		input := map[string]any{"count": 42.5}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "count", "must be an integer")
	})
}

func TestScalarBoolean(t *testing.T) {
	config := map[string]*Field{
		"active": ScalarField(TypeBoolean, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid boolean true", func(t *testing.T) {
		input := map[string]any{"active": true}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["active"] != true {
			t.Errorf("expected true, got %v", result["active"])
		}
	})

	t.Run("valid boolean false", func(t *testing.T) {
		input := map[string]any{"active": false}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["active"] != false {
			t.Errorf("expected false, got %v", result["active"])
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		input := map[string]any{"active": "true"}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "active", "must be a boolean")
	})
}

func TestScalarFloat(t *testing.T) {
	config := map[string]*Field{
		"price": ScalarField(TypeFloat, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{"price": 99.99}
	result, errs := schema.Validate(input)
	assertNoErrors(t, errs)
	if result["price"] != 99.99 {
		t.Errorf("expected 99.99, got %v", result["price"])
	}
}

// ── Date Type Tests ───────────────────────────────────────────────────────────

func TestScalarFullDate(t *testing.T) {
	config := map[string]*Field{
		"date": ScalarField(TypeFullDate, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid date", func(t *testing.T) {
		input := map[string]any{"date": "2024-01-15"}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		date, ok := result["date"].(string)
		if !ok {
			t.Fatalf("expected string, got %T", result["date"])
		}
		if date != "2024-01-15" {
			t.Errorf("expected 2024-01-15, got %q", date)
		}
	})

	t.Run("invalid format", func(t *testing.T) {
		input := map[string]any{"date": "2024/01/15"}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "date", "invalid format")
	})
}

func TestScalarMonthDate(t *testing.T) {
	config := map[string]*Field{
		"month": ScalarField(TypeMonthDate, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{"month": "2024-01"}
	result, errs := schema.Validate(input)
	assertNoErrors(t, errs)
	month, ok := result["month"].(string)
	if !ok {
		t.Fatalf("expected string, got %T", result["month"])
	}
	if month != "2024-01" {
		t.Errorf("expected 2024-01, got %q", month)
	}
}

func TestScalarYearDate(t *testing.T) {
	config := map[string]*Field{
		"year": ScalarField(TypeYearDate, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{"year": "2024"}
	result, errs := schema.Validate(input)
	assertNoErrors(t, errs)
	year, ok := result["year"].(string)
	if !ok {
		t.Fatalf("expected string, got %T", result["year"])
	}
	if year != "2024" {
		t.Errorf("expected 2024, got %q", year)
	}
}

// ── Enum Tests ──────────────────────────────────────────────────────────────────

func TestScalarEnumString(t *testing.T) {
	config := map[string]*Field{
		"status": {
			OneOf:         []OneOf{Scalar},
			ScalarType:    TypeString,
			StrEnumValues: []string{"active", "inactive", "pending"},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid enum value", func(t *testing.T) {
		input := map[string]any{"status": "active"}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["status"] != "active" {
			t.Errorf("expected 'active', got %v", result["status"])
		}
	})

	t.Run("invalid enum value", func(t *testing.T) {
		input := map[string]any{"status": "invalid"}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "status", "must be one of")
		assertErrorContains(t, errs, "status", "active")
	})
}

func TestScalarEnumInteger(t *testing.T) {
	config := map[string]*Field{
		"priority": {
			OneOf:         []OneOf{Scalar},
			ScalarType:    TypeInteger,
			IntEnumValues: []int{1, 2, 3, 4, 5},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid enum value", func(t *testing.T) {
		input := map[string]any{"priority": 3}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["priority"] != int64(3) {
			t.Errorf("expected 3, got %v", result["priority"])
		}
	})

	t.Run("invalid enum value", func(t *testing.T) {
		input := map[string]any{"priority": 10}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "priority", "must be one of")
	})
}

func TestArrayEnumOnItems(t *testing.T) {
	// Enum values are set at cfg.Items level; each array item is validated against Items enum.
	config := map[string]*Field{
		"fields": {
			OneOf: []OneOf{Array},
			Items: &Field{
				OneOf:         []OneOf{Scalar},
				ScalarType:    TypeString,
				StrEnumValues: []string{"field1", "field2", "field3"},
			},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid enum values in array", func(t *testing.T) {
		input := map[string]any{"fields": []any{"field1", "field2"}}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		fields, ok := result["fields"].([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", result["fields"])
		}
		if len(fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(fields))
		}
	})

	t.Run("invalid enum value in array", func(t *testing.T) {
		input := map[string]any{"fields": []any{"field1", "invalid_field"}}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "fields[1]", "must be one of")
	})

	t.Run("array items validated against Items enum only", func(t *testing.T) {
		// Items has StrEnumValues; only those values are allowed for each element.
		config := map[string]*Field{
			"fields": {
				OneOf: []OneOf{Array},
				Items: &Field{
					OneOf:         []OneOf{Scalar},
					ScalarType:    TypeString,
					StrEnumValues: []string{"field1", "field2"},
				},
			},
		}
		schema, err := CompileConfig(config)
		if err != nil {
			t.Fatalf("failed to compile: %v", err)
		}

		input := map[string]any{"fields": []any{"field1"}}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["fields"] == nil {
			t.Error("expected fields to be validated")
		}

		input2 := map[string]any{"fields": []any{"not_allowed"}}
		_, errs2 := schema.Validate(input2)
		assertHasErrors(t, errs2, 1)
		assertErrorContains(t, errs2, "fields[0]", "must be one of")
	})
}

// ── Array Tests ─────────────────────────────────────────────────────────────────

func TestArrayBasic(t *testing.T) {
	config := map[string]*Field{
		"tags": ScalarArray(TypeString),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid array", func(t *testing.T) {
		input := map[string]any{"tags": []any{"tag1", "tag2", "tag3"}}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		tags, ok := result["tags"].([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", result["tags"])
		}
		if len(tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(tags))
		}
	})

	t.Run("empty array not allowed", func(t *testing.T) {
		input := map[string]any{"tags": []any{}}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "tags", "cannot be empty array")
	})

	t.Run("wrong type", func(t *testing.T) {
		input := map[string]any{"tags": "not-an-array"}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "tags", "scalar not allowed")
	})
}

func TestArrayAllowEmpty(t *testing.T) {
	config := map[string]*Field{
		"tags": {
			OneOf:      []OneOf{Array},
			Items:      ScalarField(TypeString, false),
			AllowEmpty: true,
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{"tags": []any{}}
	result, errs := schema.Validate(input)
	assertNoErrors(t, errs)
	tags, ok := result["tags"].([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result["tags"])
	}
	if len(tags) != 0 {
		t.Errorf("expected empty array, got %d items", len(tags))
	}
}

func TestArrayNested(t *testing.T) {
	config := map[string]*Field{
		"matrix": {
			OneOf: []OneOf{Array},
			Items: &Field{
				OneOf: []OneOf{Array},
				Items: ScalarField(TypeInteger, false),
			},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{"matrix": []any{[]any{1, 2}, []any{3, 4}}}
	result, errs := schema.Validate(input)
	assertNoErrors(t, errs)
	matrix, ok := result["matrix"].([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result["matrix"])
	}
	if len(matrix) != 2 {
		t.Errorf("expected 2 rows, got %d", len(matrix))
	}
}

// ── Object Tests ───────────────────────────────────────────────────────────────

func TestObjectBasic(t *testing.T) {
	config := map[string]*Field{
		"user": {
			OneOf: []OneOf{Object},
			Properties: map[string]*Field{
				"name": ScalarField(TypeString, false),
				"age":  ScalarField(TypeInteger, false),
			},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid object", func(t *testing.T) {
		input := map[string]any{
			"user": map[string]any{
				"name": "John",
				"age":  30,
			},
		}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		user, ok := result["user"].(map[string]any)
		if !ok {
			t.Fatalf("expected map[string]any, got %T", result["user"])
		}
		if user["name"] != "John" {
			t.Errorf("expected 'John', got %v", user["name"])
		}
	})

	t.Run("unknown field", func(t *testing.T) {
		input := map[string]any{
			"user": map[string]any{
				"name":    "John",
				"unknown": "field",
			},
		}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "user.unknown", "unknown field")
	})

	t.Run("empty object not allowed", func(t *testing.T) {
		input := map[string]any{"user": map[string]any{}}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "user", "cannot be empty object")
	})

	t.Run("object with only wrong keys does not get 'cannot be empty object' error", func(t *testing.T) {
		// Client sent keys (wrong/unknown) — we should get specific errors only, not "cannot be empty object".
		input := map[string]any{
			"user": map[string]any{"wrong_key": "value"},
		}
		_, errs := schema.Validate(input)
		for _, e := range errs {
			if e.Msg == "cannot be empty object" {
				t.Errorf("should not report 'cannot be empty object' when client sent keys; got errors: %v", errs)
				break
			}
		}
		assertHasErrors(t, errs, 1) // unknown field only
	})
}

func TestObjectAllowEmpty(t *testing.T) {
	config := map[string]*Field{
		"filter": {
			OneOf:      []OneOf{Object},
			AllowEmpty: true,
			Properties: map[string]*Field{
				"name": ScalarField(TypeString, false),
			},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{"filter": map[string]any{}}
	result, errs := schema.Validate(input)
	assertNoErrors(t, errs)
	filter, ok := result["filter"].(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", result["filter"])
	}
	if len(filter) != 0 {
		t.Errorf("expected empty object, got %d keys", len(filter))
	}
}

// ── Required Field Tests ────────────────────────────────────────────────────────

func TestRequiredField(t *testing.T) {
	config := map[string]*Field{
		"name": ScalarField(TypeString, true),
		"age":  ScalarField(TypeInteger, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("missing required field", func(t *testing.T) {
		input := map[string]any{"age": 30}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "name", "field is required")
	})

	t.Run("present required field", func(t *testing.T) {
		input := map[string]any{"name": "John", "age": 30}
		_, errs := schema.Validate(input)
		assertNoErrors(t, errs)
	})
}

// ── Default Value Tests ────────────────────────────────────────────────────────

func TestDefaultValue(t *testing.T) {
	config := map[string]*Field{
		"limit": DefaultMinMaxInteger(10, 1, 100),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("missing field gets default", func(t *testing.T) {
		input := map[string]any{}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["limit"] != int64(10) {
			t.Errorf("expected default 10, got %v", result["limit"])
		}
	})

	t.Run("provided value overrides default", func(t *testing.T) {
		input := map[string]any{"limit": 20}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["limit"] != int64(20) {
			t.Errorf("expected 20, got %v", result["limit"])
		}
	})
}

func TestEmptyArrayWithDefaultYieldsDefault(t *testing.T) {
	defaultFields := []any{"id", "name"}
	config := map[string]*Field{
		"fields": {
			OneOf:      []OneOf{Array},
			Items:      &Field{OneOf: []OneOf{Scalar}, ScalarType: TypeString},
			AllowEmpty: true,
			Default:    defaultFields,
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("empty array with default yields default in validated output", func(t *testing.T) {
		input := map[string]any{"fields": []any{}}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		got, ok := result["fields"].([]any)
		if !ok {
			t.Fatalf("expected []any, got %T", result["fields"])
		}
		if len(got) != 2 || got[0] != "id" || got[1] != "name" {
			t.Errorf("expected default [id name], got %v", got)
		}
	})
}

// ── Range Validation Tests ────────────────────────────────────────────────────

func TestRangeOrderValidation(t *testing.T) {
	config := map[string]*Field{
		"range": RangeOrArray(TypeInteger, "gte", "lte"),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid range", func(t *testing.T) {
		input := map[string]any{
			"range": map[string]any{
				"gte": 10,
				"lte": 20,
			},
		}
		_, errs := schema.Validate(input)
		assertNoErrors(t, errs)
	})

	t.Run("invalid range order", func(t *testing.T) {
		input := map[string]any{
			"range": map[string]any{
				"gte": 20,
				"lte": 10,
			},
		}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "range", "must not be greater than")
	})

	t.Run("range as array", func(t *testing.T) {
		input := map[string]any{"range": []any{10, 20, 30}}
		_, errs := schema.Validate(input)
		assertNoErrors(t, errs)
	})
}

func TestGteZeroRangeOrArray(t *testing.T) {
	config := map[string]*Field{
		"count": GteZeroRangeOrArray(TypeInteger, "gte", "lte"),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid range with gte >= 0", func(t *testing.T) {
		input := map[string]any{
			"count": map[string]any{
				"gte": 0,
				"lte": 100,
			},
		}
		_, errs := schema.Validate(input)
		assertNoErrors(t, errs)
	})

	t.Run("invalid range with gte < 0", func(t *testing.T) {
		input := map[string]any{
			"count": map[string]any{
				"gte": -5,
				"lte": 100,
			},
		}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "count.gte", "must be >=")
	})
}

func TestRequireAtLeastOneRange(t *testing.T) {
	// Simulates timeseries filter: at least one of layoff_date or notice_date must be a range object (gte/lte)
	config := map[string]*Field{
		"filter": {
			OneOf:      []OneOf{Object},
			AllowEmpty: true, // so "at least one range" is the only requirement when empty
			Properties: map[string]*Field{
				"layoff_date": RangeOrArray(TypeDayOrMonthDate, "gte", "lte"),
				"notice_date": RangeOrArray(TypeDayOrMonthDate, "gte", "lte"),
			},
			AdditionalValidators: []ValidatorFunc{
				RequireAtLeastOneRange([]string{"layoff_date", "notice_date"}, RangeKeys{Lower: "gte", Upper: "lte"}),
			},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("valid - layoff_date is range object with gte and lte", func(t *testing.T) {
		input := map[string]any{
			"filter": map[string]any{
				"layoff_date": map[string]any{"gte": "2024-01", "lte": "2024-12"},
			},
		}
		_, errs := schema.Validate(input)
		assertNoErrors(t, errs)
	})

	t.Run("valid - notice_date is range object with gte and lte", func(t *testing.T) {
		input := map[string]any{
			"filter": map[string]any{
				"notice_date": map[string]any{"gte": "2024-01", "lte": "2024-06"},
			},
		}
		_, errs := schema.Validate(input)
		assertNoErrors(t, errs)
	})

	t.Run("invalid - field present but missing gte/lte (empty object)", func(t *testing.T) {
		input := map[string]any{
			"filter": map[string]any{
				"layoff_date": map[string]any{},
			},
		}
		_, errs := schema.Validate(input)
		if len(errs) < 1 {
			t.Errorf("expected at least 1 error, got %d", len(errs))
		}
		// Per-field validation may report "cannot be empty object", or our validator reports "at least one of ... gte and lte"
		found := false
		for _, e := range errs {
			if contains(e.Msg, "at least one of") && contains(e.Msg, "gte") && contains(e.Msg, "lte") {
				found = true
				break
			}
			if contains(e.Msg, "cannot be empty object") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error about at least one range or empty object, got: %v", errs)
		}
	})

	t.Run("invalid - neither field present", func(t *testing.T) {
		input := map[string]any{
			"filter": map[string]any{},
		}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "filter", "at least one of")
		assertErrorContains(t, errs, "filter", "gte")
		assertErrorContains(t, errs, "filter", "lte")
	})

	t.Run("invalid - field present but only gte (missing lte)", func(t *testing.T) {
		input := map[string]any{
			"filter": map[string]any{
				"layoff_date": map[string]any{"gte": "2024-01"},
			},
		}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "filter", "at least one of")
	})
}

// ── Min/Max Validation Tests ───────────────────────────────────────────────────

func TestMinMaxValidation(t *testing.T) {
	config := map[string]*Field{
		"limit": DefaultMinMaxInteger(10, 1, 100),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("value below min", func(t *testing.T) {
		input := map[string]any{"limit": 0}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "limit", "must be >=")
	})

	t.Run("value above max", func(t *testing.T) {
		input := map[string]any{"limit": 200}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "limit", "must be <=")
	})

	t.Run("value in range", func(t *testing.T) {
		input := map[string]any{"limit": 50}
		_, errs := schema.Validate(input)
		assertNoErrors(t, errs)
	})
}

// ── Nested Structure Tests ──────────────────────────────────────────────────────

func TestNestedObject(t *testing.T) {
	config := map[string]*Field{
		"user": {
			OneOf: []OneOf{Object},
			Properties: map[string]*Field{
				"profile": {
					OneOf: []OneOf{Object},
					Properties: map[string]*Field{
						"name": ScalarField(TypeString, false),
						"bio":  ScalarField(TypeString, false),
					},
				},
			},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{
		"user": map[string]any{
			"profile": map[string]any{
				"name": "John",
				"bio":  "Developer",
			},
		},
	}
	result, errs := schema.Validate(input)
	assertNoErrors(t, errs)
	user, _ := result["user"].(map[string]any)
	profile, _ := user["profile"].(map[string]any)
	if profile["name"] != "John" {
		t.Errorf("expected 'John', got %v", profile["name"])
	}
}

func TestArrayOfObjects(t *testing.T) {
	config := map[string]*Field{
		"users": {
			OneOf: []OneOf{Array},
			Items: &Field{
				OneOf: []OneOf{Object},
				Properties: map[string]*Field{
					"name": ScalarField(TypeString, false),
					"age":  ScalarField(TypeInteger, false),
				},
			},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{
		"users": []any{
			map[string]any{"name": "John", "age": 30},
			map[string]any{"name": "Jane", "age": 25},
		},
	}
	result, errs := schema.Validate(input)
	assertNoErrors(t, errs)
	users, ok := result["users"].([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result["users"])
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

// ── Error Path Tests ───────────────────────────────────────────────────────────

func TestErrorPaths(t *testing.T) {
	config := map[string]*Field{
		"users": {
			OneOf: []OneOf{Array},
			Items: &Field{
				OneOf: []OneOf{Object},
				Properties: map[string]*Field{
					"name": ScalarField(TypeString, true),
					"tags": {
						OneOf: []OneOf{Array},
						Items: ScalarField(TypeString, false),
					},
				},
			},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("nested array error path", func(t *testing.T) {
		input := map[string]any{
			"users": []any{
				map[string]any{
					"name": "John",
					"tags": []any{"tag1", 123}, // invalid type in nested array
				},
			},
		}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		// Check that error path includes array indices (format: users[0]tags[1] - no dot after array index)
		found := false
		for _, err := range errs {
			if (contains(err.Path, "users[0]tags[1]") || contains(err.Path, "users[0].tags[1]")) &&
				contains(err.Msg, "must be a string") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error path to include array indices, got: %v", errs)
		}
	})

	t.Run("missing required in nested object", func(t *testing.T) {
		input := map[string]any{
			"users": []any{
				map[string]any{
					"tags": []any{"tag1"},
				},
			},
		}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		// Path format is "users[0]name" (no dot after array index)
		found := false
		for _, err := range errs {
			if (contains(err.Path, "users[0]name") || contains(err.Path, "users[0].name")) &&
				contains(err.Msg, "field is required") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error at users[0]name or users[0].name, got: %v", errs)
		}
	})
}

// ── Root Level Tests ───────────────────────────────────────────────────────────

func TestRootMustBeObject(t *testing.T) {
	config := map[string]*Field{
		"name": ScalarField(TypeString, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := []any{"not", "an", "object"}
	_, errs := schema.Validate(input)
	assertHasErrors(t, errs, 1)
	assertErrorContains(t, errs, "", "root must be a JSON object")
}

func TestUnknownRootField(t *testing.T) {
	config := map[string]*Field{
		"name": ScalarField(TypeString, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{
		"name":    "John",
		"unknown": "field",
		"another": "one",
	}
	_, errs := schema.Validate(input)
	// All unknown root field errors are collected
	assertHasErrors(t, errs, 2)
	unknownCount := 0
	for _, e := range errs {
		if contains(e.Msg, "unknown field") {
			unknownCount++
		}
	}
	if unknownCount != 2 {
		t.Errorf("expected 2 unknown field errors, got %d", unknownCount)
	}
}

func TestCollectAllRootAndValidationErrors(t *testing.T) {
	config := map[string]*Field{
		"name": ScalarField(TypeString, true),
		"age":  ScalarField(TypeInteger, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{
		"name":    123,         // wrong type
		"age":     "not-a-num", // wrong type
		"unknown": "x",
		"extra":   "y",
	}
	_, errs := schema.Validate(input)
	// Unknown field errors for "unknown" and "extra", plus validation errors for name and age
	if len(errs) < 3 {
		t.Errorf("expected at least 3 errors (2 unknown + 1+ validation), got %d: %v", len(errs), errs)
	}
	unknownCount := 0
	for _, e := range errs {
		if contains(e.Msg, "unknown field") {
			unknownCount++
		}
	}
	if unknownCount != 2 {
		t.Errorf("expected 2 unknown field errors, got %d", unknownCount)
	}
}

func TestValidationErrorsSortedAndToAPIResponse(t *testing.T) {
	errs := ValidationErrors{
		{Path: "b", Msg: "err b"},
		{Path: "a", Msg: "err a"},
		{Path: "c", Msg: "err c"},
	}
	sorted := errs.Sorted()
	if len(sorted) != 3 {
		t.Fatalf("expected 3 errors, got %d", len(sorted))
	}
	if sorted[0].Path != "a" || sorted[1].Path != "b" || sorted[2].Path != "c" {
		t.Errorf("expected sorted by path a,b,c, got %q %q %q", sorted[0].Path, sorted[1].Path, sorted[2].Path)
	}

	api := errs.ToAPIResponse()
	if len(api.Errors) != 3 {
		t.Errorf("expected 3 API errors, got %d", len(api.Errors))
	}
	if api.Errors[0].Path != "a" && api.Errors[0].Message != "err a" {
		t.Errorf("expected first API error path=a message=err a, got path=%q message=%q", api.Errors[0].Path, api.Errors[0].Message)
	}
}

func TestDecodeValidated(t *testing.T) {
	type Req struct {
		Name string `json:"name"`
		Age  int64  `json:"age"`
	}
	validated := map[string]any{
		"name": "Jane",
		"age":  float64(30), // JSON unmarshal uses float64 for numbers
	}
	var req Req
	err := DecodeValidated(validated, &req)
	if err != nil {
		t.Fatalf("DecodeValidated: %v", err)
	}
	if req.Name != "Jane" {
		t.Errorf("expected Name=Jane, got %q", req.Name)
	}
	if req.Age != 30 {
		t.Errorf("expected Age=30, got %d", req.Age)
	}

	// nil out should error
	err = DecodeValidated(validated, nil)
	if err == nil {
		t.Error("expected error for nil out")
	}
	err = DecodeValidated(nil, &req)
	if err == nil {
		t.Error("expected error for nil validated")
	}
}

// TestDecodeValidatedWithRequestStructs ensures DecodeValidated works with a struct matching request shape (e.g. TotalsRequest).
func TestDecodeValidatedWithRequestStructs(t *testing.T) {
	type totalsReq struct {
		Filter  map[string]any `json:"filter"`
		Metrics []string       `json:"metrics"`
	}
	validated := map[string]any{
		"filter":  map[string]any{"state": []any{"CA"}},
		"metrics": []any{"total_events", "other"},
	}
	var req totalsReq
	err := DecodeValidated(validated, &req)
	if err != nil {
		t.Fatalf("DecodeValidated: %v", err)
	}
	if req.Metrics == nil || len(req.Metrics) != 2 || req.Metrics[0] != "total_events" {
		t.Errorf("expected Metrics [total_events, other], got %v", req.Metrics)
	}
	if req.Filter == nil {
		t.Error("expected Filter to be set")
	}
}

// TestToAPIResponseJSON ensures ToAPIResponse marshals to the expected JSON shape.
func TestToAPIResponseJSON(t *testing.T) {
	errs := ValidationErrors{
		{Path: "filter.a", Msg: "cannot be empty"},
		{Path: "limit", Msg: "must be <= 100"},
	}
	api := errs.ToAPIResponse()

	// Round-trip through JSON
	bytes, err := json.Marshal(api)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded struct {
		Errors []struct {
			Path    string `json:"path"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(bytes, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(decoded.Errors) != 2 {
		t.Fatalf("expected 2 errors in JSON, got %d", len(decoded.Errors))
	}
	// Sorted by path: filter.a, then limit
	if decoded.Errors[0].Path != "filter.a" || decoded.Errors[0].Message != "cannot be empty" {
		t.Errorf("first error: got path=%q message=%q", decoded.Errors[0].Path, decoded.Errors[0].Message)
	}
	if decoded.Errors[1].Path != "limit" || decoded.Errors[1].Message != "must be <= 100" {
		t.Errorf("second error: got path=%q message=%q", decoded.Errors[1].Path, decoded.Errors[1].Message)
	}
}

// TestValidationErrorsGolden runs a fixed invalid request and asserts expected error paths and message substrings.
// Change this test when you intentionally change error messages or paths.
func TestValidationErrorsGolden(t *testing.T) {
	schema, err := CompileConfig(map[string]*Field{
		"name":   ScalarField(TypeString, true),
		"limit":  DefaultMinMaxInteger(10, 1, 100),
		"tags":   ScalarArray(TypeString),
		"nested": ScalarField(TypeInteger, false),
	})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Known bad input: missing required, invalid limit, empty tags, wrong type, unknown root field
	input := map[string]any{
		"limit":  999,            // over max
		"tags":   []any{},        // empty array not allowed
		"nested": "not-a-number", // wrong type
		"extra":  "unknown field",
	}
	_, errs := schema.Validate(input)

	// Golden expectations: at least these paths and message substrings must appear
	wantSubstrings := []struct{ path, substr string }{
		{"name", "required"},
		{"limit", "must be <="},
		{"tags", "cannot be empty"},
		{"nested", "invalid integer"},
		{"extra", "unknown field"},
	}
	for _, w := range wantSubstrings {
		found := false
		for _, e := range errs {
			if e.Path == w.path && contains(e.Msg, w.substr) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("golden: expected an error at path %q containing %q, got errors: %v", w.path, w.substr, errs)
		}
	}
}

// TestMaxDepthEnforced ensures validation fails when nesting exceeds MaxDepth (default 20).
func TestMaxDepthEnforced(t *testing.T) {
	// Build a schema with 21 levels of array nesting so the leaf is at depth 21 (> MaxDepth 20)
	itemCfg := ScalarField(TypeString, false)
	for i := 0; i < 21; i++ {
		itemCfg = &Field{
			OneOf: []OneOf{Array},
			Items: itemCfg,
		}
	}
	schema, err := CompileConfig(map[string]*Field{"deep": itemCfg})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	// Build input: 21 levels of nested arrays with a string at the innermost level
	inner := any("x")
	for i := 0; i < 21; i++ {
		inner = []any{inner}
	}
	input := map[string]any{"deep": inner}

	_, errs := schema.Validate(input)
	if len(errs) == 0 {
		t.Fatal("expected validation error for depth exceeding MaxDepth")
	}
	found := false
	for _, e := range errs {
		if contains(e.Msg, "maximum nesting depth exceeded") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'maximum nesting depth exceeded' in errors, got: %v", errs)
	}
}

// ── Edge Cases ─────────────────────────────────────────────────────────────────

func TestNilValues(t *testing.T) {
	config := map[string]*Field{
		"name": ScalarField(TypeString, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{"name": nil}
	_, errs := schema.Validate(input)
	// nil produces an error for unsupported type
	assertHasErrors(t, errs, 1)
	assertErrorContains(t, errs, "name", "unsupported value type")
}

func TestDateTypesEdgeCases(t *testing.T) {
	t.Run("dayOrMonthDate - full date", func(t *testing.T) {
		config := map[string]*Field{
			"date": ScalarField(TypeDayOrMonthDate, false),
		}
		schema, err := CompileConfig(config)
		if err != nil {
			t.Fatalf("failed to compile: %v", err)
		}

		input := map[string]any{"date": "2024-01-15"}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		_, ok := result["date"].(string)
		if !ok {
			t.Errorf("expected string, got %T", result["date"])
		}
	})

	t.Run("dayOrMonthDate - month only", func(t *testing.T) {
		config := map[string]*Field{
			"date": ScalarField(TypeDayOrMonthDate, false),
		}
		schema, err := CompileConfig(config)
		if err != nil {
			t.Fatalf("failed to compile: %v", err)
		}

		input := map[string]any{"date": "2024-01"}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		date, ok := result["date"].(string)
		if !ok {
			t.Fatalf("expected string, got %T", result["date"])
		}
		if date != "2024-01" {
			t.Errorf("expected 2024-01, got %q", date)
		}
	})

	t.Run("monthOrYearDate - month", func(t *testing.T) {
		config := map[string]*Field{
			"date": ScalarField(TypeMonthOrYearDate, false),
		}
		schema, err := CompileConfig(config)
		if err != nil {
			t.Fatalf("failed to compile: %v", err)
		}

		input := map[string]any{"date": "2024-01"}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		_, ok := result["date"].(string)
		if !ok {
			t.Errorf("expected string, got %T", result["date"])
		}
	})

	t.Run("monthOrYearDate - year", func(t *testing.T) {
		config := map[string]*Field{
			"date": ScalarField(TypeMonthOrYearDate, false),
		}
		schema, err := CompileConfig(config)
		if err != nil {
			t.Fatalf("failed to compile: %v", err)
		}

		input := map[string]any{"date": "2024"}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		date, ok := result["date"].(string)
		if !ok {
			t.Fatalf("expected string, got %T", result["date"])
		}
		if date != "2024" {
			t.Errorf("expected 2024, got %q", date)
		}
	})

	t.Run("yearDate invalid length", func(t *testing.T) {
		config := map[string]*Field{
			"year": ScalarField(TypeYearDate, false),
		}
		schema, err := CompileConfig(config)
		if err != nil {
			t.Fatalf("failed to compile: %v", err)
		}

		input := map[string]any{"year": "202"}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "year", "invalid format")
	})
}

func TestIntegerFromStringEdgeCases(t *testing.T) {
	config := map[string]*Field{
		"num": ScalarField(TypeInteger, false),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("zero", func(t *testing.T) {
		input := map[string]any{"num": "0"}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["num"] != int64(0) {
			t.Errorf("expected 0, got %v", result["num"])
		}
	})

	t.Run("negative", func(t *testing.T) {
		input := map[string]any{"num": "-42"}
		result, errs := schema.Validate(input)
		assertNoErrors(t, errs)
		if result["num"] != int64(-42) {
			t.Errorf("expected -42, got %v", result["num"])
		}
	})
}

func TestArrayWithInvalidItemTypes(t *testing.T) {
	config := map[string]*Field{
		"numbers": ScalarArray(TypeInteger),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("string in integer array", func(t *testing.T) {
		input := map[string]any{"numbers": []any{1, 2, "three"}}
		_, errs := schema.Validate(input)
		assertHasErrors(t, errs, 1)
		assertErrorContains(t, errs, "numbers[2]", "invalid integer")
	})
}

func TestObjectNestedUnknownField(t *testing.T) {
	config := map[string]*Field{
		"outer": {
			OneOf: []OneOf{Object},
			Properties: map[string]*Field{
				"inner": {
					OneOf: []OneOf{Object},
					Properties: map[string]*Field{
						"value": ScalarField(TypeString, false),
					},
				},
			},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{
		"outer": map[string]any{
			"inner": map[string]any{
				"value": "test",
				"bad":   "field",
			},
		},
	}
	_, errs := schema.Validate(input)
	assertHasErrors(t, errs, 1)
	assertErrorContains(t, errs, "outer.inner.bad", "unknown field")
}

func TestRangeValidationEdgeCases(t *testing.T) {
	config := map[string]*Field{
		"range": RangeOrArray(TypeInteger, "gte", "lte"),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	t.Run("range with only lower bound", func(t *testing.T) {
		input := map[string]any{
			"range": map[string]any{
				"gte": 10,
			},
		}
		_, errs := schema.Validate(input)
		// Should not error - range order validation only checks when both are present
		assertNoErrors(t, errs)
	})

	t.Run("range with only upper bound", func(t *testing.T) {
		input := map[string]any{
			"range": map[string]any{
				"lte": 20,
			},
		}
		_, errs := schema.Validate(input)
		assertNoErrors(t, errs)
	})

	t.Run("equal bounds", func(t *testing.T) {
		input := map[string]any{
			"range": map[string]any{
				"gte": 10,
				"lte": 10,
			},
		}
		_, errs := schema.Validate(input)
		assertNoErrors(t, errs)
	})
}

func TestRequiredInNestedObject(t *testing.T) {
	config := map[string]*Field{
		"user": {
			OneOf:      []OneOf{Object},
			AllowEmpty: true, // Allow empty to avoid empty object error
			Properties: map[string]*Field{
				"name": ScalarField(TypeString, true),
			},
		},
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{
		"user": map[string]any{},
	}
	_, errs := schema.Validate(input)
	assertHasErrors(t, errs, 1)
	assertErrorContains(t, errs, "user.name", "field is required")
}

func TestMultipleErrors(t *testing.T) {
	config := map[string]*Field{
		"name":  ScalarField(TypeString, true),
		"age":   ScalarField(TypeInteger, true),
		"email": ScalarField(TypeString, true),
	}
	schema, err := CompileConfig(config)
	if err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := map[string]any{
		"name": 123,            // wrong type
		"age":  "not-a-number", // wrong type
		// email missing
	}
	_, errs := schema.Validate(input)
	if len(errs) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(errs), errs)
	}
}

// ── Helper Function Tests ─────────────────────────────────────────────────────

func TestScalarFieldHelper(t *testing.T) {
	cfg := ScalarField(TypeString, true)
	if cfg.ScalarType != TypeString {
		t.Errorf("expected TypeString, got %v", cfg.ScalarType)
	}
	if !cfg.Required {
		t.Error("expected Required=true")
	}
	if len(cfg.OneOf) != 1 || cfg.OneOf[0] != Scalar {
		t.Errorf("expected [Scalar], got %v", cfg.OneOf)
	}
}

func TestScalarArrayHelper(t *testing.T) {
	cfg := ScalarArray(TypeInteger)
	if cfg.Items == nil {
		t.Fatal("expected Items to be set")
	}
	if cfg.Items.ScalarType != TypeInteger {
		t.Errorf("expected Items.ScalarType=TypeInteger, got %v", cfg.Items.ScalarType)
	}
	if len(cfg.OneOf) != 1 || cfg.OneOf[0] != Array {
		t.Errorf("expected [Array], got %v", cfg.OneOf)
	}
}

func TestRangeOrArrayHelper(t *testing.T) {
	cfg := RangeOrArray(TypeInteger, "gte", "lte")
	if len(cfg.OneOf) != 2 {
		t.Errorf("expected 2 OneOf values, got %d", len(cfg.OneOf))
	}
	if cfg.Items == nil {
		t.Fatal("expected Items to be set")
	}
	if len(cfg.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(cfg.Properties))
	}
	if cfg.Properties["gte"] == nil || cfg.Properties["lte"] == nil {
		t.Error("expected gte and lte properties")
	}
}

// ── Compilation Error Tests ────────────────────────────────────────────────────

func TestCompilationErrors(t *testing.T) {
	t.Run("nil field config", func(t *testing.T) {
		config := map[string]*Field{
			"field": nil,
		}
		_, err := CompileConfig(config)
		if err == nil {
			t.Error("expected error for nil field config")
		}
	})

	t.Run("scalar without ScalarType", func(t *testing.T) {
		config := map[string]*Field{
			"field": {
				OneOf: []OneOf{Scalar},
			},
		}
		_, err := CompileConfig(config)
		if err == nil {
			t.Error("expected error for scalar without ScalarType")
		}
	})

	t.Run("array without Items", func(t *testing.T) {
		config := map[string]*Field{
			"field": {
				OneOf: []OneOf{Array},
			},
		}
		_, err := CompileConfig(config)
		if err == nil {
			t.Error("expected error for array without Items")
		}
	})

	t.Run("object without Properties", func(t *testing.T) {
		config := map[string]*Field{
			"field": {
				OneOf: []OneOf{Object},
			},
		}
		_, err := CompileConfig(config)
		if err == nil {
			t.Error("expected error for object without Properties")
		}
	})

	t.Run("invalid OneOf value", func(t *testing.T) {
		config := map[string]*Field{
			"field": {
				OneOf: []OneOf{"invalid"},
			},
		}
		_, err := CompileConfig(config)
		if err == nil {
			t.Error("expected error for invalid OneOf value")
		}
	})
}

// FuzzValidate ensures Validate does not panic on arbitrary JSON input.
func FuzzValidate(f *testing.F) {
	schema, err := CompileConfig(map[string]*Field{
		"filter": {
			OneOf:      []OneOf{Object},
			AllowEmpty: true,
			Properties: map[string]*Field{
				"gte": ScalarField(TypeString, false),
				"lte": ScalarField(TypeString, false),
			},
		},
		"name":  ScalarField(TypeString, false),
		"count": ScalarField(TypeInteger, false),
		"tags":  ScalarArray(TypeString),
	})
	if err != nil {
		f.Fatalf("compile seed schema: %v", err)
	}

	f.Add([]byte(`{}`))
	f.Add([]byte(`{"name":"x"}`))
	f.Add([]byte(`{"filter":{},"tags":["a","b"]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			return // skip invalid JSON
		}
		_, _ = schema.Validate(raw) // must not panic
	})
}
