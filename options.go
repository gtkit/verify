package verify

import (
	"fmt"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
)

// Option configures Gin's shared validator during [New].
//
// Options must be applied during application startup, before any Gin
// ShouldBind* or verify validation call. validator/v10 registration methods are
// not safe to run concurrently with validation.
type Option interface {
	apply(*validator.Validate, ut.Translator) error
}

type optionFunc func(*validator.Validate, ut.Translator) error

func (fn optionFunc) apply(v *validator.Validate, trans ut.Translator) error {
	return fn(v, trans)
}

// WithTranslation registers a custom field validation and its translation.
//
// Use it only as a [New] option during application startup.
func WithTranslation(method, info string, fn validator.Func) Option {
	return optionFunc(func(v *validator.Validate, trans ut.Translator) error {
		return registerValidationAndTranslationLocked(v, trans, method, info, fn)
	})
}

// WithValidationTranslation registers a translation for an existing validation tag.
//
// Use it only as a [New] option during application startup.
func WithValidationTranslation(method, info string) Option {
	return optionFunc(func(v *validator.Validate, trans ut.Translator) error {
		if err := addValidationTranslationLocked(v, trans, method, info); err != nil {
			return fmt.Errorf("register translation %q: %w", method, err)
		}
		return nil
	})
}

// WithStructValidation registers a struct-level validation function.
//
// Use it only as a [New] option during application startup.
func WithStructValidation(fn validator.StructLevelFunc, types ...any) Option {
	return optionFunc(func(v *validator.Validate, _ ut.Translator) error {
		v.RegisterStructValidation(fn, types...)
		return nil
	})
}

// EnableRequiredStructValidation enables validator's required struct option.
//
// Use it only as a [New] option during application startup.
func EnableRequiredStructValidation() Option {
	return optionFunc(func(v *validator.Validate, _ ut.Translator) error {
		validator.WithRequiredStructEnabled()(v)
		return nil
	})
}

// EnablePrivateFieldValidation enables validation of unexported fields.
//
// Use it only as a [New] option during application startup.
func EnablePrivateFieldValidation() Option {
	return optionFunc(func(v *validator.Validate, _ ut.Translator) error {
		validator.WithPrivateFieldValidation()(v)
		return nil
	})
}
