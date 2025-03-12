package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// TODO: CORS
// TODO: LOGGER

type Service struct {
	handler *http.ServeMux
}

func NewService() *Service {
	return &Service{
		handler: http.NewServeMux(),
	}
}

type Context struct {
	// Main
	W   http.ResponseWriter
	R   *http.Request
	URL url.Values
	// Session
	Session *Session // TODO
	// Readers
	Query func(v any) error
	Body  func(v any) error
	// Responses
	String func(s string) error
	JSON   func(map[string]any) error
}

func handle(h func(r *Context) error) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		req := &Context{
			// Main
			W:   w,
			R:   r,
			URL: r.URL.Query(),
			// Session
			Session: &Session{},
			// Readers
			Query: func(v any) error {
				val := reflect.ValueOf(v)
				if val.Kind() != reflect.Ptr || val.IsNil() || val.Elem().Kind() != reflect.Struct {
					return errors.New("params: argument must be a pointer to a struct")
				}

				structVal := val.Elem()
				structType := structVal.Type()

				query := r.URL.Query()
				for i := range make([]struct{}, structType.NumField()) {
					field := structType.Field(i)
					fieldVal := structVal.Field(i)

					if !fieldVal.CanSet() {
						continue
					}

					queryKey := field.Tag.Get("query")
					if queryKey == "" {
						queryKey = strings.ToLower(field.Name)
					}

					if queryVal, ok := query[queryKey]; ok && len(queryVal) > 0 {
						if err := setField(fieldVal, queryVal[0]); err != nil {
							return fmt.Errorf("failed to set field %s: %w", field.Name, err)
						}
					}
				}
				return nil
			},
			Body: func(v any) error {
				if r.Header.Get("Content-Type") != "application/json" {
					return fmt.Errorf("invalid content-type: expected 'application/json', got %q", r.Header.Get("Content-Type"))
				}

				decoder := json.NewDecoder(r.Body)
				// decoder.DisallowUnknownFields()
				if err := decoder.Decode(v); err != nil {
					return fmt.Errorf("failed to decode request body: %w", err)
				}
				return nil
			},
			// Responses
			String: func(s string) error {
				_, err := w.Write([]byte(s))
				return err
			},
			JSON: func(m map[string]any) error {
				w.Header().Set("Content-Type", "application/json")
				return json.NewEncoder(w).Encode(m)
			},
		}
		if err := h(req); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func setField(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer value %q: %w", value, err)
		}
		field.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer value %q: %w", value, err)
		}
		field.SetUint(u)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid float value %q: %w", value, err)
		}
		field.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("invalid boolean value %q: %w", value, err)
		}
		field.SetBool(b)
	default:
		return fmt.Errorf("unsupported field type: %v", field.Kind())
	}
	return nil
}

type Session struct{}

// func (s *Session) func Get() session

const (
	ANONYMOUS_SESSION = 0
	USER_SESSION      = 1
	ADMIN_SESSION     = 2
)

type Route struct {
	Path   string
	Auth   int
	GET    func(r *Context) error
	POST   func(r *Context) error
	PUT    func(r *Context) error
	PATCH  func(r *Context) error
	DELETE func(r *Context) error
}

func (s *Service) Mount(route *Route) {
	if route.Auth != ANONYMOUS_SESSION {
		//TODO: Set Session Context
	}

	if route.Path == "" {
		log.Println("Path is required to Mount Route")
		return
	}
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if route.GET != nil {
				handle(route.GET)(w, r)
				return
			}
		case http.MethodPost:
			if route.POST != nil {
				handle(route.POST)(w, r)
				return
			}
		case http.MethodPut:
			if route.PUT != nil {
				handle(route.PUT)(w, r)
				return
			}
		case http.MethodPatch:
			if route.PATCH != nil {
				handle(route.PATCH)(w, r)
				return
			}
		case http.MethodDelete:
			if route.DELETE != nil {
				handle(route.DELETE)(w, r)
				return
			}
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

	s.handler.HandleFunc(route.Path, handler)
}
