package verify

import (
	"errors"
	"slices"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/gtkit/goerr"
)

// FieldErr 把字段校验错误翻译为可读错误。
//
// 行为约定:
//   - err 为 nil → 返回 nil。
//   - err 不是 [validator.ValidationErrors] → 原样透传, 由上层决定如何处理,
//     不再包装成参数校验错误, 避免误导客户端、避免泄露内部类型名。
//   - msg 留空时, *goerr.Item 的 Message() 形如 "field 必须是一个有效的数值",
//     直接展示给客户端即可; 状态码为 StatusValidateParams。
//   - msg 非空时, 第一个非空值作为对外 Message(), 内部字段提示进入 Error()
//     供日志排障使用。
func FieldErr(field string, err error, msg ...string) error {
	if err == nil {
		return nil
	}
	var errs validator.ValidationErrors
	if !errors.As(err, &errs) {
		return err
	}
	v, ok := firstSortedMessage(translateValidationErrors(errs))
	if !ok {
		return nil
	}
	if len(msg) > 0 && msg[0] != "" {
		return goerr.New(goerr.Err(field+" "+v), goerr.StatusValidateParams(), msg[0])
	}
	return goerr.Newf(goerr.StatusValidateParams(), "%s %s", field, v)
}

// StructErr 把结构体校验错误翻译为可读错误。
// 行为约定与 [FieldErr] 一致, 非 validation 错误原样透传。
func StructErr(err error, msg ...string) error {
	if err == nil {
		return nil
	}
	var errs validator.ValidationErrors
	if !errors.As(err, &errs) {
		return err
	}
	v, ok := firstSortedMessage(translateValidationErrors(errs))
	if !ok {
		return nil
	}
	if len(msg) > 0 && msg[0] != "" {
		return goerr.New(goerr.Err(v), goerr.StatusValidateParams(), msg[0])
	}
	return goerr.New(nil, goerr.StatusValidateParams(), v)
}

// MapErr 把 map 校验结果翻译为可读错误。
// 翻译成功时 Message() 形如 "name name长度必须至少为8个字符"。
func MapErr(err map[string]any, msg ...string) error {
	if len(err) == 0 {
		return nil
	}

	keys := make([]string, 0, len(err))
	for key := range err {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, k := range keys {
		val := err[k]
		errs, ok := val.(validator.ValidationErrors)
		if !ok {
			continue
		}
		maperr := GetMapError(translateValidationErrors(errs))
		if maperr == "" {
			continue
		}
		if len(msg) > 0 && msg[0] != "" {
			return goerr.New(goerr.Err(k+" "+maperr), goerr.StatusValidateParams(), msg[0])
		}
		return goerr.Newf(goerr.StatusValidateParams(), "%s %s", k, maperr)
	}

	return nil
}

func translateValidationErrors(errs validator.ValidationErrors) map[string]string {
	fields := make(map[string]string, len(errs))
	trans := Trans()
	for _, fe := range errs {
		fields[fe.Namespace()] = translateFieldError(trans, fe)
	}
	return RemoveTopStruct(fields)
}

func translateFieldError(trans ut.Translator, fe validator.FieldError) string {
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

func firstSortedMessage(fields map[string]string) (string, bool) {
	if len(fields) == 0 {
		return "", false
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return fields[keys[0]], true
}
