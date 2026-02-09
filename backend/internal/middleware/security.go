package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeaders returns a middleware that adds security headers
//
// 実装は公開リポジトリから省略しています。
// 以下のセキュリティヘッダーを設定:
// - X-Content-Type-Options（MIME sniffing 防止）
// - Content-Security-Policy（クリックジャッキング防止）
// - Referrer-Policy（リファラー制御）
// - Strict-Transport-Security（HSTS、Cloud Run の X-Forwarded-Proto 対応）
// - Permissions-Policy（不要なブラウザ機能の無効化）
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Security headers omitted from public repository.
		c.Next()
	}
}
