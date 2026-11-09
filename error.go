package verify

import (
	"errors"
	"slices"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/gtkit/goerr"
)

// FieldError 描述单个字段的校验错误。
type FieldError struct {
	Field   string // 对外字段名（json tag）
	Message string // 翻译后的错误消息
	Tag     string // 触发的校验 tag
	Value   any    // 字段值（可选，用于调试）
}

// FieldErrors 多字段校验错误集合。
type FieldErrors []FieldError

// Error 实现 error 接口，返回所有字段错误的摘要。
func (fe FieldErrors) Error() string {
	if len(fe) == 0 {
		return ""
	}
	if len(fe) == 1 {
		return fe[0].Field + " " + fe[0].Message
	}
	var sb strings.Builder
	sb.WriteString(fe[0].Field)
	sb.WriteString(" ")
	sb.WriteString(fe[0].Message)
	for _, e := range fe[1:] {
		sb.WriteString("; ")
		sb.WriteString(e.Field)
		sb.WriteString(" ")
		sb.WriteString(e.Message)
	}
	return sb.String()
}

// First 返回第一个字段错误，集合为空时返回 nil。
func (fe FieldErrors) First() *FieldError {
	if len(fe) == 0 {
		return nil
	}
	return &fe[0]
}

// Fields 返回所有出错字段名列表。
func (fe FieldErrors) Fields() []string {
	fields := make([]string, len(fe))
	for i, e := range fe {
		fields[i] = e.Field
	}
	return fields
}

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
		return goerr.New(errors.New(field+" "+v), goerr.StatusValidateParams(), msg[0])
	}
	return goerr.Newf(goerr.StatusValidateParams(), "%s %s", field, v)
}

// StructErr 把结构体校验错误翻译为可读错误。
// 行为约定与 [FieldErr] 一致, 非 validation 错误原样透传。
// 多字段失败时返回排序后的第一个错误。
func StructErr(err error, msg ...string) error {
	if err == nil {
		return nil
	}
	var errs validator.ValidationErrors
	if !errors.As(err, &errs) {
		return err
	}
	fields := translateValidationErrors(errs)
	field, v, ok := firstSortedFieldMessage(fields)
	if !ok {
		return nil
	}
	// 统一行为：字段名 + 翻译消息
	fullMsg := field + " " + v
	if len(msg) > 0 && msg[0] != "" {
		return goerr.New(errors.New(fullMsg), goerr.StatusValidateParams(), msg[0])
	}
	return goerr.Newf(goerr.StatusValidateParams(), "%s", fullMsg)
}

// AllStructErr 把结构体校验错误翻译为包含所有字段错误的聚合错误。
//
// 行为约定:
//   - err 为 nil → 返回 nil。
//   - err 不是 [validator.ValidationErrors] → 原样透传。
//   - 返回的 *goerr.Item 的 Message() 包含所有字段错误摘要。
//   - 使用 [StructFieldErrors] 获取结构化字段错误详情。
func AllStructErr(err error, msg ...string) error {
	if err == nil {
		return nil
	}
	var errs validator.ValidationErrors
	if !errors.As(err, &errs) {
		return err
	}
	fieldErrors := toFieldErrors(errs)
	if len(fieldErrors) == 0 {
		return nil
	}
	summary := fieldErrors.Error()
	if len(msg) > 0 && msg[0] != "" {
		return goerr.New(errors.New(summary), goerr.StatusValidateParams(), msg[0])
	}
	return goerr.Newf(goerr.StatusValidateParams(), "%s", summary)
}

// StructFieldErrors 从结构体校验错误中提取所有字段错误详情。
//
// 返回的 FieldErrors 可直接用于构建详细错误响应：
//
//	if errs := verify.StructFieldErrors(err); len(errs) > 0 {
//	    for _, e := range errs {
//	        fmt.Printf("字段 %s: %s\n", e.Field, e.Message)
//	    }
//	}
func StructFieldErrors(err error) FieldErrors {
	if err == nil {
		return nil
	}
	var errs validator.ValidationErrors
	if !errors.As(err, &errs) {
		return nil
	}
	return toFieldErrors(errs)
}

