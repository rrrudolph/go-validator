// Package validation: sample client config (reference only).
// Request structs (TotalsRequest, RecordsRequest, etc.) live in sample-request-structs.go.
// Uncomment and adapt for your API.
package validation

// import (
// 	"fmt"
// 	"maps"
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// 	v "github.com/rrrudolph/go-validator"
// )

// var (
// 	// CompiledSchemas holds all pre-compiled request validation schemas
// 	CompiledSchemas = struct {
// 		Totals            *v.CompiledSchema
// 		Ranking           *v.CompiledSchema
// 		NestedRanking     *v.CompiledSchema
// 		Timeseries        *v.CompiledSchema
// 		RankingTimeseries *v.CompiledSchema
// 	}{}

// 	filtersConfig = map[string]*v.FieldConfig{
// 		"submission_date": v.RangeOrArray(v.TypeDayOrMonthDate, "gte", "lte"),

// 		"company":      v.ScalarArray(v.TypeInteger),
// 		"company_name": v.ScalarArray(v.TypeString),
// 		"title_name":   v.ScalarArray(v.TypeString),
// 		"onet":         v.ScalarArray(v.TypeString),
// 		"onet_name":    v.ScalarArray(v.TypeString),
// 		"naics2":       v.ScalarArray(v.TypeInteger),
// 		"naics3":       v.ScalarArray(v.TypeInteger),
// 		"naics4":       v.ScalarArray(v.TypeInteger),
// 		"naics5":       v.ScalarArray(v.TypeInteger),
// 		"naics6":       v.ScalarArray(v.TypeInteger),
// 		"naics2_name":  v.ScalarArray(v.TypeString),
// 		"naics3_name":  v.ScalarArray(v.TypeString),
// 		"naics4_name":  v.ScalarArray(v.TypeString),
// 		"naics5_name":  v.ScalarArray(v.TypeString),
// 		"naics6_name":  v.ScalarArray(v.TypeString),
// 		"soc2":         v.ScalarArray(v.TypeString),
// 		"soc3":         v.ScalarArray(v.TypeString),
// 		"soc4":         v.ScalarArray(v.TypeString),
// 		"soc5":         v.ScalarArray(v.TypeString),
// 		"soc2_name":    v.ScalarArray(v.TypeString),
// 		"soc3_name":    v.ScalarArray(v.TypeString),
// 		"soc4_name":    v.ScalarArray(v.TypeString),
// 		"soc5_name":    v.ScalarArray(v.TypeString),

// 		"lot_career_area":                 v.ScalarArray(v.TypeInteger),
// 		"lot_career_area_name":            v.ScalarArray(v.TypeString),
// 		"lot_occupation":                  v.ScalarArray(v.TypeInteger),
// 		"lot_occupation_name":             v.ScalarArray(v.TypeString),
// 		"lot_occupation_group":            v.ScalarArray(v.TypeInteger),
// 		"lot_occupation_group_name":       v.ScalarArray(v.TypeString),
// 		"lot_specialized_occupation":      v.ScalarArray(v.TypeInteger),
// 		"lot_specialized_occupation_name": v.ScalarArray(v.TypeString),

// 		"laa_admin_area_1":      v.ScalarArray(v.TypeString),
// 		"laa_admin_area_1_name": v.ScalarArray(v.TypeString),
// 		"laa_admin_area_2":      v.ScalarArray(v.TypeString),
// 		"laa_admin_area_2_name": v.ScalarArray(v.TypeString),
// 		"laa_country":           v.ScalarArray(v.TypeString),
// 		"laa_country_name":      v.ScalarArray(v.TypeString),

// 		"address":     v.ScalarArray(v.TypeString),
// 		"city":        v.ScalarArray(v.TypeString),
// 		"city_name":   v.ScalarArray(v.TypeString),
// 		"state":       v.ScalarArray(v.TypeInteger),
// 		"state_name":  v.ScalarArray(v.TypeString),
// 		"zip":         v.ScalarArray(v.TypeString),
// 		"county":      v.ScalarArray(v.TypeString),
// 		"county_name": v.ScalarArray(v.TypeString),
// 		"msa":         v.ScalarArray(v.TypeString),
// 		"msa_name":    v.ScalarArray(v.TypeString),

// 		"wage_level":      v.ScalarArray(v.TypeString),
// 		"case_status":     v.ScalarArray(v.TypeString),
// 		"employment_type": v.ScalarArray(v.TypeString),
// 		"is_certified":    v.ScalarField(v.TypeBoolean, false),
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
// 		Default: []any{"total_applications"}, // easier to handle this if we can expect consistent type
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
// 				StrEnumValues: GetAllRankByMetrics(),
// 				Default:       "total_applications",
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
// )

// // Custom requests schemas required by specific endpoints
// func init() {
// 	// clone base properties for timeseries, leave main FilterConfig intact
// 	timeseriesFilter := maps.Clone(FilterConfig.Properties)

// 	// submission_date is required and must be a valid date range
// 	submissionDateConfig := &v.FieldConfig{
// 		OneOf:      []v.OneOf{v.Object},
// 		AllowEmpty: false,
// 		Required:   true,
// 		Properties: map[string]*v.FieldConfig{
// 			"gte": v.ScalarField(v.TypeDayOrMonthDate, true),
// 			"lte": v.ScalarField(v.TypeDayOrMonthDate, true),
// 		},
// 		AdditionalValidators: []v.ValidatorFunc{
// 			v.ValidateRangeOrder(v.RangeKeys{Lower: "gte", Upper: "lte"}),
// 		},
// 	}

// 	timeseriesFilter["submission_date"] = submissionDateConfig

// 	TimeseriesRequestConfig := map[string]*v.FieldConfig{
// 		"filter": {
// 			OneOf:      []v.OneOf{v.Object},
// 			Required:   true,
// 			AllowEmpty: false,
// 			Properties: timeseriesFilter,
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
// }

// // panics on invalid config
// func mustCompile(name string, config map[string]*v.FieldConfig) *v.CompiledSchema {
// 	schema, err := v.CompileConfig(config)
// 	if err != nil {
// 		panic(fmt.Sprintf("invalid config for %q: %v", name, err))
// 	}
// 	return schema
// }

// // ValidateRequest validates the body, decodes into T, and sets the result on the context.
// // Register with the request type: ValidateRequest[TotalsRequest](TotalsSchema)
// func ValidateRequest[T any](schema *v.CompiledSchema) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		var raw map[string]any
// 		if err := c.ShouldBindJSON(&raw); err != nil {
// 			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 			c.Abort()
// 			return
// 		}

// 		validated, errs := schema.Validate(raw)
// 		if len(errs) > 0 {
// 			c.JSON(http.StatusBadRequest, errs.ToStandardErrors())
// 			c.Abort()
// 			return
// 		}

// 		var req T
// 		if err := v.DecodeValidated(validated, &req); err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 			c.Abort()
// 			return
// 		}

// 		c.Set("validatedBody", &req)
// 		c.Next()
// 	}
// }
