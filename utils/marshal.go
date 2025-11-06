package utils

import (
	"errors"
	"net/http"
	"reflect"
	"strconv"
	"time"
)

func Unmarshal(r *http.Request, s interface{}) error {
	// Ensure s is a pointer to a struct
	val := reflect.ValueOf(s)
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return errors.New("Unmarshal requires a pointer to a struct")
	}
	val = val.Elem()
	typ := val.Type()

	// Populate struct fields
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Process formField tag
		if formFieldName := fieldType.Tag.Get("form"); formFieldName != "" {
			if formValue := r.PostFormValue(formFieldName); formValue != "" {
				err := setFieldValue(field, formValue)
				if err != nil {
					return err
				}
			}
		}

	}

	return nil
}

// setFieldValue assigns a string value to a field, converting to the appropriate type.
func setFieldValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return errors.New("cannot set value to field")
	}

	// Handle time.Time separately since Kind() will be reflect.Struct
	if field.Type() == reflect.TypeOf(time.Time{}) {
		// Try parsing using common layouts
		layouts := []string{
			time.RFC3339,          // "2006-01-02T15:04:05Z07:00"
			time.RFC3339Nano,      // "2006-01-02T15:04:05.999999999Z07:00"
			"2006-01-02 15:04:05", // common SQL datetime format
			"2006-01-02T15:04",
			"2006-01-02", // date-only format
		}

		var parsed time.Time
		var err error
		for _, layout := range layouts {
			parsed, err = time.Parse(layout, value)
			if err == nil {
				field.Set(reflect.ValueOf(parsed))
				return nil
			}
		}
		return errors.New("invalid time value: " + value)
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			field.SetInt(intValue)
		} else {
			return errors.New("invalid integer value: " + value)
		}
	case reflect.Float32, reflect.Float64:
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			field.SetFloat(floatValue)
		} else {
			return errors.New("invalid float value: " + value)
		}
	case reflect.Bool:
		if boolValue, err := strconv.ParseBool(value); err == nil {
			field.SetBool(boolValue)
		} else {
			return errors.New("invalid boolean value: " + value)
		}
	default:
		return errors.New("unsupported field type")
	}

	return nil
}
