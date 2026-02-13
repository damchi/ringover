package apierrors_test

import (
	"ringover/pkg/apierrors"
	"ringover/pkg/translator"
	"testing"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
)

func TestMain(m *testing.M) {
	// Initialize minimal translator for tests
	translator.Translator = i18n.NewBundle(language.English)
	// Add a test message
	err := translator.Translator.AddMessages(language.English, &i18n.Message{
		ID:    "test_key",
		Other: "Test message",
	})
	if err != nil {
		return
	}
	m.Run()
}

func TestCreateError_ReturnsJsonErr(t *testing.T) {
	err := apierrors.CreateError(400, "test_key", "en")
	assert.Equal(t, 400, err.ErrDetails.Code)
	assert.Equal(t, "Test message", err.ErrDetails.Message)
}

func TestGetTransErrorMsg_ReturnsTranslation(t *testing.T) {
	msg := apierrors.GetTransErrorMsg("test_key", "en")
	assert.Equal(t, "Test message", msg)
}

func TestGetTransErrorMsg_FallbackToKey(t *testing.T) {
	// No translation exists for "unknown_key"
	msg := apierrors.GetTransErrorMsg("unknown_key", "en")
	assert.Equal(t, "unknown_key", msg)
}

func TestJsonErr_ErrorMethod(t *testing.T) {
	err := apierrors.CreateError(500, "test_key", "en")
	assert.Equal(t, "Code: 500, Message: Test message", err.Error())
}
