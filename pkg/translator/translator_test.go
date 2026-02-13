package translator_test

import (
	"os"
	"path/filepath"
	"testing"

	"ringover/pkg/translator"

	"github.com/nicksnyder/go-i18n/v2/i18n"
)

func TestInitTranslator_LoadsMessages(t *testing.T) {
	// Create a temporary directory for translations
	dir := t.TempDir()

	// Write a test en.toml file
	enFile := filepath.Join(dir, "en.toml")
	content := []byte(`
getAllUser = "Could not retrieve users."
validationFailedId = "Validation Id failed."
failFindUserById = "Fail to find the user."
hello = "Hello english"
`)
	if err := os.WriteFile(enFile, content, 0644); err != nil {
		t.Fatalf("failed to write en.toml: %v", err)
	}

	// Initialize translator with the temp dir
	translator.InitTranslator(translator.Config{
		TranslationFolder:  dir,
		SupportedLanguages: []string{translator.LanguageEn, translator.LanguageFr},
	})

	localizer := i18n.NewLocalizer(translator.Translator, translator.LanguageEn)

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: "hello",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	expected := "Hello english"
	if msg != expected {
		t.Errorf("expected %q, got %q", expected, msg)
	}
}

func TestInitTranslator_InvalidFolder(t *testing.T) {
	translator.InitTranslator(translator.Config{
		TranslationFolder:  "/path/does/not/exist",
		SupportedLanguages: []string{translator.LanguageEn},
	})
}

func TestTranslatorConstants(t *testing.T) {
	if translator.LanguageEn != "en" {
		t.Errorf("expected LanguageEn to be 'en'")
	}
	if translator.LanguageFr != "fr" {
		t.Errorf("expected LanguageFr to be 'fr'")
	}
}
