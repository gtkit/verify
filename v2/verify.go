// Package verify wraps go-playground/validator with i18n translation,
// functional options, and optional Gin integration.
//
// Instance mode (recommended):
//
//	v := verify.MustNew(verify.WithLocale("zh"), verify.WithGinBinding())
//	err := v.Struct(params)
//	if err != nil {
//	    return v.StructErr(err) // one line → translated error
//	}
//
// Package-level mode (simple projects):
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

// Verifier is a concurrency-safe validation instance.
type Verifier struct {
	validate *validator.Validate
	trans    ut.Translator
	locale   string
	mu       sync.Mutex // protects runtime registration
}

// ---------- Options ----------

// Option configures a [Verifier].
type Option func(*config)

type config struct {
	locale                 string
	useGinBinding          bool
	requiredStructEnabled  bool
	privateFieldValidation bool
	tagNameFunc            func(reflect.StructField) string
}

// WithLocale sets the translation locale. Supported: "zh" (default), "en".
func WithLocale(locale string) Option {
	return func(c *config) { c.locale = locale }
}

// WithGinBinding replaces Gin's default validator engine with this instance.
func WithGinBinding() Option {
	return func(c *config) { c.useGinBinding = true }
}

// WithRequiredStructEnabled enables required tag on non-pointer structs.
func WithRequiredStructEnabled() Option {
	return func(c *config) { c.requiredStructEnabled = true }
}

// WithPrivateFieldValidation enables validation of unexported fields.
func WithPrivateFieldValidation() Option {
	return func(c *config) { c.privateFieldValidation = true }
}

// WithTagNameFunc sets a custom field name resolver for error messages.
// Default: [JSONTagName].
func WithTagNameFunc(fn func(reflect.StructField) string) Option {
	return func(c *config) { c.tagNameFunc = fn }
}

// ---------- Constructor ----------

// New creates a new [Verifier].
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

// MustNew is like [New] but panics on error. Use only in main/init.
func MustNew(opts ...Option) *Verifier {
	v, err := New(opts...)
	if err != nil {
		panic(err)
	}
	return v
}

// ---------- Tag Name Helpers ----------

// JSONTagName extracts the JSON tag name (default).
func JSONTagName(fld reflect.StructField) string {
	name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
	if name == "-" {
		return ""
	}
	return name
}

// FormTagName extracts the "form" tag name, useful for Gin form binding.
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

// Struct validates a struct.
func (ver *Verifier) Struct(s any) error {
	return ver.validate.Struct(s)
}

// StructCtx validates a struct with context.
func (ver *Verifier) StructCtx(ctx context.Context, s any) error {
	return ver.validate.StructCtx(ctx, s)
}

// Field validates a single variable against the given tag.
func (ver *Verifier) Field(field any, tag string) error {
	return ver.validate.Var(field, tag)
}

// FieldCtx validates a single variable with context.
func (ver *Verifier) FieldCtx(ctx context.Context, field any, tag string) error {
	return ver.validate.VarCtx(ctx, field, tag)
}

// WithValue validates field1 against field2 using the tag.
func (ver *Verifier) WithValue(field1, field2 any, tag string) error {
	return ver.validate.VarWithValue(field1, field2, tag)
}

// WithValueCtx validates field1 against field2 with context.
func (ver *Verifier) WithValueCtx(ctx context.Context, field1, field2 any, tag string) error {
	return ver.validate.VarWithValueCtx(ctx, field1, field2, tag)
}

// StructFiltered validates a struct with a filter function.
func (ver *Verifier) StructFiltered(s any, fn validator.FilterFunc) error {
	return ver.validate.StructFiltered(s, fn)
}

// StructFilteredCtx validates a struct with filter and context.
func (ver *Verifier) StructFilteredCtx(ctx context.Context, s any, fn validator.FilterFunc) error {
	return ver.validate.StructFilteredCtx(ctx, s, fn)
}

// Map validates a map against rules. Returns nil if valid, otherwise
// a map of field -> error (same as validator.ValidateMap).
func (ver *Verifier) Map(m map[string]any, rules map[string]any) map[string]any {
	return ver.validate.ValidateMap(m, rules)
}

// MapCtx validates a map with context.
func (ver *Verifier) MapCtx(ctx context.Context, m map[string]any, rules map[string]any) map[string]any {
	return ver.validate.ValidateMapCtx(ctx, m, rules)
}

// ---------- Registration ----------

// SelfRegisterTranslation registers a custom validation method with translation.
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

// AddValidationTranslation adds a translation for an existing validation tag.
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

// RegisterStructValidation registers struct-level validation.
func (ver *Verifier) RegisterStructValidation(fn validator.StructLevelFunc, types ...any) {
	ver.mu.Lock()
	defer ver.mu.Unlock()
	ver.validate.RegisterStructValidation(fn, types...)
}

// ---------- Translation Helpers ----------

// RegisterTranslator returns a [validator.RegisterTranslationsFunc] for the given tag and message.
func RegisterTranslator(tag, msg string) validator.RegisterTranslationsFunc {
	return func(trans ut.Translator) error {
		return trans.Add(tag, msg, true)
	}
}

// Translate is a [validator.TranslationFunc] that translates a field error.
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

// RemoveTopStruct strips the top-level struct name from translated field keys.
// "OrderParams.name" → "name"
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

// Validate returns the underlying *validator.Validate.
func (ver *Verifier) Validate() *validator.Validate { return ver.validate }

// Trans returns the active translator.
func (ver *Verifier) Trans() ut.Translator { return ver.trans }

// Locale returns the configured locale.
func (ver *Verifier) Locale() string { return ver.locale }
