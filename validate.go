package verify

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"sync"

	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
)

// 定义一个全局翻译器T
var (
	globalState struct {
		once     sync.Once
		initErr  error
		trans    ut.Translator
		validate *validator.Validate
		regMu    sync.Mutex
	}
)

func New() {
	_ = initDefaultValidator()
}

// 初始化验证并翻译
func initDefaultValidator() error {
	globalState.once.Do(func() {
		v := validator.New()
		v.SetTagName("binding")
	// 注册一个获取json tag的自定义方法
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})

		trans, err := getTrans("zh", v)
		if err != nil {
			globalState.initErr = err
			return
		}

		globalState.validate = v
		globalState.trans = trans
	})
	return globalState.initErr
}

// WithRequiredStructEnabled在非指针结构上启用所需标记，而不是忽略。
//
// 这是选择性加入行为，以保持与之前行为的向后兼容性
// 到能够直接对结构体字段应用结构体级验证。
//
// 建议您启用此功能，因为它将成为v11+中的默认行为
func WithRequiredStructEnabled() {
	if v := Validate(); v != nil {
		validator.WithRequiredStructEnabled()(v)
	}
}

// WithPrivateFieldValidation通过使用“不安全”包激活对未导出字段的验证。
//
// 通过选择此功能，您承认您了解风险并接受任何当前或未来的风险
// 使用此功能的后果。
func WithPrivateFieldValidation() {
	if v := Validate(); v != nil {
		validator.WithPrivateFieldValidation()(v)
	}
}

func getTrans(locale string, v *validator.Validate) (ut.Translator, error) {
	zhT := zh.New() // 中文翻译器
	enT := en.New() // 英文翻译器
	// uni := ut.New(enT, zhT, enT)
	uni := ut.New(zhT, zhT, enT)

	// locale 通常取决于 http 请求头的 'Accept-Language'
	var ok bool
	// 也可以使用 uni.FindTranslator(...) 传入多个locale进行查找
	trans, ok := uni.GetTranslator(locale)
	if !ok {
		return nil, fmt.Errorf("uni.GetTranslator(%s) failed", locale)
	}
	var err error
	switch locale {
	case "en":
		err = enTranslations.RegisterDefaultTranslations(v, trans)
	case "zh":
		err = zhTranslations.RegisterDefaultTranslations(v, trans)
	default:
		err = enTranslations.RegisterDefaultTranslations(v, trans)
	}
	// err = v.RegisterTranslation("required_if", Trans, func(ut ut.Translator) error {
	// 	return ut.Add("required_if", "{0}为必填字段!", false) // see universal-translator for details
	// }, func(ut ut.Translator, fe validator.FieldError) string {
	// 	t, _ := ut.T("required_if", fe.Field())
	// 	return t
	// })
	return trans, err
}

// Validate returns the shared validator instance.
//
// Prefer package helpers such as [SelfRegisterTranslation],
// [AddValidationTranslation], and [RegisterStructValidation] for mutations.
// Treat the returned validator as read-only unless you provide your own
// external synchronization.
func Validate() *validator.Validate {
	_ = initDefaultValidator()
	return globalState.validate
}

// Trans returns the shared translator.
//
// Prefer high-level helpers such as [FieldErr], [StructErr], and [MapErr].
// Treat the returned translator as read-only.
func Trans() ut.Translator {
	_ = initDefaultValidator()
	return globalState.trans
}

// RemoveTopStruct 去除字段前面的结构体名称
func RemoveTopStruct(fields map[string]string) map[string]string {
	res := map[string]string{}
	for field, err := range fields {
		res[field[strings.Index(field, ".")+1:]] = err
	}
	return res
}

func GetMapError(fields map[string]string) string {
	if len(fields) == 0 {
		return ""
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return fields[keys[0]]
}

// RegisterTranslator 为自定义字段添加翻译功能
func RegisterTranslator(tag string, msg string) validator.RegisterTranslationsFunc {
	return func(trans ut.Translator) error {
		if err := trans.Add(tag, msg, true); err != nil {
			return err
		}
		return nil
	}
}

// Translate 自定义字段的翻译方法
func Translate(trans ut.Translator, fe validator.FieldError) string {
	msg, err := trans.T(fe.Tag(), fe.Field())
	if err != nil {
		if feErr, ok := fe.(error); ok {
			return feErr.Error()
		}
		return fe.Tag()
	}
	return msg
}

// SelfRegisterTranslation 翻译自定义校验方法
func SelfRegisterTranslation(method string, info string, myFunc validator.Func) (err error) {
	v := Validate()
	if v == nil {
		return fmt.Errorf("validator 初始化失败")
	}

	globalState.regMu.Lock()
	defer globalState.regMu.Unlock()

	if err = v.RegisterValidation(method, myFunc); err != nil {
		return
	}
	return addValidationTranslationLocked(v, globalState.trans, method, info)
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
