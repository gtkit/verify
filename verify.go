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

// FieldCtx 根据 tag 和 ctx 校验单个字段, 校验失败时返回错误。
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

// WithValueCtx 根据 tag、ctx 比较两个字段, 校验失败时返回错误。
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

// StructCtx 根据结构体字段的 tag 规则和 ctx 进行校验。
func StructCtx(ctx context.Context, s any) error {
	v, err := validatorOrError()
	if err != nil {
		return err
	}
	return v.StructCtx(ctx, s)
}

// StructFiltered 根据过滤函数选择结构体字段进行校验。
func StructFiltered(s any, fn validator.FilterFunc) error {
	v, err := validatorOrError()
	if err != nil {
		return err
	}
	return v.StructFiltered(s, fn)
}

// StructFilteredCtx 根据过滤函数和 ctx 选择结构体字段进行校验。
func StructFilteredCtx(ctx context.Context, s any, fn validator.FilterFunc) error {
	v, err := validatorOrError()
	if err != nil {
		return err
	}
	return v.StructFilteredCtx(ctx, s, fn)
}

// Map 根据规则校验 map, 返回一个 err 的 map。
//
// 如未先成功调用 [New], 本函数返回 nil, 不可与“无校验错误”语义区分。
// 生产代码应当先检查 [New] 的返回错误, 再使用本函数。
func Map(m map[string]any, rules map[string]any) map[string]any {
	v := Validate()
	if v == nil {
		return nil
	}
	return v.ValidateMap(m, rules)
}

// MapCtx 根据规则校验 map, 返回一个 err 的 map。
//
// 如未先成功调用 [New], 本函数返回 nil, 不可与“无校验错误”语义区分。
// 生产代码应当先检查 [New] 的返回错误, 再使用本函数。
func MapCtx(ctx context.Context, m map[string]any, rules map[string]any) map[string]any {
	v := Validate()
	if v == nil {
		return nil
	}
	return v.ValidateMapCtx(ctx, m, rules)
}

// TranslationFunc 返回一条自定义字段校验及其翻译注册。
//
// 与 [New] 的 [WithTranslation] Functional Option 并列, 适用于多模块各自
// 维护翻译表、不便集中到 main 的场景。注册必须在应用启动阶段、任何校验
// 发生前完成; validator/v10 不保证注册与校验并发安全。
type TranslationFunc func() Translation

// ValidationFunc 返回一条结构体级校验注册。
//
// 与 [New] 的 [WithStructValidation] Functional Option 并列, 适用于多模块
// 各自维护校验规则、不便集中到 main 的场景。注册必须在应用启动阶段、
// 任何校验发生前完成。
type ValidationFunc func() Validation

// Translation 描述一条字段校验及其翻译注册。
//
// 通常配合 [RegisterTranslation] 使用; 集中注册可改用 [New] 的
// [WithTranslation] Functional Option。
type Translation struct {
	Method string
	Info   string
	Func   validator.Func
}

// Validation 描述一条结构体级校验注册。
//
// 通常配合 [RegisterValidation] 使用; 集中注册可改用 [New] 的
// [WithStructValidation] Functional Option。
type Validation struct {
	Func validator.StructLevelFunc
	Type []any
}

// RegisterValidation 批量注册 fns 返回的结构体级校验。
//
// 与 [New] 的 [WithStructValidation] Functional Option 并列, 适用于多模块
// 各自维护校验规则、不便集中到 main 的场景。注册必须在应用启动阶段、
// 任何校验发生前完成。
func RegisterValidation(fns ...ValidationFunc) {
	for _, fn := range fns {
		v := fn()
		RegisterStructValidation(v.Func, v.Type...)
	}
}

// RegisterTranslation 批量注册 fns 返回的字段校验及其翻译。
//
// 与 [New] 的 [WithTranslation] Functional Option 并列, 适用于多模块各自
// 维护翻译表、不便集中到 main 的场景。注册必须在应用启动阶段、任何校验
// 发生前完成。多个 fn 注册失败的错误通过 [errors.Join] 聚合返回。
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
