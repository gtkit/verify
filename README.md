# verify
```
https://github.com/go-playground/validator
```
#### 用于gin框架参数验证翻译
```
type SignUpParam struct {
	Age        uint8  `json:"age" form:"age" binding:"gte=1,lte=130"`
	Name       string `json:"name" form:"name" binding:"contains=tom,checkName"`
	Email      string `json:"email" form:"email" binding:"required,email"`
	Password   string `json:"password" form:"password" binding:"required"`
	RePassword string `json:"re_password" form:"re_password" binding:"required_if=Password,eqfield=Password"`
	Date       string `json:"date" form:"date" binding:"required,datetime=2006-01-02,checkDate"`
}

// 校验方式使用 demo
func SignUp(c *gin.Context) {
	// 为SignUpParam注册自定义校验方法
	verify.Validate().RegisterStructValidation(SignUpParamStructLevelValidation, SignUpParam{})

        // 注册自定义字段级别校验方法
	if err := verify.SelfRegisterTranslation("checkDate", "必须要晚于当前日期", CustomFunc); err != nil {
		panic(err)
	}

	if err := verify.SelfRegisterTranslation("checkName", "名字格式不对", CheckName); err != nil {
		panic(err)
	}

	var s SignUpParam

	if err := c.ShouldBindQuery(&s); err != nil {
		// 获取validator.ValidationErrors类型的errors
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			// 非validator.ValidationErrors类型错误直接返回
			c.JSON(http.StatusOK, gin.H{
			 	"msg": err.Error(),
		     })
			
			return
		}

		for _, v := range verify.RemoveTopStruct(errs.Translate(verify.Trans())) {
			resp.Error(c, goerr.New(nil, goerr.ErrValidateParams, v))
			return
		}

	}
	return
}

// SignUpParamStructLevelValidation 自定义SignUpParam结构体校验函数
func SignUpParamStructLevelValidation(sl validator.StructLevel) {
	su := sl.Current().Interface().(SignUpParam)

	if su.Password != su.RePassword {
		// 输出错误提示信息，最后一个参数就是传递的param
		sl.ReportError(su.RePassword, "re_password", "RePassword", "eqfield", "填入的password")
	}
}

// CustomFunc  自定义字段级别校验方法,验证日期要在当前日期后
func CustomFunc(fl validator.FieldLevel) bool {
	date, err := time.Parse("2006-01-02", fl.Field().String())
	fmt.Println("获取到的日期")
	fmt.Println(date)
	if err != nil {
		return false
	}
	if date.Before(time.Now()) {
		return false
	}
	// return true
	return false
}

// CheckName 自定义验证函数
func CheckName(fl validator.FieldLevel) bool {
	if fl.Field().String() != "roottom" {
		return false
	}
	return true
}
```

#### VarField 验证变量字段
```
var b = "true"
err = verify.VarField(b, "bool") 

var i = "100"
err = verify.VarField(i, "number,gt=1,lt=101")

var f = "100.132"
err = verify.VarField(f, "numeric,gte=100,lte=1000")

var str = "abcdef"
err = verify.VarField(str, "string,min=4,max=10")

var arr = []string{"a", "b", "c"}
err = verify.VarField(arr, "len=3,max=5")

var map1 = map[string]string{"a": "1", "b": "2", "c": "3"}
err = verify.VarField(map1, "len=3,max=5")

var timeStr = time.Now().Format("2006-01-02  15:04:05")
err = verify.VarField(timeStr, "datetime=2006-01-02 15:04:05")
```
#### VarWithValue
```
s1 := "abc"
s2 := "cda"
err := verify.VarWithValue(s1, s2, "eqfield")
```
#### VarStruct 验证结构体字段
```
type User struct {
    Username string `json:"username" validate:"required"`
    Email    string `json:"email" validate:"required,email"`
    Age      uint8  `json:"age" validate:"gte=0,lte=120"`
}

user := User{
    Username: "example",
    Email:    "invalidemail",
    Age:      121,
}

err := verify.VarStruct(user) 

```
#### VarMap 验证map字段
```
user := map[string]interface{} {
        "name": "hdddccccc",
        "emain": "hddd@google.com",
    }

rules := map[string]interface{} {
    "name": "required,min=8,max=15",
    "email": "omitempty,email",
}
err := verify.VarMap(user, rules) // 此处 err 为 map[string]any   类型
if err!= nil {
    fmt.Println(err.Error())
}


// 集合嵌套验证
data := map[string]interface{} {
        "name": "ddkalsj",
        "email": "djsta@as.com",
        "details": map[string]interface{}{
            "contact_address": map[string]interface{}{
                "province": "湖南",
                "city":     "长沙",
            },
            "age": 18,
            "phones": []map[string]interface{}{
                {
                    "number": "11-111-1111",
                    "remark": "home",
                },
                {
                    "number": "22-222-2222",
                    "remark": "work",
                },
            },
        },
    }

rules := map[string]interface{}{
        "name":  "min=4,max=15",
        "email": "required,email",
        "details": map[string]interface{}{
            "contact_address": map[string]interface{}{
                "province": "required",
                "city":     "required",
            },
            "age": "numeric,min=18",
            "phones": map[string]interface{}{
                "number": "required,min=4,max=32",
                "remark": "required,min=1,max=32",
            },
        },
    }

err := verify.VarMap(data, rules) // 此处 err 为 map[string]any   类型
if err!= nil {
    fmt.Println(err.Error())
}
```


