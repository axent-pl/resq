package service

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"

	"axent.pl/resq/dto"
	"axent.pl/resq/i18n"
)

//go:embed templates/*.html
var templateFiles embed.FS

type TemplateVariant string

const (
	TemplateVariantStandard TemplateVariant = "standard"
	TemplateVariantXHR      TemplateVariant = "xhr"
)

type TemplateService struct {
	templates *template.Template
}

func NewTemplateService() (*TemplateService, error) {
	catalog := i18n.NewCatalog("en", "pl")
	templates, err := template.New("").
		Funcs(template.FuncMap{
			"formatTime":     formatTemplateTime,
			"fieldErrors":    fieldErrors,
			"fieldHasErrors": fieldHasErrors,
			"statusText":     statusText,
			"coalesce":       coalesce,
			"dict": func(values ...any) (map[string]any, error) {
				if len(values)%2 != 0 {
					return nil, fmt.Errorf("invalid dict call: need even number of args")
				}
				m := make(map[string]any, len(values)/2)
				for i := 0; i < len(values); i += 2 {
					key, ok := values[i].(string)
					if !ok {
						return nil, fmt.Errorf("dict keys must be strings")
					}
					m[key] = values[i+1]
				}
				return m, nil
			},
			"_": func(language, message string, args ...any) string {
				return catalog.Get(language, message, args...)
			},
		}).
		ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &TemplateService{templates: templates}, nil
}

func (s *TemplateService) Render(r *http.Request, w io.Writer, page string, variant TemplateVariant, data dto.TemplaterDTO) error {
	data.SetLang(i18n.DefaultLanguage)
	templateName := fmt.Sprintf("%s.%s", "view", variant)
	t := template.Must(s.templates.Clone())
	t = template.Must(t.ParseFS(
		templateFiles,
		"templates/layout.html",
		fmt.Sprintf("templates/%s.html", page),
	))

	return t.ExecuteTemplate(w, templateName, data)
}

func coalesce(s *string, defaultString ...string) string {
	if s == nil {
		if len(defaultString) > 0 {
			return defaultString[0]
		}
		return ""
	}
	return *s
}

func formatTemplateTime(value any, format ...string) string {
	switch v := value.(type) {
	case time.Time:
		return formatTemplateTimeTyped(v, format...)
	case *time.Time:
		if v == nil {
			return ""
		}
		return formatTemplateTimeTyped(*v, format...)
	default:
		return ""
	}
}

func formatTemplateTimeTyped(value time.Time, format ...string) string {
	if value.IsZero() {
		return ""
	}
	if len(format) > 0 {
		return value.Format(format[0])
	}
	return value.Format("2006-01-02 15:04")
}

func statusText(status string) string {
	switch status {
	case "open":
		return i18n.Mark("Open")
	case "in_progress":
		return i18n.Mark("In progress")
	case "handed_over":
		return i18n.Mark("Handed over")
	case "closed":
		return i18n.Mark("Closed")
	default:
		return status
	}
}

func fieldHasErrors(errs map[string][]string, field string) bool {
	if errs == nil {
		return false
	}
	return len(errs[field]) > 0
}

func fieldErrors(errs map[string][]string, field string) []string {
	if errs == nil {
		return nil
	}
	return errs[field]
}
