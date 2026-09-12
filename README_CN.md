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
- **输入验证器** - URL、路径、端口、host:port、枚举、数字、手机号、邮箱和用户名校验，内置 SSRF 与路径遍历防护
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

可用的 `*Pflag` 解析函数：`ResolveStringPflag`、`ResolveIntPflag`、
`ResolveBoolPflag`、`ResolveDurationPflag`、`ResolvePortPflag`、
`ResolveEnumPflag`、`ResolveIntAsStringPflag`、`ResolveIntWithValidationPflag`、
`ResolveStringWithValidationPflag`。

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
```

#### URL 与 SSRF

`ValidateURL` 默认阻止私有地址和回环地址。传 `nil` 即使用安全默认值：

```go
err := validator.ValidateURL("https://api.example.com", nil)
```

所有字段都是可选的，未设置的字段取默认值——部分填充的结构体绝不会削弱检查：

```go
opts := &validator.URLOptions{
    AllowedSchemes: []string{"http", "https", "ws", "wss"},
    AllowLocalhost: true,
    AllowPrivateIP: false,
}
err := validator.ValidateURL("http://localhost:8080", opts)
```

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `AllowedSchemes` | `http`、`https` | 未设置即使用默认列表 |
| `AllowLocalhost` | `false` | 同时匹配 `localhost.` 和 `*.localhost` |
| `AllowPrivateIP` | `false` | RFC 1918、链路本地、CGNAT、`192.0.0.0/24`、TEST-NET、`240.0.0.0/4` |
| `ResolveHostTimeout` | `5s` | DNS 查询的时限；`0` 表示使用默认值 |
| `DisableHostResolution` | `false` | 完全跳过 DNS —— 显式的退出开关 |

主机名会被解析，并检查它返回的每一个地址，因此指向 `169.254.169.254` 的名字会被
拒绝。只有在你另有管控手段、或测试必须离线时，才设置 `DisableHostResolution`。

**单靠 `ValidateURL` 无法防住 DNS 重绑定。** 它检查的是名字在*校验时刻*解析出的
地址，而你的 HTTP 客户端在真正连接时会再解析一次。短 TTL 的记录第二次可以给出不同
答案——给校验器一个公网地址，给请求一个链路本地地址。用 `SSRFDialControl` 关掉这个
窗口，它会把同一套策略应用到真正被拨号的地址上：

```go
opts := &validator.URLOptions{AllowedSchemes: []string{"https"}}

if err := validator.ValidateURL(rawURL, opts); err != nil {
    return err
}

transport := &http.Transport{
    // 特意禁用代理：见下面的注意事项。
    Proxy:       nil,
    DialContext: (&net.Dialer{Control: validator.SSRFDialControl(opts)}).DialContext,
}
client := &http.Client{Transport: transport}
```

> **代理注意事项。** 一旦代理生效——包括 `http.DefaultTransport` 从
> `HTTP_PROXY`/`HTTPS_PROXY` 继承来的那个——这个钩子看到的是*代理*的地址，不是源站
> 的。于是公网代理会把一个重绑定主机名解析到内网地址，而这个控制根本看不到；反过来，
> 企业内网代理会被直接拒绝。在必须走代理的场景下，这个钩子只能为直连关闭重绑定窗口。

#### 路径

```go
absPath, err := validator.ValidatePath("/var/log/app.log", nil)

pathOpts := &validator.PathOptions{
    AllowRelative:  false,
    AllowedDirs:    []string{"/var/log", "/tmp"},
    CheckTraversal: true,
}
absPath, err = validator.ValidatePath("../etc/passwd", pathOpts) // 被拒绝
```

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `AllowRelative` | `false` | 拒绝非绝对路径 |
| `AllowedDirs` | 无 | 包含性白名单；基准目录同样会做符号链接解析 |
| `CheckTraversal` | `false` | 拒绝 `..` 父目录引用 |

`CheckTraversal` 把 `..` 作为完整路径段来匹配，因此 `backup..2024.log`、
`..hidden` 这类正常名字是允许的。它在所有平台上都按 `/` 和 `\` 两种分隔符切分，
切分前先剥掉 Windows 盘符前缀（所以 `C:..\secret` 在 Linux 上也能抓到），并把
"只由句点和空格组成、且至少含两个句点"的段视为父目录引用——因为 Win32 会剥掉路径
组件末尾的空格和句点，使 `safe\.. \secret` 指向父目录。

包含性检查对两侧都做符号链接解析，目标尚不存在时也一样：先解析最深的已存在祖先，
再把剩余组件接回去。当 `/allowed/link` 是指向 `/etc` 的符号链接时，校验
`/allowed/link/newfile` 会被正确拒绝。

> **检查时与使用时的间隙。** `ValidatePath` 返回的是*名字*，不是已打开的句柄。
> 路径可以在检查之后、你 `os.Open` 之前被替换掉。在意这一点的场景，请打开文件后
> 对句柄本身做校验（例如 `os.OpenFile` 加 `O_NOFOLLOW`），而不是信任校验过的名字。

#### 端口与 host:port

```go
err := validator.ValidatePort(8080)                  // 1-65535
port, err := validator.ValidatePortString("8080")

