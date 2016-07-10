package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRouter(t *testing.T) {
	rt := NewRouter()
	rt.Add(`/fruits`, "*", func(w http.ResponseWriter, r *http.Request, arg PathArg) {
		fmt.Fprintf(w, "%s fruits", r.Method)
	})
	rt.Add(`/fruits/(name:\d+)`, "*", func(w http.ResponseWriter, r *http.Request, arg PathArg) {
		fmt.Fprintf(w, "%s n=%s", r.Method, arg(0))
	})
	rt.Add(`/fruits/(name)`, "GET", func(w http.ResponseWriter, r *http.Request, arg PathArg) {
		fmt.Fprintf(w, "get %s", arg(0))
	})
	rt.Add(`/fruits/(name)`, "DELETE", func(w http.ResponseWriter, r *http.Request, arg PathArg) {
		fmt.Fprintf(w, "rm %s", arg(0))
	})

	cases := map[string]struct {
		method   string
		path     string
		wantCode int
		wantBody string
	}{
		"create_fruit": {
			method:   "POST",
			path:     "/fruits",
			wantCode: http.StatusOK,
			wantBody: "POST fruits",
		},
		"list_fruits": {
			method:   "GET",
			path:     "/fruits",
			wantCode: http.StatusOK,
			wantBody: "GET fruits",
		},
		"get_apple": {
			method:   "GET",
			path:     "/fruits/apple",
			wantCode: http.StatusOK,
			wantBody: "get apple",
		},
		"get_n": {
			method:   "PUT",
			path:     "/fruits/321",
			wantCode: http.StatusOK,
			wantBody: "PUT n=321",
		},
		"delete_apple": {
			method:   "DELETE",
			path:     "/fruits/apple",
			wantCode: http.StatusOK,
			wantBody: "rm apple",
		},
		"update_apple_is_not_allowed": {
			method:   "PUT",
			path:     "/fruits/apple",
			wantCode: http.StatusMethodNotAllowed,
			wantBody: http.StatusText(http.StatusMethodNotAllowed),
		},
		"not_found": {
			method:   "GET",
			path:     "/car/land-rover",
			wantCode: http.StatusNotFound,
			wantBody: http.StatusText(http.StatusNotFound),
		},
	}

	for tname, tc := range cases {
		w := httptest.NewRecorder()
		r, err := http.NewRequest(tc.method, tc.path, nil)
		if err != nil {
			t.Errorf("%s: cannot create request: %s", tname, err)
			continue
		}

		rt.ServeHTTP(w, r)

		if w.Code != tc.wantCode {
			t.Errorf("%s: want %d status code, got %d", tname, tc.wantCode, w.Code)
		}
		if want, got := strings.TrimSpace(w.Body.String()), strings.TrimSpace(tc.wantBody); want != got {
			t.Errorf("%s: want %q, got %q", tname, want, got)
		}
	}
}

func BenchmarkRouterFastFind(b *testing.B) {
	r, _ := http.NewRequest("GET", "/books", nil)
	rt := testRouter()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		rt.ServeHTTP(w, r)
	}
}

func BenchmarkRouterSlowFind(b *testing.B) {
	r, _ := http.NewRequest("DELETE", "/fruits/apple/whatever", nil)
	rt := testRouter()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		rt.ServeHTTP(w, r)
	}
}

func BenchmarkRouterNotFound(b *testing.B) {
	r, _ := http.NewRequest("GET", "/does-not-exist", nil)
	rt := testRouter()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		rt.ServeHTTP(w, r)
	}
}

func BenchmarkRouterMethodNotAllowed(b *testing.B) {
	r, _ := http.NewRequest("POST", "/fruits/apple/whatever", nil)
	rt := testRouter()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		rt.ServeHTTP(w, r)
	}
}

func testRouter() http.Handler {
	rt := NewRouter()

	noopHandler := func(w http.ResponseWriter, r *http.Request) {}

	rt.Add(`/books`, "GET", noopHandler)
	rt.Add(`/books`, "POST", noopHandler)
	rt.Add(`/books/(book-id)`, "GET", noopHandler)
	rt.Add(`/books/(book-id)`, "POST", noopHandler)
	rt.Add(`/books/(book-id)`, "PUT", noopHandler)
	rt.Add(`/books/(book-id)`, "DELETE", noopHandler)

	// few more routes just to make it more "real life"
	for i := 0; i < 50; i++ {
		rt.Add(fmt.Sprintf("/number/%d/(action-name)", i), "*", noopHandler)
	}

	rt.Add(`/fruits`, "*", noopHandler)
	rt.Add(`/fruits/(fruit-name:\w+)`, "*", noopHandler)
	rt.Add(`/fruits/(fruit-name:\w+)/whatever`, "DELETE", noopHandler)

	return rt
}
