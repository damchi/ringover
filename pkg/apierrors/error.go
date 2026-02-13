package apierrors

import (
	"fmt"
	"ringover/pkg/translator"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"go.uber.org/zap"
)

// JsonErr represents the JSON structure for apierrors.
type JsonErr struct {
	ErrDetails Err `json:"error"`
}

// Err represents the error with a code and message.
type Err struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface for JsonErr.
func (e JsonErr) Error() string {
	return fmt.Sprintf("Code: %d, Message: %s", e.ErrDetails.Code, e.ErrDetails.Message)
}

// CreateError generates a JsonErr with a translated message.
func CreateError(code int, msgKey string, lang string) JsonErr {
	message := GetTransErrorMsg(msgKey, lang)
	return JsonErr{ErrDetails: Err{code, message}}
}

// GetTransErrorMsg retrieves the translated error message.
func GetTransErrorMsg(msgKey string, lang string) string {
	l := i18n.NewLocalizer(translator.Translator, lang, "en")
	m := i18n.LocalizeConfig{}
	m.MessageID = msgKey
	msg, err := l.Localize(&m)
	if err != nil {
		zap.L().Warn("translation not found", zap.String("lang", lang), zap.String("message_id", msgKey), zap.Error(err))
		return msgKey
	}
	return msg
}
