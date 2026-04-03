package verify

import (
	"errors"
	"slices"

	"github.com/go-playground/validator/v10"
	"github.com/gtkit/goerr"
)

// FieldErr 验证字段错误信息, 字段名, 验证时的error
func FieldErr(field string, err error, msg ...string) error {
	if err == nil {
		return nil
	}
	var errs validator.ValidationErrors
	if ok := errors.As(err, &errs); !ok {
		// 非validator.ValidationErrors类型错误直接返回
		return goerr.New(err, goerr.StatusValidateParams(), "非ValidationErrors类型错误")
	}

	if v, ok := firstSortedMessage(RemoveTopStruct(errs.Translate(Trans()))); ok {
		return goerr.New(goerr.Err(field+" "+v), goerr.StatusValidateParams(), msg...)
	}
	return nil

}

// StructErr 验证结构体错误信息
func StructErr(err error, msg ...string) error {
	if err == nil {
		return nil
	}
	var errs validator.ValidationErrors
	if ok := errors.As(err, &errs); !ok {
		// 非validator.ValidationErrors类型错误直接返回
		//return goerr.New(err, goerr.ValidateParams, "非ValidationErrors类型错误")
		return goerr.New(err, goerr.StatusValidateParams(), "非ValidationErrors类型错误")
	}

	if v, ok := firstSortedMessage(RemoveTopStruct(errs.Translate(Trans()))); ok {
		return goerr.New(goerr.Err(v), goerr.StatusValidateParams(), msg...)
	}
	return nil
}

// MapErr 验证map错误信息
func MapErr(err map[string]any, msg ...string) error {
	if err == nil {
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
			return goerr.New(nil, goerr.StatusValidateParams(), "非ValidationErrors类型错误")
		}
		if maperr := GetMapError(errs.Translate(Trans())); maperr != "" {
			return goerr.New(goerr.Err(k+" "+maperr), goerr.StatusValidateParams(), msg...)
		}
	}

	return nil
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
