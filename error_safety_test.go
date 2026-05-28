package verify

import (
	"errors"
	"strings"
	"testing"

	"github.com/gtkit/goerr"
)

type safetyPayload struct {
	Name  string `json:"name" binding:"required,min=2"`
	Email string `json:"email" binding:"required,email"`
}

// 校验通过的字段错误经过 FieldErr 后, Message() 必须含字段名和翻译后的提示,
// 不能再退化成固定的 "字段验证错误" / "请求参数错误" 之类无信息文案。
func TestFieldErr_MessageCarriesFieldHint(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err := Field("not-a-number", "required,numeric")
	if err == nil {
		t.Fatal("expected validation error")
	}

	out := FieldErr("age", err)
	if out == nil {
		t.Fatal("expected non-nil error")
	}

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
		t.Fatalf("Message() should not be the generic fallback, got %q", msg)
	}
	if strings.Contains(msg, "ValidationErrors") {
		t.Fatalf("Message() must not leak internal type name, got %q", msg)
	}
}

// 自定义业务文案应覆盖 Message(), 但字段提示仍要进入 Error() 供日志排障。
func TestFieldErr_UserMsgOverridesMessageButKeepsErrorContext(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err := Field("not-a-number", "required,numeric")
	if err == nil {
		t.Fatal("expected validation error")
	}

	out := FieldErr("age", err, "支付参数错误")
	item, ok := goerr.AsItem(out)
	if !ok {
		t.Fatalf("expected *goerr.Item, got %T", out)
	}
	if item.Message() != "支付参数错误" {
		t.Fatalf("Message() should be user msg, got %q", item.Message())
	}
	if !strings.Contains(item.Error(), "age") {
		t.Fatalf("Error() should keep field context for logs, got %q", item.Error())
	}
}

// 非 validator.ValidationErrors 的错误必须原样透传,
// 防止把数据库错误/panic 包装成 StatusValidateParams 误导客户端,
// 也防止 Message() 暴露 "非ValidationErrors类型错误" 内部细节。
func TestFieldErr_NonValidationErrorPassthrough(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	raw := errors.New("db connection refused")
	out := FieldErr("any", raw)

	if out != raw {
		t.Fatalf("expected identity passthrough, got %v", out)
	}
	if _, ok := goerr.AsItem(out); ok {
		t.Fatalf("non-validation error must not be wrapped into *goerr.Item")
	}
}

// StructErr: Message() 应该是翻译后的字段提示, 不应是固定文案。
func TestStructErr_MessageCarriesFieldHint(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err := Struct(safetyPayload{Name: "a", Email: "bad"})
	if err == nil {
		t.Fatal("expected validation error")
	}

	out := StructErr(err)
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

// StructErr 非 validation 错误透传。
func TestStructErr_NonValidationErrorPassthrough(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	raw := errors.New("internal panic recovered")
	out := StructErr(raw)
	if out != raw {
		t.Fatalf("expected identity passthrough, got %v", out)
	}
}

// MapErr: Message() 应该形如 "key 翻译后的提示"。
func TestMapErr_MessageCarriesKeyAndHint(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	data := map[string]any{"name": "ab"}
	rules := map[string]any{"name": "required,min=8"}
	result := Map(data, rules)

	out := MapErr(result)
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

// MapErr 在 result 含非 ValidationErrors 值时不应 panic, 也不应泄露内部细节。
func TestMapErr_SkipsNonValidationEntries(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// 模拟一个错误的 result 项 (实际不会发生, 但保护代码健壮性)
	result := map[string]any{"weird": "not validation errors"}
	out := MapErr(result)
	if out != nil {
		t.Fatalf("expected nil for non-validation entries, got %v", out)
	}
}

// nil/空输入路径。
func TestErrFuncs_NilInputs(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := FieldErr("x", nil); err != nil {
		t.Fatalf("FieldErr(nil) should be nil, got %v", err)
	}
	if err := StructErr(nil); err != nil {
		t.Fatalf("StructErr(nil) should be nil, got %v", err)
	}
	if err := MapErr(nil); err != nil {
		t.Fatalf("MapErr(nil) should be nil, got %v", err)
	}
	if err := MapErr(map[string]any{}); err != nil {
		t.Fatalf("MapErr(empty) should be nil, got %v", err)
	}
}
