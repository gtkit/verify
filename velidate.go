// @Author xiaozhaofu 2023/3/2 11:22:00
package verify

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
	"github.com/gtkit/goerr"
)

// 定义一个全局翻译器T
var (
	trans    ut.Translator
	validate *validator.Validate
)

func New() {
	transValidate()
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
		panic(err)
	}
	validate = v
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

func Validate() *validator.Validate {
	return validate
}
func Trans() ut.Translator {
	return trans
}

func RemoveTopStruct(fields map[string]string) map[string]string {
	res := map[string]string{}
	for field, err := range fields {
		res[field[strings.Index(field, ".")+1:]] = err
	}
	return res
}

// registerTranslator 为自定义字段添加翻译功能
func RegisterTranslator(tag string, msg string) validator.RegisterTranslationsFunc {
	return func(trans ut.Translator) error {
		if err := trans.Add(tag, msg, false); err != nil {
			return err
		}
		return nil
	}
}

// translate 自定义字段的翻译方法
func Translate(trans ut.Translator, fe validator.FieldError) string {
	msg, err := trans.T(fe.Tag(), fe.Field())
	if err != nil {
		panic(fe.(error).Error())
	}
	return msg
}

// 翻译自定义校验方法
func SelfRegisterTranslation(method string, info string, myFunc validator.Func) (err error) {

	if err = validate.RegisterValidation(method, myFunc); err != nil {
		return
	}

	err = AddValidationTranslation(method, info)
	return
}

// 完善未有的验证方法的翻译
func AddValidationTranslation(method, info string) error {
	return validate.RegisterTranslation(
		method,
		trans,
		RegisterTranslator(method, "{0}"+info),
		Translate,
	)
}

func Field(field interface{}, tag string) error {
	return Validate().Var(field, tag)
}

// ErrorInfo 普通验证字段错误信息, 字段名, 验证时的error
func ErrorInfo(field string, err error) goerr.Error {
	if err == nil {
		return nil
	}
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		// 非validator.ValidationErrors类型错误直接返回
		return goerr.New(err, goerr.ErrValidateParams, "字段验证错误")
	}

	for _, v := range RemoveTopStruct(errs.Translate(trans)) {
		return goerr.New(goerr.Custom(field+" "+v), goerr.ErrValidateParams, "字段验证错误")

	}
	return nil

}

// Error 验证字段错误信息, 字段名, 验证时的error
func Error(field string, err error) goerr.Error {
	if err == nil {
		return nil
	}
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		// 非validator.ValidationErrors类型错误直接返回
		return goerr.New(err, goerr.ErrValidateParams, "字段验证错误")
	}

	for _, v := range RemoveTopStruct(errs.Translate(trans)) {
		return goerr.New(goerr.Custom(field+" "+v), goerr.ErrValidateParams, "字段验证错误")

	}
	return nil

}

// ErrorStruct 验证结构体错误信息
func ErrorStruct(err error) goerr.Error {
	if err == nil {
		return nil
	}
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		// 非validator.ValidationErrors类型错误直接返回
		return goerr.New(err, goerr.ErrValidateParams, "字段验证错误")
	}

	for _, v := range RemoveTopStruct(errs.Translate(trans)) {
		return goerr.New(goerr.Err(v), goerr.ErrValidateParams, "字段验证错误")

	}

	return nil

}

// Struct 验证结构体错误信息
func Struct(err error) goerr.Error {
	if err == nil {
		return nil
	}
	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		// 非validator.ValidationErrors类型错误直接返回
		return goerr.New(err, goerr.ErrValidateParams, "字段验证错误")
	}

	for _, v := range RemoveTopStruct(errs.Translate(trans)) {
		return goerr.New(goerr.Err(v), goerr.ErrValidateParams, "字段验证错误")

	}

	return nil

}
