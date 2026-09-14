package configutil

import (
	"flag"
	"strconv"
	"strings"
	"time"

	"github.com/lj020326/cli-kit/env"
	"github.com/lj020326/cli-kit/flagutil"
	"github.com/lj020326/cli-kit/validator"
)

// ResolveString resolves a configuration value with priority: CLI flag > environment variable > default value.
// Returns the resolved string value.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag (e.g., "port")
//   - envKey: Name of the environment variable (e.g., "PORT")
//   - defaultValue: Default value to use if neither CLI nor ENV is set
//   - trimmed: If true, trim whitespace from environment variable value
func ResolveString(fs *flag.FlagSet, flagName, envKey, defaultValue string, trimmed bool) string {
	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) {
		return flagutil.GetString(fs, flagName, defaultValue)
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		if trimmed {
			value := env.GetTrimmed(envKey, "")
			if value != "" {
				return value
			}
		} else {
			value := env.Get(envKey, "")
			if value != "" {
				return value
			}
		}
	}

	// Priority 3: Default value
	return defaultValue
}

// ResolveInt resolves an integer configuration value with priority: CLI flag > environment variable > default value.
// Returns the resolved integer value.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag (e.g., "port")
//   - envKey: Name of the environment variable (e.g., "PORT")
//   - defaultValue: Default value to use if neither CLI nor ENV is set
//   - allowZero: If false, zero values from ENV are treated as "not set" and default is used
func ResolveInt(fs *flag.FlagSet, flagName, envKey string, defaultValue int, allowZero bool) int {
	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) {
		value := flagutil.GetInt(fs, flagName, defaultValue)
		// CLI flag value is always used if flag is set, even if zero
		return value
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		value := env.GetInt(envKey, defaultValue)
		// If allowZero is false and value is 0, treat as "not set" and use default
		if !allowZero && value == 0 {
			return defaultValue
		}
		return value
	}

	// Priority 3: Default value
	return defaultValue
}

// ResolveInt64 resolves an int64 configuration value with priority: CLI flag > environment variable > default value.
// Returns the resolved int64 value.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag (e.g., "max-size")
//   - envKey: Name of the environment variable (e.g., "MAX_SIZE")
//   - defaultValue: Default value to use if neither CLI nor ENV is set
//   - allowZero: If false, zero values from ENV are treated as "not set" and default is used
func ResolveInt64(fs *flag.FlagSet, flagName, envKey string, defaultValue int64, allowZero bool) int64 {
	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) {
		value := flagutil.GetInt64(fs, flagName, defaultValue)
		// CLI flag value is always used if flag is set, even if zero
		return value
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		value := env.GetInt64(envKey, defaultValue)
		// If allowZero is false and value is 0, treat as "not set" and use default
		if !allowZero && value == 0 {
			return defaultValue
		}
		return value
	}

	// Priority 3: Default value
	return defaultValue
}

// ResolveInt64WithValidation resolves an int64 configuration with custom validation function.
// Priority: CLI flag > environment variable > default value.
// If validator returns an error, the value is rejected and default is used.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag
//   - envKey: Name of the environment variable
//   - defaultValue: Default value to use
//   - allowZero: If false, zero values from ENV are treated as "not set" and default is used
//   - validator: Function to validate the resolved value (returns error if invalid)
//
// Returns:
//   - int64: The resolved and validated value
//   - error: Returns error if validation fails for all sources
func ResolveInt64WithValidation(
	fs *flag.FlagSet,
	flagName, envKey string,
	defaultValue int64,
	allowZero bool,
	validator func(int64) error,
) (int64, error) {
	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) {
		value := flagutil.GetInt64(fs, flagName, defaultValue)
		if err := validator(value); err == nil {
			return value, nil
		}
		// Invalid CLI value, try ENV
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		value := env.GetInt64(envKey, defaultValue)
		if !allowZero && value == 0 {
			// Treat as not set, try default
		} else {
			if err := validator(value); err == nil {
				return value, nil
			}
		}
		// Invalid ENV value, try default
	}

	// Priority 3: Default value
	if err := validator(defaultValue); err == nil {
		return defaultValue, nil
	}

	// All sources failed validation
	return defaultValue, validator(defaultValue)
}

// ResolveBool resolves a boolean configuration value with priority: CLI flag > environment variable > default value.
// Returns the resolved boolean value.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag (e.g., "redis-enabled")
//   - envKey: Name of the environment variable (e.g., "REDIS_ENABLED")
//   - defaultValue: Default value to use if neither CLI nor ENV is set
func ResolveBool(fs *flag.FlagSet, flagName, envKey string, defaultValue bool) bool {
	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) {
		return flagutil.GetBool(fs, flagName, defaultValue)
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		return env.GetBool(envKey, defaultValue)
	}

	// Priority 3: Default value
	return defaultValue
}

