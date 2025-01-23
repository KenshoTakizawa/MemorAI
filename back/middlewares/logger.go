package middlewares

import (
    "github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
    return func(c *gin.Context) {
        // リクエストのログを記録
        c.Next()
    }
}
