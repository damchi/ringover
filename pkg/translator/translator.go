package translator

import (
	"fmt"
	"os"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml/v2"
	"go.uber.org/zap"
	"golang.org/x/text/language"
)

var Translator *i18n.Bundle

type Config struct {
	TranslationFolder  string
	SupportedLanguages []string // List of supported languages
}

const (
	LanguageFr = "fr"
	LanguageEn = "en"
	// Add more language constants as needed, e.g., "de", "es", etc.
)

func InitTranslator(cfg Config) {
	Translator = i18n.NewBundle(language.English)
	Translator.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	var err error

	// List files in the translation folder
	lstFiles, err := os.ReadDir(cfg.TranslationFolder)
	if err != nil {
		zap.L().Error("failed to list translation folder", zap.String("folder", cfg.TranslationFolder), zap.Error(err))
		return
	}

	// Load all translation files
	for _, f := range lstFiles {
		// Check if the file is a valid language translation file
		if f.IsDir() {
			continue
		}
		filepath := fmt.Sprintf("%s/%s", cfg.TranslationFolder, f.Name())

		// Load the message file into the Translator bundle
		_, err := Translator.LoadMessageFile(filepath)
		if err != nil {
			zap.L().Warn("failed to load translation file", zap.String("file", f.Name()), zap.Error(err))
		}
	}
}
