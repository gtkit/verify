# Changelog

遵循 [Keep a Changelog 1.1.0](https://keepachangelog.com/zh-CN/1.1.0/) 和 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [Unreleased]

### Fixed
- ⚠ 修复 `AllMapErr` 的字段路径错误。该 API 自 v1.3.0 引入，缺陷随 v1.3.0 / v1.3.1 一并发布。两处症状：
  - 扁平 map 下字段名被拼成 `email.email`——`k` 与 `translateValidationErrors` 返回的字段名本就是同一个名字，旧实现仍做 `k + "." + field` 拼接。
  - 嵌套 map 下外层前缀没有下发到内层字段，整段内层摘要被塞进单个 `FieldError{Field: "user"}`，产出 `"user email.email ...; name.name ..."` 而不是 `"user.email ...; user.name ..."`。
- 根因是旧实现从 `goerr.AsItem(nested).Message()` 反取内层结果，等于把已渲染的字符串当结构化数据用。现改为内部递归 helper `mapFieldErrors(err, prefix)` 直接返回 `FieldErrors`，每层按键名排序，`AllMapErr` 只负责渲染与包装。
- 修复后 `AllMapErr(Map({"user":{"name":"ab","email":"invalid"}}, {"user":{"name":"required,min=8","email":"required,email"}}))` 的 `Message()` 为 `user.email email必须是一个有效的邮箱; user.name name长度必须至少为8个字符`。
- 原有测试只断言 `strings.Contains(msg, "name")`，对上述两种错误输出同样通过，因此未能拦截。现改为精确断言完整字符串，并补上真正嵌套的用例与自定义文案覆盖用例。
- README 中 `AllMapErr` 的示例输出与实测不符（顺序与文案均错），已改为实测值。

## [1.3.1] - 2026-11-09

### Removed
- ⚠ **破坏性变更**：移除 4 个自 v1.2.0 起标记 deprecated 的导出 helper —— `Translate`、`RegisterTranslator`、`RemoveTopStruct`、`GetMapError`。按 SemVer 这属于 major 变更，本包不发布 v2.0.0，故在 v1.x 内直接移除；升级时按下表改写调用点。
- 对应的内部实现 `translate`、`registerTranslator`、`removeTopStruct`、`getMapError` 保留为包内私有函数，`FieldErr`/`StructErr`/`AllStructErr`/`MapErr`/`AllMapErr` 的行为与消息文本完全不变。
- 移除源码与文档中所有 "将在 v2.0.0 移除" 的表述；v1.2.2 条目里"维持 deprecated"的记录作为历史保留。

#### 迁移对照

| 移除的调用 | 改写为 |
|-----------|-------|
| `verify.RemoveTopStruct(errs.Translate(verify.Trans()))` 后遍历消息 | `verify.StructFieldErrors(errs)`，遍历 `FieldErrors` 的 `Field` / `Message` |
| 同上，只需要第一条消息 | `verify.StructErr(errs)`（返回 `*goerr.Item`，`Message()` 即字段名 + 中文提示） |
| `verify.GetMapError(fields)` | `verify.MapErr(result)` 或 `verify.AllMapErr(result)` |
| `verify.Translate(trans, fe)` | `verify.FieldErr(field, err)` / `StructErr` / `AllStructErr` |
| `verify.RegisterTranslator(tag, msg)` | `verify.New(verify.WithTranslation(...))` 或 `verify.WithValidationTranslation(tag, msg)` |

## [1.3.0] - 2026-11-09

### Added
- 新增 `FieldError` 结构体和 `FieldErrors` 类型，支持结构化多字段错误返回。
- 新增 `AllStructErr(err error, msg ...string) error`，返回包含所有字段错误的聚合错误。
- 新增 `StructFieldErrors(err error) FieldErrors`，从结构体校验错误中提取所有字段错误详情。
- 新增 `AllMapErr(err map[string]any, msg ...string) error`，支持嵌套 map 的错误聚合。
- `FieldErrors` 类型提供 `Error()`、`First()`、`Fields()` 方法，便于构建详细错误响应。

### Changed
- `StructErr` 行为统一：现在返回的 Message() 包含字段名前缀（如 "email 必须是一个有效的邮箱"），与 `FieldErr` 保持一致。
- 内部优化 `firstSortedMessage`，新增 `firstSortedFieldMessage` 同时返回字段名和消息，避免重复排序。

## [1.2.5] - 2026-09-17

### Changed
- 依赖 `github.com/gtkit/goerr` 升级到 v1.4.0。该版本删除了 `goerr.Err`, 本包内部错误构造改用标准库 `errors.New`; `FieldErr`/`StructErr`/`MapErr` 的返回类型、业务状态码与 `Error()`/`Message()` 文本均不变。
- 依赖 `github.com/go-playground/validator/v10` 升级到 v10.30.4。
- `RegisterTagNameFunc` 内解析 json tag 改用 `strings.Cut`, 解析结果不变。

## [1.2.2] - 2026-05-28

### Changed
- 撤回 `RegisterTranslation`、`SelfRegisterTranslation`、`AddValidationTranslation`、`RegisterValidation`、`RegisterStructValidation`、`WithRequiredStructEnabled`、`WithPrivateFieldValidation` 以及 `TranslationFunc`/`ValidationFunc`/`Translation`/`Validation` 类型的 deprecation 标记, 它们与 `New(opts...)` Functional Options 并列, 为多模块各自维护校验/翻译、不便集中到 main 的场景提供散注册路径。v2.0.0 不再计划移除。
- 内部错误消息 helper `Translate`、`RegisterTranslator`、`RemoveTopStruct`、`GetMapError` 维持 deprecated, 仍建议改用 `FieldErr`/`StructErr`/`MapErr`。

## [1.2.1] - 2026-05-28

### Fixed
- 修复 `len`、`min`、`max`、`oneof`、`eq`、`ne`、`gt`、`lt`、`gte`、`lte` 等带参数内置 tag 翻译失败、退化成 `字段 tag`(如 `SceneID len`)的问题; 翻译路径改用 `validator.FieldError.Translate`, 正确触发 zh 包注册的 `customTransFunc`。

## [1.2.0] - 2026-05-28

### Added
- 新增 `New(opts ...Option)` 启动期 Functional Options API, 支持在 Gin 开始校验前注册自定义字段校验、翻译、结构体级校验和 validator 配置。
- 新增 `WithTranslation`、`WithValidationTranslation`、`WithStructValidation`、`EnableRequiredStructValidation`、`EnablePrivateFieldValidation` 五个 Option。
- 新增 `LICENSE`、包级 GoDoc 文档和可验证 Example 测试。
- 新增 `BenchmarkStructErr` 和 `BenchmarkFieldErr`, 用于观察错误翻译路径的分配情况。

### Changed
- `FieldErr`、`StructErr`、`MapErr` 现在严格区分翻译失败 fallback, 不再泄露 validator 内部 key 或 namespace。
- `regMu` 明确定位为只串行化 verify 包内注册调用, 不提供注册与 Gin/validator 校验并发执行时的安全保证。
- README 和 GoDoc 明确注册类 API 仅应在应用启动阶段调用, 并推荐通过 `New(opts...)` 完成。

### Deprecated
- 废弃 `SelfRegisterTranslation`、`AddValidationTranslation`、`RegisterStructValidation`、`RegisterValidation`、`RegisterTranslation`、`WithRequiredStructEnabled`、`WithPrivateFieldValidation`, 请改用 `New` 的 Functional Options。
- 废弃 `Translate`、`RegisterTranslator`、`RemoveTopStruct`、`GetMapError` 4 个内部 helper 的对外导出, 请改用 `FieldErr`、`StructErr`、`MapErr` 高层错误处理函数。

### Removed
- 移除导入阶段自动初始化行为, 统一由调用方在启动阶段显式调用 `New()`。

### Fixed
- `Translate` fallback 不再触发 validator 默认 `fe.Error()` 的英文 `Key: '...' Error:` 格式输出。
- `RemoveTopStruct` 对不含 `.` 的字段名也能正确返回。

### Security
- 错误消息不再暴露内部 namespace 或结构体名, 对外只展示字段名和翻译后的提示。
