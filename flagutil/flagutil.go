package flagutil

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lj020326/cli-kit/validator"
)

// HasFlag checks if a command-line flag is set in the given FlagSet
func HasFlag(fs *flag.FlagSet, name string) bool {
	found := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// HasFlagInArgs checks if a flag is present in args (supports -name, --name, -name=value, --name=value)
func HasFlagInArgs(args []string, name string) bool {
	if name == "" {
		return false
	}

	longForm := "--" + name
	shortForm := "-" + name
	longPrefix := longForm + "="
	shortPrefix := shortForm + "="

	for _, arg := range args {
		if arg == longForm || arg == shortForm || strings.HasPrefix(arg, longPrefix) || strings.HasPrefix(arg, shortPrefix) {
			return true
		}
	}

	return false
}

// HasFlagInOSArgs checks if a flag is present in os.Args
func HasFlagInOSArgs(name string) bool {
	return HasFlagInArgs(os.Args[1:], name)
}

// GetFlagValue returns the string value for a flag if it was set.
func GetFlagValue(fs *flag.FlagSet, name string) (string, bool) {
	if fs == nil || name == "" {
		return "", false
	}
	// Lookup first to check if flag exists, then verify it was set
	found := fs.Lookup(name)
	if found == nil {
		return "", false
	}
	if !HasFlag(fs, name) {
		return "", false
	}
	return found.Value.String(), true
}

// GetString returns flag value or defaultValue when not set.
func GetString(fs *flag.FlagSet, name, defaultValue string) string {
	value, ok := GetFlagValue(fs, name)
	if !ok {
		return defaultValue
	}
	return value
}

// GetInt returns flag value as int or defaultValue when not set/invalid.
func GetInt(fs *flag.FlagSet, name string, defaultValue int) int {
	value, ok := GetFlagValue(fs, name)
	if !ok {
		return defaultValue
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return defaultValue
}

// GetInt64 returns flag value as int64 or defaultValue when not set/invalid.
func GetInt64(fs *flag.FlagSet, name string, defaultValue int64) int64 {
	value, ok := GetFlagValue(fs, name)
	if !ok {
		return defaultValue
	}
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsed
	}
	return defaultValue
}

// GetUint returns flag value as uint or defaultValue when not set/invalid.
func GetUint(fs *flag.FlagSet, name string, defaultValue uint) uint {
	value, ok := GetFlagValue(fs, name)
	if !ok {
		return defaultValue
	}
	if parsed, err := strconv.ParseUint(value, 10, 0); err == nil {
		return uint(parsed)
	}
	return defaultValue
}

// GetUint64 returns flag value as uint64 or defaultValue when not set/invalid.
func GetUint64(fs *flag.FlagSet, name string, defaultValue uint64) uint64 {
	value, ok := GetFlagValue(fs, name)
	if !ok {
		return defaultValue
	}
	if parsed, err := strconv.ParseUint(value, 10, 64); err == nil {
		return parsed
	}
	return defaultValue
}

// GetFloat64 returns flag value as float64 or defaultValue when not set/invalid.
func GetFloat64(fs *flag.FlagSet, name string, defaultValue float64) float64 {
	value, ok := GetFlagValue(fs, name)
	if !ok {
		return defaultValue
	}
	if parsed, err := strconv.ParseFloat(value, 64); err == nil {
		return parsed
	}
	return defaultValue
}

// GetBool returns flag value as bool or defaultValue when not set/invalid.
func GetBool(fs *flag.FlagSet, name string, defaultValue bool) bool {
	value, ok := GetFlagValue(fs, name)
	if !ok {
		return defaultValue
	}
	if parsed, err := strconv.ParseBool(value); err == nil {
		return parsed
	}
	return defaultValue
}

// GetDuration returns flag value as time.Duration or defaultValue when not set/invalid.
func GetDuration(fs *flag.FlagSet, name string, defaultValue time.Duration) time.Duration {
	value, ok := GetFlagValue(fs, name)
	if !ok {
		return defaultValue
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed
	}
	return defaultValue
}

// ReadPasswordFromFile reads password from file (security improvement).
// Path is validated with path traversal check; relative paths are resolved to absolute.
// File content is trimmed of leading and trailing whitespace.
func ReadPasswordFromFile(filePath string) (string, error) {
	// Security: reject path traversal and resolve to absolute path
	safePath, err := validator.ValidatePath(filePath, &validator.PathOptions{CheckTraversal: true})
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(safePath)
	if err != nil {
		return "", err
	}

	password := strings.TrimSpace(string(data))
	return password, nil
}
