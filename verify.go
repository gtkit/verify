package verify

import (
	"context"

	"github.com/go-playground/validator/v10"
)

// Field 根据 tag 校验单个字段, 校验失败时返回错误。
func Field(field any, tag string) error {
	return Validate().Var(field, tag)
}

func FieldCtx(ctx context.Context, field any, tag string) error {
	return Validate().VarCtx(ctx, field, tag)
}

// WithValue 根据 tag 比较两个字段, 校验失败时返回错误。
func WithValue(field1, field2 any, tag string) error {
	return Validate().VarWithValue(field1, field2, tag)
}

func WithValueCtx(ctx context.Context, field1, field2 any, tag string) error {
	return Validate().VarWithValueCtx(ctx, field1, field2, tag)
}

// Struct 根据结构体字段的 tag 规则进行校验。
func Struct(s any) error {
	return Validate().Struct(s)
}

func StructCtx(ctx context.Context, s any) error {
	return Validate().StructCtx(ctx, s)
}

func StructFiltered(s any, fn validator.FilterFunc) error {
	return Validate().StructFiltered(s, fn)
}

func StructFilteredCtx(ctx context.Context, s any, fn validator.FilterFunc) error {
	return Validate().StructFilteredCtx(ctx, s, fn)
}

// Map 根据规则校验 map, 返回一个 err 的 map。
func Map(m map[string]any, rules map[string]any) map[string]any {
	return Validate().ValidateMap(m, rules)
}

func MapCtx(ctx context.Context, m map[string]any, rules map[string]any) map[string]any {
	return Validate().ValidateMapCtx(ctx, m, rules)
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

func RegisterValidation(fns ...ValidationFunc) {
	for _, fn := range fns {
		v := fn()
		RegisterStructValidation(v.Func, v.Type)
	}
}

func RegisterTranslation(fns ...TranslationFunc) {
	for _, fn := range fns {
		v := fn()
		if err := SelfRegisterTranslation(v.Method, v.Info, v.Func); err != nil {
			panic(err)
		}
	}

}
