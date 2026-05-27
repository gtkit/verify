// Package verify 在 go-playground/validator 基础上封装了 i18n 翻译、
// functional options 和可选的 Gin 集成。
//
// 实例模式（推荐）:
//
//	v := verify.MustNew(verify.WithLocale("zh"), verify.WithGinBinding())
//	err := v.Struct(params)
//	if err != nil {
//	    return v.StructErr(err) // 一行即可得到翻译后的错误
//	}
//
// 包级模式（适用于简单项目）:
//
//	verify.Init(verify.WithLocale("zh"))
//	err := verify.Struct(params)
//	if err != nil {
//	    return verify.StructErr(err)
//	}
package verify

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
)

// Verifier 是并发安全的校验器实例。
type Verifier struct {
	validate *validator.Validate
	trans    ut.Translator
	locale   string
	mu       sync.Mutex // 保护运行时注册操作
}

// ---------- Options ----------

// Option 用于配置 [Verifier] 的函数选项。
type Option func(*config)

type config struct {
	locale                 string
	useGinBinding          bool
	requiredStructEnabled  bool
	privateFieldValidation bool
	tagNameFunc            func(reflect.StructField) string
}

// WithLocale 设置翻译使用的 locale, 支持: "zh"（默认）、"en"。
func WithLocale(locale string) Option {
	return func(c *config) { c.locale = locale }
}

// WithGinBinding 用当前实例替换 Gin 的默认校验器引擎。
func WithGinBinding() Option {
	return func(c *config) { c.useGinBinding = true }
}

// WithRequiredStructEnabled 为非指针结构体启用 required 标签。
func WithRequiredStructEnabled() Option {
	return func(c *config) { c.requiredStructEnabled = true }
}

// WithPrivateFieldValidation 启用对未导出字段的校验。
func WithPrivateFieldValidation() Option {
	return func(c *config) { c.privateFieldValidation = true }
}

// WithTagNameFunc 设置错误信息中显示字段名所使用的自定义解析函数。
// 默认: [JSONTagName]。
func WithTagNameFunc(fn func(reflect.StructField) string) Option {
	return func(c *config) { c.tagNameFunc = fn }
}

// ---------- Constructor ----------

// New 创建一个新的 [Verifier]。
func New(opts ...Option) (*Verifier, error) {
	cfg := &config{locale: "zh"}
	for _, opt := range opts {
		opt(cfg)
	}

	v := validator.New()
	v.SetTagName("binding")
	if cfg.requiredStructEnabled {
		validator.WithRequiredStructEnabled()(v)
	}
	if cfg.privateFieldValidation {
		validator.WithPrivateFieldValidation()(v)
	}

	tagFn := cfg.tagNameFunc
	if tagFn == nil {
		tagFn = JSONTagName
	}
	v.RegisterTagNameFunc(tagFn)

	trans, err := setupTranslator(cfg.locale, v)
	if err != nil {
		return nil, fmt.Errorf("verify: %w", err)
	}

	ver := &Verifier{validate: v, trans: trans, locale: cfg.locale}

	if cfg.useGinBinding {
		if err := bindToGin(v); err != nil {
			return nil, fmt.Errorf("verify: %w", err)
		}
	}

	return ver, nil
}

// MustNew 与 [New] 类似, 但出错时直接 panic。仅在 main/init 中使用。
func MustNew(opts ...Option) *Verifier {
	v, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return v
}

// ---------- Tag Name Helpers ----------

// JSONTagName 提取字段的 json tag 名称（默认实现）。
func JSONTagName(fld reflect.StructField) string {
	name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}

// FormTagName 提取字段的 form tag 名称, 适用于 Gin 表单绑定。
func FormTagName(fld reflect.StructField) string {
	name, _, _ := strings.Cut(fld.Tag.Get("form"), ",")
	if name == "-" {
		return ""
	}
	return name
}

// ---------- Translator Setup ----------

func setupTranslator(locale string, v *validator.Validate) (ut.Translator, error) {
	zhT := zh.New()
	enT := en.New()
	uni := ut.New(zhT, zhT, enT)

	trans, ok := uni.GetTranslator(locale)
	if !ok {
		return nil, fmt.Errorf("unsupported locale: %s", locale)
	}

	var err error
	switch locale {
	case "zh":
		err = zhTranslations.RegisterDefaultTranslations(v, trans)
	case "en":
		err = enTranslations.RegisterDefaultTranslations(v, trans)
	default:
		err = enTranslations.RegisterDefaultTranslations(v, trans)
	}
	if err != nil {
		return nil, fmt.Errorf("register translations for %q: %w", locale, err)
	}
	return trans, nil
}

// ---------- Validation Methods ----------

// Struct 校验一个结构体。
func (ver *Verifier) Struct(s any) error {
	return ver.validate.Struct(s)
}

// StructCtx 带 context 地校验一个结构体。
func (ver *Verifier) StructCtx(ctx context.Context, s any) error {
	return ver.validate.StructCtx(ctx, s)
}

