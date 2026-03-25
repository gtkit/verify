package verify

import (
	"context"
	"sync"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

var (
	defaultVerifier *Verifier
	defaultMu       sync.RWMutex
)

// Init initializes the package-level default [Verifier].
// Safe to call concurrently. Successful initialization is sticky;
// failed initialization can be retried.
func Init(opts ...Option) error {
	defaultMu.RLock()
	if defaultVerifier != nil {
		defaultMu.RUnlock()
		return nil
	}
	defaultMu.RUnlock()

	v, err := New(opts...)
	if err != nil {
		return err
	}

	defaultMu.Lock()
	defer defaultMu.Unlock()
	if defaultVerifier == nil {
		defaultVerifier = v
	}
	return nil
}

func mustDefault() *Verifier {
	if v := Default(); v != nil {
		return v
	}
	panic("verify: not initialized — call verify.Init() first")
}

// Default returns the package-level [Verifier], or nil if not initialized.
func Default() *Verifier {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultVerifier
}

// ---------- Validation ----------

func Struct(s any) error                                  { return mustDefault().Struct(s) }
func StructCtx(ctx context.Context, s any) error          { return mustDefault().StructCtx(ctx, s) }
func Field(field any, tag string) error                   { return mustDefault().Field(field, tag) }
func FieldCtx(ctx context.Context, f any, t string) error { return mustDefault().FieldCtx(ctx, f, t) }
func WithValue(f1, f2 any, tag string) error              { return mustDefault().WithValue(f1, f2, tag) }
func WithValueCtx(ctx context.Context, f1, f2 any, tag string) error {
	return mustDefault().WithValueCtx(ctx, f1, f2, tag)
}
func StructFiltered(s any, fn validator.FilterFunc) error {
	return mustDefault().StructFiltered(s, fn)
}
func StructFilteredCtx(ctx context.Context, s any, fn validator.FilterFunc) error {
	return mustDefault().StructFilteredCtx(ctx, s, fn)
}
func Map(m map[string]any, rules map[string]any) map[string]any {
	return mustDefault().Map(m, rules)
}
func MapCtx(ctx context.Context, m map[string]any, rules map[string]any) map[string]any {
	return mustDefault().MapCtx(ctx, m, rules)
}

// ---------- Error helpers ----------

func FieldErr(field string, err error) error     { return mustDefault().FieldErr(field, err) }
func StructErr(err error) error                  { return mustDefault().StructErr(err) }
func MapErr(result map[string]any) error         { return mustDefault().MapErr(result) }
func AllFieldErrors(err error) map[string]string { return mustDefault().AllFieldErrors(err) }
func AllMapErrors(result map[string]any) map[string]string {
	return mustDefault().AllMapErrors(result)
}

// ---------- Registration ----------

func SelfRegisterTranslation(method, info string, fn validator.Func) error {
	return mustDefault().SelfRegisterTranslation(method, info, fn)
}
func AddValidationTranslation(method, info string) error {
	return mustDefault().AddValidationTranslation(method, info)
}
func RegisterStructValidation(fn validator.StructLevelFunc, types ...any) {
	mustDefault().RegisterStructValidation(fn, types...)
}

// ---------- Accessors ----------

func Validate() *validator.Validate { return mustDefault().Validate() }
func Trans() ut.Translator          { return mustDefault().Trans() }
