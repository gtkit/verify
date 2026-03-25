package verify

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestTranslateFallsBackWithoutPanic(t *testing.T) {
	New()

	if err := Validate().RegisterValidation("custom_missing_translation", func(fl validator.FieldLevel) bool {
		return false
	}); err != nil {
		t.Fatal(err)
	}

	err := Field("bad", "custom_missing_translation")
	if err == nil {
		t.Fatal("expected validation error")
	}

	msg := FieldErr("field", err)
	if msg == nil {
		t.Fatal("expected translated error")
	}
	if !strings.Contains(msg.Error(), "custom_missing_translation") {
		t.Fatalf("expected fallback message to mention tag, got %q", msg.Error())
	}
}

func TestBindingTagIsUsed(t *testing.T) {
	New()

	type payload struct {
		Name string `binding:"required"`
	}

	if err := Struct(payload{}); err == nil {
		t.Fatal("expected binding tag validation to run")
	}
}

func TestMapErrDeterministicOrder(t *testing.T) {
	New()

	result := map[string]any{
		"z_field": validator.ValidationErrors{
			mustFieldError(t, "payload.z_field", "z_field", "required"),
		},
		"a_field": validator.ValidationErrors{
			mustFieldError(t, "payload.a_field", "a_field", "required"),
		},
	}

	err := MapErr(result)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "a_field") {
		t.Fatalf("expected deterministic first key a_field, got %q", err.Error())
	}
}

func mustFieldError(t *testing.T, namespace, field, tag string) validator.FieldError {
	t.Helper()

	type payload struct {
		AField string `json:"a_field" binding:"required"`
		ZField string `json:"z_field" binding:"required"`
	}

	err := Struct(payload{})
	if err == nil {
		t.Fatal("expected validation error")
	}

	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		t.Fatalf("expected validator.ValidationErrors, got %T", err)
	}

	for _, fe := range errs {
		if fe.Namespace() == namespace && fe.Field() == field && fe.Tag() == tag {
			return fe
		}
	}
	t.Fatalf("expected field error %s/%s/%s not found", namespace, field, tag)
	return nil
}