// Field 根据给定 tag 校验单个变量。
func (ver *Verifier) Field(field any, tag string) error {
	return ver.validate.Var(field, tag)
}

// FieldCtx 带 context 地校验单个变量。
func (ver *Verifier) FieldCtx(ctx context.Context, field any, tag string) error {
	return ver.validate.VarCtx(ctx, field, tag)
}

// WithValue 根据 tag 比较 field1 与 field2。
func (ver *Verifier) WithValue(field1, field2 any, tag string) error {
	return ver.validate.VarWithValue(field1, field2, tag)
}

// WithValueCtx 带 context 地比较 field1 与 field2。
func (ver *Verifier) WithValueCtx(ctx context.Context, field1, field2 any, tag string) error {
	return ver.validate.VarWithValueCtx(ctx, field1, field2, tag)
}

// StructFiltered 使用过滤函数校验结构体。
func (ver *Verifier) StructFiltered(s any, fn validator.FilterFunc) error {
	return ver.validate.StructFiltered(s, fn)
}

// StructFilteredCtx 使用过滤函数并带 context 地校验结构体。
func (ver *Verifier) StructFilteredCtx(ctx context.Context, s any, fn validator.FilterFunc) error {
	return ver.validate.StructFilteredCtx(ctx, s, fn)
}

// Map 根据规则校验 map。校验通过时返回 nil, 否则返回
// 字段 -> 错误 的 map（与 validator.ValidateMap 行为一致）。
func (ver *Verifier) Map(m map[string]any, rules map[string]any) map[string]any {
	return ver.validate.ValidateMap(m, rules)
}

// MapCtx 带 context 地校验 map。
func (ver *Verifier) MapCtx(ctx context.Context, m map[string]any, rules map[string]any) map[string]any {
	return ver.validate.ValidateMapCtx(ctx, m, rules)
}

// ---------- Registration ----------

// SelfRegisterTranslation 注册自定义校验方法及其翻译。
//
//	v.SelfRegisterTranslation("checkDate", "必须要晚于当前日期", CheckDate)
func (ver *Verifier) SelfRegisterTranslation(method, info string, fn validator.Func) error {
	ver.mu.Lock()
	defer ver.mu.Unlock()

	if err := ver.validate.RegisterValidation(method, fn); err != nil {
		return err
	}
	return ver.addValidationTranslationLocked(method, info)
}

// AddValidationTranslation 为已有的校验 tag 追加翻译。
//
//	v.AddValidationTranslation("required_if", "{0}为必填字段")
func (ver *Verifier) AddValidationTranslation(method, info string) error {
	ver.mu.Lock()
	defer ver.mu.Unlock()

	return ver.addValidationTranslationLocked(method, info)
}

func (ver *Verifier) addValidationTranslationLocked(method, info string) error {
	return ver.validate.RegisterTranslation(
		method,
		ver.trans,
		RegisterTranslator(method, info),
		Translate,
	)
}

// RegisterStructValidation 注册结构体级别的校验方法。
func (ver *Verifier) RegisterStructValidation(fn validator.StructLevelFunc, types ...any) {
	ver.mu.Lock()
	defer ver.mu.Unlock()
	ver.validate.RegisterStructValidation(fn, types...)
}

// ---------- Translation Helpers ----------

// RegisterTranslator 根据给定的 tag 与提示信息返回一个 [validator.RegisterTranslationsFunc]。
func RegisterTranslator(tag, msg string) validator.RegisterTranslationsFunc {
	return func(trans ut.Translator) error {
		return trans.Add(tag, msg, true)
	}
}

// Translate 是一个 [validator.TranslationFunc], 用于翻译字段错误。
func Translate(trans ut.Translator, fe validator.FieldError) string {
	msg, err := trans.T(fe.Tag(), fe.Field())
	if err != nil {
		if feErr, ok := fe.(error); ok {
			return feErr.Error()
		}
		return fe.Tag()
	}
	return msg
}

// RemoveTopStruct 去掉翻译后字段 key 前面的顶层结构体名。
// 例如 "OrderParams.name" → "name"
func RemoveTopStruct(fields map[string]string) map[string]string {
	res := make(map[string]string, len(fields))
	for field, msg := range fields {
		if _, after, ok := strings.Cut(field, "."); ok {
			field = after
		}
		res[field] = msg
	}
	return res
}

func firstSortedMessage(fields map[string]string) (string, bool) {
	if len(fields) == 0 {
		return "", false
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return fields[keys[0]], true
}

// ---------- Accessors ----------

// Validate 返回底层的 *validator.Validate。
func (ver *Verifier) Validate() *validator.Validate { return ver.validate }

// Trans 返回当前生效的翻译器。
func (ver *Verifier) Trans() ut.Translator { return ver.trans }

// Locale 返回配置的 locale。
func (ver *Verifier) Locale() string { return ver.locale }
