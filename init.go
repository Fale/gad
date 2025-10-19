package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Patterns
var (
	logFilenamePattern = regexp.MustCompile(`([0-9]{4})-([0-9]{2})-([0-9]{2})T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}-.+\.log`)
	logDatePattern     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// Errors
var errNoFilesInBucket = errors.New("no files in the bucket")

const (
	appName       = "gad"
	configFileExt = "yaml"
)

func initConfig() error {
	// Default values
	viper.SetDefault("batch-size", 500)
	viper.SetDefault("day-until", time.Now().UTC().Add(-2*time.Hour).Format("2006-01-02"))
	viper.SetDefault("profile", "default")

	// ---- Resolve XDG config dir (with fallback to ~/.config) ----
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home dir: %w", err)
	}
	userConfigDir := os.Getenv("XDG_CONFIG_HOME")
	if userConfigDir == "" {
		userConfigDir = filepath.Join(home, ".config")
	}

	// ---- Viper config search order (lowest to highest before env/flags) ----
	viper.SetConfigName(appName)
	viper.SetConfigType(configFileExt)
	viper.AddConfigPath(filepath.Join("/etc", appName)) // /etc/gad
	viper.AddConfigPath(userConfigDir)                  // $XDG_CONFIG_HOME

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("read config: %w", err)
		}
	}

	// ---- Environment variables (override config) ----
	// Env to set in shell: GAD_BATCH_SIZE, GAD_BUCKET, GAD_DAY_UNTIL, GAD_LOGS_FOLDER, GAD_PROFILE
	viper.SetEnvPrefix(strings.ToUpper(appName))
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// ---- CLI flags (highest precedence) ----
	pflag.Int("batch-size", viper.GetInt("batch-size"), "Number of items to process per batch")
	pflag.String("bucket", viper.GetString("bucket"), "AWS bucket where logs are stored")
	pflag.String("day-until", viper.GetString("day-until"), "Process data up to this UTC day (excluded), format YYYY-MM-DD")
	pflag.String("logs-folder", viper.GetString("logs-folder"), "Folder to store processed logs")
	pflag.String("profile", viper.GetString("profile"), "AWS profile to be used")

	// Custom help with config paths & env info
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, `%s – configuration and flags

Config search order (lowest to highest before env/flags):
      1) Defaults (built-in)
      2) %s
      3) %s

Environment variables (override config):
      GAD_BATCH_SIZE    → batch-size
      GAD_BUCKET        → bucket
      GAD_DAY_UNTIL     → day-until
      GAD_LOGS_FOLDER   → logs-folder
      GAD_PROFILE       → profile

Flags (highest precedence):
`, appName, filepath.Join("/etc", appName, appName+"."+configFileExt), filepath.Join(userConfigDir, appName+"."+configFileExt))

		pflag.PrintDefaults()
		fmt.Fprintln(os.Stderr)

		if cf := viper.ConfigFileUsed(); cf != "" {
			fmt.Fprintf(os.Stderr, "Active config file: %s\n", cf)
		} else {
			fmt.Fprintf(os.Stderr, "Active config file: (none found)\n")
		}
		fmt.Fprintf(os.Stderr, "Precedence: defaults < config file < environment < flags\n")
	}

	// Apply CLI overrides only if provided
	pflag.Parse()
	for _, key := range []string{"profile", "bucket", "logs-folder"} {
		if f := pflag.Lookup(key); f != nil && f.Changed {
			viper.Set(key, f.Value.String())
		}
	}
	// String flag for date (YYYY-MM-DD)
	if f := pflag.Lookup("day-until"); f != nil && f.Changed {
		viper.Set("day-until", f.Value.String())
	}
	// Int flag for batch size
	if f := pflag.Lookup("batch-size"); f != nil && f.Changed {
		i, _ := strconv.Atoi(f.Value.String())
		viper.Set("batch-size", i)
	}

	// ---- Validation ----
	if viper.GetInt("batch-size") <= 0 {
		return fmt.Errorf("invalid batch-size: must be > 0")
	}
	if viper.GetString("bucket") == "" {
		return fmt.Errorf("missing required configuration: bucket")
	}
	if viper.GetString("logs-folder") == "" {
		return fmt.Errorf("missing required configuration: logs-folder")
	}
	dayUntil, err := time.Parse("2006-01-02", viper.GetString("day-until"))
	if err != nil {
		return fmt.Errorf("invalid day-until (expected YYYY-MM-DD): %w", err)
	}
	if dayUntil.After(time.Now().UTC()) {
		return fmt.Errorf("day-until needs to be in the past or now")
	}

	return nil
}
