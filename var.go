package verify

import (
	"context"

	"github.com/go-playground/validator/v10"
)

// VarField validates a single field based on the tag string.
// It returns an error if the validation fails.
func Field(field any, tag string) error {
	return Validate().Var(field, tag)
}

func FieldCtx(ctx context.Context, field any, tag string) error {
	return Validate().VarCtx(ctx, field, tag)
}

// VarWithValue validates a single field based on the tag string, with a comparison value.
// It returns an error if the validation fails.
func WithValue(field1, field2 any, tag string) error {
	return Validate().VarWithValue(field1, field2, tag)
}

func WithValueCtx(ctx context.Context, field1, field2 any, tag string) error {
	return Validate().VarWithValueCtx(ctx, field1, field2, tag)
}

// VarStruct validates a struct based on the validation rules defined in the struct's tags.
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

// VarMap validates a map based on the validation rules defined in the map's tags, 返回一个 err 的 map.
func Map(m map[string]any, rules map[string]any) map[string]any {
	return Validate().ValidateMap(m, rules)
}

func MapCtx(ctx context.Context, m map[string]any, rules map[string]any) map[string]any {
	return Validate().ValidateMapCtx(ctx, m, rules)
}
