package verify

import (
	"context"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestContextValidationHelpers(t *testing.T) {
	resetGlobalStateForTest(t)
	ctx := context.Background()

	if err := FieldCtx(ctx, "123", "required,numeric"); err != nil {
		t.Fatalf("FieldCtx() unexpected error = %v", err)
	}
	if err := FieldCtx(ctx, "bad", "numeric"); err == nil {
		t.Fatal("FieldCtx() expected validation error")
	}

	if err := StructCtx(ctx, safetyPayload{Name: "ok", Email: "ok@example.com"}); err != nil {
		t.Fatalf("StructCtx() unexpected error = %v", err)
	}
	if err := StructCtx(ctx, safetyPayload{Name: "x", Email: "bad"}); err == nil {
		t.Fatal("StructCtx() expected validation error")
	}
}

func TestWithValueHelpers(t *testing.T) {
	resetGlobalStateForTest(t)
	ctx := context.Background()

	if err := WithValue("same", "same", "eqfield"); err != nil {
		t.Fatalf("WithValue() unexpected error = %v", err)
	}
	if err := WithValue("left", "right", "eqfield"); err == nil {
		t.Fatal("WithValue() expected validation error")
	}
	if err := WithValueCtx(ctx, "same", "same", "eqfield"); err != nil {
		t.Fatalf("WithValueCtx() unexpected error = %v", err)
	}
	if err := WithValueCtx(ctx, "left", "right", "eqfield"); err == nil {
		t.Fatal("WithValueCtx() expected validation error")
	}
}

func TestStructFilteredHelpers(t *testing.T) {
	resetGlobalStateForTest(t)

	type payload struct {
		Name  string `json:"name" binding:"required"`
		Email string `json:"email" binding:"required,email"`
	}
	filterEmail := func(ns []byte) bool {
		return strings.Contains(string(ns), ".Email")
	}

	if err := StructFiltered(payload{}, filterEmail); err == nil {
		t.Fatal("StructFiltered() expected email validation error")
	}
	if err := StructFiltered(payload{Name: ""}, func(ns []byte) bool {
		return strings.Contains(string(ns), ".Name")
	}); err == nil {
		t.Fatal("StructFiltered() expected name validation error")
	}
	if err := StructFilteredCtx(context.Background(), payload{}, filterEmail); err == nil {
		t.Fatal("StructFilteredCtx() expected email validation error")
	}
}

func TestMapCtx(t *testing.T) {
	resetGlobalStateForTest(t)

	result := MapCtx(context.Background(), map[string]any{"name": "ab"}, map[string]any{"name": "min=4"})
	if len(result) == 0 {
		t.Fatal("MapCtx() expected validation errors")
	}
	if err := MapErr(result); err == nil {
		t.Fatal("MapErr(MapCtx()) expected translated error")
	}
}

func TestRegistrationHelpers(t *testing.T) {
	resetGlobalStateForTest(t)

	type payload struct {
		Name string `json:"name" binding:"bulk_check"`
	}

	if err := RegisterTranslation(func() Translation {
		return Translation{
			Method: "bulk_check",
			Info:   "{0}批量校验失败",
			Func: func(fl validator.FieldLevel) bool {
				return fl.Field().String() == "ok"
			},
		}
	}); err != nil {
		t.Fatalf("RegisterTranslation() error = %v", err)
	}

	err := Struct(payload{Name: "bad"})
	if err == nil {
		t.Fatal("Struct() expected bulk_check validation error")
	}
	out := StructErr(err)
	if out == nil {
		t.Fatal("StructErr() expected translated error")
	}
	if !strings.Contains(out.Error(), "name批量校验失败") {
		t.Fatalf("expected bulk registration translation, got %q", out.Error())
	}
}

func TestRegisterTranslationReturnsErrorWithoutPanic(t *testing.T) {
	resetGlobalStateForTest(t)

	err := RegisterTranslation(func() Translation {
		return Translation{
			Method: "",
			Info:   "{0}无效校验失败",
			Func: func(fl validator.FieldLevel) bool {
				return true
			},
		}
	}, func() Translation {
		return Translation{
			Method: "invalid|tag",
			Info:   "{0}无效校验失败",
			Func: func(fl validator.FieldLevel) bool {
				return true
			},
		}
	})

	if err == nil {
		t.Fatal("RegisterTranslation() expected registration error")
	}
	if got := err.Error(); !strings.Contains(got, "register validation") || !strings.Contains(got, "invalid|tag") {
		t.Fatalf("RegisterTranslation() error should include failed registration context, got %q", got)
	}
}

func TestRegisterStructValidationHelper(t *testing.T) {
	resetGlobalStateForTest(t)

	type payload struct {
		Password string `json:"password"`
		Confirm  string `json:"confirm"`
	}

	if err := AddValidationTranslation("password_match", "{0}必须和密码一致"); err != nil {
		t.Fatalf("AddValidationTranslation() error = %v", err)
	}
	RegisterValidation(func() Validation {
		return Validation{
			Type: []any{payload{}},
			Func: func(sl validator.StructLevel) {
				p := sl.Current().Interface().(payload)
				if p.Password != p.Confirm {
					sl.ReportError(p.Confirm, "confirm", "Confirm", "password_match", "")
				}
			},
		}
	})

	err := Struct(payload{Password: "a", Confirm: "b"})
	if err == nil {
		t.Fatal("Struct() expected struct-level validation error")
	}
	out := StructErr(err)
	if out == nil {
		t.Fatal("StructErr() expected translated error")
	}
	if !strings.Contains(out.Error(), "confirm必须和密码一致") {
		t.Fatalf("expected struct-level translation, got %q", out.Error())
	}
}

func TestNewAppliesStartupOptions(t *testing.T) {
	resetGlobalStateForTest(t)

	type payload struct {
		Name    string `json:"name" binding:"startup_check"`
		Confirm string `json:"confirm"`
	}

	err := New(
		WithTranslation("startup_check", "{0}启动期校验失败", func(fl validator.FieldLevel) bool {
			return fl.Field().String() == "ok"
		}),
		WithValidationTranslation("confirm_match", "{0}必须和 name 一致"),
		WithStructValidation(func(sl validator.StructLevel) {
			p := sl.Current().Interface().(payload)
			if p.Name != p.Confirm {
				sl.ReportError(p.Confirm, "confirm", "Confirm", "confirm_match", "")
			}
		}, payload{}),
	)
	if err != nil {
		t.Fatalf("New(options) error = %v", err)
	}

	err = Struct(payload{Name: "bad", Confirm: "other"})
	if err == nil {
		t.Fatal("Struct() expected startup option validation error")
	}
	out := StructErr(err)
	if out == nil {
		t.Fatal("StructErr() expected translated startup option error")
	}
	got := out.Error()
	if !strings.Contains(got, "name启动期校验失败") && !strings.Contains(got, "confirm必须和 name 一致") {
		t.Fatalf("expected startup option translation, got %q", got)
	}
}

func TestNewReturnsStartupOptionError(t *testing.T) {
	resetGlobalStateForTest(t)

	err := New(WithTranslation("invalid|startup", "{0}启动期校验失败", func(fl validator.FieldLevel) bool {
		return true
	}))
	if err == nil {
		t.Fatal("New(WithTranslation(invalid tag)) expected error")
	}
	if got := err.Error(); !strings.Contains(got, "register validation") || !strings.Contains(got, "invalid|startup") {
		t.Fatalf("New option error should include failed registration context, got %q", got)
	}
}

func TestNewAppliesValidatorConfigOptions(t *testing.T) {
	resetGlobalStateForTest(t)

	type nested struct {
		Value string `json:"value" binding:"required"`
	}
	type payload struct {
		Nested nested `json:"nested" binding:"required"`
	}

	if err := New(EnableRequiredStructValidation()); err != nil {
		t.Fatalf("New(EnableRequiredStructValidation()) error = %v", err)
	}
	if err := Struct(payload{}); err == nil {
		t.Fatal("Struct() expected nested required struct validation error")
	}
}