host, port, err := validator.ValidateHostPort("localhost:8080")
host, port, err = validator.ValidateHostPortWithDefaults("myhost", "localhost", 8080)
host, port, err = validator.ParseHostPort("example.com:443") // 只解析不校验
```

带作用域的 IPv6 地址会保留 zone：`[fe80::1%eth0]:443` 可以正确解析。

#### 枚举与数字

```go
err := validator.ValidateEnum("production",
    []string{"development", "production", "staging"},
    false, // 不区分大小写
)
err = validator.ValidateEnumCaseInsensitive("PRODUCTION", allowed)
err = validator.ValidateEnumCaseSensitive("production", allowed)

err = validator.ValidatePositive(42)            // > 0
err = validator.ValidatePositiveInt64(100)
err = validator.ValidateNonNegative(0)          // >= 0
err = validator.ValidateNonNegativeInt64(0)
err = validator.ValidateInRange(port, 1, 65535) // 闭区间
err = validator.ValidateInRangeInt64(n, 0, 100)
```

#### 手机号

```go
err := validator.ValidatePhone("13800138000", nil) // 任意格式
err = validator.ValidatePhoneCN("13800138000")
err = validator.ValidatePhoneUS("+12025551234")
err = validator.ValidatePhoneUK("+447911123456")
err = validator.ValidatePhoneInternational("+8613800138000")

err = validator.ValidatePhone("13800138000", &validator.PhoneOptions{
    AllowEmpty: true,
    Region:     validator.PhoneRegionCN, // 或 PhoneRegionUS/UK/International/Any
})
```

#### 邮箱地址

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

#### 用户名

```go
err := validator.ValidateUsername("john_doe", nil)   // 默认风格，3-32 字符
err = validator.ValidateUsernameSimple("johndoe")    // 仅字母数字
err = validator.ValidateUsernameRelaxed("john.doe")  // 允许点号，3-64 字符
err = validator.ValidateUsernameWithReserved("admin", []string{"admin", "root"})

err = validator.ValidateUsername("john_doe", &validator.UsernameOptions{
    Style:         validator.UsernameStyleCustom, // 或 Default/Simple/Relaxed
    CustomPattern: regexp.MustCompile(`^[a-z][a-z0-9-]{2,15}$`),
    MinLength:     3,
    MaxLength:     16,
    ReservedNames: []string{"admin", "root"},
    AllowEmpty:    false,
})

