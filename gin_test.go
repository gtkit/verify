package verify

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

func TestImportDoesNotInitializeGinValidator(t *testing.T) {
	tmp := t.TempDir()
	mainFile := filepath.Join(tmp, "main.go")
	src := `package main

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin/binding"
	_ "github.com/gtkit/verify"
)

type payload struct {
	Name string ` + "`json:\"name\" binding:\"required\"`" + `
}

func main() {
	err := binding.Validator.ValidateStruct(payload{})
	if err == nil {
		panic("expected validation error")
	}
	if strings.Contains(err.Error(), "'name'") {
		panic(fmt.Sprintf("verify import initialized gin validator: %s", err))
	}
}
`
	if err := os.WriteFile(mainFile, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", mainFile)
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run import check failed: %v\n%s", err, out)
	}
}

func TestNewReturnsErrorForUnsupportedGinValidatorEngine(t *testing.T) {
	oldValidator := binding.Validator
	binding.Validator = unsupportedGinValidator{}
	defer func() {
		binding.Validator = oldValidator
		resetGlobalStateForTest(t)
	}()
	resetGlobalStateForTest(t)

	if err := New(); err == nil {
		t.Fatal("New() expected unsupported engine error")
	}
	if err := Field("value", "required"); err == nil {
		t.Fatal("Field() expected initialization error")
	}
	if err := Struct(struct{}{}); err == nil {
		t.Fatal("Struct() expected initialization error")
	}
	if got := Map(map[string]any{"name": ""}, map[string]any{"name": "required"}); got != nil {
		t.Fatalf("Map() expected nil on initialization error, got %#v", got)
	}
}

func TestValidateUsesGinBindingValidatorByDefault(t *testing.T) {
	resetGlobalStateForTest(t)

	ginValidator, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		t.Fatal("expected gin binding validator engine to be *validator.Validate")
	}
	if Validate() != ginValidator {
		t.Fatal("verify should use gin binding validator by default")
	}

	type payload struct {
		Email string `json:"email" binding:"required,email"`
	}
	err := binding.Validator.ValidateStruct(payload{Email: "bad"})
	if err == nil {
		t.Fatal("expected gin binding validation error")
	}

	out := StructErr(err)
	if out == nil {
		t.Fatal("expected translated error")
	}
	if !strings.Contains(out.Error(), "email") {
		t.Fatalf("expected translated error to contain json field name, got %q", out.Error())
	}
}

func TestSelfRegisterTranslationAppliesToGinBindingValidator(t *testing.T) {
	resetGlobalStateForTest(t)

	if err := SelfRegisterTranslation("not_bad", "{0}不能是bad", func(fl validator.FieldLevel) bool {
		return fl.Field().String() != "bad"
	}); err != nil {
		t.Fatalf("SelfRegisterTranslation() error = %v", err)
	}

	type payload struct {
		Name string `json:"name" binding:"not_bad"`
	}
	err := binding.Validator.ValidateStruct(payload{Name: "bad"})
	if err == nil {
		t.Fatal("expected gin binding validation error")
	}

	out := StructErr(err)
	if out == nil {
		t.Fatal("expected translated error")
	}
	if !strings.Contains(out.Error(), "name不能是bad") {
		t.Fatalf("expected custom translated gin binding error, got %q", out.Error())
	}
}

func TestGinShouldBindJSONUsesVerifyTranslations(t *testing.T) {
	resetGlobalStateForTest(t)

	if err := SelfRegisterTranslation("not_bad_json", "{0}不能是bad", func(fl validator.FieldLevel) bool {
		return fl.Field().String() != "bad"
	}); err != nil {
		t.Fatalf("SelfRegisterTranslation() error = %v", err)
	}

	type payload struct {
		Name string `json:"name" binding:"required,not_bad_json"`
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	var boundErr error
	router.POST("/payload", func(c *gin.Context) {
		var req payload
		boundErr = c.ShouldBindJSON(&req)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/payload", strings.NewReader(`{"name":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if boundErr == nil {
		t.Fatal("expected ShouldBindJSON validation error")
	}
	out := StructErr(boundErr)
	if out == nil {
		t.Fatal("expected translated error")
	}
	if !strings.Contains(out.Error(), "name不能是bad") {
		t.Fatalf("expected custom translated ShouldBindJSON error, got %q", out.Error())
	}
}

func TestStructErrAfterFirstGinBindStillUsesJSONFieldName(t *testing.T) {
	resetGlobalStateForTest(t)

	type payload struct {
		Name string `json:"name" binding:"required"`
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	var boundErr error
	router.POST("/payload", func(c *gin.Context) {
		var req payload
		boundErr = c.ShouldBindJSON(&req)
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/payload", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if boundErr == nil {
		t.Fatal("expected ShouldBindJSON validation error")
	}
	out := StructErr(boundErr)
	if out == nil {
		t.Fatal("expected translated error")
	}
	if !strings.Contains(out.Error(), "name") {
		t.Fatalf("expected json field name after first Gin bind, got %q", out.Error())
	}
	if strings.Contains(out.Error(), "Name") {
		t.Fatalf("expected translated error not to use struct field name after first Gin bind, got %q", out.Error())
	}
}

func TestValidateConcurrentInitializationUsesSingleGinValidator(t *testing.T) {
	resetGlobalStateForTest(t)

	ginValidator, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		t.Fatal("expected gin binding validator engine to be *validator.Validate")
	}

	const goroutines = 64
	start := make(chan struct{})
	errCh := make(chan string, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			<-start
			if Validate() != ginValidator {
				errCh <- "Validate() returned a different validator"
			}
			if Trans() == nil {
				errCh <- "Trans() returned nil"
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)

	for msg := range errCh {
		t.Fatal(msg)
	}
}

func TestRepeatedNewDoesNotReregisterTranslations(t *testing.T) {
	resetGlobalStateForTest(t)

	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}
	firstValidator := Validate()
	firstTrans := Trans()
	if firstValidator == nil {
		t.Fatal("expected validator after New")
	}
	if firstTrans == nil {
		t.Fatal("expected translator after New")
	}

	if err := New(); err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if Validate() != firstValidator {
		t.Fatal("New should keep the first gin validator")
	}
	if Trans() != firstTrans {
		t.Fatal("New should keep the first translator")
	}
}

func resetGlobalStateForTest(t *testing.T) {
	t.Helper()

	globalState.once = sync.Once{}
	globalState.initErr = nil
	globalState.trans = nil
	globalState.validate = nil
	globalState.regMu = sync.Mutex{}
}

type unsupportedGinValidator struct{}

func (unsupportedGinValidator) ValidateStruct(any) error {
	return nil
}

func (unsupportedGinValidator) Engine() any {
	return unsupportedGinValidator{}
}
