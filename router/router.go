package router

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

	"github.com/brightsidedeveloper/serve/session"
	"github.com/golang-jwt/jwt/v5"
)

type Router struct {
	Handler *http.ServeMux
}

func NewRouter() *Router {
	return &Router{
		Handler: http.NewServeMux(),
	}
}

type Context struct {
	// Main
	W   http.ResponseWriter
	R   *http.Request
	URL url.Values
	// Session
	Claims *jwt.RegisteredClaims
	// Readers
	Params func(v any) error
	Body   func(v any) error
	// Responses
	String func(s string) error
	JSON   func(map[string]any) error
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

const (
	AUTH_ANON  = 0
	AUTH_USER  = 1
	AUTH_ADMIN = 2
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

func selectHandler(route *Route, method string) func(*Context) error {
	switch method {
	case http.MethodGet:
		return route.GET
	case http.MethodPost:
		return route.POST
	case http.MethodPut:
		return route.PUT
	case http.MethodPatch:
		return route.PATCH
	case http.MethodDelete:
		return route.DELETE
	default:
		return nil
	}
}

func buildContext(w http.ResponseWriter, r *http.Request) *Context {
	return &Context{
		W:   w,
		R:   r,
		URL: r.URL.Query(),
		Params: func(v any) error {
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
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(v); err != nil {
				return fmt.Errorf("failed to decode request body: %w", err)
			}
			return nil
		},
		String: func(s string) error {
			_, err := w.Write([]byte(s))
			return err
		},
		JSON: func(m map[string]any) error {
			w.Header().Set("Content-Type", "application/json")
			return json.NewEncoder(w).Encode(m)
		},
	}
}

func authMiddleware(next func(*Context) error, authLevel int) func(*Context) error {
	return func(req *Context) error {
		if authLevel == AUTH_ANON {
			return next(req)
		}
		authHeader := req.R.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return fmt.Errorf("unauthorized: missing or invalid Authorization header")
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := session.VerifyToken(tokenString)
		if err != nil {
			return fmt.Errorf("unauthorized: %v", err)
		}
		req.Claims = claims
		return next(req)
	}
}

func magic(req *Context) error {

	sub := req.Claims.Subject
	fmt.Println(sub)

	return nil
}

func (s *Router) Mount(route *Route) {
	if route.Path == "" {
		log.Println("Path is required to Mount Route")
		return
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Handling %s request for %s", r.Method, r.URL.Path)

		serveHandler := selectHandler(route, r.Method)
		if serveHandler == nil {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		req := buildContext(w, r)
		if err := magic(req); err != nil {
			fmt.Println(err)
		}

		authenticatedHandler := authMiddleware(serveHandler, route.Auth)

		magic(req)

		if err := authenticatedHandler(req); err != nil {
			if strings.HasPrefix(err.Error(), "unauthorized") {
				http.Error(w, err.Error(), http.StatusUnauthorized)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}

	}

	s.Handler.HandleFunc(route.Path, handler)
}
