package verify

import (
	"errors"
	"slices"

	"github.com/go-playground/validator/v10"
	"github.com/gtkit/goerr"
)

// FieldErr 把字段校验错误翻译为可读错误。
//
// 行为约定:
//   - err 为 nil → 返回 nil。
//   - err 不是 [validator.ValidationErrors] → 原样透传, 由上层决定如何处理,
//     不再包装成参数校验错误, 避免误导客户端、避免泄露内部类型名。
//   - 翻译成功 → 返回 *goerr.Item, Message() 形如 "field 必须是一个有效的数值",
//     适合直接给客户端展示; 状态码为 StatusValidateParams。
//
// field 为提示信息中前置的字段展示名:
//
//	err := v.Field(p, "required,numeric")
//	if err != nil {
//	    return v.FieldErr("type", err) // Message() → "type 必须是一个有效的数值"
//	}
func (ver *Verifier) FieldErr(field string, err error) error {
	if err == nil {
		return nil
	}
	valErrs, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return err
	}
	msg, ok := firstSortedMessage(RemoveTopStruct(valErrs.Translate(ver.trans)))
	if !ok {
		return nil
	}
	return goerr.Newf(goerr.StatusValidateParams(), "%s %s", field, msg)
}

// StructErr 把结构体校验错误翻译为可读错误。
//
// 行为约定与 [Verifier.FieldErr] 一致, 非 validation 错误原样透传。
// 翻译成功时 Message() 形如 "name长度必须至少为2个字符"。
//
//	err := v.Struct(params)
//	if err != nil {
//	    return v.StructErr(err)
//	}
func (ver *Verifier) StructErr(err error) error {
	if err == nil {
		return nil
	}
	valErrs, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return err
	}
	msg, ok := firstSortedMessage(RemoveTopStruct(valErrs.Translate(ver.trans)))
	if !ok {
		return nil
	}
	return goerr.New(nil, goerr.StatusValidateParams(), msg)
}

// MapErr 把 map 校验结果翻译为可读错误。
// result 为 [Verifier.Map] 的返回值。
// 翻译成功时 Message() 形如 "name name长度必须至少为8个字符"。
//
//	result := v.Map(data, rules)
//	if len(result) > 0 {
//	    return v.MapErr(result)
//	}
func (ver *Verifier) MapErr(result map[string]any) error {
	if len(result) == 0 {
		return nil
	}
	msgs := ver.AllMapErrors(result)
	key := firstSortedKey(msgs)
	if key == "" {
		return nil
	}
	return goerr.Newf(goerr.StatusValidateParams(), "%s %s", key, msgs[key])
}

// AllFieldErrors 翻译全部字段校验错误。
// 返回 字段名 → 翻译后提示 的 map; 当 err 为 nil 或不是 validation 错误时返回 nil。
//
//	err := v.Struct(params)
//	if err != nil {
//	    for field, msg := range v.AllFieldErrors(err) {
//	        log.Printf("%s: %s", field, msg)
//	    }
//	}
func (ver *Verifier) AllFieldErrors(err error) map[string]string {
	if err == nil {
		return nil
	}
	valErrs, ok := errors.AsType[validator.ValidationErrors](err)
	if !ok {
		return nil
	}
	return RemoveTopStruct(valErrs.Translate(ver.trans))
}

// AllMapErrors 翻译全部 map 校验错误。
// 返回 key → 翻译后提示 的 map; 当 result 为空时返回 nil。
func (ver *Verifier) AllMapErrors(result map[string]any) map[string]string {
	if len(result) == 0 {
		return nil
	}
	out := make(map[string]string, len(result))
	for key, val := range result {
		valErrs, ok := val.(validator.ValidationErrors)
		if !ok {
			continue
		}
		if msg, ok := firstSortedMessage(valErrs.Translate(ver.trans)); ok {
			out[key] = msg
		}
	}
	return out
}

func firstSortedKey(fields map[string]string) string {
	if len(fields) == 0 {
		return ""
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys[0]
}