// toFieldErrors 把 validator.ValidationErrors 转换为 FieldErrors。
func toFieldErrors(errs validator.ValidationErrors) FieldErrors {
	fields := translateValidationErrors(errs)
	if len(fields) == 0 {
		return nil
	}
	// 按字段名排序保证确定性
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	result := make(FieldErrors, 0, len(keys))
	for _, k := range keys {
		result = append(result, FieldError{
			Field:   k,
			Message: fields[k],
		})
	}
	return result
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
		maperr := getMapError(translateValidationErrors(errs))
		if maperr == "" {
			continue
		}
		if len(msg) > 0 && msg[0] != "" {
			return goerr.New(errors.New(k+" "+maperr), goerr.StatusValidateParams(), msg[0])
		}
		return goerr.Newf(goerr.StatusValidateParams(), "%s %s", k, maperr)
	}

	return nil
}

// mapFieldErrors 递归收集 map 校验结果中的所有字段错误。
//
// prefix 是外层键路径（顶层为空），用于拼出 "user.email" 这样的对外字段名。
// 每层都按键名排序，保证同一输入的产出确定。
func mapFieldErrors(err map[string]any, prefix string) FieldErrors {
	keys := make([]string, 0, len(err))
	for key := range err {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	var out FieldErrors
	for _, k := range keys {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		switch v := err[k].(type) {
		case validator.ValidationErrors:
			fields := translateValidationErrors(v)
			fieldKeys := make([]string, 0, len(fields))
			for field := range fields {
				fieldKeys = append(fieldKeys, field)
			}
			slices.Sort(fieldKeys)
			for _, field := range fieldKeys {
				// removeTopStruct 已剥掉 namespace 前缀，对 map 校验而言
				// 翻译后的字段名通常就是 k 本身。相等时直接用 path，否则会
				// 拼出 "email.email" 这种重复路径；不等说明规则指向更深的
				// 字段，追加相对路径。
				name := path
				if field != k {
					name = path + "." + field
				}
				out = append(out, FieldError{Field: name, Message: fields[field]})
			}
		case map[string]any:
			out = append(out, mapFieldErrors(v, path)...)
		}
	}
	return out
}

// AllMapErr 把 map 校验结果翻译为包含所有字段错误的聚合错误。
//
// 嵌套 map 的对外字段名按点号拼接（如 "user.email"）。Message() 是所有字段
// 错误的摘要，形如：
//
//	user.email email必须是一个有效的邮箱; user.name name长度必须至少为8个字符
func AllMapErr(err map[string]any, msg ...string) error {
	if len(err) == 0 {
		return nil
	}

	fieldErrors := mapFieldErrors(err, "")
	if len(fieldErrors) == 0 {
		return nil
	}
	summary := fieldErrors.Error()
	if len(msg) > 0 && msg[0] != "" {
		return goerr.New(errors.New(summary), goerr.StatusValidateParams(), msg[0])
	}
	return goerr.Newf(goerr.StatusValidateParams(), "%s", summary)
}

func translateValidationErrors(errs validator.ValidationErrors) map[string]string {
	fields := make(map[string]string, len(errs))
	trans := Trans()
	for _, fe := range errs {
		fields[fe.Namespace()] = translate(trans, fe)
	}
	return removeTopStruct(fields)
}

// firstSortedFieldMessage 返回排序后第一个字段名和消息。
func firstSortedFieldMessage(fields map[string]string) (field, message string, ok bool) {
	if len(fields) == 0 {
		return "", "", false
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys[0], fields[keys[0]], true
}

func firstSortedMessage(fields map[string]string) (string, bool) {
	_, msg, ok := firstSortedFieldMessage(fields)
	return msg, ok
}
