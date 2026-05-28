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

type state struct {
	once     sync.Once
	initErr  error
	trans    ut.Translator
	validate *validator.Validate
	regMu    sync.Mutex
}

// globalState 保存 Gin 共享 validator 的初始化状态和注册锁。
var globalState state

func validatorOrError() (*validator.Validate, error) {
	if err := initDefaultValidator(); err != nil {
		return nil, err
	}
	return globalState.validate, nil
}

// New initializes verify with Gin's shared binding validator and applies opts.
//
// Call New during application startup before any Gin ShouldBind* call, so
// validator field-name caches use the external json field names. Options and
// registration helpers must also run before any validation; validator/v10 does
// not make registration safe while validation is running. Nil options are
// ignored.
func New(opts ...Option) error {
	if err := initDefaultValidator(); err != nil {
		return err
	}
	if len(opts) == 0 {
		return nil
	}
	globalState.regMu.Lock()
	defer globalState.regMu.Unlock()
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt.apply(globalState.validate, globalState.trans); err != nil {
			return err
		}
	}
	return nil
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
// 与 [New] 的 [EnableRequiredStructValidation] Functional Option 并列,
// 适用于多模块各自配置、不便集中到 main 的场景。仅应在应用启动阶段、
// 任何校验发生前调用。
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
// 使用此功能的后果。
//
// 与 [New] 的 [EnablePrivateFieldValidation] Functional Option 并列,
// 适用于多模块各自配置、不便集中到 main 的场景。仅应在应用启动阶段、
// 任何校验发生前调用。
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
// 如需变更, 建议优先使用 [New] 的 [WithTranslation]、
// [WithStructValidation]、[WithValidationTranslation] 等 Functional Option,
// 在应用启动阶段一次性完成所有注册。除非你提供外部同步, 否则应将返回的
// 校验器视为只读。
//
// 如未先成功调用 [New], 本函数可能返回 nil。生产代码应当先检查
// [New] 的返回错误, 再使用本函数。
func Validate() *validator.Validate {
	_ = initDefaultValidator()
	return globalState.validate
}

// Trans 返回共享的翻译器。
//
// 建议优先使用 [FieldErr]、[StructErr]、[MapErr] 等高层辅助函数,
// 应将返回的翻译器视为只读。
//
// 如未先成功调用 [New], 本函数可能返回 nil。生产代码应当先检查
// [New] 的返回错误, 再使用本函数。
func Trans() ut.Translator {
	_ = initDefaultValidator()
	return globalState.trans
}

// RemoveTopStruct 去除字段前面的结构体名称。
//
// Deprecated: 请使用 [FieldErr]、[StructErr] 或 [MapErr] 获取对外错误消息。
// 本函数将在 v2.0.0 移除。
func RemoveTopStruct(fields map[string]string) map[string]string {
	return removeTopStruct(fields)
}

