package i18n

import (
	"embed"
	"strings"

	"github.com/leonelquinteros/gotext"
)

const (
	DefaultLanguage = "pl"
	domain          = "resq"
)

//go:embed locales/*/LC_MESSAGES/*.po
var localeFiles embed.FS

type Catalog struct {
	locales map[string]*gotext.Locale
}

// Mark identifies a message for extraction when it is translated indirectly.
// It deliberately returns the message unchanged at runtime.
func Mark(message string) string {
	return message
}

func NewCatalog(languages ...string) *Catalog {
	locales := make(map[string]*gotext.Locale, len(languages))
	for _, language := range languages {
		locale := gotext.NewLocaleFSWithPath(language, localeFiles, "locales")
		locale.AddDomain(domain)
		locales[language] = locale
	}
	return &Catalog{locales: locales}
}

func (c *Catalog) Get(language, message string, args ...any) string {
	locale := c.locales[language]
	if locale == nil {
		if separator := strings.IndexAny(language, "_-"); separator > 0 {
			locale = c.locales[language[:separator]]
		}
	}
	if locale == nil {
		return gotext.FormatString(message, args...)
	}
	return locale.Get(message, args...)
}
