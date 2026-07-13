package dto

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

func UnmarshalDTO[T any](r *http.Request) (T, error) {
	var d T
	if r == nil {
		return d, ValidationError{Errors: map[string][]string{"request": {"is required"}}}
	}
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return d, ValidationError{Errors: map[string][]string{"form": {err.Error()}}}
		}
	} else {
		if err := r.ParseForm(); err != nil {
			return d, ValidationError{Errors: map[string][]string{"form": {err.Error()}}}
		}
	}

	errs := make(map[string][]string)
	value := reflect.ValueOf(&d).Elem()
	unmarshalFormValue(value, "", r.Form, errs)
	if len(errs) > 0 {
		return d, ValidationError{Errors: errs}
	}

	return d, nil
}

func unmarshalFormValue(value reflect.Value, prefix string, form url.Values, errs map[string][]string) {
	value = dereferenceForSet(value)
	if !value.IsValid() {
		return
	}

	if value.Kind() != reflect.Struct || isTimeValue(value) {
		setScalarFormValue(value, prefix, form, errs)
		return
	}

	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := valueType.Field(i)
		if fieldType.PkgPath != "" {
			continue
		}

		formName := formFieldName(fieldType)
		if formName == "-" {
			continue
		}
		fieldPath := joinFormPath(prefix, formName)
		fieldValue := value.Field(i)

		if isStructLikeField(fieldValue) {
			unmarshalFormValue(fieldValue, fieldPath, form, errs)
			continue
		}

		if isStructSlice(fieldValue) {
			setStructSliceFormValue(fieldValue, fieldPath, form, errs)
			continue
		}

		setScalarFormValue(fieldValue, fieldPath, form, errs)
	}
}

func setStructSliceFormValue(value reflect.Value, path string, form url.Values, errs map[string][]string) {
	indices := formSliceIndices(path, form)
	if len(indices) == 0 {
		return
	}

	slice := reflect.MakeSlice(value.Type(), len(indices), len(indices))
	for pos, idx := range indices {
		item := slice.Index(pos)
		unmarshalFormValue(item, fmt.Sprintf("%s[%d]", path, idx), form, errs)
	}

	value.Set(slice)
}

func setScalarFormValue(value reflect.Value, path string, form url.Values, errs map[string][]string) {
	if path == "" {
		return
	}

	values, ok := form[path]
	if !ok || len(values) == 0 {
		return
	}
	raw := values[0]
	if value.Kind() == reflect.Pointer {
		if raw == "" {
			return
		}
		pointer := reflect.New(value.Type().Elem())
		setScalarFormValue(pointer.Elem(), path, url.Values{path: []string{raw}}, errs)
		if len(errs[path]) > 0 {
			return
		}
		value.Set(pointer)
		return
	}

	if !value.CanSet() {
		return
	}

	if isTimeValue(value) {
		parsed, err := parseFormTime(raw)
		if err != nil {
			addValidationError(errs, path, "must be a valid date/time")
			return
		}
		value.Set(reflect.ValueOf(parsed))
		return
	}

	switch value.Kind() {
	case reflect.String:
		value.SetString(raw)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			addValidationError(errs, path, "must be a valid boolean")
			return
		}
		value.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(raw, 10, value.Type().Bits())
		if err != nil {
			addValidationError(errs, path, "must be a valid integer")
			return
		}
		value.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		parsed, err := strconv.ParseUint(raw, 10, value.Type().Bits())
		if err != nil {
			addValidationError(errs, path, "must be a valid unsigned integer")
			return
		}
		value.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(raw, value.Type().Bits())
		if err != nil {
			addValidationError(errs, path, "must be a valid number")
			return
		}
		value.SetFloat(parsed)
	default:
		addValidationError(errs, path, "has unsupported field type "+value.Type().String())
	}
}

func parseFormTime(raw string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
		"2006/01/02",
	}

	var lastErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}

	return time.Time{}, lastErr
}

func dereferenceForSet(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			if !value.CanSet() {
				return reflect.Value{}
			}
			value.Set(reflect.New(value.Type().Elem()))
		}
		value = value.Elem()
	}
	return value
}

func isStructLikeField(value reflect.Value) bool {
	value = unwrapValue(value)
	return value.IsValid() && value.Kind() == reflect.Struct && !isTimeValue(value)
}

func isStructSlice(value reflect.Value) bool {
	value = unwrapValue(value)
	if !value.IsValid() || value.Kind() != reflect.Slice {
		return false
	}

	itemType := value.Type().Elem()
	if itemType.Kind() == reflect.Pointer {
		itemType = itemType.Elem()
	}

	return itemType.Kind() == reflect.Struct && itemType != reflect.TypeOf(time.Time{})
}

func formSliceIndices(path string, form url.Values) []int {
	prefix := path + "["
	seen := map[int]struct{}{}
	for key := range form {
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		rest := strings.TrimPrefix(key, prefix)
		end := strings.Index(rest, "]")
		if end < 0 {
			continue
		}

		idx, err := strconv.Atoi(rest[:end])
		if err != nil || idx < 0 {
			continue
		}
		seen[idx] = struct{}{}
	}

	indices := make([]int, 0, len(seen))
	for idx := range seen {
		indices = append(indices, idx)
	}
	sort.Ints(indices)

	return indices
}

func formFieldName(field reflect.StructField) string {
	formTag := field.Tag.Get("form")
	if formTag == "-" {
		return "-"
	}
	if name, _, _ := strings.Cut(formTag, ","); name != "" {
		return name
	}

	jsonTag := field.Tag.Get("json")
	if jsonTag == "-" {
		return "-"
	}
	if name, _, _ := strings.Cut(jsonTag, ","); name != "" {
		return name
	}

	return field.Name
}

func joinFormPath(prefix string, name string) string {
	if name == "" {
		return prefix
	}
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}