func removeTopStruct(fields map[string]string) map[string]string {
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

// GetMapError 返回按字段名排序后的第一条 map 校验错误消息。
//
// Deprecated: 请使用 [MapErr] 获取对外错误消息。本函数将在 v2.0.0 移除。
func GetMapError(fields map[string]string) string {
	return getMapError(fields)
}

func getMapError(fields map[string]string) string {
	msg, _ := firstSortedMessage(fields)
	return msg
}

// RegisterTranslator 为自定义字段添加翻译功能。
//
// Deprecated: 请使用 [New] 的 [WithTranslation] 或
// [WithValidationTranslation] Functional Option, 在应用启动阶段一次性完成注册。
// 本函数将在 v2.0.0 移除。
func RegisterTranslator(tag string, msg string) validator.RegisterTranslationsFunc {
	return registerTranslator(tag, msg)
}

func registerTranslator(tag string, msg string) validator.RegisterTranslationsFunc {
	return func(trans ut.Translator) error {
		return trans.Add(tag, msg, true)
	}
}

// Translate 自定义字段的翻译方法。
//
// Deprecated: 请使用 [FieldErr]、[StructErr] 或 [MapErr] 获取对外错误消息。
// 本函数将在 v2.0.0 移除。
func Translate(trans ut.Translator, fe validator.FieldError) string {
	return translate(trans, fe)
}

func translate(trans ut.Translator, fe validator.FieldError) string {
	if trans != nil {
		// fe.Translate 会优先调用 RegisterTranslation 注册过的 customTransFunc,
		// 这对 len/min/max/oneof 等"翻译表里没有直接以 tag 命名的 key"的
		// 带参数内置 tag 是必要的: 它们的中文消息靠 customTransFunc 从
		// len-string/len-number/... 等子 key 拼装。
		// 未注册时 validator 内部回退到 fe.Error()(英文 Key:'...' 长串),
		// 视作未翻译, 再退到下方的 field+tag 兜底。
		if msg := fe.Translate(trans); msg != "" && msg != fe.Error() {
			return msg
		}
	}
	if field := fe.Field(); field != "" {
		return field + " " + fe.Tag()
	}
	return fe.Tag()
}

// simpleTransFunc 作为 customTransFunc 注册给用户自定义 tag(经
// [SelfRegisterTranslation] / [WithTranslation] / [AddValidationTranslation]
// / [RegisterTranslation]), 走 ut.T(tag, field) 简单替换。不直接调
// translate, 以避免 translate 顶层调用 fe.Translate 时回到此处造成无限递归。
func simpleTransFunc(ut ut.Translator, fe validator.FieldError) string {
	if ut != nil {
		if msg, err := ut.T(fe.Tag(), fe.Field()); err == nil {
			return msg
		}
	}
	if field := fe.Field(); field != "" {
		return field + " " + fe.Tag()
	}
	return fe.Tag()
}

// SelfRegisterTranslation 注册自定义校验函数及其翻译。
//
// 与 [New] 的 [WithTranslation] Functional Option 并列, 适用于多模块各自
// 维护翻译表、不便集中到 main 的场景。仅应在应用启动阶段、任何校验发生
// 前调用。该函数只串行化 verify 包内的注册写操作, 不提供注册与 Gin/
// validator 校验并发执行时的安全保证。
func SelfRegisterTranslation(method string, info string, myFunc validator.Func) (err error) {
	v := Validate()
	if v == nil {
		return fmt.Errorf("validator 初始化失败")
	}

	globalState.regMu.Lock()
	defer globalState.regMu.Unlock()
	return registerValidationAndTranslationLocked(v, globalState.trans, method, info, myFunc)
}

// AddValidationTranslation 为已存在的校验 tag 注册翻译模板。
//
// 与 [New] 的 [WithValidationTranslation] Functional Option 并列, 适用于
// 多模块各自维护翻译表、不便集中到 main 的场景。仅应在应用启动阶段、
// 任何校验发生前调用。该函数只串行化 verify 包内的注册写操作, 不提供
// 注册与 Gin/validator 校验并发执行时的安全保证。
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
		registerTranslator(method, info),
		simpleTransFunc,
	)
}

// registerValidationAndTranslationLocked 注册校验函数和对应翻译。
// 调用方必须在 globalState.regMu 锁内调用。
func registerValidationAndTranslationLocked(v *validator.Validate, trans ut.Translator, method, info string, fn validator.Func) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("register validation %q: %v", method, r)
		}
	}()
	if err := v.RegisterValidation(method, fn); err != nil {
		return fmt.Errorf("register validation %q: %w", method, err)
	}
	if err := addValidationTranslationLocked(v, trans, method, info); err != nil {
		return fmt.Errorf("register translation %q: %w", method, err)
	}
	return nil
}

// RegisterStructValidation 注册结构体级校验函数。
//
// 与 [New] 的 [WithStructValidation] Functional Option 并列, 适用于多模块
// 各自维护校验规则、不便集中到 main 的场景。仅应在应用启动阶段、任何
// 校验发生前调用。该函数只串行化 verify 包内的注册写操作, 不提供注册与
// Gin/validator 校验并发执行时的安全保证。
//
// 如未先成功调用 [New], 本函数会静默退化为不注册。生产代码应当先检查
// [New] 的返回错误, 再使用本函数。
func RegisterStructValidation(sl validator.StructLevelFunc, types ...any) {
	v := Validate()
	if v == nil {
		return
	}
	globalState.regMu.Lock()
	defer globalState.regMu.Unlock()
	v.RegisterStructValidation(sl, types...)
}
