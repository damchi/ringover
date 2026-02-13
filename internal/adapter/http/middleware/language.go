package middleware

import (
	"ringover/pkg/translator"

	"github.com/gin-gonic/gin"
)

// LanguageMiddleware is a Gin language that sets the language based on the Accept-Language header.
func LanguageMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Keep language handling simple for now: use raw header value, fallback to en.
		lang := c.GetHeader("Accept-Language")
		if lang == "" {
			lang = translator.LanguageEn
		}
		c.Set("lang", lang)
		c.Next()
	}
}

func GetLang(c *gin.Context) string {
	if lang, exists := c.Get("lang"); exists {
		if s, ok := lang.(string); ok {
			return s
		}
	}
	return translator.LanguageEn
}
