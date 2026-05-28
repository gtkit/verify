package verify_test

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/gtkit/verify"
)

func ExampleNew() {
	err := verify.New(
		verify.WithTranslation("example_name", "名字格式不对", func(fl validator.FieldLevel) bool {
			return fl.Field().String() == "tom"
		}),
	)
	fmt.Println(err == nil)
	// Output: true
}

func ExampleStructErr() {
	type User struct {
		Email string `json:"email" binding:"required,email"`
	}

	fmt.Println(verify.StructErr(verify.Struct(User{Email: "bad"})))
	// Output: email必须是一个有效的邮箱
}
