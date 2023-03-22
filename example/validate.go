// @Author xiaozhaofu 2023/3/22 11:30:00
package example

import (
	"github.com/gin-gonic/gin"
)

type Params struct {
	OrderSn  string `json:"order_sn" form:"order_sn" binding:"required"`
	OutSkuSn string `json:"out_sku_sn" form:"out_sku_sn" binding:"required"`
}

func vali(c *gin.Context) {
	var p Params
	if err := c.ShouldBind(&p); err != nil {
		// 获取validator.ValidationErrors类型的errors
		// errs, ok := err.(validator.ValidationErrors)
		// if !ok {
		// 	response.Error(c, goerr.ErrParams, err)
		// 	return
		// }
		// for _, v := range verify.RemoveTopStruct(errs.Translate(verify.Trans())) {
		// 	response.Error(c, goerr.ErrValidateParams, goerr.Custom(v))
		// 	return
		// }

	}
}
