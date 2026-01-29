// Package validation: sample client config (reference only).
// Uncomment and adapt for your API.
package validation

// import (
// 	"fmt"
// 	"maps"

// 	"github.com/gin-gonic/gin"
// 	apimodel "gitlab.economicmodeling.com/ltc/microservices/warn/app/lib/model"
// 	v "gitlab.economicmodeling.com/rudy.selman/go-validator"
// )

// var (
// 	// CompiledSchemas holds all pre-compiled request validation schemas
// 	CompiledSchemas = struct {
// 		Totals            *v.CompiledSchema
// 		Ranking           *v.CompiledSchema
// 		NestedRanking     *v.CompiledSchema
// 		Timeseries        *v.CompiledSchema
// 		RankingTimeseries *v.CompiledSchema
// 		Records           *v.CompiledSchema
// 	}{}

// 	filtersConfig = map[string]*v.FieldConfig{
// 		"number_employees_affected": v.GteZeroRangeOrArray(v.TypeInteger, "gte", "lte"),

// 		"notice_date": v.RangeOrArray(v.TypeDayOrMonthDate, "gte", "lte"),
// 		"layoff_date": v.RangeOrArray(v.TypeDayOrMonthDate, "gte", "lte"),
// 		// "year":        v.RangeOrArray(v.TypeYearDate, "gte", "lte"),

// 		"city":                 v.ScalarArray(v.TypeString),
// 		"city_name":            v.ScalarArray(v.TypeString),
// 		"state":                v.ScalarArray(v.TypeString),
// 		"zip":                  v.ScalarArray(v.TypeString),
// 		"county":               v.ScalarArray(v.TypeString),
// 		"company":              v.ScalarArray(v.TypeString),
// 		"company_name":         v.ScalarArray(v.TypeString),
// 		"layoff_type":          v.ScalarArray(v.TypeString),
// 		"notice_type":          v.ScalarArray(v.TypeString),
// 		"occupations_impacted": v.ScalarArray(v.TypeString),
// 		// "reason_for_layoff":    v.ScalarArray(v.TypeString), no values
// 		"trade_union": v.ScalarArray(v.TypeString),
// 		"naics2":      v.ScalarArray(v.TypeString),
// 		"naics3":      v.ScalarArray(v.TypeString),
// 		"naics4":      v.ScalarArray(v.TypeString),
// 		"naics5":      v.ScalarArray(v.TypeString),
// 		"naics6":      v.ScalarArray(v.TypeString),
// 		"naics2_name": v.ScalarArray(v.TypeString),
// 		"naics3_name": v.ScalarArray(v.TypeString),
// 		"naics4_name": v.ScalarArray(v.TypeString),
// 		"naics5_name": v.ScalarArray(v.TypeString),
// 		"naics6_name": v.ScalarArray(v.TypeString),
// 	}

// 	// ── Reusable FieldConfig objects

// 	FilterConfig = v.FieldConfig{
// 		OneOf:      []v.OneOf{v.Object},
// 		Required:   true,
// 		AllowEmpty: true,
// 		Properties: filtersConfig,
// 	}

// 	MetricsConfig = v.FieldConfig{
// 		OneOf: []v.OneOf{v.Array},
// 		Items: &v.FieldConfig{
// 			OneOf:         []v.OneOf{v.Scalar},
// 			ScalarType:    v.TypeString,
// 			StrEnumValues: GetAllMetrics(),
// 		},
// 		Default: []string{"total_events"},
// 	}

// 	RankingConfig = v.FieldConfig{
// 		OneOf:      []v.OneOf{v.Object},
// 		Required:   true,
// 		AllowEmpty: true,
// 		Properties: map[string]*v.FieldConfig{
// 			"limit": v.DefaultMinMaxInteger(10, 1, 10000),
// 			"by": {
// 				OneOf:         []v.OneOf{v.Scalar},
// 				ScalarType:    v.TypeString,
// 				StrEnumValues: GetAllMetrics(),
// 				Default:       "total_events",
// 			},
// 			"extra_metrics": &MetricsConfig,
// 		},
// 	}

// 	// ── Endpoint Requests

// 	TotalsRequestConfig = map[string]*v.FieldConfig{
// 		"filter":  &FilterConfig,
// 		"metrics": &MetricsConfig,
// 	}

// 	RankingRequestConfig = map[string]*v.FieldConfig{
// 		"filter": &FilterConfig,
// 		"rank":   &RankingConfig,
// 	}

// 	NestedRankingRequestConfig = map[string]*v.FieldConfig{
// 		"filter":      &FilterConfig,
// 		"rank":        &RankingConfig,
// 		"nested_rank": &RankingConfig,
// 	}

