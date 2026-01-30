// Package validation: sample request structs (reference only).
// Use with DecodeValidated(validated, &req) after schema.Validate.
// Uncomment and adapt for your API.
package validation

// Filter is the shared filter object (many optional keys; keep as map for flexibility).
type Filter map[string]any

// TotalsRequest is the validated body for a totals endpoint.
type TotalsRequest struct {
	Filter  Filter   `json:"filter"`
	Metrics []string `json:"metrics"`
}

// RankRequest is the validated rank object (limit, by, extra_metrics).
type RankRequest struct {
	Limit        int64    `json:"limit"`
	By           string   `json:"by"`
	ExtraMetrics []string `json:"extra_metrics"`
}

// RankingRequest is the validated body for a ranking endpoint.
type RankingRequest struct {
	Filter Filter      `json:"filter"`
	Rank   RankRequest `json:"rank"`
}

// NestedRankingRequest is the validated body for a nested-ranking endpoint.
type NestedRankingRequest struct {
	Filter     Filter      `json:"filter"`
	Rank       RankRequest `json:"rank"`
	NestedRank RankRequest `json:"nested_rank"`
}

// RecordsRequest is the validated body for a records endpoint.
type RecordsRequest struct {
	Filter Filter   `json:"filter"`
	Fields []string `json:"fields"`
	Order  string   `json:"order"`
	Limit  int64    `json:"limit"`
}

// TimeseriesRequest is the validated body for a timeseries endpoint.
type TimeseriesRequest struct {
	Filter   Filter   `json:"filter"`
	Metrics  []string `json:"metrics"`
	Interval string   `json:"interval"`
}

// TimeseriesOptions is the nested timeseries object (interval, metrics).
type TimeseriesOptions struct {
	Interval string   `json:"interval"`
	Metrics  []string `json:"metrics"`
}

// RankingTimeseriesRequest is the validated body for a ranking-timeseries endpoint.
type RankingTimeseriesRequest struct {
	Filter     Filter            `json:"filter"`
	Metrics    []string          `json:"metrics"`
	Rank       RankRequest       `json:"rank"`
	Timeseries TimeseriesOptions `json:"timeseries"`
}
