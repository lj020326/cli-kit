# cli-kit

[![Go Reference](https://pkg.go.dev/badge/github.com/soulteary/cli-kit.svg)](https://pkg.go.dev/github.com/soulteary/cli-kit)
[![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/soulteary/cli-kit/graph/badge.svg)](https://codecov.io/gh/soulteary/cli-kit)

[中文文档](README_CN.md)

A comprehensive Go library for building robust command-line applications. This toolkit provides utilities for environment variable management, command-line flag handling, priority-based configuration resolution, input validation, and testing support.

## Features

- **Environment Variable Management** - Safe and flexible environment variable operations with type conversion
- **Flag Utilities** - Enhanced command-line flag handling with type-safe getters
- **Configuration Resolution** - Priority-based configuration resolution (CLI flags > environment variables > defaults)
- **Validators** - Comprehensive validation for URLs, paths, ports, host:port, and enums with SSRF protection
- **Test Utilities** - Helper functions for testing CLI applications and configuration resolution

## Installation

```bash
go get github.com/soulteary/cli-kit
```

## Quick Start

### Environment Variables

```go
import "github.com/soulteary/cli-kit/env"

// Check if environment variable exists
if env.Has("PORT") {
    // Variable is set
}

// Get with default value
port := env.Get("PORT", "8080")

// Get typed values
portInt := env.GetInt("PORT", 8080)
timeout := env.GetDuration("TIMEOUT", 5*time.Second)
enabled := env.GetBool("ENABLED", false)
ratio := env.GetFloat64("RATIO", 0.5)

// Get trimmed string (removes leading/trailing whitespace)
value := env.GetTrimmed("CONFIG_PATH", "")

// Get string slice from comma-separated value
hosts := env.GetStringSlice("HOSTS", []string{"localhost"}, ",")

// Lookup (distinguish between not set and empty)
value, ok := env.Lookup("API_KEY")

// More typed getters: GetInt64, GetUint, GetUint64
```

### Flag Utilities

```go
import "github.com/soulteary/cli-kit/flagutil"

fs := flag.NewFlagSet("app", flag.ContinueOnError)
port := fs.Int("port", 8080, "Server port")

// Check if flag was set
if flagutil.HasFlag(fs, "port") {
    // Flag was provided
}

// Get flag value with type conversion
portValue := flagutil.GetInt(fs, "port", 8080)
timeout := flagutil.GetDuration(fs, "timeout", 5*time.Second)
enabled := flagutil.GetBool(fs, "enabled", false)

// Check if flag exists in command-line arguments
if flagutil.HasFlagInOSArgs("verbose") {
    // -verbose or --verbose was provided
}

// Read password from file (with security checks)
password, err := flagutil.ReadPasswordFromFile("/path/to/password.txt")

// More: HasFlagInArgs(args, name), GetFlagValue, GetString, GetInt64, GetUint, GetUint64, GetFloat64
```

**pflag support**: When using [spf13/pflag](https://github.com/spf13/pflag) (short flags, deprecated marks, etc.), use the `*Pflag` helpers with the same semantics:

```go
import "github.com/soulteary/cli-kit/flagutil"
"github.com/spf13/pflag"

fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
fs.IntP("port", "p", 8080, "Server port")
fs.Parse(os.Args)

if flagutil.HasFlagPflag(fs, "port") {
    port := flagutil.GetIntPflag(fs, "port", 8080)
}
// Also: GetStringPflag, GetBoolPflag, GetDurationPflag, GetFlagValuePflag
```

### Configuration Resolution

The `configutil` package resolves configuration values with a clear priority order: **CLI flags > Environment variables > Default values**.

```go
import "github.com/soulteary/cli-kit/configutil"

fs := flag.NewFlagSet("app", flag.ContinueOnError)
portFlag := fs.Int("port", 0, "Server port")
fs.Parse(os.Args[1:])

// Resolve with priority: CLI flag > ENV > default
port := configutil.ResolveInt(fs, "port", "PORT", 8080, false)
host := configutil.ResolveString(fs, "host", "HOST", "localhost", true)
debug := configutil.ResolveBool(fs, "debug", "DEBUG", false)
timeout := configutil.ResolveDuration(fs, "timeout", "TIMEOUT", 30*time.Second)

// Resolve with validation
url, err := configutil.ResolveStringWithValidation(
    fs, "url", "API_URL", "https://api.example.com",
    true, // trimmed
    func(s string) error {
        return validator.ValidateURL(s, nil)
    },
)

// Resolve enum value
mode, err := configutil.ResolveEnum(
    fs, "mode", "APP_MODE", "production",
    []string{"development", "production", "staging"},
    false, // case-insensitive
)

// Resolve host:port with validation
host, port, err := configutil.ResolveHostPort(
    fs, "addr", "SERVER_ADDR", "localhost:8080",
)

// Resolve port with automatic range validation
port, err := configutil.ResolvePort(fs, "port", "PORT", 8080)
```

**pflag**: With `*pflag.FlagSet`, use `Resolve*Pflag` with the same semantics. Pass an empty `envKey` to use only CLI and default (no environment lookup):

```go
fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
// ... define and Parse ...
port, err := configutil.ResolvePortPflag(fs, "port", "PORT", 8080)
base := configutil.ResolveBoolPflag(fs, "debug", "", false) // empty envKey: CLI or default only
```

Additional configutil APIs (same priority: CLI > ENV > default):

- **ResolveInt64** / **ResolveInt64WithValidation** - int64 and with custom validator
- **ResolveIntAsString** - resolve as int but return string
- **ResolveStringWithValidator** - validator `func(string) bool`, returns string (invalid falls back to next source)
- **ResolveStringWithValidation** - validator `func(string) error`, returns `(string, error)` (documented above)
- **ResolveStringNonEmpty** - use CLI/ENV only when value is non-empty, else default
- **ResolveIntWithValidation** - int with custom validation
- **ResolveStringSlice** / **ResolveStringSliceMulti** - slice from comma-separated (or multi-source merge)

### Validators

```go
import "github.com/soulteary/cli-kit/validator"

// Validate URL (with SSRF protection by default)
err := validator.ValidateURL("https://api.example.com", nil)

// With custom options
opts := &validator.URLOptions{
    AllowedSchemes: []string{"http", "https", "ws", "wss"},
    AllowLocalhost: true,
    AllowPrivateIP: false,
}
err := validator.ValidateURL("http://localhost:8080", opts)

// Validate port (range: 1-65535)
err := validator.ValidatePort(8080)
err := validator.ValidatePortString("8080")

// Validate host:port
host, port, err := validator.ValidateHostPort("localhost:8080")

// Validate host:port with defaults
host, port, err := validator.ValidateHostPortWithDefaults("myhost", "localhost", 8080)

// Validate path (with security checks)
absPath, err := validator.ValidatePath("/var/log/app.log", nil)

// With custom options
pathOpts := &validator.PathOptions{
    AllowRelative:  false,
    AllowedDirs:    []string{"/var/log", "/tmp"},
    CheckTraversal: true,
}
absPath, err := validator.ValidatePath("../etc/passwd", pathOpts) // Error: path traversal

// Validate enum
err := validator.ValidateEnum("production", 
    []string{"development", "production", "staging"},
    false, // case-insensitive
)

// Validate numbers (positive, non-negative, range)
err := validator.ValidatePositive(42)                      // > 0
err = validator.ValidatePositiveInt64(100)
err = validator.ValidateNonNegative(0)                    // >= 0
err = validator.ValidateNonNegativeInt64(0)
err = validator.ValidateInRange(port, 1, 65535)          // [min, max] inclusive
err = validator.ValidateInRangeInt64(n, 0, 100)
// Errors: validator.ErrNotPositive, validator.ErrNegative

// Validate phone number (supports multiple regions)
err := validator.ValidatePhone("13800138000", nil) // Any format
err := validator.ValidatePhoneCN("13800138000")    // Chinese mainland
err := validator.ValidatePhoneUS("+12025551234")   // US format
err := validator.ValidatePhoneUK("+447911123456")  // UK format

// With custom options
phoneOpts := &validator.PhoneOptions{
    AllowEmpty: true,
    Region:     validator.PhoneRegionCN,
}
err := validator.ValidatePhone("13800138000", phoneOpts)

// Validate email
err := validator.ValidateEmailSimple("user@example.com")

// With domain restrictions
err := validator.ValidateEmailWithDomains("user@company.com", []string{"company.com"})

// With full options
emailOpts := &validator.EmailOptions{
    AllowEmpty:     false,
    AllowedDomains: []string{"company.com", "corp.com"},
    BlockedDomains: []string{"spam.com"},
}
err := validator.ValidateEmail("user@company.com", emailOpts)

// Validate username
err := validator.ValidateUsername("john_doe", nil)           // Default style (3-32 chars)
err := validator.ValidateUsernameSimple("johndoe")           // Alphanumeric only
err := validator.ValidateUsernameRelaxed("john.doe")         // Allows dots (3-64 chars)

// With reserved names
err := validator.ValidateUsernameWithReserved("admin", []string{"admin", "root", "system"})
```

### Test Utilities

```go
import (
    "github.com/soulteary/cli-kit/testutil"
    "testing"
)

// Environment variable management in tests
func TestMyFunction(t *testing.T) {
    envMgr := testutil.NewEnvManager()
    defer envMgr.Cleanup() // Automatically restores original values
    
    envMgr.Set("PORT", "8080")
    envMgr.SetMultiple(map[string]string{
        "HOST":  "localhost",
        "DEBUG": "true",
    })
    
    // Your test code here
}

// Flag parsing helper
func TestFlags(t *testing.T) {
    fs := testutil.NewTestFlagSet("test")
    port := fs.Int("port", 8080, "port")
    
    err := testutil.ParseFlags(fs, []string{"-port", "9090"})
    if err != nil {
        t.Fatal(err)
    }
    
    // Or use MustParseFlags to panic on error
    testutil.MustParseFlags(fs, []string{"-port", "9090"})
}

// Table-driven configuration tests (ENV and default only).
// RunConfigTests injects EnvVars for each case; it does NOT pass CLIArgs to the resolver.
// To test "CLI flag takes priority", use a separate test that parses flags and asserts.
func TestConfigResolution(t *testing.T) {
    cases := []testutil.ConfigTestCase{
        {
            Name:     "ENV used when set",
            EnvVars:  map[string]string{"PORT": "8080"},
            Expected: 8080,
        },
        {
            Name:     "Default used when neither set",
            EnvVars:  map[string]string{},
            Expected: 3000,
        },
    }

    resolver := func(fs *flag.FlagSet, envVars map[string]string) (interface{}, error) {
        fs.Int("port", 0, "Port")
        if err := fs.Parse([]string{}); err != nil {
            return nil, err
        }
        return configutil.ResolveInt(fs, "port", "PORT", 3000, false), nil
    }

    testutil.RunConfigTests(t, cases, resolver)
}
```

## Project Structure

```
cli-kit/
├── env/              # Environment variable utilities
│   └── env.go        # Get, GetInt, GetBool, GetDuration, etc.
├── flagutil/         # Command-line flag utilities
│   └── flagutil.go   # HasFlag, GetInt, ReadPasswordFromFile, etc.
├── configutil/       # Configuration resolution with priority
│   └── priority.go   # ResolveString, ResolveInt, ResolveEnum, etc.
├── validator/        # Input validation
│   ├── url.go        # URL validation with SSRF protection
│   ├── path.go       # Path validation with traversal protection
│   ├── port.go       # Port range validation
│   ├── hostport.go   # Host:port format validation
│   ├── enum.go       # Enum value validation
│   ├── number.go     # Numeric validation (positive, non-negative, range)
│   ├── phone.go      # Phone number validation (CN/US/UK/International)
│   ├── email.go      # Email address validation with domain control
│   └── username.go   # Username format validation with styles
└── testutil/         # Testing utilities
    ├── env.go        # Environment variable test helpers
    ├── flag.go       # Flag parsing test helpers
    └── config.go     # Configuration test helpers
```

## Security Features

| Feature | Description |
|---------|-------------|
| **SSRF Protection** | URL validator blocks private IPs and localhost by default |
| **Path Traversal Prevention** | Path validator detects and blocks `..` sequences |
| **Directory Restrictions** | Optional allowlist for permitted directories |
| **Safe File Reading** | Password file reading with path validation |

## Test Coverage

This project maintains high test coverage:

| Package | Coverage |
|---------|----------|
| configutil | 100% |
| env | 100% |
| flagutil | 100% |
| validator | 91.9% |
| testutil | 86.7% |
| **Total** | **94.6%** |

Run tests with coverage:

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Requirements

- Go 1.26 or later

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