### 错误处理
```
ErrorInfo | FieldError 处理变量字段错误
StructErr 处理结构体字段错误
MapErr 处理map字段错误

返回 goerr.Error 类型
github.com/gtkit/goerr
```
### 验证器
```
多个验证标签之间符号说明
逗号（,）：多个验证标签分隔符。例如：validate:"required,email
横线（-）：跳过该字段不验证
竖线（|）：使用多个验证标记，但只需满足其中一个即可
omitempty：如果字段未设置，则忽略该符号后面的校验
dive：表示深入一层验证，用于切片与集合（slice、map）
```
### 常见网络相关tag
```
1. fqdn：完全限定域名，例如：www.baddu.com
2. hostname：主机名称
3. hostname_port：主机端口
4. ip：ip地址，包括ipv4和ipv6
5. ipv4
6. ipv6
7. mac：mac地址
8. uri：统一资源标识符
9. url：统一资源定位符
10. url_encode

```
### 常见字符串tag
```
1. alpha：字母字符串
2. alphanum ：字母与数字字符串
3. alphaunicode：字母和Unicode字符
4. alphanumunicode：字母、数字和Unicode字符
5. ascii：ascii字符串
6. boolean：布尔类型字符
7. contains：字符串包含指定内容，例如：contains=www，表示被校验字符串应包含“www”
8. endsnotwith：字符串不以指定内容结尾，例如：endsnotwith=www
9. startsnotwith：字符串不以指定内容开头
10. endswith：字符串以指定内容结尾，例如：endswith=com
11. startswith：字符串以指定内容开头
12. excludes：字符串不包含指定内容，与contains相反
13. lowercase：字符串全小写
14. uppercase：字符串全大写
15. multibyte：多字节字符串
16. number：数字类型，数字如："12345678"
17. numeric：数值类型，数字如："123.345"

```
### 常见格式相关tag
```
1. base64：base64字符串
2. base64url：base64URL字符串
3. credit_card：信用卡号
4. datetime：根据给定时间格式验证，例如：datetime=2006-01-02 15:04:05
5. e164：手机号码验证，手机号码格式如：+8613800138000
6. email：电子邮箱地址
7. hexadecimal：16进制字符串
8. json：json字符串
9. jwt：jwt字符串
10. latitude：纬度
11. longitude：经度
12. timezone：时区，时区如：Asia/Shanghai
13. iscolor：是否为颜色
14. country_code：国家代码

```
### 比较相关tag
```
lte：小于等于参数值，validate:"lte=3"  (小于等于3)

gte：大于等于参数值，validate:"lte=120,gte=0" (大于等于0小于等于120)

lt：小于参数值，validate:"lt=3" (小于3)

gt：大于参数值，validate:"lt=120,gt=0" (大于0小于120)

len：等于参数值，validate:"len=2"

max：最大值，小于等于参数值，validate:"max=20" (小于等于20)

min：最小值，大于等于参数值，validate:"min=2,max=20" (大于等于2小于等于20)

ne：不等于，validate:"ne=2" (不等于2)

oneof：只能是列举出的值其中一个，这些值必须是数值或字符串，以空格分隔，如果字符串中有空格，将字符串用单引号包围，validate:"oneof=red green"


```
### 字段相关tag
```
1. eqcsfield：字段与其他字段相等
2. eqfield：字段与其他字段相等
3. fieldcontains：字段包含其他字段值
4. fieldexcludes：字段不包含其他字段值
5. gtcsfield：字段大于其他字段值
6. gtecsfield：字段大于等于其他字段值
7. gtefield：字段大于等于其他字段值
8. gtfield：字段大于其他字段值
9. ltcsfield：字段小于其他字段值
10. ltecsfield：字段小于等于其他字段值
11. ltefield：字段小于等于其他字段值
12. ltfield：字段小于其他字段值
13. necsfield：字段不等于其他字段
14. nefield：字段不等于其他字段
注：字段相关tag，包括两种情况，
1. tag包含cs：表示与对象内部其他对象字段的字段比较。
2. tag不包含cs：表示与对象内部其他基础类型字段比较

```
### 其他tag
```
1. dir：目录
2. file：文件路径
3. isdefault：验证值是否为默认值（零值），相当于 eq=零值
4. len：长度，可用于字符串、切片、集合
5. max：最大值、长度，可用于数字、字符串、切片、集合
6. min：最小值、长度，可用于数字、字符串、切片、集合
7. oneof：指定内容中的某一个值，例如：oneof=red green，那么只有取值为 red或green才能验证
通过
8. required：必填
9. required_if：满足条件必填，例如：required_if=Field1 test，当Field1=test，则必填
10. required_unless：除指定条件外，其他情况必填，例如：required_unless=Field1 test，当
Field1!=test，则必填
11. required_with：如果指定的字段有值，则必填，例如：required_with=Field1，当Field1不为零
值，则必填
12. required_with_all：如果指定的字段都有值，则必填，例如：required_with_all=Field1 Field2，
当Field1 Field2 两个字段都不为零值，则必填
13. required_without：如果指定的字段没有值，则必填，例如：required_without=Field1，当Field1
为零值，则必填
14. required_without_all：如果指定的字段都没有值，则必填，例如：required_without_all=Field1
Field2，当Field1 Field2 两个字段都为零值，则必填
15. excluded_with、excluded_with_all、excluded_without、excluded_without_all 与required相
反，表示满足某些条件，必须不填（即必须为零值）


```
