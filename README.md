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
- **Validators** - URLs, paths, ports, host:port, enums, numbers, phones, emails and usernames, with SSRF and path-traversal protection
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

Available `*Pflag` resolvers: `ResolveStringPflag`, `ResolveIntPflag`,
`ResolveBoolPflag`, `ResolveDurationPflag`, `ResolvePortPflag`,
`ResolveEnumPflag`, `ResolveIntAsStringPflag`, `ResolveIntWithValidationPflag`,
`ResolveStringWithValidationPflag`.

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
```

#### URLs and SSRF

`ValidateURL` blocks private addresses and loopback by default. Passing `nil`
gets the safe defaults:

```go
err := validator.ValidateURL("https://api.example.com", nil)
```

Every field is optional, and an unset field takes its default — a partial
struct never weakens the check:

```go
opts := &validator.URLOptions{
    AllowedSchemes: []string{"http", "https", "ws", "wss"},
    AllowLocalhost: true,
    AllowPrivateIP: false,
}
err := validator.ValidateURL("http://localhost:8080", opts)
```

| Option | Default | Notes |
|--------|---------|-------|
| `AllowedSchemes` | `http`, `https` | unset means the default list |
| `AllowLocalhost` | `false` | also matches `localhost.` and `*.localhost` |
| `AllowPrivateIP` | `false` | RFC 1918, link-local, CGNAT, `192.0.0.0/24`, TEST-NET, `240.0.0.0/4` |
| `ResolveHostTimeout` | `5s` | bound on the DNS lookup; `0` means the default |
| `DisableHostResolution` | `false` | skip DNS entirely — the explicit opt-out |

A hostname is resolved and every address it returns is checked, so a name
pointing at `169.254.169.254` is rejected. Set `DisableHostResolution` only
where you have another control in place, or in tests that must stay offline.

**`ValidateURL` alone is not sufficient against DNS rebinding.** It inspects
the addresses a name resolved to *at validation time*, while your HTTP client
resolves again when it connects. A short-TTL record can answer differently the
second time — a public address for the validator, a link-local one for the
request. Close that window with `SSRFDialControl`, which applies the same
policy to the address actually being dialled:

```go
opts := &validator.URLOptions{AllowedSchemes: []string{"https"}}

if err := validator.ValidateURL(rawURL, opts); err != nil {
    return err
}

transport := &http.Transport{
    // Proxying is disabled deliberately: see the caveat below.
    Proxy:       nil,
    DialContext: (&net.Dialer{Control: validator.SSRFDialControl(opts)}).DialContext,
}
client := &http.Client{Transport: transport}
```

> **Proxy caveat.** With a proxy in effect — including the one
> `http.DefaultTransport` picks up from `HTTP_PROXY`/`HTTPS_PROXY` — the hook
> sees the *proxy's* address, not the origin's. A public proxy would then
> resolve a rebinding hostname to an internal address without this control ever
> seeing it, while a private corporate proxy is rejected outright. Where a proxy
> is required, the hook only closes the rebinding window for direct connections.

#### Paths

```go
absPath, err := validator.ValidatePath("/var/log/app.log", nil)

pathOpts := &validator.PathOptions{
    AllowRelative:  false,
    AllowedDirs:    []string{"/var/log", "/tmp"},
    CheckTraversal: true,
}
absPath, err = validator.ValidatePath("../etc/passwd", pathOpts) // rejected
```

| Option | Default | Notes |
|--------|---------|-------|
| `AllowRelative` | `false` | reject a path that is not absolute |
| `AllowedDirs` | none | containment allowlist; bases are symlink-resolved too |
| `CheckTraversal` | `false` | reject a `..` parent reference |

`CheckTraversal` matches `..` as a whole path segment, so ordinary names like
`backup..2024.log` and `..hidden` are accepted. It splits on both `/` and `\`
on every platform, strips a Windows drive prefix before splitting (so
`C:..\secret` is caught on Linux too), and treats a segment of only periods and
spaces with at least two periods as a parent reference — because Win32 strips
trailing spaces and periods from a component, making `safe\.. \secret` open the
parent.

Containment resolves symlinks on both sides, including for a target that does
not exist yet: the deepest existing ancestor is resolved and the remaining
components re-attached. With `/allowed/link` a symlink to `/etc`, validating
`/allowed/link/newfile` is correctly rejected.

> **Time-of-check/time-of-use.** `ValidatePath` returns a *name*, not an open
> handle. The path can be replaced between the check and your `os.Open`. Where
> that matters, open the file and verify the handle (for example with
> `os.OpenFile` plus `O_NOFOLLOW`) rather than trusting the validated name.

#### Ports and host:port

```go
err := validator.ValidatePort(8080)                  // 1-65535
port, err := validator.ValidatePortString("8080")

