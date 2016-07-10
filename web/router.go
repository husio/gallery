package web

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

type Router struct {
	endpoints []endpoint

	// NotFound is called when none of defined handlers match current route.
	NotFound http.Handler

	// MethodNotAllowed is called when there is path match but not for current
	// method.
	MethodNotAllowed http.Handler
}

func NewRouter() *Router {
	return &Router{
		NotFound: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}),
		MethodNotAllowed: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}),
	}
}

// Add registers handler to be called whenever request matching given path
// regexp and method must be handled.
//
// Use (name) or (name:regexp) to match part of the path and pass result to
// handler. First example will use [^/]+ to make match, in second one provided
// regexp is used. Name is not used and is required only for documentation
// purposes.
//
// Using '*' as methods will match any method.
func (r *Router) Add(path, methods string, handler interface{}) {
	builder := regexp.MustCompile(`\(.*?\)`)
	raw := builder.ReplaceAllStringFunc(path, func(s string) string {
		s = s[1 : len(s)-1]
		// every (<name>) can be optionally contain separate regexp
		// definition using notation (<name>:<regexp>)
		chunks := strings.SplitN(s, ":", 2)
		if len(chunks) == 1 {
			return `([^/]+)`
		}
		return `(` + chunks[1] + `)`
	})
	// replace {} with regular expressions syntax
	rx, err := regexp.Compile(`^` + raw + `$`)
	if err != nil {
		panic(fmt.Sprintf("invalid routing path %q: %s", path, err))
	}

	methodsSet := make(map[string]struct{})
	for _, method := range strings.Split(methods, ",") {
		methodsSet[strings.TrimSpace(method)] = struct{}{}
	}

	var fn Handler
	switch h := handler.(type) {
	case Handler:
		fn = h
	case func(http.ResponseWriter, *http.Request, PathArg):
		fn = h
	case func(http.ResponseWriter, *http.Request, func(int) string):
		fn = func(w http.ResponseWriter, r *http.Request, a PathArg) { h(w, r, a) }
	case http.Handler:
		fn = func(w http.ResponseWriter, r *http.Request, _ PathArg) { h.ServeHTTP(w, r) }
	case http.HandlerFunc:
		fn = func(w http.ResponseWriter, r *http.Request, _ PathArg) { h(w, r) }
	case func(http.ResponseWriter, *http.Request):
		fn = func(w http.ResponseWriter, r *http.Request, _ PathArg) { h(w, r) }
	default:
		panic(fmt.Sprintf("invalid handler for %s %s: %T", methods, path, handler))
	}

	r.endpoints = append(r.endpoints, endpoint{
		methods: methodsSet,
		rx:      rx,
		handler: fn,
	})

}

type endpoint struct {
	methods map[string]struct{}
	rx      *regexp.Regexp
	handler Handler
}

// after 1.7, argument values can be attached to request context, but for now,
// pass them as one of the arguments
type Handler func(http.ResponseWriter, *http.Request, PathArg)

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var pathMatch bool

	for _, endpoint := range rt.endpoints {
		match := endpoint.rx.FindAllStringSubmatch(r.URL.Path, 1)
		if len(match) == 0 {
			continue
		}

		pathMatch = true

		_, ok := endpoint.methods[r.Method]
		if !ok {
			_, ok = endpoint.methods["*"]
		}
		if !ok {
			continue
		}

		args := match[0][1:]
		endpoint.handler(w, r, pathArg(args))
		return
	}

	if pathMatch {
		rt.MethodNotAllowed.ServeHTTP(w, r)
	} else {
		rt.NotFound.ServeHTTP(w, r)
	}
}

// PathArg return value as matched by path regexp at given index.
type PathArg func(index int) string

func pathArg(args []string) PathArg {
	return func(index int) string {
		if len(args) < index {
			return ""
		}
		return args[index]
	}
}
