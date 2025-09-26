package error

import (
	"fmt"
	"go-notify/model"
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func GinErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		errs := c.Errors
		if len(errs) > 0 {
			for _, e := range errs {
				switch e.Type {
				case gin.ErrorTypeBind:
					errArray, ok := e.Err.(validator.ValidationErrors)
					if !ok {
						writeError(c, e.Error())
						return
					}
					var stringErrors []string
					for _, err := range errArray {
						stringErrors = append(stringErrors, validationErrorToText(err))
					}
					writeError(c, strings.Join(stringErrors, "; "))
				default:
					writeError(c, e.Err.Error()) // 其他类型的错误直接返回

				}
			}
		}
	}
}

// 用于将验证错误转换为可读文本
func validationErrorToText(e validator.FieldError) string {
	// 获取错误字段名
	runes := []rune(e.Field())           // 转出为rune切片以支持中文
	runes[0] = unicode.ToLower(runes[0]) // 首字母小写
	fieldName := string(runes)           // 转回字符串

	switch e.Tag() {
	case "required":
		return fmt.Sprintf("Field '%s' is required", fieldName)
	case "max":
		return fmt.Sprintf("Field '%s' must be less or equal to %s", fieldName, e.Param())
	case "min":
		return fmt.Sprintf("Field '%s' must be more or equal to %s", fieldName, e.Param())
	}
	return fmt.Sprintf("Field '%s' is not valid", fieldName)

}

func writeError(ctx *gin.Context, errString string) {
	if ctx.Writer.Status() == http.StatusOK {
		return
	}
	status := ctx.Writer.Status()
	ctx.JSON(status, &model.Error{Error: http.StatusText(status), ErrorCode: status, ErrorDescription: errString})
}
