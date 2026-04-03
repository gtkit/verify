package verify

import (
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/gtkit/goerr"
)

// bindToGin replaces Gin's built-in validator engine.
func bindToGin(v *validator.Validate) error {
	binding.Validator = &ginValidator{v: v}
	return nil
}

type ginValidator struct{ v *validator.Validate }

func (g *ginValidator) ValidateStruct(obj any) error { return goerr.WithStack(g.v.Struct(obj)) }
func (g *ginValidator) Engine() any                  { return g.v }

// GinStructErr translates an error from Gin's c.ShouldBind into a
// human-readable error, same as [Verifier.StructErr].
//
//	if err := c.ShouldBindJSON(&params); err != nil {
//	    return v.GinStructErr(err)
//	}
func (ver *Verifier) GinStructErr(err error) error {
	return ver.StructErr(err)
}

// GinFieldErr translates a Gin binding error for a specific field.
func (ver *Verifier) GinFieldErr(field string, err error) error {
	return ver.FieldErr(field, err)
}
