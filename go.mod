module github.com/hapkiduki/order-go

go 1.25.1

require (
	// Router - Lightweight, idiomatic and composable router
	github.com/go-chi/chi/v5 v5.2.3

	// Chi middleware
	github.com/go-chi/cors v1.2.2
	github.com/go-chi/render v1.0.3

	// UUID generation
	github.com/google/uuid v1.6.0

	// Configuration management (12-Factor: III. Config)
	github.com/spf13/viper v1.21.0

	// Testing
	github.com/stretchr/testify v1.11.1

	// Logging (12-Factor: XI. Logs)
	go.uber.org/zap v1.27.1

	// Rate limiting
	golang.org/x/time v0.14.0
)

require (
	github.com/ajg/form v1.5.1 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.28.0 // indirect
)
