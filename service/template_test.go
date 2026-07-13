package service

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"axent.pl/resq/dto"
	"axent.pl/resq/model"
)

func TestNewTemplateServiceParsesTemplates(t *testing.T) {
	if _, err := NewTemplateService(); err != nil {
		t.Fatalf("new template service: %v", err)
	}
}

func TestTemplateServiceRenderUsesPolishCatalog(t *testing.T) {
	service, err := NewTemplateService()
	if err != nil {
		t.Fatalf("new template service: %v", err)
	}

	data := &dto.UserListTemplateDTO{
		BaseTemplateDTO: &dto.BaseTemplateDTO{Title: "Users"},
		Paging: model.PagingResult{
			Number:     1,
			Size:       10,
			TotalPages: 1,
		},
	}
	var output bytes.Buffer
	if err := service.Render(httptest.NewRequest("GET", "/users", nil), &output, "users_list", TemplateVariantStandard, data); err != nil {
		t.Fatalf("render template: %v", err)
	}

	if data.Lang != "pl" {
		t.Fatalf("language = %q, want %q", data.Lang, "pl")
	}
	for _, translated := range []string{"<html lang=\"pl\">", "Uzytkownicy", "Strona 1 z 1 - razem 0"} {
		if !strings.Contains(output.String(), translated) {
			t.Errorf("rendered output does not contain %q", translated)
		}
	}
}
