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
	if err := AllStructErr(nil); err != nil {
		t.Fatalf("AllStructErr(nil) should be nil, got %v", err)
	}
	if err := AllMapErr(nil); err != nil {
		t.Fatalf("AllMapErr(nil) should be nil, got %v", err)
	}
	if errs := StructFieldErrors(nil); errs != nil {
		t.Fatalf("StructFieldErrors(nil) should be nil, got %v", errs)
	}
}

// StructErr 统一行为测试：Message() 应包含字段名前缀。
func TestStructErr_MessageIncludesFieldName(t *testing.T) {
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

	msg := item.Message()
	// 应该包含字段名（email 或 name）
	if !strings.Contains(msg, "email") && !strings.Contains(msg, "name") {
		t.Fatalf("Message() should contain field name, got %q", msg)
	}
}

// AllStructErr 测试：返回所有字段错误聚合。
func TestAllStructErr_ReturnsAllFieldErrors(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err := Struct(safetyPayload{Name: "a", Email: "bad"})
	if err == nil {
		t.Fatal("expected validation error")
	}

	out := AllStructErr(err)
	if out == nil {
		t.Fatal("expected non-nil error")
	}

	item, ok := goerr.AsItem(out)
	if !ok {
		t.Fatalf("expected *goerr.Item, got %T", out)
	}

	msg := item.Message()
	// 应该包含两个字段错误
	if !strings.Contains(msg, "name") || !strings.Contains(msg, "email") {
		t.Fatalf("Message() should contain both field errors, got %q", msg)
	}

	// 使用 StructFieldErrors 获取结构化错误
	fieldErrs := StructFieldErrors(err)
	if len(fieldErrs) != 2 {
		t.Fatalf("expected 2 field errors, got %d", len(fieldErrs))
	}
}

// StructFieldErrors 测试：提取结构化字段错误。
func TestStructFieldErrors_ReturnsStructuredErrors(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err := Struct(safetyPayload{Name: "a", Email: "bad"})
	if err == nil {
		t.Fatal("expected validation error")
	}

	fieldErrs := StructFieldErrors(err)
	if len(fieldErrs) == 0 {
		t.Fatal("expected non-empty FieldErrors")
	}

	// 检查字段名和消息
	foundName := false
	foundEmail := false
	for _, fe := range fieldErrs {
		if fe.Field == "name" {
			foundName = true
			if fe.Message == "" {
				t.Fatal("name field error message should not be empty")
			}
		}
		if fe.Field == "email" {
			foundEmail = true
			if fe.Message == "" {
				t.Fatal("email field error message should not be empty")
			}
		}
	}
	if !foundName || !foundEmail {
		t.Fatalf("expected both name and email field errors, got %+v", fieldErrs)
	}
}

// FieldErrors 类型方法测试。
func TestFieldErrors_Methods(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	err := Struct(safetyPayload{Name: "a", Email: "bad"})
	fieldErrs := StructFieldErrors(err)

	// Test First()
	first := fieldErrs.First()
	if first == nil {
		t.Fatal("First() should not return nil for non-empty FieldErrors")
	}
	if first.Field == "" {
		t.Fatal("First().Field should not be empty")
	}

	// Test Fields()
	fields := fieldErrs.Fields()
	if len(fields) != len(fieldErrs) {
		t.Fatalf("Fields() length mismatch: got %d, want %d", len(fields), len(fieldErrs))
	}

	// Test Error()
	errMsg := fieldErrs.Error()
	if errMsg == "" {
		t.Fatal("Error() should not be empty for non-empty FieldErrors")
	}
	if !strings.Contains(errMsg, ";") && len(fieldErrs) > 1 {
		t.Fatal("Error() should contain ';' separator for multiple errors")
	}
}

// AllMapErr 测试：扁平 map 的多字段错误聚合。
//
// 断言精确输出而不是 Contains：此前实现把字段名拼成 "email.email"
// （k 与翻译后字段名相等时重复拼接），Contains("email") 照样通过，
// 缺陷因此漏网。
func TestAllMapErr_FlatMapErrors(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := Map(
		map[string]any{"name": "ab", "email": "invalid"},
		map[string]any{"name": "required,min=8", "email": "required,email"},
	)

	item, ok := goerr.AsItem(AllMapErr(result))
	if !ok {
		t.Fatalf("expected *goerr.Item, got %T", AllMapErr(result))
	}
	if got := item.Message(); got != "email email必须是一个有效的邮箱; name name长度必须至少为8个字符" {
		t.Fatalf("Message() = %q", got)
	}
}

// AllMapErr 测试：嵌套 map 的字段名按点号拼接。
//
// 此前实现把内层整段摘要塞进 Field:"user"，产出
// "user email.email ...; name.name ..."——外层前缀没有下发到内层字段。
func TestAllMapErr_NestedMapErrors(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := Map(
		map[string]any{"user": map[string]any{"name": "ab", "email": "invalid"}},
		map[string]any{"user": map[string]any{"name": "required,min=8", "email": "required,email"}},
	)

	item, ok := goerr.AsItem(AllMapErr(result))
	if !ok {
		t.Fatalf("expected *goerr.Item")
	}
	want := "user.email email必须是一个有效的邮箱; user.name name长度必须至少为8个字符"
	if got := item.Message(); got != want {
		t.Fatalf("Message() = %q, want %q", got, want)
	}
}

// AllMapErr 自定义文案覆盖 Message()，字段摘要保留在 Error() 供日志排障。
func TestAllMapErr_UserMsgOverridesMessage(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result := Map(
		map[string]any{"user": map[string]any{"email": "invalid"}},
		map[string]any{"user": map[string]any{"email": "required,email"}},
	)

	item, ok := goerr.AsItem(AllMapErr(result, "注册参数错误"))
	if !ok {
		t.Fatalf("expected *goerr.Item")
	}
	if item.Message() != "注册参数错误" {
		t.Fatalf("Message() = %q, want user msg", item.Message())
	}
	if !strings.Contains(item.Error(), "user.email") {
		t.Fatalf("Error() should keep the nested field path, got %q", item.Error())
	}
}

// AllStructErr 非 validation 错误透传。
func TestAllStructErr_NonValidationErrorPassthrough(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	raw := errors.New("internal error")
	out := AllStructErr(raw)
	if out != raw {
		t.Fatalf("expected identity passthrough, got %v", out)
	}
}

// AllMapErr 空输入。
func TestAllMapErr_EmptyInput(t *testing.T) {
	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := AllMapErr(map[string]any{}); err != nil {
		t.Fatalf("AllMapErr(empty) should be nil, got %v", err)
	}
}
