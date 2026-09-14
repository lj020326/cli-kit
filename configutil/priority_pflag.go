// Package configutil provides pflag-based config resolution with the same
// priority semantics as the flag-based API: CLI > environment variable > default.
// When envKey is empty, only CLI and default are used (no environment lookup).
package configutil

import (
	"strconv"
	"time"

	"github.com/lj020326/cli-kit/env"
	"github.com/lj020326/cli-kit/flagutil"
	"github.com/lj020326/cli-kit/validator"
	"github.com/spf13/pflag"
)

// ResolveStringPflag resolves a string with priority: CLI flag > env (if envKey set) > default.
func ResolveStringPflag(fs *pflag.FlagSet, flagName, envKey, defaultValue string, trimmed bool) string {
	if flagutil.HasFlagPflag(fs, flagName) {
		return flagutil.GetStringPflag(fs, flagName, defaultValue)
	}
	if envKey != "" && env.Has(envKey) {
		if trimmed {
			if v := env.GetTrimmed(envKey, ""); v != "" {
				return v
			}
		} else {
			if v := env.Get(envKey, ""); v != "" {
				return v
			}
		}
	}
	return defaultValue
}

// ResolveIntPflag resolves an int with priority: CLI flag > env (if envKey set) > default.
func ResolveIntPflag(fs *pflag.FlagSet, flagName, envKey string, defaultValue int, allowZero bool) int {
	if flagutil.HasFlagPflag(fs, flagName) {
		return flagutil.GetIntPflag(fs, flagName, defaultValue)
	}
	if envKey != "" && env.Has(envKey) {
		v := env.GetInt(envKey, defaultValue)
		if !allowZero && v == 0 {
			return defaultValue
		}
		return v
	}
	return defaultValue
}

// ResolveBoolPflag resolves a bool with priority: CLI flag > env (if envKey set) > default.
func ResolveBoolPflag(fs *pflag.FlagSet, flagName, envKey string, defaultValue bool) bool {
	if flagutil.HasFlagPflag(fs, flagName) {
		return flagutil.GetBoolPflag(fs, flagName, defaultValue)
	}
	if envKey != "" && env.Has(envKey) {
		return env.GetBool(envKey, defaultValue)
	}
	return defaultValue
}

// ResolveEnumPflag resolves an enum string with validation.
// When envKey is empty, only CLI and default are used.
func ResolveEnumPflag(
	fs *pflag.FlagSet,
	flagName, envKey, defaultValue string,
	allowedValues []string,
	caseSensitive bool,
) (string, error) {
	validate := func(s string) error {
		return validator.ValidateEnum(s, allowedValues, caseSensitive)
	}
	return ResolveStringWithValidationPflag(fs, flagName, envKey, defaultValue, true, validate)
}

// ResolveStringWithValidationPflag resolves a string with custom validation.
func ResolveStringWithValidationPflag(
	fs *pflag.FlagSet,
	flagName, envKey, defaultValue string,
	trimmed bool,
	validate func(string) error,
) (string, error) {
	if flagutil.HasFlagPflag(fs, flagName) {
		v := flagutil.GetStringPflag(fs, flagName, defaultValue)
		if err := validate(v); err == nil {
			return v, nil
		}
	}
	if envKey != "" && env.Has(envKey) {
		var v string
		if trimmed {
			v = env.GetTrimmed(envKey, "")
		} else {
			v = env.Get(envKey, "")
		}
		if v != "" {
			if err := validate(v); err == nil {
				return v, nil
			}
		}
	}
	if err := validate(defaultValue); err == nil {
		return defaultValue, nil
	}
	return defaultValue, validate(defaultValue)
}

// ResolveIntWithValidationPflag resolves an int with custom validation.
func ResolveIntWithValidationPflag(
	fs *pflag.FlagSet,
	flagName, envKey string,
	defaultValue int,
	allowZero bool,
	validate func(int) error,
) (int, error) {
	if flagutil.HasFlagPflag(fs, flagName) {
		v := flagutil.GetIntPflag(fs, flagName, defaultValue)
		if err := validate(v); err == nil {
			return v, nil
		}
	}
	if envKey != "" && env.Has(envKey) {
		v := env.GetInt(envKey, defaultValue)
		if allowZero || v != 0 {
			if err := validate(v); err == nil {
				return v, nil
			}
		}
	}
	if err := validate(defaultValue); err == nil {
		return defaultValue, nil
	}
	return defaultValue, validate(defaultValue)
}

// ResolvePortPflag resolves a port (1-65535) with validation.
func ResolvePortPflag(fs *pflag.FlagSet, flagName, envKey string, defaultValue int) (int, error) {
	validate := func(port int) error {
		return validator.ValidatePort(port)
	}
	return ResolveIntWithValidationPflag(fs, flagName, envKey, defaultValue, false, validate)
}

// ResolveDurationPflag resolves a duration with priority: CLI > env (if envKey set) > default.
func ResolveDurationPflag(fs *pflag.FlagSet, flagName, envKey string, defaultValue time.Duration) time.Duration {
	if flagutil.HasFlagPflag(fs, flagName) {
		return flagutil.GetDurationPflag(fs, flagName, defaultValue)
	}
	if envKey != "" && env.Has(envKey) {
		return env.GetDuration(envKey, defaultValue)
	}
	return defaultValue
}

// ResolveIntAsStringPflag resolves an int and returns it as string.
func ResolveIntAsStringPflag(fs *pflag.FlagSet, flagName, envKey string, defaultValue int, allowZero bool) string {
	return strconv.Itoa(ResolveIntPflag(fs, flagName, envKey, defaultValue, allowZero))
}
