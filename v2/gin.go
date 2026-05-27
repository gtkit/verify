package verify

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/gtkit/goerr"
)

// bindToGin 替换 Gin 内置的校验器引擎。
func bindToGin(v *validator.Validate) error {
	binding.Validator = &ginValidator{v: v}
	return nil
}

type ginValidator struct{ v *validator.Validate }

func (g *ginValidator) ValidateStruct(obj any) error { return goerr.WithStack(g.v.Struct(obj)) }
func (g *ginValidator) Engine() any                  { return g.v }

// GinStructErr 把 Gin 的 c.ShouldBind 返回的错误翻译为可读错误,
// 行为与 [Verifier.StructErr] 一致。
//
//	if err := c.ShouldBindJSON(&params); err != nil {
//	    return v.GinStructErr(err)
//	}
func (ver *Verifier) GinStructErr(err error) error {
	return ver.StructErr(err)
}

// GinFieldErr 把 Gin 的字段绑定错误翻译为可读错误。
func (ver *Verifier) GinFieldErr(field string, err error) error {
	return ver.FieldErr(field, err)
}
