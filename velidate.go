// @Author xiaozhaofu 2023/3/2 11:22:00
package verify

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"gitlab.superjq.com/go-tools/logger"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
	"gitlab.superjq.com/go-tools/goerr"
)

var _ Verify = (*verify)(nil)

type Verify interface {
	i()
	RemoveTopStruct(fields map[string]string) map[string]string
	RegisterTranslator(tag string, msg string) validator.RegisterTranslationsFunc
	Translate(trans ut.Translator, fe validator.FieldError) string
	SelfRegisterTranslation(method string, info string, myFunc validator.Func) (err error)
	AddValidationTranslation(method, info string) error
	ErrorInfo(field string, err error) error
}

type verify struct {
	Trans    ut.Translator
	Validate *validator.Validate
}

// 定义一个全局翻译器T
var (
	trans    ut.Translator
	validate *validator.Validate
)

func New() Verify {
	initlogger()
	transValidate()
	return &verify{
		Trans:    trans,
		Validate: validate,
	}
}

func initlogger() {
	if logger.Zlog() == nil {
		opt := &logger.Option{
			FileStdout: true,
			Division:   "size",
		}
		logger.NewZap(opt)
		log.Println("redis new zap logger")
	}
}

// 初始化验证并翻译
func transValidate() {
	v, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		panic("validator 初始化失败")
		// return nil, fmt.Errorf("validator 初始化失败")
	}
	// 注册一个获取json tag的自定义方法
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	// 翻译
	err := getTrans("zh", v)
	if err != nil {
		logger.Info("初始化验证翻译 error: ", err)
		panic(err)
	}
	validate = v
	logger.Info("初始化验证翻译 success")

}

func getTrans(locale string, v *validator.Validate) (err error) {
	zhT := zh.New() // 中文翻译器
	enT := en.New() // 英文翻译器
	uni := ut.New(enT, zhT, enT)

	// locale 通常取决于 http 请求头的 'Accept-Language'
	var ok bool
	// 也可以使用 uni.FindTranslator(...) 传入多个locale进行查找
	trans, ok = uni.GetTranslator(locale)
	if !ok {
		return fmt.Errorf("uni.GetTranslator(%s) failed", locale)
	}
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
	return err
}

func (v *verify) i() {}

func (v *verify) RemoveTopStruct(fields map[string]string) map[string]string {
	res := map[string]string{}
	for field, err := range fields {
		res[field[strings.Index(field, ".")+1:]] = err
	}
	return res
}

// registerTranslator 为自定义字段添加翻译功能
func (v *verify) RegisterTranslator(tag string, msg string) validator.RegisterTranslationsFunc {
	return func(trans ut.Translator) error {
		if err := trans.Add(tag, msg, false); err != nil {
			return err
		}
		return nil
	}
}

// translate 自定义字段的翻译方法
func (v *verify) Translate(trans ut.Translator, fe validator.FieldError) string {
	msg, err := trans.T(fe.Tag(), fe.Field())
	if err != nil {
		panic(fe.(error).Error())
	}
	return msg
}

// 翻译自定义校验方法
func (v *verify) SelfRegisterTranslation(method string, info string, myFunc validator.Func) (err error) {

	if err = v.Validate.RegisterValidation(method, myFunc); err != nil {
		return
	}

	err = v.AddValidationTranslation(method, info)
	return
}

// 完善未有的验证方法的翻译
func (v *verify) AddValidationTranslation(method, info string) error {
	return v.Validate.RegisterTranslation(
		method,
		trans,
		v.RegisterTranslator(method, "{0}"+info),
		v.Translate,
	)
}

// 普通验证字段错误信息, 字段名, 验证时的error
func (v *verify) ErrorInfo(field string, err error) error {

	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		// 非validator.ValidationErrors类型错误直接返回
		return goerr.Wrap(err, "非validator类型错误")
	}
	for _, v := range v.RemoveTopStruct(errs.Translate(trans)) {

		return goerr.Custom(field + " " + v)
	}
	return nil

}