host, port, err := validator.ValidateHostPort("localhost:8080")
host, port, err = validator.ValidateHostPortWithDefaults("myhost", "localhost", 8080)
host, port, err = validator.ParseHostPort("example.com:443") // parse without validating
```

Scoped IPv6 addresses keep their zone: `[fe80::1%eth0]:443` parses correctly.

#### Enums and numbers

```go
err := validator.ValidateEnum("production",
    []string{"development", "production", "staging"},
    false, // case-insensitive
)
err = validator.ValidateEnumCaseInsensitive("PRODUCTION", allowed)
err = validator.ValidateEnumCaseSensitive("production", allowed)

err = validator.ValidatePositive(42)            // > 0
err = validator.ValidatePositiveInt64(100)
err = validator.ValidateNonNegative(0)          // >= 0
err = validator.ValidateNonNegativeInt64(0)
err = validator.ValidateInRange(port, 1, 65535) // inclusive
err = validator.ValidateInRangeInt64(n, 0, 100)
```

#### Phone numbers

```go
err := validator.ValidatePhone("13800138000", nil) // any format
err = validator.ValidatePhoneCN("13800138000")
err = validator.ValidatePhoneUS("+12025551234")
err = validator.ValidatePhoneUK("+447911123456")
err = validator.ValidatePhoneInternational("+8613800138000")

err = validator.ValidatePhone("13800138000", &validator.PhoneOptions{
    AllowEmpty: true,
    Region:     validator.PhoneRegionCN, // or PhoneRegionUS/UK/International/Any
})
```

#### Email addresses

```go
err := validator.ValidateEmailSimple("user@example.com")
err = validator.ValidateEmailWithDomains("user@company.com", []string{"company.com"})

err = validator.ValidateEmail("user@company.com", &validator.EmailOptions{
    AllowEmpty:     false,
    AllowedDomains: []string{"company.com", "corp.com"},
    BlockedDomains: []string{"spam.com"},
})

domain := validator.ExtractEmailDomain("user@example.com") // "example.com"
```

#### Usernames

```go
err := validator.ValidateUsername("john_doe", nil)   // default style, 3-32 chars
err = validator.ValidateUsernameSimple("johndoe")    // alphanumeric only
err = validator.ValidateUsernameRelaxed("john.doe")  // allows dots, 3-64 chars
err = validator.ValidateUsernameWithReserved("admin", []string{"admin", "root"})

err = validator.ValidateUsername("john_doe", &validator.UsernameOptions{
    Style:         validator.UsernameStyleCustom, // or Default/Simple/Relaxed
    CustomPattern: regexp.MustCompile(`^[a-z][a-z0-9-]{2,15}$`),
    MinLength:     3,
    MaxLength:     16,
    ReservedNames: []string{"admin", "root"},
    AllowEmpty:    false,
})

