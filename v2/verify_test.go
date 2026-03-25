package verify_test

import (
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	verify "github.com/gtkit/verify/v2"
)

func newVerifier(t *testing.T) *verify.Verifier {
	t.Helper()
	return verify.MustNew(verify.WithLocale("zh"))
}

// ---------- Field ----------

func TestField_Valid(t *testing.T) {
	v := newVerifier(t)
	if err := v.Field("12345", "required,numeric"); err != nil {
		t.Fatal(err)
	}
}

func TestFieldErr(t *testing.T) {
	v := newVerifier(t)
	err := v.Field("www.google.com", "required,numeric")
	if err == nil {
		t.Fatal("expected error")
	}

	goerr := v.FieldErr("type", err)
	if goerr == nil {
		t.Fatal("FieldErr should return non-nil")
	}
	t.Logf("FieldErr: %v", goerr)
}

func TestFieldErr_Nil(t *testing.T) {
	v := newVerifier(t)
	if err := v.FieldErr("x", nil); err != nil {
		t.Fatal("expected nil for nil input")
	}
}

// ---------- Struct ----------

type SignUpParams struct {
	Name       string `json:"name" binding:"required,min=2,max=20"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=6"`
	RePassword string `json:"re_password" binding:"required,eqfield=Password"`
	Age        uint8  `json:"age" binding:"gte=1,lte=130"`
}

func TestStruct_Valid(t *testing.T) {
	v := newVerifier(t)
	p := SignUpParams{
		Name: "alice", Email: "a@b.com",
		Password: "123456", RePassword: "123456", Age: 25,
	}
	if err := v.Struct(p); err != nil {
		t.Fatal(err)
	}
}

func TestStructErr(t *testing.T) {
	v := newVerifier(t)
	p := SignUpParams{Name: "a", Email: "bad", Password: "1", RePassword: "2", Age: 200}
	err := v.Struct(p)
	if err == nil {
		t.Fatal("expected error")
	}

	goerr := v.StructErr(err)
	if goerr == nil {
		t.Fatal("StructErr should return non-nil")
	}
	t.Logf("StructErr: %v", goerr)
}

func TestStructErr_Nil(t *testing.T) {
	v := newVerifier(t)
	if err := v.StructErr(nil); err != nil {
		t.Fatal("expected nil")
	}
}

// ---------- AllFieldErrors ----------

func TestAllFieldErrors(t *testing.T) {
	v := newVerifier(t)
	p := SignUpParams{Name: "a", Email: "bad", Password: "1", RePassword: "2", Age: 200}
	err := v.Struct(p)
	if err == nil {
		t.Fatal("expected error")
	}

	all := v.AllFieldErrors(err)
	if len(all) == 0 {
		t.Fatal("expected multiple errors")
	}
	for field, msg := range all {
		t.Logf("  %s: %s", field, msg)
	}
}

// ---------- WithValue ----------

func TestWithValue(t *testing.T) {
	v := newVerifier(t)
	if err := v.WithValue("abc", "abc", "eqfield"); err != nil {
		t.Fatal(err)
	}
	if err := v.WithValue("abc", "xyz", "eqfield"); err == nil {
		t.Fatal("expected error")
	}
}

// ---------- Map ----------

func TestMapErr(t *testing.T) {
	v := newVerifier(t)
	data := map[string]any{"name": "ab", "email": "not-email"}
	rules := map[string]any{"name": "required,min=8,max=15", "email": "required,email"}

	result := v.Map(data, rules)
	if len(result) == 0 {
		t.Fatal("expected errors")
	}

	goerr := v.MapErr(result)
	if goerr == nil {
		t.Fatal("MapErr should return non-nil")
	}
	t.Logf("MapErr: %v", goerr)
}

func TestMapErr_Valid(t *testing.T) {
	v := newVerifier(t)
	data := map[string]any{"name": "alice12345"}
	rules := map[string]any{"name": "required,min=4"}
	result := v.Map(data, rules)
	if err := v.MapErr(result); err != nil {
		t.Fatal(err)
	}
}

func TestMapErr_Nil(t *testing.T) {
	v := newVerifier(t)
	if err := v.MapErr(nil); err != nil {
		t.Fatal("expected nil")
	}
}

func TestAllMapErrors(t *testing.T) {
	v := newVerifier(t)
	data := map[string]any{"name": "ab", "email": "bad"}
	rules := map[string]any{"name": "required,min=8", "email": "required,email"}
	result := v.Map(data, rules)

	all := v.AllMapErrors(result)
	if len(all) == 0 {
		t.Fatal("expected errors")
	}
	for k, msg := range all {
		t.Logf("  %s: %s", k, msg)
	}
}

// ---------- Custom Validation ----------

type OrderParams struct {
	Name     string `json:"name" binding:"required,checkName"`
	Date     string `json:"date" binding:"required,datetime=2006-01-02,checkDate"`
	Password string `json:"password" binding:"required"`
}

