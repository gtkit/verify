# Changelog

遵循 [Keep a Changelog 1.1.0](https://keepachangelog.com/zh-CN/1.1.0/) 和 [Semantic Versioning](https://semver.org/lang/zh-CN/)。

## [Unreleased]

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

### Removed
- 移除导入阶段自动初始化行为, 统一由调用方在启动阶段显式调用 `New()`。

### Fixed
- `Translate` fallback 不再触发 validator 默认 `fe.Error()` 的英文 `Key: '...' Error:` 格式输出。
- `RemoveTopStruct` 对不含 `.` 的字段名也能正确返回。

### Security
- 错误消息不再暴露内部 namespace 或结构体名, 对外只展示字段名和翻译后的提示。
