package verify

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

func resetDefaultForTest() {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultVerifier = nil
}

func TestInitRetryAfterFailure(t *testing.T) {
	resetDefaultForTest()
	t.Cleanup(resetDefaultForTest)

	if err := Init(WithLocale("xx")); err == nil {
		t.Fatal("expected init error")
	}
	if Default() != nil {
		t.Fatal("default verifier should remain nil after failed init")
	}

	if err := Init(WithLocale("zh")); err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if Default() == nil {
		t.Fatal("default verifier should be initialized after retry")
	}
}

func TestTranslateFallsBackWithoutPanic(t *testing.T) {
	v := MustNew(WithLocale("zh"))
	if err := v.validate.RegisterValidation("custom_missing_translation", func(fl validator.FieldLevel) bool {
		return false
	}); err != nil {
		t.Fatal(err)
	}

	err := v.Field("bad", "custom_missing_translation")
	if err == nil {
		t.Fatal("expected validation error")
	}

	msg := v.FieldErr("field", err)
	if msg == nil {
		t.Fatal("expected fallback error message")
	}
	if !strings.Contains(msg.Error(), "custom_missing_translation") {
		t.Fatalf("expected fallback message to mention tag, got %q", msg.Error())
	}
}
