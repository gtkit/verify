package verify_test

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gtkit/verify"
)

func ExampleNew() {
	if err := verify.New(
		verify.WithTranslation("name_is_tom", "{0}必须是 tom", func(fl validator.FieldLevel) bool {
			return fl.Field().String() == "tom"
		}),
	); err != nil {
		return
	}

	type User struct {
		Name string `json:"name" binding:"name_is_tom"`
	}

	fmt.Println(verify.StructErr(verify.Struct(User{Name: "alice"})))
	// Output: name必须是 tom
}

func ExampleStructErr() {
	type User struct {
		Email string `json:"email" binding:"required,email"`
	}

	fmt.Println(verify.StructErr(verify.Struct(User{Email: "bad"})))
	// Output: email必须是一个有效的邮箱
}
