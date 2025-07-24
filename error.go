package verify

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gtkit/goerr"
)

// FieldError 验证字段错误信息, 字段名, 验证时的error
func FieldError(field string, err error, msg ...string) goerr.Error {
	if err == nil {
		return nil
	}
	var errs validator.ValidationErrors
	if ok := errors.As(err, &errs); !ok {
		// 非validator.ValidationErrors类型错误直接返回
		return goerr.New(err, goerr.ValidateParams, "非ValidationErrors类型错误")
	}

	for _, v := range RemoveTopStruct(errs.Translate(trans)) {
		return goerr.New(goerr.Err(field+" "+v), goerr.ValidateParams, msg...)

	}
	return nil

}

// StructErr 验证结构体错误信息
func StructErr(err error, msg ...string) goerr.Error {
	if err == nil {
		return nil
	}
	var errs validator.ValidationErrors
	if ok := errors.As(err, &errs); !ok {
		// 非validator.ValidationErrors类型错误直接返回
		return goerr.New(err, goerr.ValidateParams, "非ValidationErrors类型错误")
	}

	for _, v := range RemoveTopStruct(errs.Translate(Trans())) {
		return goerr.New(goerr.Err(v), goerr.ValidateParams, msg...)

	}
	return nil
}

// MapErr 验证map错误信息
func MapErr(err map[string]any, msg ...string) goerr.Error {
	var (
		errs validator.ValidationErrors
		ok   bool
	)
	if err == nil {
		return nil
	}

	for k, v := range err {
		if errs, ok = v.(validator.ValidationErrors); !ok {
			return goerr.New(nil, goerr.ValidateParams, "非ValidationErrors类型错误")
		}
		if maperr := GetMapError(errs.Translate(Trans())); maperr != "" {
			return goerr.New(goerr.Err(k+" "+maperr), goerr.ValidateParams, msg...)
		}
	}

	return nil
}
