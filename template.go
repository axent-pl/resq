package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"time"
)

type ListPageData struct {
	CSRFToken   string
	CurrentUser string
	Filters     struct {
		Author string
		Day    string
	}
	Authors    []string
	Reports    []Report
	Pagination struct {
		Page, TotalPages int
	}
}

type DetailPageData struct {
	CSRFToken   string
	CurrentUser string
	Report      Report
}

type FormPageData struct {
	CSRFToken   string
	CurrentUser string
	Action      string // URL to submit the form to
	Mode        string // "create", "edit", "new-version"
	Report      Report
	Errors      map[string]string
}

var (
	templateService *template.Template
	pages           map[string]*template.Template
)

func ExecuteTemplate(wr io.Writer, page string, data any) error {
	if tpl, ok := pages[page]; ok {
		return tpl.ExecuteTemplate(wr, page, data)
	}
	t := template.Must(templateService.Clone())
	t = template.Must(t.ParseFiles(fmt.Sprintf("templates/%s.tmpl", page)))
	pages[page] = t
	return pages[page].ExecuteTemplate(wr, page, data)
}

func init() {
	var err error

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"formatDate": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return t.Format("2006-01-02T15:04")
		},
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
	}

	templateService, err = template.New("").Funcs(funcMap).ParseGlob("templates/*.tmpl")
	if err != nil {
		log.Fatalf("template parse error: %v", err)
	}

	pages = map[string]*template.Template{}
}
