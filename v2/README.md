# verify

Go 参数验证库，封装 [go-playground/validator](https://github.com/go-playground/validator)，
提供中英文翻译、Gin 集成、自定义验证注册。

## 安装

```bash
go get github.com/gtkit/verify@latest
```

## 快速开始

```go
// 初始化（选一种方式）

// 方式一：实例模式（推荐）
v := verify.MustNew(verify.WithLocale("zh"), verify.WithGinBinding())

// 方式二：包级模式
verify.Init(verify.WithLocale("zh"), verify.WithGinBinding())
```

## 结构体验证

```go
type User struct {
    Name  string `json:"name" binding:"required,min=2,max=20"`
    Email string `json:"email" binding:"required,email"`
}

// 验证
err := v.Struct(user)
if err != nil {
    return v.StructErr(err)  // → "name长度必须至少为2个字符"
}

// 需要全部错误时
all := v.AllFieldErrors(err)
// map[string]string{"name": "...", "email": "..."}
```

## 字段验证

```go
err := v.Field("www.google.com", "required,numeric")
if err != nil {
    return v.FieldErr("type", err)  // → "type 必须是一个有效的数值"
}
```

## Map 验证

```go
data := map[string]any{"name": "ab", "email": "bad"}
rules := map[string]any{"name": "required,min=8,max=15", "email": "required,email"}

result := v.Map(data, rules)
if len(result) > 0 {
    return v.MapErr(result)  // → "name name长度必须至少为8个字符"
}

// 需要全部错误时
all := v.AllMapErrors(result)
```

## 字段比较验证

```go
err := v.WithValue(password, confirmPassword, "eqfield")
```

## Gin 集成

```go
func main() {
    v := verify.MustNew(verify.WithLocale("zh"), verify.WithGinBinding())

    r := gin.Default()
    r.POST("/signup", func(c *gin.Context) {
        var params SignUpParams
        if err := c.ShouldBindJSON(&params); err != nil {
            // Gin binding 返回的 err 可以直接用 StructErr 翻译
            c.JSON(400, gin.H{"error": v.StructErr(err).Error()})
            return
        }
        c.JSON(200, gin.H{"msg": "ok"})
    })
}
```

## 自定义验证

```go
// 注册自定义验证方法 + 翻译
v.SelfRegisterTranslation("checkDate", "必须要晚于当前日期", func(fl validator.FieldLevel) bool {
    date, err := time.Parse(time.DateOnly, fl.Field().String())
    if err != nil {
        return false
    }
    return date.After(time.Now())
})

// 注册结构体级验证
v.RegisterStructValidation(func(sl validator.StructLevel) {
    su := sl.Current().Interface().(OrderParams)
    if su.Password != su.RePassword {
        sl.ReportError(su.RePassword, "re_password", "RePassword", "eqfield", "Password")
    }
}, OrderParams{})

// 补充已有 tag 的翻译
v.AddValidationTranslation("required_if", "{0}为必填字段")
```

## 配置选项

| Option | 说明 | 默认 |
|--------|------|------|
| `WithLocale("zh")` | 翻译语言：`"zh"` / `"en"` | `"zh"` |
| `WithGinBinding()` | 替换 Gin 默认验证器 | 不启用 |
| `WithRequiredStructEnabled()` | 非指针 struct 启用 required | 不启用 |
| `WithPrivateFieldValidation()` | 验证未导出字段 | 不启用 |
| `WithTagNameFunc(fn)` | 自定义字段名解析 | `JSONTagName` |

内置 TagNameFunc：`verify.JSONTagName`（默认）、`verify.FormTagName`（Gin 表单）。

## v1 → v2 迁移

调用方式几乎不变，主要差异：

```go
// v1                                    → v2
verify.New()                             → verify.Init(verify.WithLocale("zh"))
verify.FieldErr("type", err)             → v.FieldErr("type", err)  // 返回 error 而非 goerr.Error
verify.StructErr(err)                    → v.StructErr(err)
verify.MapErr(result)                    → v.MapErr(result)
verify.SelfRegisterTranslation(...)      → v.SelfRegisterTranslation(...)
verify.AddValidationTranslation(...)     → v.AddValidationTranslation(...)
verify.RegisterStructValidation(...)     → v.RegisterStructValidation(...)
verify.Validate()                        → v.Validate()
verify.Trans()                           → v.Trans()
verify.RemoveTopStruct(fields)           → verify.RemoveTopStruct(fields)  // 不变
verify.RegisterTranslator(tag, msg)      → verify.RegisterTranslator(tag, msg)  // 不变
```

**核心变化：**

1. **初始化**：`verify.New()` → `verify.MustNew(opts...)` 或 `verify.Init(opts...)`
2. **返回类型**：`goerr.Error` → 标准 `error`（不再依赖 goerr）
3. **Gin 解耦**：`WithGinBinding()` 可选，非 Gin 项目不引入 Gin
4. **并发安全**：`Init()` 用 `sync.Once` 保护，运行时注册用 `sync.Mutex` 保护
5. **不再 panic**：翻译失败返回原始错误文本而非 panic

如果需要 goerr 集成，在调用层适配：

```go
if err := v.Struct(params); err != nil {
    return goerr.New(v.StructErr(err), goerr.ValidateParams)
}
```

包级函数全部保留，简单项目可以 `verify.Init()` 后直接用 `verify.StructErr(err)` 等。

## API 一览

### 验证
- `v.Struct(s)` / `v.StructCtx(ctx, s)`
- `v.Field(val, tag)` / `v.FieldCtx(ctx, val, tag)`
- `v.WithValue(f1, f2, tag)` / `v.WithValueCtx(ctx, f1, f2, tag)`
- `v.StructFiltered(s, fn)` / `v.StructFilteredCtx(ctx, s, fn)`
- `v.Map(data, rules)` / `v.MapCtx(ctx, data, rules)`

### 错误翻译
- `v.FieldErr(field, err)` → 单个字段翻译后的 error
- `v.StructErr(err)` → 结构体第一个翻译后的 error
- `v.MapErr(result)` → Map 第一个翻译后的 error
- `v.AllFieldErrors(err)` → 全部字段错误 `map[string]string`
- `v.AllMapErrors(result)` → 全部 Map 错误 `map[string]string`

### 注册
- `v.SelfRegisterTranslation(method, info, fn)` → 注册自定义验证 + 翻译
- `v.AddValidationTranslation(method, info)` → 补充已有 tag 翻译
- `v.RegisterStructValidation(fn, types...)` → 注册结构体级验证
- `verify.RegisterTranslator(tag, msg)` → 返回翻译注册函数
- `verify.Translate(trans, fe)` → 翻译函数

### 访问器
- `v.Validate()` → `*validator.Validate`
- `v.Trans()` → `ut.Translator`
- `v.Locale()` → `string`

## License

MIT
