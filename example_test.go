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
	// Output: name name必须是 tom
}

func ExampleStructErr() {
	type User struct {
		Email string `json:"email" binding:"required,email"`
	}

	fmt.Println(verify.StructErr(verify.Struct(User{Email: "bad"})))
	// Output: email email必须是一个有效的邮箱
}

func ExampleAllStructErr() {
	type User struct {
		Name  string `json:"name" binding:"required,min=2"`
		Email string `json:"email" binding:"required,email"`
	}

	err := verify.AllStructErr(verify.Struct(User{Name: "a", Email: "bad"}))
	fmt.Println(err)
	// Output: email email必须是一个有效的邮箱; name name长度必须至少为2个字符
}

func ExampleStructFieldErrors() {
	type User struct {
		Name  string `json:"name" binding:"required,min=2"`
		Email string `json:"email" binding:"required,email"`
	}

	fieldErrs := verify.StructFieldErrors(verify.Struct(User{Name: "a", Email: "bad"}))
	for _, fe := range fieldErrs {
		fmt.Printf("%s: %s\n", fe.Field, fe.Message)
	}
	// Output:
	// email: email必须是一个有效的邮箱
	// name: name长度必须至少为2个字符
}
