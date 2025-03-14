package router

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestSetField(t *testing.T) {
	tests := []struct {
		name     string
		field    reflect.Value
		value    string
		wantErr  bool
		expected any
	}{
		{
			name:     "set string",
			field:    reflect.ValueOf(new(string)).Elem(),
			value:    "test",
			wantErr:  false,
			expected: "test",
		},
		{
			name:     "set int",
			field:    reflect.ValueOf(new(int)).Elem(),
			value:    "123",
			wantErr:  false,
			expected: 123,
		},
		{
			name:     "set float",
			field:    reflect.ValueOf(new(float64)).Elem(),
			value:    "45.67",
			wantErr:  false,
			expected: 45.67,
		},
		{
			name:     "set bool",
			field:    reflect.ValueOf(new(bool)).Elem(),
			value:    "true",
			wantErr:  false,
			expected: true,
		},
		{
			name:    "invalid int",
			field:   reflect.ValueOf(new(int)).Elem(),
			value:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := setField(tt.field, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("setField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				switch tt.field.Kind() {
				case reflect.String:
					if got := tt.field.String(); got != tt.expected {
						t.Errorf("setField() = %v, want %v", got, tt.expected)
					}
				case reflect.Int:
					if got := tt.field.Int(); got != int64(tt.expected.(int)) {
						t.Errorf("setField() = %v, want %v", got, tt.expected)
					}
				case reflect.Float64:
					if got := tt.field.Float(); got != tt.expected {
						t.Errorf("setField() = %v, want %v", got, tt.expected)
					}
				case reflect.Bool:
					if got := tt.field.Bool(); got != tt.expected {
						t.Errorf("setField() = %v, want %v", got, tt.expected)
					}
				}
			}
		})
	}
}

func TestParams(t *testing.T) {
	type TestParams struct {
		ID   int    `query:"id"`
		Name string `query:"name"`
	}

	req := httptest.NewRequest("GET", "/test?id=69&name=Josh", nil)
	w := httptest.NewRecorder()
	ctx := buildContext(w, req)

	var params TestParams
	err := ctx.Params(&params)
	if err != nil {
		t.Fatalf("Params() error = %v", err)
	}

	if params.ID != 69 {
		t.Errorf("Params() ID = %v, want 69", params.ID)
	}
	if params.Name != "Josh" {
		t.Errorf("Params() Name = %v, want Josh", params.Name)
	}

	// Test with invalid input
	req = httptest.NewRequest("GET", "/test?id=invalid", nil)
	ctx = buildContext(httptest.NewRecorder(), req)
	err = ctx.Params(&params)
	if err == nil {
		t.Error("Params() expected error for invalid integer, got nil")
	}
}

func TestBody(t *testing.T) {
	type TestBody struct {
		Age  int    `json:"age"`
		City string `json:"city"`
	}

	jsonBody := `{"age": 30, "city": "Kansas City"}`
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	ctx := buildContext(w, req)

	var body TestBody
	err := ctx.Body(&body)
	if err != nil {
		t.Fatalf("Body() error = %v", err)
	}

	if body.Age != 30 {
		t.Errorf("Body() Age = %v, want 30", body.Age)
	}
	if body.City != "Kansas City" {
		t.Errorf("Body() City = %v, want Kansas City", body.City)
	}

	// Test invalid JSON
	req = httptest.NewRequest("POST", "/test", bytes.NewBufferString("{invalid"))
	req.Header.Set("Content-Type", "application/json")
	ctx = buildContext(httptest.NewRecorder(), req)
	err = ctx.Body(&body)
	if err == nil {
		t.Error("Body() expected error for invalid JSON, got nil")
	}
}

func TestResponseMethods(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := buildContext(w, req)

	// Test String
	err := ctx.String("Hello, World!")
	if err != nil {
		t.Fatalf("String() error = %v", err)
	}
	if w.Body.String() != "Hello, World!" {
		t.Errorf("String() body = %v, want Hello, World!", w.Body.String())
	}

	// Test JSON
	w = httptest.NewRecorder()
	ctx = buildContext(w, req)
	data := map[string]any{"message": "Success"}
	err = ctx.JSON(data)
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}
	expected := `{"message":"Success"}` + "\n"
	if w.Body.String() != expected {
		t.Errorf("JSON() body = %v, want %v", w.Body.String(), expected)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("JSON() Content-Type = %v, want application/json", w.Header().Get("Content-Type"))
	}
}

func TestAuthMiddleware(t *testing.T) {
	handler := func(ctx *Context) error {
		return ctx.String("OK")
	}

	// Test AUTH_ANON
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	ctx := buildContext(w, req)
	err := authMiddleware(handler, AUTH_ANON)(ctx)
	if err != nil {
		t.Errorf("authMiddleware(AUTH_ANON) error = %v", err)
	}
	if w.Body.String() != "OK" {
		t.Errorf("authMiddleware(AUTH_ANON) body = %v, want OK", w.Body.String())
	}

	// Test AUTH_USER with valid token
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	w = httptest.NewRecorder()
	ctx = buildContext(w, req)
	err = authMiddleware(handler, AUTH_USER)(ctx)
	if err != nil {
		t.Errorf("authMiddleware(AUTH_USER) error = %v", err)
	}
	if ctx.Claims.Subject != "user1" {
		t.Errorf("authMiddleware(AUTH_USER) Claims.Subject = %v, want user1", ctx.Claims.Subject)
	}

	// Test AUTH_USER with invalid token
	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	w = httptest.NewRecorder()
	ctx = buildContext(w, req)
	err = authMiddleware(handler, AUTH_USER)(ctx)
	if err == nil || !strings.HasPrefix(err.Error(), "unauthorized") {
		t.Errorf("authMiddleware(AUTH_USER) expected unauthorized error, got %v", err)
	}
}

func TestMount(t *testing.T) {
	router := NewRouter()
	route := &Route{
		Path: "/test",
		Auth: AUTH_USER,
		GET: func(ctx *Context) error {
			return ctx.String("Hello, " + ctx.Claims.Subject)
		},
	}
	router.Mount(route)

	server := httptest.NewServer(router.Handler)
	defer server.Close()

	// Test GET with valid token
	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Request error = %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %v, want %v", resp.StatusCode, http.StatusOK)
	}
	if string(body) != "Hello, user1" {
		t.Errorf("Body = %v, want Hello, user1", string(body))
	}

	// Test POST (unsupported method)
	req, _ = http.NewRequest("POST", server.URL+"/test", nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("StatusCode = %v, want %v", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}