normalized := validator.NormalizeUsername("  John_Doe ") // trimmed, lower-cased
ok := validator.IsValidUsernameChar('_')
```

#### Files and directories

```go
err := validator.ValidateFileExists("/etc/app.conf")
err = validator.ValidateFileReadable("/etc/app.conf")
err = validator.ValidateDirExists("/var/log")
err = validator.ValidateDirWritable("/var/log")
```

#### Error sentinels

Match with `errors.Is`:

`ErrInvalidPort`, `ErrInvalidHostPort`, `ErrInvalidEnumValue`, `ErrInvalidEmail`,
`ErrInvalidPhone`, `ErrInvalidUsername`, `ErrNotPositive`, `ErrNegative`,
`ErrFileNotFound`, `ErrFileNotReadable`, `ErrNotAFile`, `ErrDirNotFound`,
`ErrDirNotWritable`, `ErrNotADirectory`.

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

## Upgrade Notes (v1.9.0)

All changes are in `validator`. One field and one function were added; nothing
was removed. Some inputs that used to pass are now correctly rejected, and some
that used to be rejected now pass.

- **A partial `URLOptions` no longer switches off hostname checking.**
  `ResolveHostTimeout` was copied unconditionally and its zero value meant "skip
  DNS resolution", so

  ```go
  validator.ValidateURL(u, &validator.URLOptions{
      AllowedSchemes: []string{"https"},
  })
  ```

  which reads as "narrow the scheme list", also accepted any hostname without
  resolving it — including one pointing at `169.254.169.254`. Zero now means
  "use the default 5s". **If you relied on the old behaviour to keep a test
  offline, set `DisableHostResolution: true`** — otherwise those calls will
  start performing real DNS lookups.
- **`SSRFDialControl` is new, and `ValidateURL` is documented as insufficient
  on its own.** Validation checks the addresses a name resolved to; the client
  resolves again when it connects. Add the dial hook wherever you fetch a
  caller-supplied URL.
- **`CheckTraversal` accepts legal names containing dots.** It used
  `strings.Contains(path, "..")`, which rejected `backup..2024.log` and
  `..hidden`. It now matches `..` as a whole segment.
- **`CheckTraversal` catches traversal it previously missed.** Windows
  drive-relative paths (`C:..\secret`), backslash-separated paths validated on
  Linux, and the spellings Win32 normalizes into `..` (`safe\.. \secret`) all
  escaped the segment check. On POSIX this refuses one otherwise legal name,
  `...`, which is the right side to err on under an explicitly requested
  traversal check.
- **A symlinked parent no longer escapes `AllowedDirs` for a file that does not
  exist yet.** `EvalSymlinks` fails on a non-existent path and the fallback
  checked the unresolved name, so with `/allowed/link` pointing at `/etc`,
  validating `/allowed/link/newfile` looked contained while the write landed in
  `/etc/newfile`. Reading existing files was protected; creating new ones was
  not.
- **Allowed bases are resolved the same way as the request.** With
  `alias -> /real` and `AllowedDirs: ["alias/future"]`, a genuinely contained
  creation used to be rejected because the base stayed lexical.
- **`AllowLocalhost` matches more spellings.** `localhost.` and `*.localhost`
  both resolve to loopback and were missed by the bare string compare.
- **More ranges count as private.** `192.0.0.0/24`, `192.0.2.0/24` (TEST-NET-1)
  and `240.0.0.0/4` are blocked unless `AllowPrivateIP` is set. Two documented
  globally reachable addresses in that first block are excepted: `192.0.0.9`
  (PCP anycast, RFC 7723) and `192.0.0.10` (TURN anycast, RFC 8155), so reaching
  them no longer means enabling all private addresses.
- **Scoped IPv6 addresses validate.** `[fe80::1%eth0]:443` was rejected as "not
  an IP" even with `AllowPrivateIP` set, because the zone was left on the string
  handed to `net.ParseIP`.
- **`ValidatePath`'s time-of-check/time-of-use gap is documented.** It returns a
  name, not an open handle.

## Security Features

| Feature | Description |
|---------|-------------|
| **SSRF protection** | `ValidateURL` resolves hostnames and rejects private, loopback, link-local, CGNAT and reserved addresses by default. A partial `URLOptions` cannot weaken it. |
| **DNS rebinding** | `SSRFDialControl` re-applies the policy to the address actually dialled, closing the check-then-use window `ValidateURL` cannot |
| **Path traversal** | `CheckTraversal` matches `..` as a path segment, on both separators, after a Windows drive prefix, including the spellings Win32 normalizes into `..` |
| **Symlink containment** | `AllowedDirs` resolves symlinks on both sides, including for a target that does not exist yet |
| **Directory restrictions** | Optional allowlist of permitted directories |
| **Safe file reading** | `flagutil.ReadPasswordFromFile` validates the path before reading |

## Test Coverage

| Package | Coverage |
|---------|----------|
| configutil | 100% |
| env | 100% |
| flagutil | 96.7% |
| validator | 92.9% |
| testutil | 88.7% |
| **Total** | **95.2%** |

Run tests with coverage:

```bash
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
```

A few `validator` and `flagutil` tests simulate I/O failures with `chmod`, which
uid 0 ignores; those sections skip when the suite runs as root.

## Requirements

- **Go 1.27+** (`go.mod` declares `go 1.27.0`)
- Optional: `github.com/spf13/pflag` for the `*Pflag` helpers

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
