package resources

import (
	_ "embed"
)

//go:embed swagger.json
var OpenAPISpec string

//go:embed real-time-badge-flow.md
var RealTimeBadgeFlow string
