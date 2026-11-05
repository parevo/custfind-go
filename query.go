package custfind

import (
	"net/url"
	"reflect"
	"strconv"
)

// encodeQuery turns a params struct with `url:"name"` tags into query values,
// leaving out zero values and nil pointers.
func encodeQuery(params any) url.Values {
	q := url.Values{}
	v := reflect.ValueOf(params)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return q
		}
		v = v.Elem()
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Tag.Get("url")
		if name == "" {
			continue
		}
		f := v.Field(i)
		if f.Kind() == reflect.Pointer {
			if f.IsNil() {
				continue
			}
			f = f.Elem()
		} else if f.IsZero() {
			continue
		}
		switch f.Kind() {
		case reflect.String:
			q.Set(name, f.String())
		case reflect.Int, reflect.Int64, reflect.Int32:
			q.Set(name, strconv.FormatInt(f.Int(), 10))
		case reflect.Float64, reflect.Float32:
			q.Set(name, strconv.FormatFloat(f.Float(), 'f', -1, 64))
		case reflect.Bool:
			q.Set(name, strconv.FormatBool(f.Bool()))
		}
	}
	return q
}
