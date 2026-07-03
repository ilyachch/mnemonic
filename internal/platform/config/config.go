package config

// Config represents the schema v1 configuration structure.
type Config struct {
	Version int           `toml:"version" json:"version"`
	Paths   PathsConfig   `toml:"paths" json:"paths"`
	Notes   NotesConfig   `toml:"notes" json:"notes"`
	Index   IndexConfig   `toml:"index" json:"index"`
	Output  OutputConfig  `toml:"output" json:"output"`
	Logging LoggingConfig `toml:"logging" json:"logging"`
}

// PathsConfig holds configuration for paths.
type PathsConfig struct {
	MemoriesHome string `toml:"memories_home" json:"memories_home"`
}

// NotesConfig holds configuration for notes behavior.
type NotesConfig struct {
	DeleteBehavior string `toml:"delete_behavior" json:"delete_behavior"`
	TrashDirName   string `toml:"trash_dir_name" json:"trash_dir_name"`
}

// IndexConfig holds index and SQLite configuration.
type IndexConfig struct {
	FTS           bool `toml:"fts" json:"fts"`
	WAL           bool `toml:"wal" json:"wal"`
	BusyTimeoutMs int  `toml:"busy_timeout_ms" json:"busy_timeout_ms"`
}

// OutputConfig holds formatting settings for CLI output.
type OutputConfig struct {
	JSONPretty bool `toml:"json_pretty" json:"json_pretty"`
}

// LoggingConfig holds settings for log level and outputs.
type LoggingConfig struct {
	Level  string `toml:"level" json:"level"`
	Format string `toml:"format" json:"format"`
}

// DefaultConfig returns a Config populated with default values.
func DefaultConfig() *Config {
	return &Config{
		Version: 1,
		Paths: PathsConfig{
			MemoriesHome: "", // Will be resolved dynamically by paths package if empty.
		},
		Notes: NotesConfig{
			DeleteBehavior: "trash",
			TrashDirName:   ".trash",
		},
		Index: IndexConfig{
			FTS:           true,
			WAL:           true,
			BusyTimeoutMs: 5000,
		},
		Output: OutputConfig{
			JSONPretty: false,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
}
