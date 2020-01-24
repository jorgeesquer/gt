package lib

import (
	"fmt"
	"path/filepath"
	"github.com/gtlang/gt/core"
	"strconv"
	"strings"
)

func init() {
	core.RegisterLib(Router, ``)
}

var Router = []core.NativeFunction{
	core.NativeFunction{
		Name: "http.newRouter",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			r := newRouter()
			return core.NewObject(r), nil
		},
	},
}

type httpRouter struct {
	node *routeNode
}

func (r httpRouter) Type() string {
	return "http.Router"
}

func (r httpRouter) GetMethod(name string) core.NativeMethod {
	switch name {
	case "reset":
		return r.reset
	case "add":
		return r.add
	case "match":
		return r.match
	}
	return nil
}

func (r httpRouter) match(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	url := args[0].ToString()
	m, ok := r.Match(url)
	if ok {
		return core.NewObject(m), nil
	}

	return core.NullValue, nil
}

func (r httpRouter) reset(args []core.Value, vm *core.VM) (core.Value, error) {
	r.Reset()
	return core.NullValue, nil
}

func (r httpRouter) add(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	var route *httpRoute

	v := args[0]

	switch v.Type {

	case core.Object:
		o, ok := v.ToObject().(*httpRoute)
		if !ok {
			return core.NullValue, fmt.Errorf("invalid type for a route: %v", v.TypeName())
		}
		route = o

	case core.Map:
		route = &httpRoute{}

		mo := v.ToMap()
		mo.Mutex.RLock()
		defer mo.Mutex.RUnlock()
		m := mo.Map
		for k, p := range m {
			if err := route.SetProperty(k, p); err != nil {
				return core.NullValue, err
			}
		}

	default:
		return core.NullValue, fmt.Errorf("invalid type for route")
	}

	r.Add(route)

	return core.NullValue, nil
}

func newRouter() *httpRouter {
	return &httpRouter{node: newNode()}
}

// remove all routes
func (r *httpRouter) Reset() {
	r.node = newNode()
}

func extensionAsSegment(url string) string {
	ext := filepath.Ext(url)
	if ext != "" {
		url = url[:len(url)-len(ext)] + "/" + ext[1:]
	}
	return url
}

func (r httpRouter) Add(t *httpRoute) {
	url := extensionAsSegment(t.URL)
	url = strings.ToLower(url)
	segments := strings.Split(url, "/")
	t.Params = nil

	node := r.node

	for _, s := range segments {
		if s == "" {
			continue
		}
		if s[0] == ':' {
			t.Params = append(t.Params, s[1:])
			s = ":"
		}

		n, ok := node.child[s]
		if ok {
			node = n
			continue
		}

		n = newNode()
		node.child[s] = n
		node = n
	}

	node.route = t
}

func (r httpRouter) Match(url string) (routeMatch, bool) {
	url = extensionAsSegment(url)
	segments := strings.Split(url, "/")

	var params []string

	var lastNotMatched bool
	var lastWildcardNode *routeNode
	node := r.node

	for _, s := range segments {
		if s == "" {
			continue
		}

		if len(node.child) == 0 {
			break
		}

		if n, ok := node.child["*"]; ok {
			lastWildcardNode = n
		}

		n, ok := node.child[strings.ToLower(s)]
		if ok {
			node = n
			continue
		}

		n, ok = node.child[":"]
		if ok {
			params = append(params, s)
			node = n
			continue
		}

		lastNotMatched = true
		break
	}

	if node.route == nil {
		if n, ok := node.child["*"]; ok {
			return routeMatch{Route: n.route, Params: params}, true
		}

		if node.route == nil && lastWildcardNode != nil {
			return routeMatch{Route: lastWildcardNode.route, Params: params}, true
		}

		if node.route == nil {
			return routeMatch{}, false
		}
	}

	if lastNotMatched {
		if lastWildcardNode != nil {
			return routeMatch{Route: lastWildcardNode.route}, true
		}

		if node.route.URL == "/" {
			return routeMatch{Route: node.route}, true
		}

		return routeMatch{}, false
	}

	return routeMatch{Route: node.route, Params: params}, true
}

func newNode() *routeNode {
	return &routeNode{child: make(map[string]*routeNode)}
}

type routeNode struct {
	child map[string]*routeNode
	route *httpRoute
}

type routeMatch struct {
	Route  *httpRoute
	Params []string
}

func (r routeMatch) Type() string {
	return "http.RouteMatch"
}

func (r routeMatch) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "route":
		return core.NewObject(r.Route), nil
	case "values":
		p := make([]core.Value, len(r.Params))
		for i, v := range r.Params {
			p[i] = core.NewString(v)
		}
		return core.NewArrayValues(p), nil
	}

	return core.UndefinedValue, nil
}

func (r routeMatch) GetMethod(name string) core.NativeMethod {
	switch name {
	case "value":
		return r.value
	case "int":
		return r.int
	}
	return nil
}

func (r routeMatch) int(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	name := args[0].ToString()
	for i, k := range r.Route.Params {
		if k == name {
			s := r.Params[i]
			if s == "" {
				return core.NullValue, nil
			}

			i, err := strconv.Atoi(s)
			if err != nil {
				return core.NullValue, err
			}

			return core.NewInt(i), nil
		}
	}
	return core.NullValue, nil
}

func (r routeMatch) value(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	name := args[0].ToString()
	for i, k := range r.Route.Params {
		if k == name {
			return core.NewString(r.Params[i]), nil
		}
	}
	return core.NullValue, nil
}

func (m routeMatch) GetParam(name string) string {
	for i, k := range m.Route.Params {
		if k == name {
			return m.Params[i]
		}
	}
	return ""
}

type httpRoute struct {
	URL         string
	Handler     core.Value
	Filter      core.Value
	Params      []string
	Permissions []core.Value
}

func NewRoute(url string) *httpRoute {
	return &httpRoute{URL: url}
}

func (r *httpRoute) Type() string {
	return "http.Route"
}

func (r *httpRoute) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "url":
		return core.NewString(r.URL), nil
	case "handler":
		return r.Handler, nil
	case "filter":
		if r.Filter.Type == core.Null {
			return core.NullValue, nil
		}
		return r.Filter, nil
	case "permissions":
		return core.NewArrayValues(r.Permissions), nil
	}

	return core.UndefinedValue, nil
}

func (r *httpRoute) SetProperty(name string, v core.Value) error {
	switch name {
	case "url":
		r.URL = v.ToString()
		return nil
	case "handler":
		if v.Type != core.Func && v.TypeName() != "Closure" {
			return fmt.Errorf("invalid handler. Must be a function, got %v", v.TypeName())
		}
		r.Handler = v
		return nil
	case "filter":
		switch v.Type {
		case core.Null, core.Undefined, core.Func:
			r.Filter = v
			return nil
		case core.Object:
			if v.TypeName() != "Closure" {
				return fmt.Errorf("invalid filter. Must be a function, got %v", v.Type)
			}
			r.Filter = v
			return nil
		default:
			return fmt.Errorf("invalid filter. Must be a function, got %v", v.Type)
		}
	case "permissions":
		switch v.Type {
		case core.Null, core.Undefined:
			r.Permissions = nil
			return nil
		case core.Array:
			r.Permissions = v.ToArray()
			return nil
		default:
			return fmt.Errorf("invalid permissions, expected array, got: %s", v.TypeName())
		}
	}

	return ErrReadOnlyOrUndefined
}
