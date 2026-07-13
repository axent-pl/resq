package dto

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type ValidationError struct {
	Errors map[string][]string `json:"errors"`
}

type ValidationFunc func(value any) bool

func (e ValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation failed: %d field(s) invalid", len(e.Errors))
}

func ValidateDTO[T any](d T, funcs ...map[string]ValidationFunc) error {
	var customFuncs map[string]ValidationFunc
	if len(funcs) > 0 {
		customFuncs = funcs[0]
	}

	errs := make(map[string][]string)
	validateValue(reflect.ValueOf(d), "", errs, customFuncs)
	if len(errs) > 0 {
		return ValidationError{Errors: errs}
	}
	return nil
}

func validateValue(value reflect.Value, path string, errs map[string][]string, customFuncs map[string]ValidationFunc) {
	value = unwrapValue(value)
	if !value.IsValid() {
		return
	}

	if value.Kind() != reflect.Struct || isTimeValue(value) {
		return
	}

	valueType := value.Type()
	for i := 0; i < value.NumField(); i++ {
		fieldType := valueType.Field(i)
		if fieldType.PkgPath != "" {
			continue
		}

		fieldValue := value.Field(i)
		fieldPath := joinPath(path, fieldName(fieldType))
		validateField(fieldValue, fieldPath, fieldType.Tag.Get("validate"), errs, customFuncs)
		validateNested(fieldValue, fieldPath, errs, customFuncs)
	}
}

func validateField(value reflect.Value, path string, tag string, errs map[string][]string, customFuncs map[string]ValidationFunc) {
	if tag == "" || tag == "-" {
		return
	}

	rules := strings.Split(tag, ",")
	required := hasRule(rules, "required")
	if required && isEmptyValue(value) {
		addValidationError(errs, path, "is required")
		return
	}
	if isOptionalEmptyValue(value) {
		return
	}

	value = unwrapValue(value)
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" || rule == "required" {
			continue
		}

		name, arg, hasArg := strings.Cut(rule, "=")
		switch name {
		case "min":
			if !hasArg || !validateMin(value, arg) {
				addValidationError(errs, path, "must be at least "+arg)
			}
		case "max":
			if !hasArg || !validateMax(value, arg) {
				addValidationError(errs, path, "must be at most "+arg)
			}
		case "oneof":
			if !hasArg || !validateOneOf(value, arg) {
				addValidationError(errs, path, "must be one of: "+arg)
			}
		case "fn":
			if !hasArg || !validateFunc(value, arg, customFuncs) {
				addValidationError(errs, path, "failed custom validation: "+arg)
			}
		default:
			addValidationError(errs, path, "has unknown validation rule: "+name)
		}
	}
}

func validateNested(value reflect.Value, path string, errs map[string][]string, customFuncs map[string]ValidationFunc) {
	value = unwrapValue(value)
	if !value.IsValid() {
		return
	}

	switch value.Kind() {
	case reflect.Struct:
		if !isTimeValue(value) {
			validateValue(value, path, errs, customFuncs)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			item := unwrapValue(value.Index(i))
			if item.IsValid() && item.Kind() == reflect.Struct && !isTimeValue(item) {
				validateValue(item, fmt.Sprintf("%s[%d]", path, i), errs, customFuncs)
			}
		}
	}
}

func validateMin(value reflect.Value, arg string) bool {
	min, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return false
	}

	switch value.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return float64(value.Len()) >= min
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int()) >= min
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(value.Uint()) >= min
	case reflect.Float32, reflect.Float64:
		return value.Float() >= min
	default:
		return false
	}
}

func validateMax(value reflect.Value, arg string) bool {
	max, err := strconv.ParseFloat(arg, 64)
	if err != nil {
		return false
	}

	switch value.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return float64(value.Len()) <= max
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int()) <= max
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return float64(value.Uint()) <= max
	case reflect.Float32, reflect.Float64:
		return value.Float() <= max
	default:
		return false
	}
}

func validateOneOf(value reflect.Value, arg string) bool {
	actual := fmt.Sprint(value.Interface())
	for _, allowed := range strings.Fields(arg) {
		if actual == allowed {
			return true
		}
	}
	return false
}

func validateFunc(value reflect.Value, name string, customFuncs map[string]ValidationFunc) bool {
	if len(customFuncs) == 0 {
		return false
	}

	fn, ok := customFuncs[name]
	if !ok || fn == nil {
		return false
	}
	return fn(value.Interface())
}

func hasRule(rules []string, name string) bool {
	for _, rule := range rules {
		if strings.TrimSpace(rule) == name {
			return true
		}
	}
	return false
}

func isOptionalEmptyValue(value reflect.Value) bool {
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return true
	}
	if value.Kind() == reflect.String && value.Len() == 0 {
		return true
	}
	return false
}

func isEmptyValue(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}

	switch value.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Slice, reflect.Map:
		return value.IsNil()
	case reflect.String, reflect.Array:
		return value.Len() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Struct:
		if isTimeValue(value) {
			return value.Interface().(time.Time).IsZero()
		}
		return value.IsZero()
	default:
		return value.IsZero()
	}
}

func unwrapValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func isTimeValue(value reflect.Value) bool {
	return value.IsValid() && value.Type() == reflect.TypeOf(time.Time{})
}

func fieldName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "-" {
		return field.Name
	}
	if name, _, _ := strings.Cut(jsonTag, ","); name != "" {
		return name
	}
	return field.Name
}

func joinPath(prefix string, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

func addValidationError(errs map[string][]string, path string, message string) {
	errs[path] = append(errs[path], message)
}
