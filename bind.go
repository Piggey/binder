package binder

import (
	"encoding"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strconv"
)

func (b *Binder) Bind(r *http.Request, obj any) error {
	err := bindReflect(r, obj)
	if err != nil {
		return err
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("io.ReadAll: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	contentType := r.Header.Get("Content-Type")
	mimetype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("mime.ParseMediaType: %w", err)
	}

	err = bindBody(mimetype, data, obj)
	if err != nil {
		return fmt.Errorf("body: %w", err)
	}

	err = b.validate.StructCtx(r.Context(), obj)
	if err != nil {
		return fmt.Errorf("validation: %w", err)
	}

	return nil
}

const (
	mimetypeApplicationJson = "application/json"
	mimetypeApplicationXml  = "application/xml"
)

func bindBody(mimetype string, data []byte, obj any) error {
	switch mimetype {
	case mimetypeApplicationJson:
		return json.Unmarshal(data, obj)
	case mimetypeApplicationXml:
		return xml.Unmarshal(data, obj)
	default:
		return fmt.Errorf("unsupported mimetype: %s", mimetype)
	}
}

const (
	tagHeader = "header"
	tagQuery  = "query"
	tagPath   = "path"
	tagCookie = "cookie"
)

func bindReflect(r *http.Request, obj any) error {
	if obj == nil {
		return fmt.Errorf("obj must be a non-nil pointer to a struct")
	}
	
	v := reflect.ValueOf(obj)
	if v.IsNil() || v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("obj must be a non-nil pointer to a struct")
	}

	v = v.Elem()
	t := v.Type()
	for i := range v.NumField() {
		fv := v.Field(i)
		ft := t.Field(i)

		if !fv.CanSet() {
			continue // skip private fields
		}

		if tag := ft.Tag.Get(tagHeader); tag != "" {
			err := setFieldSlice(fv, r.Header.Values(tag))
			if err != nil {
				return fmt.Errorf("header: %s: %w", tag, err)
			}
		}

		if tag := ft.Tag.Get(tagQuery); tag != "" {
			err := setFieldSlice(fv, r.URL.Query()[tag])
			if err != nil {
				return fmt.Errorf("query: %s: %w", tag, err)
			}
		}

		if tag := ft.Tag.Get(tagPath); tag != "" {
			err := setField(fv, r.PathValue(tag))
			if err != nil {
				return fmt.Errorf("path: %s: %w", tag, err)
			}
		}

		if tag := ft.Tag.Get(tagCookie); tag != "" {
			cookie, err := r.Cookie(tag)
			if err != nil {
				return fmt.Errorf("r.Cookie: %w", err)
			}

			err = setField(fv, cookie.Value)
			if err != nil {
				return fmt.Errorf("cookie: %s: %w", tag, err)
			}
		}
	}

	return nil
}

var (
	textUnmarshalerType   = reflect.TypeFor[encoding.TextUnmarshaler]()
	binaryUnmarshalerType = reflect.TypeFor[encoding.BinaryUnmarshaler]()
)

func setField(field reflect.Value, val string) error {
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}

		return setField(field.Elem(), val)
	}

	if val == "" {
		return nil
	}

	if field.CanAddr() {
		addr := field.Addr()

		if addr.Type().Implements(textUnmarshalerType) {
			unmarshaler := addr.Interface().(encoding.TextUnmarshaler)
			return unmarshaler.UnmarshalText([]byte(val))
		}

		if addr.Type().Implements(binaryUnmarshalerType) {
			unmarshaler := addr.Interface().(encoding.BinaryUnmarshaler)
			return unmarshaler.UnmarshalBinary([]byte(val))
		}
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(val)

	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return fmt.Errorf("strconv.ParseInt: %w", err)
		}

		field.SetInt(i)

	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		i, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return fmt.Errorf("strconv.ParseUint: %w", err)
		}

		field.SetUint(i)

	case reflect.Bool:
		b, err := strconv.ParseBool(val)
		if err != nil {
			return fmt.Errorf("strconv.ParseBool: %w", err)
		}

		field.SetBool(b)

	case reflect.Float64, reflect.Float32:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return fmt.Errorf("strconv.ParseFloat: %w", err)
		}

		field.SetFloat(f)

	default:
		return fmt.Errorf("unsupported type: %s", field.Type())
	}

	return nil
}

func setFieldSlice(field reflect.Value, vals []string) error {
	if field.Kind() != reflect.Slice {
		if len(vals) == 0 {
			return nil
		}
		return setField(field, vals[0])
	}

	slice := reflect.MakeSlice(field.Type(), 0, len(vals))

	for _, v := range vals {
		elem := reflect.New(field.Type().Elem()).Elem()

		if err := setField(elem, v); err != nil {
			return err
		}

		slice = reflect.Append(slice, elem)
	}

	field.Set(slice)
	return nil
}
