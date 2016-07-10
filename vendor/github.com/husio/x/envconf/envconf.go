package envconf

import (
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Load assign values from settings mapping into given structure.
func Load(dest interface{}, settings map[string]string) error {
	s := reflect.ValueOf(dest)
	if s.Kind() != reflect.Ptr {
		return fmt.Errorf("expected pointer to struct, got %T", dest)
	}

	s = s.Elem()
	if s.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %T", dest)
	}

	splitListMu.Lock()
	defer splitListMu.Unlock()

	var errs ParseErrors
	tp := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if !f.CanSet() {
			continue
		}
		tags := strings.Split(tp.Field(i).Tag.Get("envconf"), ",")

		var name string
		if len(tags) > 0 && tags[0] != "" {
			name = tags[0]
		} else {
			name = convertName(tp.Field(i).Name)
		}

		required := false
		if len(tags) > 1 && contains(tags[1:], "required") {
			required = true
		}

		value, ok := settings[name]
		if !ok {
			if required {
				errs = append(errs, &ParseError{
					Field: tp.Field(i).Name,
					Name:  name,
					Value: value,
					Err:   errors.New("required"),
					Kind:  f.Kind(),
				})
			}
			continue
		}

		switch f.Kind() {
		case reflect.String:
			f.SetString(value)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			var intValue int64
			var err error
			if value != "" {
				intValue, err = strconv.ParseInt(value, 0, f.Type().Bits())
			}
			if err != nil {
				errs = append(errs, &ParseError{
					Field: tp.Field(i).Name,
					Name:  name,
					Value: value,
					Err:   err,
					Kind:  f.Kind(),
				})
				continue
			}
			f.SetInt(intValue)
		case reflect.Bool:
			var boolValue bool
			var err error
			if value != "" {
				boolValue, err = strconv.ParseBool(value)
			}
			if err != nil {
				errs = append(errs, &ParseError{
					Field: tp.Field(i).Name,
					Name:  name,
					Value: value,
					Err:   err,
					Kind:  f.Kind(),
				})
				continue
			}
			f.SetBool(boolValue)
		case reflect.Float32, reflect.Float64:
			var floatValue float64
			var err error
			if value != "" {
				floatValue, err = strconv.ParseFloat(value, f.Type().Bits())
			}
			if err != nil {
				errs = append(errs, &ParseError{
					Field: tp.Field(i).Name,
					Name:  name,
					Value: value,
					Err:   err,
					Kind:  f.Kind(),
				})
				continue
			}
			f.SetFloat(floatValue)
		case reflect.Slice:
			if value == "" {
				continue
			}
			vals := splitList(value)

			switch f.Type().Elem().Kind() {
			case reflect.String:
				for _, v := range vals {
					f.Set(reflect.Append(f, reflect.ValueOf(v)))
				}
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				for _, v := range vals {
					intValue, err := strconv.ParseInt(v, 0, f.Type().Elem().Bits())
					if err != nil {
						errs = append(errs, &ParseError{
							Field: tp.Field(i).Name,
							Name:  name,
							Value: v,
							Err:   err,
							Kind:  f.Kind(),
						})
						continue
					}
					iv := reflect.New(f.Type().Elem()).Elem()
					iv.SetInt(intValue)
					f.Set(reflect.Append(f, iv))
				}
			case reflect.Bool:
				for _, v := range vals {
					boolValue, err := strconv.ParseBool(v)
					if err != nil {
						errs = append(errs, &ParseError{
							Field: tp.Field(i).Name,
							Name:  name,
							Value: v,
							Err:   err,
							Kind:  f.Kind(),
						})
						continue
					}
					f.Set(reflect.Append(f, reflect.ValueOf(boolValue)))
				}
			case reflect.Float32, reflect.Float64:
				for _, v := range vals {
					floatValue, err := strconv.ParseFloat(v, f.Type().Elem().Bits())
					if err != nil {
						errs = append(errs, &ParseError{
							Field: tp.Field(i).Name,
							Name:  name,
							Value: v,
							Err:   err,
							Kind:  f.Kind(),
						})
						continue
					}
					fv := reflect.New(f.Type().Elem()).Elem()
					fv.SetFloat(floatValue)
					f.Set(reflect.Append(f, fv))
				}
			default:
				log.Printf("unsupported type: %s", f.Kind())
			}
		default:
			log.Printf("unsupported type: %s", f.Kind())
		}

	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

func Must(err error) {
	if err == nil {
		return
	}
	if errs, ok := err.(ParseErrors); !ok {
		fmt.Fprintf(os.Stderr, "cannot parse config: %s\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "cannot parse config")
		for _, err := range errs {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", err.Name, err.Err)
		}
	}
	os.Exit(1)
}

var (
	splitListMu sync.Mutex
	splitList   = func(s string) []string {
		return strings.Split(s, ";")
	}
)

// Separator set list separator function. By default, values are expected to be
// separated by ;
func SeparatorFunc(f func(string) []string) {
	splitListMu.Lock()
	splitList = f
	splitListMu.Unlock()
}

type ParseErrors []*ParseError

func (e ParseErrors) Error() string {
	switch n := len(e); n {
	case 0:
		return ""
	case 1:
		return "1 parse error"
	default:
		return fmt.Sprintf("%d parse errors", n)
	}
}

type ParseError struct {
	Field string
	Name  string
	Value string
	Err   error
	Kind  reflect.Kind
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("cannot parse %s: %s", e.Field, e.Err)
}

// LoadEnv load configuration using environment variables.
func LoadEnv(dest interface{}) error {
	env := make(map[string]string)
	for _, kv := range os.Environ() {
		pair := strings.SplitN(kv, "=", 2)
		if len(pair) != 2 {
			continue
		}
		env[pair[0]] = pair[1]
	}
	return Load(dest, env)
}

func contains(arr []string, s string) bool {
	for _, el := range arr {
		if el == s {
			return true
		}
	}
	return false
}

func convertName(s string) string {
	s = conv1.ReplaceAllStringFunc(s, func(val string) string {
		return val[:1] + "_" + val[1:]
	})
	s = conv2.ReplaceAllStringFunc(s, func(val string) string {
		return val[:1] + "_" + val[1:]
	})
	return strings.ToUpper(s)
}

var (
	conv1 = regexp.MustCompile(`.([A-Z][a-z]+)`)
	conv2 = regexp.MustCompile(`([a-z0-9])([A-Z])`)
)
