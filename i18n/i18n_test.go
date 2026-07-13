package i18n

import "testing"

func TestCatalogGet(t *testing.T) {
	catalog := NewCatalog("en", "pl")

	tests := []struct {
		name     string
		language string
		message  string
		args     []any
		want     string
	}{
		{name: "Polish", language: "pl", message: "Events", want: "Wydarzenia"},
		{name: "regional locale", language: "pl-PL", message: "Events", want: "Wydarzenia"},
		{name: "English label", language: "en", message: "initial_assessment", want: "Initial assessment"},
		{name: "formatted", language: "pl", message: "Page %d of %d - %d total", args: []any{2, 4, 37}, want: "Strona 2 z 4 - razem 37"},
		{name: "unknown language fallback", language: "de", message: "Page %d", args: []any{3}, want: "Page 3"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := catalog.Get(test.language, test.message, test.args...); got != test.want {
				t.Fatalf("Get() = %q, want %q", got, test.want)
			}
		})
	}
}
