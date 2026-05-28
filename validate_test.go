package verify

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

// 复现并防回归: len/min/max/oneof 等带参数内置 tag 必须被翻译成中文长度提示,
// 而不是退化成 "Field tag" 形式。validator/v10 的 zh 包对这类 tag 用
// customTransFunc 拼装翻译, 翻译表里并不存在直接以 tag 命名的 key,
// 故顶层翻译入口必须走 fe.Translate(trans), 不能简单 trans.T(tag, field)。
func TestTranslateBuiltinParamTags(t *testing.T) {
	resetGlobalStateForTest(t)
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	type payload struct {
		SceneID string `json:"scene_id" binding:"required,len=20"`
		Name    string `json:"name" binding:"required,min=8"`
		Title   string `json:"title" binding:"required,max=3"`
		Channel string `json:"channel" binding:"required,oneof=a b c"`
	}

	err := Struct(payload{SceneID: "abc", Name: "ab", Title: "abcd", Channel: "x"})
	if err == nil {
		t.Fatal("expected validation error")
	}

	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		t.Fatalf("expected validator.ValidationErrors, got %T", err)
	}

	want := map[string]string{
		"scene_id": "长度必须是20",
		"name":     "长度必须至少为8",
		"title":    "长度不能超过3",
		"channel":  "必须是[a b c]中的一个",
	}

	trans := Trans()
	for _, fe := range errs {
		got := translate(trans, fe)
		sub, ok := want[fe.Field()]
		if !ok {
			continue
		}
		if !strings.Contains(got, sub) {
			t.Fatalf("translate(%s/%s) = %q, want contains %q", fe.Field(), fe.Tag(), got, sub)
		}
		if strings.Contains(got, fe.Field()+" "+fe.Tag()) {
			t.Fatalf("translate(%s/%s) fell back to 'field tag': %q", fe.Field(), fe.Tag(), got)
		}
	}
}

func TestTranslateFallsBackWithoutPanic(t *testing.T) {
	resetGlobalStateForTest(t)
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

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
	if strings.Contains(msg.Error(), "Key:") || strings.Contains(msg.Error(), "payload") {
		t.Fatalf("fallback message must not leak validator namespace, got %q", msg.Error())
	}
}

func TestBindingTagIsUsed(t *testing.T) {
	resetGlobalStateForTest(t)
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	type payload struct {
		Name string `binding:"required"`
	}

	if err := Struct(payload{}); err == nil {
		t.Fatal("expected binding tag validation to run")
	}
}

func TestMapErrDeterministicOrder(t *testing.T) {
	resetGlobalStateForTest(t)
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

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

func TestTranslateFallsBackWithNilTranslator(t *testing.T) {
	resetGlobalStateForTest(t)
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	fe := mustFieldError(t, "payload.a_field", "a_field", "required")
	if got := Translate(nil, fe); got != "a_field required" {
		t.Fatalf("Translate(nil) = %q, want %q", got, "a_field required")
	}
}

func TestRemoveTopStructKeepsBareFieldName(t *testing.T) {
	got := RemoveTopStruct(map[string]string{
		"name":          "required",
		"payload.email": "email",
	})

	if got["name"] != "required" {
		t.Fatalf("expected bare field to be kept, got %#v", got)
	}
	if got["email"] != "email" {
		t.Fatalf("expected top struct to be removed, got %#v", got)
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