// ResolveDuration resolves a duration configuration value with priority: CLI flag > environment variable > default value.
// Returns the resolved duration value.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag (e.g., "timeout")
//   - envKey: Name of the environment variable (e.g., "TIMEOUT")
//   - defaultValue: Default value to use if neither CLI nor ENV is set
func ResolveDuration(fs *flag.FlagSet, flagName, envKey string, defaultValue time.Duration) time.Duration {
	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) {
		return flagutil.GetDuration(fs, flagName, defaultValue)
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		return env.GetDuration(envKey, defaultValue)
	}

	// Priority 3: Default value
	return defaultValue
}

// ResolveIntAsString resolves an integer configuration and converts it to string.
// Useful for cases where the config struct expects a string but the value is an integer.
// Priority: CLI flag > environment variable > default value.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag (e.g., "port")
//   - envKey: Name of the environment variable (e.g., "PORT")
//   - defaultValue: Default integer value to use if neither CLI nor ENV is set
//   - allowZero: If false, zero values from ENV are treated as "not set" and default is used
func ResolveIntAsString(fs *flag.FlagSet, flagName, envKey string, defaultValue int, allowZero bool) string {
	intValue := ResolveInt(fs, flagName, envKey, defaultValue, allowZero)
	return strconv.Itoa(intValue)
}

// ResolveStringWithValidator resolves a string configuration with custom validation.
// Priority: CLI flag > environment variable > default value.
// If validator returns false, the value is rejected and default is used.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag
//   - envKey: Name of the environment variable
//   - defaultValue: Default value to use
//   - trimmed: If true, trim whitespace from environment variable value
//   - validator: Function to validate the resolved value (returns true if valid)
func ResolveStringWithValidator(
	fs *flag.FlagSet,
	flagName, envKey, defaultValue string,
	trimmed bool,
	validator func(string) bool,
) string {
	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) {
		value := flagutil.GetString(fs, flagName, defaultValue)
		if validator(value) {
			return value
		}
		// Invalid CLI value, fall back to default (don't try ENV)
		return defaultValue
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		var value string
		if trimmed {
			value = env.GetTrimmed(envKey, "")
		} else {
			value = env.Get(envKey, "")
		}
		if value != "" && validator(value) {
			return value
		}
		// Invalid or empty ENV value, fall back to default
	}

	// Priority 3: Default value
	return defaultValue
}

// ResolveStringNonEmpty resolves a string configuration, ensuring the result is non-empty.
// If both CLI and ENV yield empty strings, returns defaultValue.
// Priority: CLI flag > environment variable > default value.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag
//   - envKey: Name of the environment variable
//   - defaultValue: Default value to use
//   - trimmed: If true, trim whitespace from environment variable value
func ResolveStringNonEmpty(fs *flag.FlagSet, flagName, envKey, defaultValue string, trimmed bool) string {
	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) {
		value := flagutil.GetString(fs, flagName, defaultValue)
		// Check if value is non-empty
		if trimmed {
			if strings.TrimSpace(value) != "" {
				return value
			}
		} else {
			if value != "" {
				return value
			}
		}
		// Empty CLI value, try ENV next
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		var value string
		if trimmed {
			value = env.GetTrimmed(envKey, "")
		} else {
			value = env.Get(envKey, "")
		}
		if value != "" {
			return value
		}
		// Empty ENV value, fall back to default
	}

	// Priority 3: Default value
	return defaultValue
}

// ResolveStringWithValidation resolves a string configuration with custom validation function.
// Priority: CLI flag > environment variable > default value.
// If validator returns an error, the value is rejected and default is used.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag
//   - envKey: Name of the environment variable
//   - defaultValue: Default value to use
//   - trimmed: If true, trim whitespace from environment variable value
//   - validator: Function to validate the resolved value (returns error if invalid)
//
// Returns:
//   - string: The resolved and validated value
//   - error: Returns error if validation fails for all sources (CLI, ENV, default)
func ResolveStringWithValidation(
	fs *flag.FlagSet,
	flagName, envKey, defaultValue string,
	trimmed bool,
	validator func(string) error,
) (string, error) {
	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) {
		value := flagutil.GetString(fs, flagName, defaultValue)
		if err := validator(value); err == nil {
			return value, nil
		}
		// Invalid CLI value, try ENV
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		var value string
		if trimmed {
			value = env.GetTrimmed(envKey, "")
		} else {
			value = env.Get(envKey, "")
		}
		if value != "" {
			if err := validator(value); err == nil {
				return value, nil
			}
		}
		// Invalid ENV value, try default
	}

	// Priority 3: Default value
	if err := validator(defaultValue); err == nil {
		return defaultValue, nil
	}

	// All sources failed validation
	return defaultValue, validator(defaultValue)
}

// ResolveIntWithValidation resolves an integer configuration with custom validation function.
// Priority: CLI flag > environment variable > default value.
// If validator returns an error, the value is rejected and default is used.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag
//   - envKey: Name of the environment variable
//   - defaultValue: Default value to use
//   - allowZero: If false, zero values from ENV are treated as "not set" and default is used
//   - validator: Function to validate the resolved value (returns error if invalid)
//
// Returns:
//   - int: The resolved and validated value
//   - error: Returns error if validation fails for all sources
func ResolveIntWithValidation(
	fs *flag.FlagSet,
	flagName, envKey string,
	defaultValue int,
	allowZero bool,
	validator func(int) error,
) (int, error) {
	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) {
		value := flagutil.GetInt(fs, flagName, defaultValue)
		if err := validator(value); err == nil {
			return value, nil
		}
		// Invalid CLI value, try ENV
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		value := env.GetInt(envKey, defaultValue)
		if !allowZero && value == 0 {
			// Treat as not set, try default
		} else {
			if err := validator(value); err == nil {
				return value, nil
			}
		}
		// Invalid ENV value, try default
	}

	// Priority 3: Default value
	if err := validator(defaultValue); err == nil {
		return defaultValue, nil
	}

	// All sources failed validation
	return defaultValue, validator(defaultValue)
}

