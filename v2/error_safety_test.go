package verify_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/gtkit/goerr"
	verify "github.com/gtkit/verify/v2"
)

type safetyPayload struct {
	Name  string `json:"name" binding:"required,min=2"`
	Email string `json:"email" binding:"required,email"`
}

func newSafetyVerifier(t *testing.T) *verify.Verifier {
	t.Helper()
	return verify.MustNew(verify.WithLocale("zh"))
}

func TestFieldErr_MessageCarriesFieldHint(t *testing.T) {
	v := newSafetyVerifier(t)

	err := v.Field("not-a-number", "required,numeric")
	if err == nil {
		t.Fatal("expected validation error")
	}

	out := v.FieldErr("age", err)
	item, ok := goerr.AsItem(out)
	if !ok {
		t.Fatalf("expected *goerr.Item, got %T", out)
	}

	if item.Code() != goerr.StatusValidateParams().Code() {
		t.Fatalf("unexpected status code: %d", item.Code())
	}

	msg := item.Message()
	if !strings.Contains(msg, "age") {
		t.Fatalf("Message() should contain field name 'age', got %q", msg)
	}
	if strings.Contains(msg, "字段验证错误") || strings.Contains(msg, "请求参数错误") {
		t.Fatalf("Message() should be the translated hint, got %q", msg)
	}
	if strings.Contains(msg, "ValidationErrors") {
		t.Fatalf("Message() must not leak internal type name, got %q", msg)
	}
}

func TestFieldErr_NonValidationErrorPassthrough(t *testing.T) {
	v := newSafetyVerifier(t)

	raw := errors.New("db connection refused")
	out := v.FieldErr("any", raw)

	if out != raw {
		t.Fatalf("expected identity passthrough, got %v", out)
	}
	if _, ok := goerr.AsItem(out); ok {
		t.Fatalf("non-validation error must not be wrapped into *goerr.Item")
	}
}

func TestStructErr_MessageCarriesFieldHint(t *testing.T) {
	v := newSafetyVerifier(t)

	err := v.Struct(safetyPayload{Name: "a", Email: "bad"})
	if err == nil {
		t.Fatal("expected validation error")
	}

	out := v.StructErr(err)
	item, ok := goerr.AsItem(out)
	if !ok {
		t.Fatalf("expected *goerr.Item, got %T", out)
	}
	if item.Code() != goerr.StatusValidateParams().Code() {
		t.Fatalf("unexpected status code: %d", item.Code())
	}
	msg := item.Message()
	if strings.Contains(msg, "结构验证错误") || strings.Contains(msg, "请求参数错误") {
		t.Fatalf("Message() should be the translated hint, got %q", msg)
	}
	if strings.Contains(msg, "ValidationErrors") {
		t.Fatalf("Message() must not leak internal type name, got %q", msg)
	}
	if msg == "" {
		t.Fatal("Message() should not be empty")
	}
}

func TestStructErr_NonValidationErrorPassthrough(t *testing.T) {
	v := newSafetyVerifier(t)
	raw := errors.New("internal panic recovered")
	out := v.StructErr(raw)
	if out != raw {
		t.Fatalf("expected identity passthrough, got %v", out)
	}
}

func TestMapErr_MessageCarriesKeyAndHint(t *testing.T) {
	v := newSafetyVerifier(t)

	data := map[string]any{"name": "ab"}
	rules := map[string]any{"name": "required,min=8"}
	result := v.Map(data, rules)

	out := v.MapErr(result)
	item, ok := goerr.AsItem(out)
	if !ok {
		t.Fatalf("expected *goerr.Item, got %T", out)
	}
	if item.Code() != goerr.StatusValidateParams().Code() {
		t.Fatalf("unexpected status code: %d", item.Code())
	}
	msg := item.Message()
	if !strings.Contains(msg, "name") {
		t.Fatalf("Message() should contain key 'name', got %q", msg)
	}
	if strings.Contains(msg, "映射验证错误") || strings.Contains(msg, "请求参数错误") {
		t.Fatalf("Message() should be the translated hint, got %q", msg)
	}
}

func TestMapErr_SkipsNonValidationEntries(t *testing.T) {
	v := newSafetyVerifier(t)

	result := map[string]any{
		"internal": errors.New("db password leaked"),
	}
	out := v.MapErr(result)
	if out != nil {
		t.Fatalf("expected nil for non-validation entries, got %v", out)
	}
}

func TestErrFuncs_NilInputs(t *testing.T) {
	v := newSafetyVerifier(t)
	if err := v.FieldErr("x", nil); err != nil {
		t.Fatalf("FieldErr(nil) should be nil, got %v", err)
	}
	if err := v.StructErr(nil); err != nil {
		t.Fatalf("StructErr(nil) should be nil, got %v", err)
	}
	if err := v.MapErr(nil); err != nil {
		t.Fatalf("MapErr(nil) should be nil, got %v", err)
	}
	if err := v.MapErr(map[string]any{}); err != nil {
		t.Fatalf("MapErr(empty) should be nil, got %v", err)
	}
}

// 确认无论字段提示如何, 客户端拿到的 Message() 不会含原始字段值,
// 避免回显用户输入造成 XSS / PII 泄露。
func TestStructErr_DoesNotEchoFieldValue(t *testing.T) {
	v := newSafetyVerifier(t)
	secret := "very-secret-token-12345"
	err := v.Struct(safetyPayload{Name: secret, Email: "not-email-" + secret})
	if err == nil {
		t.Fatal("expected validation error")
	}
	out := v.StructErr(err)
	item, ok := goerr.AsItem(out)
	if !ok {
		t.Fatalf("expected *goerr.Item, got %T", out)
	}
	if strings.Contains(item.Message(), secret) {
		t.Fatalf("Message() must not echo user input value, got %q", item.Message())
	}
}