func TestCustomValidation(t *testing.T) {
	v := newVerifier(t)

	// Register custom validators — same API as v1.
	if err := v.SelfRegisterTranslation("checkDate", "必须要晚于当前日期", checkDate); err != nil {
		t.Fatal(err)
	}
	if err := v.SelfRegisterTranslation("checkName", "名字格式不对", checkName); err != nil {
		t.Fatal(err)
	}

	v.RegisterStructValidation(paramValidation, OrderParams{})

	p := OrderParams{Name: "wrong", Date: "2020-01-01", Password: "123"}
	err := v.Struct(p)
	if err == nil {
		t.Fatal("expected error")
	}

	goerr := v.StructErr(err)
	t.Logf("StructErr: %v", goerr)
}

func checkName(fl validator.FieldLevel) bool { return fl.Field().String() == "roottom" }
func checkDate(fl validator.FieldLevel) bool {
	date, err := time.Parse(time.DateOnly, fl.Field().String())
	if err != nil {
		return false
	}
	return date.After(time.Now())
}
func paramValidation(sl validator.StructLevel) {
	su, ok := sl.Current().Interface().(OrderParams)
	if !ok {
		return
	}
	if su.Name == "" {
		sl.ReportError(su.Name, "name", "Name", "required", "")
	}
}

// ---------- AddValidationTranslation ----------

func TestAddValidationTranslation(t *testing.T) {
	v := newVerifier(t)
	// Add a custom translation for an existing tag.
	if err := v.AddValidationTranslation("required", "{0}不能为空哦"); err != nil {
		t.Fatal(err)
	}
	err := v.Field("", "required")
	if err == nil {
		t.Fatal("expected error")
	}
	goerr := v.FieldErr("name", err)
	t.Logf("custom translation: %v", goerr)
}

// ---------- RegisterTranslator / Translate ----------

func TestRegisterTranslator(t *testing.T) {
	// Verify RegisterTranslator returns a valid function.
	fn := verify.RegisterTranslator("test_tag", "测试消息")
	if fn == nil {
		t.Fatal("expected non-nil function")
	}
}

// ---------- RemoveTopStruct ----------

func TestRemoveTopStruct(t *testing.T) {
	input := map[string]string{
		"OrderParams.name":     "名字太短",
		"OrderParams.password": "必填",
		"bare_field":           "no dot",
	}
	result := verify.RemoveTopStruct(input)
	if result["name"] != "名字太短" {
		t.Fatalf("expected '名字太短', got %q", result["name"])
	}
	if result["password"] != "必填" {
		t.Fatalf("expected '必填', got %q", result["password"])
	}
	if result["bare_field"] != "no dot" {
		t.Fatalf("expected 'no dot', got %q", result["bare_field"])
	}
}

// ---------- Accessors ----------

func TestAccessors(t *testing.T) {
	v := newVerifier(t)
	if v.Validate() == nil {
		t.Fatal("Validate() nil")
	}
	if v.Trans() == nil {
		t.Fatal("Trans() nil")
	}
	if v.Locale() != "zh" {
		t.Fatalf("expected 'zh', got %q", v.Locale())
	}
}

// ---------- English locale ----------

func TestEnglishLocale(t *testing.T) {
	v := verify.MustNew(verify.WithLocale("en"))
	err := v.Field("", "required")
	if err == nil {
		t.Fatal("expected error")
	}
	goerr := v.FieldErr("name", err)
	t.Logf("english: %v", goerr)
}

// ---------- FormTagName ----------

func TestFormTagName(t *testing.T) {
	type P struct {
		UserName string `form:"user_name" binding:"required"`
	}
	v := verify.MustNew(verify.WithLocale("zh"), verify.WithTagNameFunc(verify.FormTagName))
	err := v.Struct(P{})
	if err == nil {
		t.Fatal("expected error")
	}
	all := v.AllFieldErrors(err)
	if _, ok := all["user_name"]; !ok {
		t.Fatalf("expected field 'user_name', got %v", all)
	}
}

// ---------- Package-level API ----------

func TestPackageLevelAPI(t *testing.T) {
	if err := verify.Init(verify.WithLocale("zh")); err != nil {
		t.Fatal(err)
	}

	err := verify.Field("bad", "required,numeric")
	if err == nil {
		t.Fatal("expected error")
	}

	goerr := verify.FieldErr("val", err)
	t.Logf("package-level FieldErr: %v", goerr)

	goerr = verify.StructErr(err)
	t.Logf("package-level StructErr: %v", goerr)

	if verify.Validate() == nil {
		t.Fatal("Validate() nil")
	}
	if verify.Trans() == nil {
		t.Fatal("Trans() nil")
	}
}

// ---------- MustNew panic ----------

func TestMustNew_BadLocale(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = verify.MustNew(verify.WithLocale("xx"))
}

// ---------- Version ----------

func TestVersion(t *testing.T) {
	if verify.Version == "" {
		t.Fatal("empty version")
	}
	t.Logf("version: %s", verify.Version)
}
