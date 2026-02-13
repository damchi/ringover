package tests

import (
	"os"
	"testing"

	"ringover/pkg/translator"

	"github.com/gin-gonic/gin"
)

const translationFolder = "../../../../../pkg/translator/translation"

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	translator.InitTranslator(translator.Config{
		TranslationFolder:  translationFolder,
		SupportedLanguages: []string{translator.LanguageFr, translator.LanguageEn},
	})
	os.Exit(m.Run())
}
