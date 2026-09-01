# cli-kit

[![Go Reference](https://pkg.go.dev/badge/github.com/soulteary/cli-kit.svg)](https://pkg.go.dev/github.com/soulteary/cli-kit)
[![Go Report Card](.github/goreportcard.svg)](.github/goreportcard-report.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![codecov](https://codecov.io/gh/soulteary/cli-kit/graph/badge.svg)](https://codecov.io/gh/soulteary/cli-kit)

[English](README.md)

一个用于构建健壮命令行应用的 Go 工具库。提供环境变量管理、命令行参数处理、优先级配置解析、输入验证和测试辅助等功能。

## 功能特性

- **环境变量管理** - 安全灵活的环境变量操作，支持类型转换
- **命令行参数工具** - 增强的命令行参数处理，类型安全的取值方法
- **配置优先级解析** - 支持优先级的配置解析（CLI 参数 > 环境变量 > 默认值）
- **输入验证器** - 全面的验证功能，支持 URL、路径、端口、host:port、枚举值，内置 SSRF 防护
- **测试工具** - 用于测试 CLI 应用和配置解析的辅助函数

## 安装

```bash
go get github.com/soulteary/cli-kit
```

## 快速开始

### 环境变量

```go
import "github.com/soulteary/cli-kit/env"

// 检查环境变量是否存在
if env.Has("PORT") {
    // 变量已设置
}

// 获取值，支持默认值
port := env.Get("PORT", "8080")

// 获取类型化的值
portInt := env.GetInt("PORT", 8080)
timeout := env.GetDuration("TIMEOUT", 5*time.Second)
enabled := env.GetBool("ENABLED", false)
ratio := env.GetFloat64("RATIO", 0.5)

// 获取去除空白的字符串
value := env.GetTrimmed("CONFIG_PATH", "")

// 从逗号分隔的值获取字符串切片
hosts := env.GetStringSlice("HOSTS", []string{"localhost"}, ",")

// Lookup（区分"未设置"和"设置为空"）
value, ok := env.Lookup("API_KEY")

// 更多类型化获取：GetInt64、GetUint、GetUint64
```

### 命令行参数工具

```go
import "github.com/soulteary/cli-kit/flagutil"

fs := flag.NewFlagSet("app", flag.ContinueOnError)
port := fs.Int("port", 8080, "服务端口")

// 检查参数是否已设置
if flagutil.HasFlag(fs, "port") {
    // 参数已提供
}

// 获取带类型转换的参数值
portValue := flagutil.GetInt(fs, "port", 8080)
timeout := flagutil.GetDuration(fs, "timeout", 5*time.Second)
enabled := flagutil.GetBool(fs, "enabled", false)

// 检查参数是否存在于命令行中
if flagutil.HasFlagInOSArgs("verbose") {
    // 提供了 -verbose 或 --verbose
}

// 从文件读取密码（带安全检查）
password, err := flagutil.ReadPasswordFromFile("/path/to/password.txt")

// 更多：HasFlagInArgs(args, name)、GetFlagValue、GetString、GetInt64、GetUint、GetUint64、GetFloat64
```

**pflag 支持**：若使用 [spf13/pflag](https://github.com/spf13/pflag)（支持短选项、废弃标记等），可使用同名语义的 `*Pflag` 函数：

```go
import "github.com/soulteary/cli-kit/flagutil"
"github.com/spf13/pflag"

fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
fs.IntP("port", "p", 8080, "服务端口")
fs.Parse(os.Args)

if flagutil.HasFlagPflag(fs, "port") {
    port := flagutil.GetIntPflag(fs, "port", 8080)
}
// 同样提供：GetStringPflag、GetBoolPflag、GetDurationPflag、GetFlagValuePflag
```

### 配置优先级解析

`configutil` 包按照明确的优先级顺序解析配置值：**CLI 参数 > 环境变量 > 默认值**。

```go
import "github.com/soulteary/cli-kit/configutil"

fs := flag.NewFlagSet("app", flag.ContinueOnError)
portFlag := fs.Int("port", 0, "服务端口")
fs.Parse(os.Args[1:])

// 按优先级解析：CLI 参数 > 环境变量 > 默认值
port := configutil.ResolveInt(fs, "port", "PORT", 8080, false)
host := configutil.ResolveString(fs, "host", "HOST", "localhost", true)
debug := configutil.ResolveBool(fs, "debug", "DEBUG", false)
timeout := configutil.ResolveDuration(fs, "timeout", "TIMEOUT", 30*time.Second)

// 带验证的解析
url, err := configutil.ResolveStringWithValidation(
    fs, "url", "API_URL", "https://api.example.com",
    true, // 去除空白
    func(s string) error {
        return validator.ValidateURL(s, nil)
    },
)

// 解析枚举值
mode, err := configutil.ResolveEnum(
    fs, "mode", "APP_MODE", "production",
    []string{"development", "production", "staging"},
    false, // 不区分大小写
)

// 解析 host:port 并验证
host, port, err := configutil.ResolveHostPort(
    fs, "addr", "SERVER_ADDR", "localhost:8080",
)

// 解析端口并自动验证范围
port, err := configutil.ResolvePort(fs, "port", "PORT", 8080)
```

**pflag**：使用 `*pflag.FlagSet` 时可用 `Resolve*Pflag`，语义相同；传空字符串 `envKey` 时仅用 CLI 与默认值（不读环境变量）：

```go
fs := pflag.NewFlagSet("app", pflag.ContinueOnError)
// ... 定义并 Parse ...
port, err := configutil.ResolvePortPflag(fs, "port", "PORT", 8080)
base := configutil.ResolveBoolPflag(fs, "debug", "", false) // envKey 为空：仅 CLI 或 default
```

更多 configutil API（优先级均为：CLI > 环境变量 > 默认值）：

- **ResolveInt64** / **ResolveInt64WithValidation** - int64 及带自定义校验
- **ResolveIntAsString** - 按 int 解析但返回字符串
- **ResolveStringWithValidator** - 校验函数 `func(string) bool`，返回 string（无效则回退到下一来源）
- **ResolveStringWithValidation** - 校验函数 `func(string) error`，返回 `(string, error)`（见上）
- **ResolveStringNonEmpty** - 仅当值非空时采用 CLI/ENV，否则用默认值
- **ResolveIntWithValidation** - 带自定义校验的 int
- **ResolveStringSlice** / **ResolveStringSliceMulti** - 逗号分隔的切片（或多源合并）

### 验证器

```go
import "github.com/soulteary/cli-kit/validator"

// 验证 URL（默认启用 SSRF 防护）
err := validator.ValidateURL("https://api.example.com", nil)

// 自定义选项
opts := &validator.URLOptions{
    AllowedSchemes: []string{"http", "https", "ws", "wss"},
    AllowLocalhost: true,
    AllowPrivateIP: false,
}
err := validator.ValidateURL("http://localhost:8080", opts)

// 验证端口（范围：1-65535）
err := validator.ValidatePort(8080)
err := validator.ValidatePortString("8080")

// 验证 host:port
host, port, err := validator.ValidateHostPort("localhost:8080")

// 带默认值验证 host:port
host, port, err := validator.ValidateHostPortWithDefaults("myhost", "localhost", 8080)

// 验证路径（带安全检查）
absPath, err := validator.ValidatePath("/var/log/app.log", nil)

// 自定义选项
pathOpts := &validator.PathOptions{
    AllowRelative:  false,
    AllowedDirs:    []string{"/var/log", "/tmp"},
    CheckTraversal: true,
}
absPath, err := validator.ValidatePath("../etc/passwd", pathOpts) // 错误：路径遍历攻击

// 验证枚举
err := validator.ValidateEnum("production", 
    []string{"development", "production", "staging"},
    false, // 不区分大小写
)

// 数值验证（正数、非负、区间）
err := validator.ValidatePositive(42)                      // > 0
err = validator.ValidatePositiveInt64(100)
err = validator.ValidateNonNegative(0)                    // >= 0
err = validator.ValidateNonNegativeInt64(0)
err = validator.ValidateInRange(port, 1, 65535)          // [min, max] 闭区间
err = validator.ValidateInRangeInt64(n, 0, 100)
// 错误变量：validator.ErrNotPositive、validator.ErrNegative

// 验证手机号（支持多个地区）
err := validator.ValidatePhone("13800138000", nil) // 任意格式
err := validator.ValidatePhoneCN("13800138000")    // 中国大陆格式
err := validator.ValidatePhoneUS("+12025551234")   // 美国格式
err := validator.ValidatePhoneUK("+447911123456")  // 英国格式

// 自定义选项
phoneOpts := &validator.PhoneOptions{
    AllowEmpty: true,
    Region:     validator.PhoneRegionCN,
}
err := validator.ValidatePhone("13800138000", phoneOpts)

// 验证邮箱
err := validator.ValidateEmailSimple("user@example.com")

// 带域名限制
err := validator.ValidateEmailWithDomains("user@company.com", []string{"company.com"})

// 完整选项
emailOpts := &validator.EmailOptions{
    AllowEmpty:     false,
    AllowedDomains: []string{"company.com", "corp.com"},
    BlockedDomains: []string{"spam.com"},
}
err := validator.ValidateEmail("user@company.com", emailOpts)

// 验证用户名
err := validator.ValidateUsername("john_doe", nil)           // 默认风格（3-32字符）
err := validator.ValidateUsernameSimple("johndoe")           // 仅字母数字
err := validator.ValidateUsernameRelaxed("john.doe")         // 允许点号（3-64字符）

// 带保留名检查
err := validator.ValidateUsernameWithReserved("admin", []string{"admin", "root", "system"})
```

### 测试工具

```go
import (
    "github.com/soulteary/cli-kit/testutil"
    "testing"
)

// 测试中的环境变量管理
func TestMyFunction(t *testing.T) {
    envMgr := testutil.NewEnvManager()
    defer envMgr.Cleanup() // 自动恢复原始值
    
    envMgr.Set("PORT", "8080")
    envMgr.SetMultiple(map[string]string{
        "HOST":  "localhost",
        "DEBUG": "true",
    })
    
    // 你的测试代码
}

// 参数解析辅助
func TestFlags(t *testing.T) {
    fs := testutil.NewTestFlagSet("test")
    port := fs.Int("port", 8080, "端口")
    
    err := testutil.ParseFlags(fs, []string{"-port", "9090"})
    if err != nil {
        t.Fatal(err)
    }
    
    // 或使用 MustParseFlags，出错时 panic
    testutil.MustParseFlags(fs, []string{"-port", "9090"})
}

// 表驱动的配置测试（仅环境变量与默认值）。
// RunConfigTests 会为每个 case 注入 EnvVars，不会向 resolver 传递 CLIArgs。
// 若需测试「CLI 参数优先」，请单独写测试并自行 Parse 参数后断言。
func TestConfigResolution(t *testing.T) {
    cases := []testutil.ConfigTestCase{
        {
            Name:     "设置了环境变量时使用环境变量",
            EnvVars:  map[string]string{"PORT": "8080"},
            Expected: 8080,
        },
        {
            Name:     "都未设置时使用默认值",
            EnvVars:  map[string]string{},
            Expected: 3000,
        },
    }

    resolver := func(fs *flag.FlagSet, envVars map[string]string) (interface{}, error) {
        fs.Int("port", 0, "端口")
        if err := fs.Parse([]string{}); err != nil {
            return nil, err
        }
        return configutil.ResolveInt(fs, "port", "PORT", 3000, false), nil
    }

    testutil.RunConfigTests(t, cases, resolver)
}
```

## 项目结构

```
cli-kit/
├── env/              # 环境变量工具
│   └── env.go        # Get, GetInt, GetBool, GetDuration 等
├── flagutil/         # 命令行参数工具
│   └── flagutil.go   # HasFlag, GetInt, ReadPasswordFromFile 等
├── configutil/       # 优先级配置解析
│   └── priority.go   # ResolveString, ResolveInt, ResolveEnum 等
├── validator/        # 输入验证
│   ├── url.go        # URL 验证，支持 SSRF 防护
│   ├── path.go       # 路径验证，支持遍历攻击防护
│   ├── port.go       # 端口范围验证
│   ├── hostport.go   # host:port 格式验证
│   ├── enum.go       # 枚举值验证
│   ├── number.go     # 数值验证（正数、非负、区间）
│   ├── phone.go      # 手机号验证（中国/美国/英国/国际格式）
│   ├── email.go      # 邮箱验证，支持域名白名单/黑名单
│   └── username.go   # 用户名格式验证，支持多种风格
└── testutil/         # 测试工具
    ├── env.go        # 环境变量测试辅助
    ├── flag.go       # 参数解析测试辅助
    └── config.go     # 配置测试辅助
```

## 安全特性

| 特性 | 描述 |
|------|------|
| **SSRF 防护** | URL 验证器默认阻止私有 IP 和 localhost |
| **路径遍历防护** | 路径验证器检测并阻止 `..` 序列 |
| **目录限制** | 可选的允许目录白名单 |
| **安全文件读取** | 带路径验证的密码文件读取 |

## 测试覆盖率

本项目保持较高的测试覆盖率：

| 包 | 覆盖率 |
|------|--------|
| configutil | 100% |
| env | 100% |
| flagutil | 100% |
| validator | 91.9% |
| testutil | 86.7% |
| **总计** | **94.6%** |

运行测试并查看覆盖率：

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## 环境要求

- Go 1.26 或更高版本

## 许可证

本项目采用 Apache License 2.0 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 贡献

欢迎贡献！请随时提交 Pull Request。