normalized := validator.NormalizeUsername("  John_Doe ") // 去首尾空白并转小写
ok := validator.IsValidUsernameChar('_')
```

#### 文件与目录

```go
err := validator.ValidateFileExists("/etc/app.conf")
err = validator.ValidateFileReadable("/etc/app.conf")
err = validator.ValidateDirExists("/var/log")
err = validator.ValidateDirWritable("/var/log")
```

#### 错误哨兵

用 `errors.Is` 判断：

`ErrInvalidPort`、`ErrInvalidHostPort`、`ErrInvalidEnumValue`、`ErrInvalidEmail`、
`ErrInvalidPhone`、`ErrInvalidUsername`、`ErrNotPositive`、`ErrNegative`、
`ErrFileNotFound`、`ErrFileNotReadable`、`ErrNotAFile`、`ErrDirNotFound`、
`ErrDirNotWritable`、`ErrNotADirectory`。

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

## 升级说明（v1.9.0）

改动全部在 `validator` 包。新增一个字段和一个函数，没有删除任何东西。一些原本能通过
的输入现在会被正确拒绝，一些原本被拒绝的现在能通过。

- **部分填充的 `URLOptions` 不再关掉主机名检查。** `ResolveHostTimeout` 此前被
  无条件复制，而它的零值意味着"跳过 DNS 解析"，于是

  ```go
  validator.ValidateURL(u, &validator.URLOptions{
      AllowedSchemes: []string{"https"},
  })
  ```

  这段读起来是"收紧 scheme 列表"的代码，同时也让任何主机名都无需解析即被接受——包括
  指向 `169.254.169.254` 的。现在零值表示"使用默认的 5s"。**如果你原本靠这个行为让
  测试保持离线，请设置 `DisableHostResolution: true`**——否则这些调用会开始真正发起
  DNS 查询。
- **新增 `SSRFDialControl`，并明确写出 `ValidateURL` 单独不足够。** 校验检查的是
  名字当时解析出的地址，客户端连接时会再解析一次。凡是要抓取调用方提供的 URL 的地方，
  都请加上这个拨号钩子。
- **`CheckTraversal` 接受含点号的合法名字。** 它此前用
  `strings.Contains(path, "..")`，会拒绝 `backup..2024.log` 和 `..hidden`。现在按
  完整路径段匹配 `..`。
- **`CheckTraversal` 能抓到此前漏掉的遍历。** Windows 盘符相对路径
  （`C:..\secret`）、在 Linux 上校验的反斜杠分隔路径，以及 Win32 会归一化成 `..`
  的写法（`safe\.. \secret`）都曾绕过段检查。在 POSIX 上这会多拒绝一个本来合法的
  名字 `...`——在一个被显式要求做遍历检查的场景下，这是该偏向的一边。
- **目标尚不存在的文件，符号链接父目录不再能逃出 `AllowedDirs`。**
  `EvalSymlinks` 对不存在的路径会失败，而兜底逻辑检查的是未解析的名字，于是当
  `/allowed/link` 指向 `/etc` 时，校验 `/allowed/link/newfile` 看起来是被包含的，
  而写入落到了 `/etc/newfile`。读已有文件是受保护的，创建新文件不是。
- **允许的基准目录与请求用同一套方式解析。** 当 `alias -> /real` 且
  `AllowedDirs: ["alias/future"]` 时，一次真正被包含的创建此前会因为基准目录停留在
  字面形式而被拒绝。
- **`AllowLocalhost` 匹配更多写法。** `localhost.` 和 `*.localhost` 都解析到回环
  地址，而此前的字符串直接比较漏掉了它们。
- **更多地址段算作私有。** 未设置 `AllowPrivateIP` 时，`192.0.0.0/24`、
  `192.0.2.0/24`（TEST-NET-1）和 `240.0.0.0/4` 会被阻止。第一段里有两个文档明确
  全球可达的地址被豁免：`192.0.0.9`（PCP anycast，RFC 7723）和 `192.0.0.10`
  （TURN anycast，RFC 8155），因此访问它们不再需要放开所有私有地址。
- **带作用域的 IPv6 地址能通过校验。** `[fe80::1%eth0]:443` 此前即便设置了
  `AllowPrivateIP` 也会被报成"不是 IP"，因为交给 `net.ParseIP` 的字符串还带着 zone。
- **文档写明了 `ValidatePath` 的检查时/使用时间隙。** 它返回的是名字，不是已打开的
  句柄。

## 安全特性

| 特性 | 描述 |
|------|------|
| **SSRF 防护** | `ValidateURL` 会解析主机名，默认拒绝私有、回环、链路本地、CGNAT 和保留地址。部分填充的 `URLOptions` 无法削弱它。 |
| **DNS 重绑定** | `SSRFDialControl` 把策略重新应用到真正被拨号的地址上，关掉 `ValidateURL` 无法关掉的"检查后使用"窗口 |
| **路径遍历** | `CheckTraversal` 把 `..` 当作完整路径段匹配，覆盖两种分隔符、盘符前缀之后，以及 Win32 会归一化成 `..` 的各种写法 |
| **符号链接包含性** | `AllowedDirs` 对两侧都做符号链接解析，目标尚不存在时也一样 |
| **目录限制** | 可选的允许目录白名单 |
| **安全文件读取** | `flagutil.ReadPasswordFromFile` 读取前先校验路径 |

## 测试覆盖率

| 包 | 覆盖率 |
|------|--------|
| configutil | 100% |
| env | 100% |
| flagutil | 96.7% |
| validator | 92.9% |
| testutil | 88.7% |
| **总计** | **95.2%** |

运行测试并查看覆盖率：

```bash
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out
```

`validator` 和 `flagutil` 里有少数测试用 `chmod` 模拟 I/O 失败，而 uid 0 会忽略
权限位，因此以 root 运行时这些部分会跳过。

## 环境要求

- **Go 1.27+**（`go.mod` 声明 `go 1.27.0`）
- 可选：`github.com/spf13/pflag`（使用 `*Pflag` 辅助函数时）

## 许可证

本项目采用 Apache License 2.0 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 贡献

欢迎贡献！请随时提交 Pull Request。