// ResolveStringSlice resolves a string slice configuration value with priority: CLI flag > environment variable > default value.
// For CLI flags, it expects a flag.Value implementation that collects multiple values (e.g., can be specified multiple times).
// For environment variables, it splits the value by the specified separator.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag (e.g., "hooks")
//   - envKey: Name of the environment variable (e.g., "HOOKS")
//   - defaultValue: Default value to use if neither CLI nor ENV is set
//   - sep: Separator for environment variable parsing (default ",")
func ResolveStringSlice(fs *flag.FlagSet, flagName, envKey string, defaultValue []string, sep string) []string {
	if sep == "" {
		sep = ","
	}

	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) {
		value := flagutil.GetString(fs, flagName, "")
		if value != "" {
			// Single value from flag, return as slice
			// Note: For multi-value flags, the caller should use flag.Var with a custom type
			return []string{value}
		}
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		result := env.GetStringSlice(envKey, nil, sep)
		if len(result) > 0 {
			return result
		}
	}

	// Priority 3: Default value
	return defaultValue
}

// ResolveStringSliceMulti resolves a string slice from a multi-value flag (flag.Value interface).
// This function reads the current value from a flag that implements the flag.Value interface
// and can collect multiple values (specified multiple times on command line).
// For environment variables, it splits the value by the specified separator.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag (e.g., "hooks")
//   - envKey: Name of the environment variable (e.g., "HOOKS")
//   - currentFlagValue: Current slice value from the flag (already parsed by flag.Var)
//   - defaultValue: Default value to use if neither CLI nor ENV is set
//   - sep: Separator for environment variable parsing (default ",")
func ResolveStringSliceMulti(fs *flag.FlagSet, flagName, envKey string, currentFlagValue, defaultValue []string, sep string) []string {
	if sep == "" {
		sep = ","
	}

	// Priority 1: CLI flag (highest priority)
	if flagutil.HasFlag(fs, flagName) && len(currentFlagValue) > 0 {
		return currentFlagValue
	}

	// Priority 2: Environment variable
	if env.Has(envKey) {
		result := env.GetStringSlice(envKey, nil, sep)
		if len(result) > 0 {
			return result
		}
	}

	// Priority 3: Default value
	return defaultValue
}

// ResolveEnum resolves an enum configuration value with validation.
// Priority: CLI flag > environment variable > default value.
// Validates that the value is in the allowed enum list.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag
//   - envKey: Name of the environment variable
//   - defaultValue: Default value to use
//   - allowedValues: List of allowed enum values
//   - caseSensitive: Whether enum comparison should be case-sensitive
//
// Returns:
//   - string: The resolved enum value
//   - error: Returns error if value is not in allowed list
func ResolveEnum(
	fs *flag.FlagSet,
	flagName, envKey, defaultValue string,
	allowedValues []string,
	caseSensitive bool,
) (string, error) {
	validateEnum := func(s string) error {
		return validator.ValidateEnum(s, allowedValues, caseSensitive)
	}
	return ResolveStringWithValidation(fs, flagName, envKey, defaultValue, true, validateEnum)
}

// ResolveHostPort resolves a host:port configuration with validation.
// Priority: CLI flag > environment variable > default value.
// Validates that the value is in valid host:port format.
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag
//   - envKey: Name of the environment variable
//   - defaultValue: Default host:port value to use
//
// Returns:
//   - host: The host part of the address
//   - port: The port number
//   - error: Returns error if format is invalid or port is out of range
func ResolveHostPort(
	fs *flag.FlagSet,
	flagName, envKey, defaultValue string,
) (host string, port int, err error) {
	value := ResolveString(fs, flagName, envKey, defaultValue, true)
	return validator.ValidateHostPort(value)
}

// ResolvePort resolves a port configuration with automatic validation.
// Priority: CLI flag > environment variable > default value.
// Automatically validates that the port is in the valid range (1-65535).
//
// Parameters:
//   - fs: FlagSet to check for CLI flag
//   - flagName: Name of the CLI flag
//   - envKey: Name of the environment variable
//   - defaultValue: Default port value to use
//
// Returns:
//   - int: The resolved and validated port number
//   - error: Returns error if port is out of range
func ResolvePort(
	fs *flag.FlagSet,
	flagName, envKey string,
	defaultValue int,
) (int, error) {
	validatePort := func(port int) error {
		return validator.ValidatePort(port)
	}
	return ResolveIntWithValidation(fs, flagName, envKey, defaultValue, false, validatePort)
}
