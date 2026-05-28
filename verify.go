package verify

import (
	"context"
	"errors"

	"github.com/go-playground/validator/v10"
)

// Field 根据 tag 校验单个字段, 校验失败时返回错误。
func Field(field any, tag string) error {
	v, err := validatorOrError()
	if err != nil {
		return err
	}
	return v.Var(field, tag)
}

func FieldCtx(ctx context.Context, field any, tag string) error {
	v, err := validatorOrError()
	if err != nil {
		return err
	}
	return v.VarCtx(ctx, field, tag)
}

// WithValue 根据 tag 比较两个字段, 校验失败时返回错误。
func WithValue(field1, field2 any, tag string) error {
	v, err := validatorOrError()
	if err != nil {
		return err
	}
	return v.VarWithValue(field1, field2, tag)
}

func WithValueCtx(ctx context.Context, field1, field2 any, tag string) error {
	v, err := validatorOrError()
	if err != nil {
		return err
	}
	return v.VarWithValueCtx(ctx, field1, field2, tag)
}

// Struct 根据结构体字段的 tag 规则进行校验。
func Struct(s any) error {
	v, err := validatorOrError()
	if err != nil {
		return err
	}
	return v.Struct(s)
}

func StructCtx(ctx context.Context, s any) error {
	v, err := validatorOrError()
	if err != nil {
		return err
	}
	return v.StructCtx(ctx, s)
}

func StructFiltered(s any, fn validator.FilterFunc) error {
	v, err := validatorOrError()
	if err != nil {
		return err
	}
	return v.StructFiltered(s, fn)
}

func StructFilteredCtx(ctx context.Context, s any, fn validator.FilterFunc) error {
	v, err := validatorOrError()
	if err != nil {
		return err
	}
	return v.StructFilteredCtx(ctx, s, fn)
}

// Map 根据规则校验 map, 返回一个 err 的 map。
func Map(m map[string]any, rules map[string]any) map[string]any {
	v := Validate()
	if v == nil {
		return nil
	}
	return v.ValidateMap(m, rules)
}

func MapCtx(ctx context.Context, m map[string]any, rules map[string]any) map[string]any {
	v := Validate()
	if v == nil {
		return nil
	}
	return v.ValidateMapCtx(ctx, m, rules)
}

type TranslationFunc func() Translation
type ValidationFunc func() Validation

type Translation struct {
	Method string
	Info   string
	Func   validator.Func
}

type Validation struct {
	Func validator.StructLevelFunc
	Type []any
}

// RegisterValidation registers struct-level validations returned by fns.
//
// Deprecated: 请使用 [New] 的 [WithStructValidation] Functional Option, 在应用
// 启动阶段一次性完成所有注册。本函数将在 v2.0.0 移除。
func RegisterValidation(fns ...ValidationFunc) {
	for _, fn := range fns {
		v := fn()
		RegisterStructValidation(v.Func, v.Type...)
	}
}

// RegisterTranslation registers field validations and translations returned by fns.
//
// Deprecated: 请使用 [New] 的 [WithTranslation] Functional Option, 在应用
// 启动阶段一次性完成所有注册。本函数将在 v2.0.0 移除。
func RegisterTranslation(fns ...TranslationFunc) error {
	errs := make([]error, 0, len(fns))
	for _, fn := range fns {
		v := fn()
		if err := SelfRegisterTranslation(v.Method, v.Info, v.Func); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
