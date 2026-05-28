package verify

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
)

// globalState 保存 Gin 共享 validator 的初始化状态和注册锁。
var (
	globalState struct {
		once     sync.Once
		initErr  error
		trans    ut.Translator
		validate *validator.Validate
		regMu    sync.Mutex
	}
)

func validatorOrError() (*validator.Validate, error) {
	if err := initDefaultValidator(); err != nil {
		return nil, err
	}
	return globalState.validate, nil
}

// New initializes verify with Gin's shared binding validator.
//
// Call New during application startup before any Gin ShouldBind* call, so
// validator field-name caches use the external json field names.
func New() error {
	return initDefaultValidator()
}

// 初始化验证并翻译
func initDefaultValidator() error {
	globalState.once.Do(func() {
		engine := binding.Validator.Engine()
		v, ok := engine.(*validator.Validate)
		if !ok {
			globalState.initErr = fmt.Errorf("gin binding validator engine %T is not *validator.Validate", engine)
			return
		}

		trans, err := initValidator(v)
		if err != nil {
			globalState.initErr = err
			return
		}

		globalState.validate = v
		globalState.trans = trans
	})
	return globalState.initErr
}

func initValidator(v *validator.Validate) (ut.Translator, error) {
	// 与 Gin 默认校验 tag 保持一致; 显式设置避免外部替换默认值后语义漂移。
	v.SetTagName("binding")
	// 注册一个获取json tag的自定义方法
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return getTrans(v)
}

// WithRequiredStructEnabled 在非指针结构上启用所需标记，而不是忽略。
//
// 这是选择性加入行为，以保持与之前行为的向后兼容性
// 到能够直接对结构体字段应用结构体级验证。
//
// 建议在应用启动阶段、任何校验发生前调用。
func WithRequiredStructEnabled() {
	if v := Validate(); v != nil {
		globalState.regMu.Lock()
		defer globalState.regMu.Unlock()
		validator.WithRequiredStructEnabled()(v)
	}
}

// WithPrivateFieldValidation 通过使用“不安全”包激活对未导出字段的验证。
//
// 通过选择此功能，您承认您了解风险并接受任何当前或未来的风险
// 使用此功能的后果。建议在应用启动阶段、任何校验发生前调用。
func WithPrivateFieldValidation() {
	if v := Validate(); v != nil {
		globalState.regMu.Lock()
		defer globalState.regMu.Unlock()
		validator.WithPrivateFieldValidation()(v)
	}
}

func getTrans(v *validator.Validate) (ut.Translator, error) {
	zhT := zh.New() // 中文翻译器
	uni := ut.New(zhT, zhT)

	trans, ok := uni.GetTranslator("zh")
	if !ok {
		return nil, fmt.Errorf("uni.GetTranslator(zh) failed")
	}
	return trans, zhTranslations.RegisterDefaultTranslations(v, trans)
}

// Validate 返回共享的校验器实例。
//
// 如需变更，建议使用 [SelfRegisterTranslation]、[AddValidationTranslation]、
// [RegisterStructValidation] 等包级辅助函数。
// 除非你提供外部同步，否则应将返回的校验器视为只读。
func Validate() *validator.Validate {
	_ = initDefaultValidator()
	return globalState.validate
}

// Trans 返回共享的翻译器。
//
// 建议优先使用 [FieldErr]、[StructErr]、[MapErr] 等高层辅助函数，
// 应将返回的翻译器视为只读。
func Trans() ut.Translator {
	_ = initDefaultValidator()
	return globalState.trans
}

// RemoveTopStruct 去除字段前面的结构体名称
func RemoveTopStruct(fields map[string]string) map[string]string {
	res := map[string]string{}
	for field, err := range fields {
		_, key, ok := strings.Cut(field, ".")
		if !ok {
			key = field
		}
		res[key] = err
	}
	return res
}

func GetMapError(fields map[string]string) string {
	msg, _ := firstSortedMessage(fields)
	return msg
}

// RegisterTranslator 为自定义字段添加翻译功能
func RegisterTranslator(tag string, msg string) validator.RegisterTranslationsFunc {
	return func(trans ut.Translator) error {
		return trans.Add(tag, msg, true)
	}
}

// Translate 自定义字段的翻译方法
func Translate(trans ut.Translator, fe validator.FieldError) string {
	if trans != nil {
		if msg, err := trans.T(fe.Tag(), fe.Field()); err == nil {
			return msg
		}
	}
	if field := fe.Field(); field != "" {
		return field + " " + fe.Tag()
	}
	return fe.Tag()
}

// SelfRegisterTranslation 翻译自定义校验方法
func SelfRegisterTranslation(method string, info string, myFunc validator.Func) (err error) {
	v := Validate()
	if v == nil {
		return fmt.Errorf("validator 初始化失败")
	}

	globalState.regMu.Lock()
	defer globalState.regMu.Unlock()
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("register validation %q: %v", method, r)
		}
	}()

	if err = v.RegisterValidation(method, myFunc); err != nil {
		return fmt.Errorf("register validation %q: %w", method, err)
	}
	if err = addValidationTranslationLocked(v, globalState.trans, method, info); err != nil {
		return fmt.Errorf("register translation %q: %w", method, err)
	}
	return nil
}

// AddValidationTranslation 完善未有的验证方法的翻译
func AddValidationTranslation(method, info string) error {
	v := Validate()
	if v == nil {
		return fmt.Errorf("validator 初始化失败")
	}

	globalState.regMu.Lock()
	defer globalState.regMu.Unlock()

	return addValidationTranslationLocked(v, globalState.trans, method, info)
}

func addValidationTranslationLocked(v *validator.Validate, trans ut.Translator, method, info string) error {
	return v.RegisterTranslation(
		method,
		trans,
		RegisterTranslator(method, info),
		Translate,
	)
}

// RegisterStructValidation 自定义结构体验证方法
func RegisterStructValidation(sl validator.StructLevelFunc, types ...any) {
	v := Validate()
	if v == nil {
		return
	}
	globalState.regMu.Lock()
	defer globalState.regMu.Unlock()
	v.RegisterStructValidation(sl, types...)
}