// 	// DefaultRecordsFields is used when "fields" is missing or empty on records requests.
// 	DefaultRecordsFields = []string{"id", "layoff_date", "state", "company", "description", "number_employees_affected"}

// 	RecordsRequestConfig = map[string]*v.FieldConfig{
// 		"filter": &FilterConfig,
// 		"fields": {
// 			OneOf:      []v.OneOf{v.Array},
// 			Items: &v.FieldConfig{
// 				OneOf:         []v.OneOf{v.Scalar},
// 				ScalarType:    v.TypeString,
// 				StrEnumValues: GetAllRecordFields(),
// 			},
// 			Default: DefaultRecordsFields,
// 		},
// 		"order": {
// 			OneOf:         []v.OneOf{v.Scalar},
// 			ScalarType:    v.TypeString,
// 			StrEnumValues: []string{"score", "layoff_date"},
// 			Default:       "score",
// 		},
// 		"limit": v.DefaultMinMaxInteger(10, 1, 100),
// 	}
// )

// // Custom requests schemas required by specific endpoints, and compilation
// func init() {
// 	// RecordsRequestConfig["fields"].Items.StrEnumValues is set at compile time when GetAllRecordFields()
// 	// was still nil (meta.init() runs after var init). Patch it now so enum validation works.
// 	if fc := RecordsRequestConfig["fields"]; fc != nil && fc.Items != nil {
// 		fc.Items.StrEnumValues = GetAllRecordFields()
// 	}

// 	// clone base properties for timeseries; require "at least one of" layoff_date or notice_date
// 	timeseriesFilter := maps.Clone(FilterConfig.Properties)

// 	TimeseriesRequestConfig := map[string]*v.FieldConfig{
// 		"filter": {
// 			OneOf:      []v.OneOf{v.Object},
// 			Required:   true,
// 			AllowEmpty: false,
// 			Properties: timeseriesFilter,
// 			AdditionalValidators: []v.ValidatorFunc{
// 				v.RequireAtLeastOneRange([]string{"layoff_date", "notice_date"}, v.RangeKeys{Lower: "gte", Upper: "lte"}),
// 			},
// 		},
// 		"metrics": &MetricsConfig,
// 		"interval": {
// 			OneOf:         []v.OneOf{v.Scalar},
// 			ScalarType:    v.TypeString,
// 			StrEnumValues: []string{"year", "month"},
// 			Default:       "month",
// 		},
// 	}

// 	RankingTimeseriesRequestConfig := map[string]*v.FieldConfig{
// 		"filter":  TimeseriesRequestConfig["filter"],
// 		"metrics": &MetricsConfig,
// 		"rank":    &RankingConfig,
// 		"timeseries": {
// 			OneOf: []v.OneOf{v.Object},
// 			Properties: map[string]*v.FieldConfig{
// 				"interval": TimeseriesRequestConfig["interval"],
// 				"metrics":  &MetricsConfig,
// 			},
// 			Required: true,
// 		},
// 	}

// 	// precompile schemas
// 	CompiledSchemas.Totals = mustCompile("totals", TotalsRequestConfig)
// 	CompiledSchemas.Ranking = mustCompile("ranking", RankingRequestConfig)
// 	CompiledSchemas.NestedRanking = mustCompile("nested-ranking", NestedRankingRequestConfig)
// 	CompiledSchemas.Timeseries = mustCompile("timeseries", TimeseriesRequestConfig)
// 	CompiledSchemas.RankingTimeseries = mustCompile("ranking-timeseries", RankingTimeseriesRequestConfig)
// 	CompiledSchemas.Records = mustCompile("records", RecordsRequestConfig)
// }

// // panics on invalid config
// func mustCompile(name string, config map[string]*v.FieldConfig) *v.CompiledSchema {
// 	schema, err := v.CompileConfig(config)
// 	if err != nil {
// 		panic(fmt.Sprintf("invalid config for %q: %v", name, err))
// 	}
// 	return schema
// }

// // this would live in an API client for error response handling
// func ValidateRequest(schema *v.CompiledSchema) gin.HandlerFunc {
// 	return func(gctx *gin.Context) {
// 		var raw map[string]any
// 		if err := gctx.ShouldBindJSON(&raw); err != nil {
// 			gctx.Error(apimodel.NewInvalidRequestError(err))
// 			gctx.Abort()
// 			return
// 		}

// 		validated, errs := schema.Validate(raw)
// 		if len(errs) > 0 {
// 			gctx.Error(errs)
// 			gctx.Abort()
// 			return
// 		}

// 		gctx.Set("validatedBody", validated)
// 		gctx.Next()
// 	}
// }
