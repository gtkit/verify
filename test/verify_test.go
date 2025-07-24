package test_test

import (
	"log"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"

	"github.com/gtkit/verify"
)

func init() {
	verify.New()
	log.Println("new verify instance")
}
func TestVarField(t *testing.T) {
	p := "www.google.com"
	err := verify.Field(p, "required,numeric")
	if err != nil {
		goerr := verify.FieldError("type", err)
		log.Printf("error: %+v\n", goerr.Status())
		t.Logf("error: %v", goerr)
		// t.Error(goerr)
		return
	}
	t.Log("success")
}

type OrderParams struct {
	Name        string `json:"name" form:"name" binding:"contains=tom,checkName"`
	OrderShopId string `form:"order_shop_id"  binding:"required,numeric"`
	Type        string `form:"type"  binding:"required,numeric"`
	Password    string `json:"password" form:"password" binding:"required"`
	RePassword  string `json:"re_password" form:"re_password" binding:"required_with=Password,eqfield=Password"`
	Date        string `json:"date" form:"date" binding:"required,datetime=2006-01-02,checkDate"`
	Imei        string `json:"imei,omitempty" form:"imei"`
	Oaid        string `json:"oaid" form:"oaid" binding:"required_if=Imei __IMEI__ ,required_without=Imei"` // 设备OAID
}

func TestVarStruct(t *testing.T) {
	params := OrderParams{
		Name:        "roottom",
		OrderShopId: "123",
		Type:        "1",
		Password:    "123",
		RePassword:  "123",
		Date:        "2025-01-02",
		Imei:        "",
	}
	// 注册结构体验证器
	verify.Validate().RegisterStructValidation(ParamStructlValidation, params)

	// 注册自定义验证器
	if err := verify.SelfRegisterTranslation("checkDate", "必须要晚于当前日期", CheckDate); err != nil {
		panic(err)
	}
	if err := verify.SelfRegisterTranslation("checkName", "名字格式不对", CheckName); err != nil {
		panic(err)
	}

	// 验证结构体
	if err := verify.Struct(params); err != nil {
		goerr := verify.StructErr(err)
		t.Logf("error: %+v\n", goerr.Status())
		t.Error(goerr)
		return
	}
	t.Log("success")
}

func CheckName(fl validator.FieldLevel) bool {
	return fl.Field().String() == "roottom"
}

func CheckDate(fl validator.FieldLevel) bool {
	date, err := time.Parse(time.DateOnly, fl.Field().String())
	if err != nil {
		return false
	}
	// 验证日期是否晚于当前日期
	return date.After(time.Now())
}

func ParamStructlValidation(sl validator.StructLevel) {
	su := sl.Current().Interface().(OrderParams)

	if su.Password != su.RePassword {
		// 输出错误提示信息，最后一个参数就是传递的param
		sl.ReportError(su.RePassword, "re_password", "RePassword", "eqfield", "填入的password")
	}
}

func TestVarMap(t *testing.T) {
	user := map[string]interface{}{
		"name":  "htttereeee",
		"emain": "hddd@google.com",
		// "email": "1",
	}

	rules := map[string]interface{}{
		"name":  "required,min=8,max=15",
		"email": "omitempty,email",
	}

	// 此处err为map[string]any
	if err := verify.Map(user, rules); len(err) > 0 {
		log.Printf("verify error: %+v, len: %d\n", err, len(err))
		goerr := verify.MapErr(err)
		t.Logf("error: %+v\n", goerr.Status())
		t.Error(goerr)
		return
	}
	t.Log("success")
}
