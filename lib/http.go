package lib

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gtlang/gt/core"
)

func init() {
	// set a default timeout for the whole app
	http.DefaultClient.Timeout = time.Second * 60

	cacheBreaker = RandString(9)

	core.RegisterLib(HTTP, `


declare namespace http {
    export const CONTENT_TYPE_JAVASCRIPT: string

    export function get(url: string, timeout?: time.Duration | number, config?: tls.Config): string
    export function post(url: string, data?: any): string

    export function getJSON(url: string): any

    export function cacheBreaker(): string
    export function resetCacheBreaker(): string

    export function encodeURIComponent(url: string): string
    export function decodeURIComponent(url: string): string

    export function parseURL(url?: string): URL

    export function serveReverseProxy(url: URL, w: ResponseWriter, r: Request): void

    export function newRouter(): Router

    export interface Router {
        reset(): void
        add(r: Route): void
        match(url: string): RouteMatch | null
    }

    export interface RouteMatch {
        route: Route
        values: string[]
        int(name: string): number
        value(name: string): string
    }

    export interface Route {
        url: string
        handler: Function
        filter?: Function
        permissions?: string[]
    }

    export type Handler = (w: ResponseWriter, r: Request) => void

    export interface Server {
        address: string
        addressTLS: string
        tlsConfig: tls.Config
        handler: Handler
		readHeaderTimeout: time.Duration | number
        writeTimeout: time.Duration | number
		readTimeout: time.Duration | number
        idleTimeout: time.Duration | number
        start(): void
        close(): void
        shutdown(duration?: time.Duration | number): void
    }

    export function newServer(): Server

    export type METHOD = "GET" | "POST" | "PUT" | "PATCH" | "DELETE" | "OPTIONS"

    export function newRequest(method: METHOD, url: string, data?: any): Request

    export function newContext(method: METHOD, url: string, data?: any): Context

    export interface Context {
        request: Request
        response: ResponseWriter
    }

    export interface Request {
        /**
         * If the request is using a TLS connection
         */
        tls: boolean

        /**
         * The http method.
         */
        method: METHOD

        host: string

        /**
         * scheme + host + port
         * https://stackoverflow.com/a/37366696/4264
         */
        origin: string

        url: URL

        referer: string

        userAgent: string

        body: io.ReaderCloser

        remoteAddr: string
        remoteIP: string

		/**
		 * The extension of the URL
		 */
        extension: string

        // string returns the first value for the named component of the query.
        // POST and PUT body parameters take precedence over URL query string values.
        string(key: string): string

        // int works as value but deserializes the value into an object.
        json(key: string): any

        // int works as value but converts the value to an int.
        int(key: string): number

        // float works as value but converts the value to a float.
        float(key: string): number

        // currency works as value but converts the value to a float.
        currency(key: string): number

        // bool works as value but converts the value to an bool.
        bool(key: string): boolean

        // date works as value but converts the value formated as "yyyy-mm-dd" to an time.Time.
        date(key: string): time.Time

        header(key: string): string
        setHeader(key: string, value: string): void

        file(name: string): File

        // The parsed form data, including both the URL
        // field's query parameters and the POST or PUT form data.
        // This field is only available after ParseForm is called.
        // The HTTP client ignores Form and uses Body instead.
        values: StringMap

        formValues: StringMap

        cookie(key: string): Cookie | null

        addCookie(c: Cookie): void
        addCookieValue(name: string, value: string): void

        setBasicAuth(user: string, password: string): void
        basicAuth(): { user: string, password: string }

        execute(timeout?: number | time.Duration, tlsconf?: tls.Config): Response

        mustExecute(timeout?: number | time.Duration, tlsconf?: tls.Config): string
    }


    export interface File {
        name: string
        contentType: string
        size: number
        read(b: byte[]): number
		ReadAt(p: byte[], off: number): number
        close(): void
    }

    export function newCookie(): Cookie

    export interface Cookie {
        domain: string
        path: string
        expires: time.Time
        name: string
        value: string
        secure: boolean
        httpOnly: boolean
    }

    export interface URL {
        scheme: string
        host: string
        port: string

        /**
         * The host without the port number if present
         */
        hostName: string

        /**
         * returns the subdomain part *only if* the host has a format xxx.xxxx.xx.
         */
        subdomain: string

        path: string
        query: string
        pathAndQuery: string
    }

    // interface FormValues {
    //     [key: string]: any
    // }  


    export interface Response {
        status: number
        handled: boolean
        proto: string
        body(): string
        cookies(): Cookie[]
    }


    export interface ResponseWriter {
        readonly status: number

        handled: boolean

        /**
         * Only will have a value if it is a Recorder
         */
        readonly body: io.Buffer

        readonly headers: Map<string[]>

        cookie(name: string): Cookie

        cookies(): Cookie[]

        addCookie(c: Cookie): void

        /**
         * Writes v to the server response.
         */
        write(v: any): number

        /**
         * Writes v HTML escaped to the server response.
         */
        writeHTML(v: any): void

        /**
         * Writes v to the server response setting json content type if
         * the header is not already set.
         */
        writeJSON(v: any, skipCacheHeader?: boolean): void

        /**
         * Writes v to the server response setting json content type if
         * the header is not already set.
         */
        writeJSONStatus(status: number, v: any, skipCacheHeader?: boolean): void

        /**
         * Serves a static file
         */
        writeFile(name: string, data: byte[] | string | io.File | io.FileSystem): void

        /**
         * Sets the http status header.
         */
        setStatus(status: number): void

        /**
         * Sets the content type header.
         */
        setContentType(type: string): void

        /**
         * Sets the content type header.
         */
        setHeader(name: string, value: string): void

        /**
         * Send a error to the client
         */
        writeError(status: number, msg?: string): void

        /**
         * Send a error with json content-type to the client
         */
        writeJSONError(status: number, msg?: string): void

        redirect(url: string): void
    }


}

`)
}

const MAX_PARSE_FORM_MEMORY = 10000

var cacheBreaker string

var HTTP = []core.NativeFunction{
	core.NativeFunction{
		Name: "->http.CONTENT_TYPE_JAVASCRIPT",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewString("application/x-javascript"), nil
		},
	},
	core.NativeFunction{
		Name: "->http.CONTENT_TYPE_JAVASCRIPT",
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewString("application/x-javascript"), nil
		},
	},
	core.NativeFunction{
		Name:      "http.cacheBreaker",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			return core.NewString(cacheBreaker), nil
		},
	},
	core.NativeFunction{
		Name:      "http.resetCacheBreaker",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("trusted") {
				return core.NullValue, ErrUnauthorized
			}
			cacheBreaker = RandString(9)
			return core.NullValue, nil
		},
	},
	core.NativeFunction{
		Name:      "http.newServer",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("netListen") {
				return core.NullValue, ErrUnauthorized
			}
			s := &server{vm: vm}
			return core.NewObject(s), nil
		},
	},
	core.NativeFunction{
		Name:      "websocket.upgrade",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("networking") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateArgs(args, core.Object); err != nil {
				return core.NullValue, err
			}
			r, ok := args[0].ToObject().(*request)
			if !ok {
				return core.NullValue, fmt.Errorf("invalid Request, got %s", args[1].TypeName())
			}

			c, err := upgrader.Upgrade(r.writer, r.request, nil)
			if err != nil {
				return core.NullValue, err
			}

			// Maximum message size allowed from peer.
			c.SetReadLimit(8192)

			return core.NewObject(newWebsocketConn(c, vm)), nil
		},
	},
	core.NativeFunction{
		Name:      "http.newCookie",
		Arguments: 0,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			c := &cookie{}
			return core.NewObject(c), nil
		},
	},
	core.NativeFunction{
		Name:      "http.encodeURIComponent",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}
			v := args[0].ToString()
			u := url.QueryEscape(v)
			return core.NewString(u), nil
		},
	},
	core.NativeFunction{
		Name:      "http.decodeURIComponent",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}
			v := args[0].ToString()
			u, err := url.QueryUnescape(v)
			if err != nil {
				return core.NullValue, err
			}
			return core.NewString(u), nil
		},
	},
	// core.NativeFunc{
	// 	Name:      "http.newWriter",
	// 	Arguments: 0,
	// 	Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
	// 		b := NewBuffer()
	// 		return core.Object(b), nil
	// 	},
	// },
	core.NativeFunction{
		Name:      "http.newContext",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateArgRange(args, 2, 3); err != nil {
				return core.NullValue, err
			}

			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected argument 1 to be string, got %v", args[0].Type)
			}
			if args[1].Type != core.String {
				return core.NullValue, fmt.Errorf("expected argument 1 to be string, got %v", args[1].Type)
			}

			var method string
			var urlStr string
			var queryMap map[string]core.Value
			var reader io.Reader

			switch len(args) {
			case 2:
				method = args[0].ToString()
				urlStr = args[1].ToString()
			case 3:
				method = args[0].ToString()
				urlStr = args[1].ToString()
				form := url.Values{}

				v := args[2]

				switch v.Type {
				case core.Null, core.Undefined:
				case core.String:
					reader = strings.NewReader(v.ToString())
					if method != "POST" {
						return core.NullValue, fmt.Errorf("can only pass a data string with POST")
					}
				case core.Map:
					m := v.ToMap()
					if method == "GET" {
						queryMap = m.Map
					} else {
						m.Mutex.RLock()
						for k, v := range m.Map {
							vs, err := serialize(v)
							if err != nil {
								return core.NullValue, fmt.Errorf("error serializign parameter: %v", v.Type)
							}
							form.Add(k, vs)
						}
						m.Mutex.RUnlock()
						reader = strings.NewReader(form.Encode())
					}
				default:
					return core.NullValue, fmt.Errorf("expected argument 3 to be object, got %v", v.Type)
				}
			}

			r, err := http.NewRequest(method, urlStr, reader)
			if err != nil {
				return core.NullValue, err
			}

			if method == "POST" {
				r.Header.Add("Content-Type", "application/x-www-form-urlencoded")
			} else if method == "GET" && queryMap != nil {
				q := r.URL.Query()
				for k, v := range queryMap {
					vs, err := serialize(v)
					if err != nil {
						return core.NullValue, fmt.Errorf("error serializign parameter: %v", v.Type)
					}
					q.Add(k, vs)
				}
				r.URL.RawQuery = q.Encode()
			}

			w := httptest.NewRecorder()

			m := make(map[string]core.Value, 2)
			m["request"] = core.NewObject(&request{request: r, writer: w})
			m["response"] = core.NewObject(&responseWriter{writer: w, request: r})

			return core.NewMapValues(m), nil
		},
	},
	core.NativeFunction{
		Name:      "http.newRequest",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("networking") {
				fmt.Println(23423)
				return core.NullValue, ErrUnauthorized
			}

			if err := ValidateArgRange(args, 2, 3); err != nil {
				return core.NullValue, err
			}

			if args[0].Type != core.String {
				return core.NullValue, fmt.Errorf("expected argument 1 to be string, got %v", args[0].Type)
			}

			if args[1].Type != core.String {
				return core.NullValue, fmt.Errorf("expected argument 2 to be string, got %v", args[1].Type)
			}

			var method string
			var urlStr string
			var queryMap map[string]core.Value
			var reader io.Reader
			var contentType string

			switch len(args) {
			case 2:
				method = args[0].ToString()
				urlStr = args[1].ToString()
			case 3:
				method = args[0].ToString()
				urlStr = args[1].ToString()
				form := url.Values{}

				v := args[2]

				switch v.Type {
				case core.Null, core.Undefined:
				case core.String:
					if method != "POST" {
						return core.NullValue, fmt.Errorf("can only pass a data string with POST")
					}
					reader = strings.NewReader(v.ToString())
					contentType = "application/json; charset=UTF-8"
				case core.Map:
					m := v.ToMap()
					if method == "GET" {
						queryMap = m.Map
					} else {
						m.Mutex.RLock()
						for k, v := range m.Map {
							vs, err := serialize(v)
							if err != nil {
								return core.NullValue, fmt.Errorf("error serializign parameter: %v", v.Type)
							}
							form.Add(k, vs)
						}
						m.Mutex.RUnlock()
						reader = strings.NewReader(form.Encode())
						contentType = "application/x-www-form-urlencoded"
					}
				default:
					return core.NullValue, fmt.Errorf("expected argument 3 to be object, got %v", v.Type)
				}
			}

			r, err := http.NewRequest(method, urlStr, reader)
			if err != nil {
				return core.NullValue, err
			}

			if method == "POST" {
				r.Header.Add("Content-Type", contentType)
			} else if method == "GET" && queryMap != nil {
				q := r.URL.Query()
				for k, v := range queryMap {
					vs, err := serialize(v)
					if err != nil {
						return core.NullValue, fmt.Errorf("error serializign parameter: %v", v.Type)
					}
					q.Add(k, vs)
				}
				r.URL.RawQuery = q.Encode()
			}

			return core.NewObject(&request{request: r}), nil
		},
	},
	core.NativeFunction{
		Name:      "http.get",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("networking") {
				return core.NullValue, ErrUnauthorized
			}

			client := &http.Client{}
			timeout := 20 * time.Second

			ln := len(args)

			if ln == 0 {
				return core.NullValue, fmt.Errorf("expected 1 to 3 arguments, got %d", len(args))
			}

			a := args[0]
			if a.Type != core.String {
				return core.NullValue, fmt.Errorf("expected argument 0 to be string, got %s", a.TypeName())
			}
			url := a.ToString()

			if ln == 0 {
			} else if ln > 1 {
				a := args[1]
				switch a.Type {
				case core.Undefined, core.Null:
				case core.Int, core.Object:
					var err error
					timeout, err = ToDuration(args[1])
					if err != nil {
						return core.NullValue, err
					}
				default:
					return core.NullValue, fmt.Errorf("expected argument 1 to be duration")
				}
			}

			if ln > 2 {
				b := args[2]
				switch b.Type {
				case core.Null, core.Undefined:
				case core.Object:
					t, ok := args[2].ToObjectOrNil().(*tlsConfig)
					if !ok {
						return core.NullValue, fmt.Errorf("expected argument 2 to be tls.Config")
					}
					client.Transport = &http.Transport{TLSClientConfig: t.conf}
				default:
					return core.NullValue, fmt.Errorf("expected argument 2 to be string, got %s", b.TypeName())
				}
			}

			client.Timeout = timeout

			resp, err := client.Get(url)
			if err != nil {
				return core.NullValue, err
			}

			b, err := ioutil.ReadAll(resp.Body)

			resp.Body.Close()

			if err != nil {
				return core.NullValue, err
			}

			if resp.StatusCode != 200 {
				return core.NullValue, fmt.Errorf("hTTP Error %d: %v", resp.StatusCode, string(b))
			}

			return core.NewString(string(b)), nil
		},
	},
	core.NativeFunction{
		Name:      "http.post",
		Arguments: 2,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("networking") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateArgs(args, core.String, core.Map); err != nil {
				return core.NullValue, err
			}
			u := args[0].ToString()

			data := url.Values{}

			m := args[1].ToMap()
			m.Mutex.RLock()
			for k, v := range m.Map {
				data.Add(k, v.ToString())
			}
			m.Mutex.RUnlock()

			resp, err := http.PostForm(u, data)
			if err != nil {
				return core.NullValue, err
			}

			b, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return core.NullValue, err
			}

			return core.NewString(string(b)), nil
		},
	},
	core.NativeFunction{
		Name:      "http.getJSON",
		Arguments: 1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("networking") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateArgs(args, core.String); err != nil {
				return core.NullValue, err
			}
			url := args[0].ToString()

			resp, err := http.Get(url)
			if err != nil {
				return core.NullValue, err
			}

			b, err := ioutil.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return core.NullValue, err
			}

			v, err := unmarshal(b)
			if err != nil {
				return core.NullValue, err
			}

			return v, nil
		},
	},
	core.NativeFunction{
		Name:      "http.parseURL",
		Arguments: -1,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if err := ValidateOptionalArgs(args, core.String); err != nil {
				return core.NullValue, err
			}

			if len(args) == 0 {
				u := &url.URL{}
				return core.NewObject(&URL{u}), nil
			}

			rawURL := args[0].ToString()
			u, err := url.Parse(rawURL)
			if err != nil {
				return core.NullValue, err
			}
			return core.NewObject(&URL{u}), nil
		},
	},
	core.NativeFunction{
		Name:      "http.serveReverseProxy",
		Arguments: 3,
		Function: func(this core.Value, args []core.Value, vm *core.VM) (core.Value, error) {
			if !vm.HasPermission("networking") {
				return core.NullValue, ErrUnauthorized
			}
			if err := ValidateArgs(args, core.Object, core.Object, core.Object); err != nil {
				return core.NullValue, err
			}
			url := args[0].ToObject().(*URL)
			if url == nil {
				return core.NullValue, fmt.Errorf("invalid URL, got %s", args[0].TypeName())
			}
			resp, ok := args[1].ToObject().(*responseWriter)
			if !ok {
				return core.NullValue, fmt.Errorf("invalid Response, got %s", args[2].TypeName())
			}

			req, ok := args[2].ToObject().(*request)
			if !ok {
				return core.NullValue, fmt.Errorf("invalid Request, got %s", args[1].TypeName())
			}
			proxy := httputil.NewSingleHostReverseProxy(url.url)
			proxy.ServeHTTP(resp.writer, req.request)
			return core.NullValue, nil
		},
	},
}

func serialize(v core.Value) (string, error) {
	switch v.Type {
	case core.Int, core.Float, core.String, core.Bool, core.Rune:
		return v.ToString(), nil
	}

	b, err := json.Marshal(v.Export(0))
	if err != nil {
		return "", err
	}

	// return strings without quotes, only the content of the string
	ln := len(b)
	if ln >= 2 && b[0] == '"' && b[ln-1] == '"' {
		b = b[1 : ln-1]
	}

	return string(b), nil
}

type server struct {
	address           string
	addressTLS        string
	handler           int
	tlsConfig         *tlsConfig
	server            *http.Server
	tlsServer         *http.Server
	readHeaderTimeout time.Duration
	writeTimeout      time.Duration
	readTimeout       time.Duration
	idleTimeout       time.Duration
	vm                *core.VM
}

func (s *server) Type() string {
	return "http.Server"
}

func (s *server) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "address":
		return core.NewString(s.Address()), nil
	case "addressTLS":
		return core.NewString(s.AddressTLS()), nil
	case "tlsConfig":
		return core.NewObject(s.tlsConfig), nil
	case "handler":
		return core.NewFunction(s.handler), nil
	case "readHeaderTimeout":
		return core.NewObject(Duration(s.readHeaderTimeout)), nil
	case "writeTimeout":
		return core.NewObject(Duration(s.writeTimeout)), nil
	case "readTimeout":
		return core.NewObject(Duration(s.readTimeout)), nil
	case "idleTimeout":
		return core.NewObject(Duration(s.idleTimeout)), nil
	}
	return core.UndefinedValue, nil
}

func (s *server) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "address":
		if v.Type != core.String {
			return fmt.Errorf("invalid type, expected string")
		}
		s.address = v.ToString()
		return nil

	case "addressTLS":
		if v.Type != core.String {
			return fmt.Errorf("invalid type, expected string")
		}
		s.addressTLS = v.ToString()
		return nil

	case "handler":
		if v.Type != core.Func {
			if _, ok := v.ToObject().(core.Closure); ok {
				return fmt.Errorf("expected a function, got a closure")
			}
			return fmt.Errorf("invalid type, expected a function, got %v", v.Type)
		}
		s.handler = v.ToFunction()
		return nil

	case "tlsConfig":
		if v.Type != core.Object {
			return fmt.Errorf("invalid type, expected a tls object")
		}
		tls, ok := v.ToObjectOrNil().(*tlsConfig)
		if !ok {
			return fmt.Errorf("invalid type, expected a tls object")
		}
		s.tlsConfig = tls
		return nil

	case "readHeaderTimeout":
		d, err := ToDuration(v)
		if err != nil {
			return err
		}
		s.readHeaderTimeout = time.Duration(d)
		return nil

	case "writeTimeout":
		d, err := ToDuration(v)
		if err != nil {
			return err
		}
		s.writeTimeout = time.Duration(d)
		return nil

	case "readTimeout":
		d, err := ToDuration(v)
		if err != nil {
			return err
		}
		s.readTimeout = time.Duration(d)
		return nil

	case "idleTimeout":
		d, err := ToDuration(v)
		if err != nil {
			return err
		}
		s.idleTimeout = time.Duration(d)
		return nil
	}

	return ErrReadOnlyOrUndefined
}

func (s *server) GetMethod(name string) core.NativeMethod {
	switch name {
	case "start":
		return s.start
	case "close":
		return s.close
	case "shutdown":
		return s.shutdown
	}
	return nil
}

func (s *server) Address() string {
	if s.address == "" {
		return ":8080"
	}
	return s.address
}

func (s *server) AddressTLS() string {
	if s.addressTLS == "" {
		return ":443"
	}
	return s.addressTLS
}

func (s *server) shutdown(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)

	var d time.Duration

	switch l {
	case 0:
		d = time.Second
	case 1:
		var a = args[0]
		switch a.Type {
		case core.Int:
			d = time.Duration(a.ToInt())
		case core.Object:
			dur, ok := a.ToObject().(Duration)
			if !ok {
				return core.NullValue, fmt.Errorf("expected duration, got %s", a.TypeName())
			}
			d = time.Duration(dur)
		}
	default:
		return core.NullValue, fmt.Errorf("expected 0 or 1 argument, got %d", l)
	}

	ctx, cancel := context.WithTimeout(context.Background(), d)
	err := s.server.Shutdown(ctx)

	var err2 error
	if s.tlsServer != nil {
		err2 = s.tlsServer.Shutdown(ctx)
	}

	if err == nil {
		err = err2
	}

	cancel()
	return core.NullValue, err
}
func (s *server) close(args []core.Value, vm *core.VM) (core.Value, error) {
	err := s.server.Close()

	var err2 error
	if s.tlsServer != nil {
		err2 = s.tlsServer.Close()
	}

	if err == nil {
		err = err2
	}

	return core.NullValue, err
}

func (s *server) start(args []core.Value, vm *core.VM) (core.Value, error) {
	if s.tlsConfig != nil {
		s.tlsServer = &http.Server{
			ReadHeaderTimeout: s.readHeaderTimeout,
			ReadTimeout:       s.readTimeout,
			WriteTimeout:      s.writeTimeout,
			IdleTimeout:       s.idleTimeout,
			TLSConfig:         s.tlsConfig.conf,
			Addr:              s.AddressTLS(),
			Handler:           s,
		}

		go func() {
			if err := s.tlsServer.ListenAndServeTLS("", ""); err != nil {
				// setting the error will make it stop in the next step
				vm.Error = err
				fmt.Println(err)
			}
		}()
	}

	s.server = &http.Server{
		ReadHeaderTimeout: s.readHeaderTimeout,
		ReadTimeout:       s.readTimeout,
		WriteTimeout:      s.writeTimeout,
		IdleTimeout:       s.idleTimeout,
		Addr:              s.Address(),
		Handler:           s,
	}

	if err := s.server.ListenAndServe(); err != nil {
		return core.NullValue, err
	}

	return core.NullValue, nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.handler == 0 {
		return
	}

	// create a new VM initialized with the global values
	sVM := s.vm
	p := sVM.Program
	g := sVM.Globals()

	vm := core.NewInitializedVM(p, g)
	vm.FileSystem = sVM.FileSystem
	vm.Trusted = sVM.Trusted
	ctx, ok := sVM.Context.(Context)
	if ok {
		vm.Context = ctx.Clone()
	} else {
		vm.Context = sVM.Context
	}

	rr := &responseWriter{
		writer:  w,
		request: r,
	}

	req := core.NewObject(&request{
		request: r,
		writer:  w,
	})

	if _, err := vm.RunFuncIndex(s.handler, core.NewObject(rr), req); err != nil {
		// the VM is paused at http.Listen so it has
		// no effect to pass the error to sVM
		fmt.Println(err)
	}
}

type cookie struct {
	domain   string
	path     string
	expires  time.Time
	name     string
	value    string
	secure   bool
	httpOnly bool
}

func (c *cookie) Type() string {
	return "http.Cookie"
}

func (c *cookie) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "domain":
		return core.NewString(c.name), nil
	case "path":
		return core.NewString(c.path), nil
	case "expires":
		return core.NewObject(TimeObj(c.expires)), nil
	case "name":
		return core.NewString(c.name), nil
	case "value":
		return core.NewString(c.value), nil
	case "secure":
		return core.NewBool(c.secure), nil
	case "httpOnly":
		return core.NewBool(c.httpOnly), nil
	}
	return core.UndefinedValue, nil
}

func (c *cookie) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "domain":
		if v.Type != core.String {
			return ErrInvalidType
		}
		c.domain = v.ToString()
		return nil
	case "path":
		if v.Type != core.String {
			return ErrInvalidType
		}
		c.path = v.ToString()
		return nil
	case "expires":
		if v.Type != core.Object {
			return ErrInvalidType
		}
		t, ok := v.ToObject().(TimeObj)
		if !ok {
			return ErrInvalidType
		}
		c.expires = time.Time(t)
		return nil
	case "name":
		if v.Type != core.String {
			return ErrInvalidType
		}
		c.name = v.ToString()
		return nil
	case "value":
		if v.Type != core.String {
			return ErrInvalidType
		}
		c.value = v.ToString()
		return nil
	case "secure":
		if v.Type != core.Bool {
			return ErrInvalidType
		}
		c.secure = v.ToBool()
		return nil
	case "httpOnly":
		if v.Type != core.Bool {
			return ErrInvalidType
		}
		c.httpOnly = v.ToBool()
		return nil
	}
	return ErrReadOnlyOrUndefined
}

type response struct {
	r       *http.Response
	handled bool
}

func (r *response) Type() string {
	return "http.Response"
}

func (r *response) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "handled":
		return core.NewBool(r.handled), nil
	case "status":
		return core.NewInt(r.r.StatusCode), nil
	case "proto":
		return core.NewString(r.r.Proto), nil
	}
	return core.UndefinedValue, nil
}

func (r *response) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "handled":
		if v.Type != core.Bool {
			return fmt.Errorf("invalid type. Expected boolean")
		}
		r.handled = v.ToBool()
		return nil
	}
	return ErrReadOnlyOrUndefined
}

func (r *response) GetMethod(name string) core.NativeMethod {
	switch name {
	case "body":
		return r.body
	case "cookies":
		return r.cookies
	}
	return nil
}

func (r *response) cookies(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	cookies := r.r.Cookies()

	v := make([]core.Value, len(cookies))

	for i, k := range cookies {
		c := &cookie{
			domain:   k.Domain,
			expires:  k.Expires,
			secure:   k.Secure,
			httpOnly: k.HttpOnly,
			name:     k.Name,
			value:    k.Value,
		}

		v[i] = core.NewObject(c)
	}

	return core.NewArrayValues(v), nil
}

func (r *response) body(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgRange(args, 0, 0); err != nil {
		return core.NullValue, err
	}

	resp := r.r

	b, err := ioutil.ReadAll(resp.Body)

	resp.Body.Close()

	if err != nil {
		return core.NullValue, err
	}

	return core.NewString(string(b)), nil
}

type request struct {
	request *http.Request
	writer  http.ResponseWriter
}

func (r *request) Type() string {
	return "http.Request"
}

func (r *request) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "origin":
		var s string
		rq := r.request
		if rq.TLS != nil {
			s = "https://"
		} else {
			s = "http://"
		}
		host := rq.Host
		if host == "" {
			host = rq.URL.Host
		}
		s += host
		return core.NewString(s), nil
	case "body":
		return core.NewObject(&readerCloser{r.request.Body}), nil
	case "tls":
		return core.NewBool(r.request.TLS != nil), nil
	case "host":
		return core.NewString(r.request.Host), nil
	case "method":
		return core.NewString(r.request.Method), nil
	case "userAgent":
		return core.NewString(r.request.UserAgent()), nil
	case "referer":
		return core.NewString(r.request.Referer()), nil
	case "remoteAddr":
		return core.NewString(r.request.RemoteAddr), nil
	case "remoteIP":
		a := r.request.RemoteAddr
		if !strings.ContainsRune(a, ':') {
			return core.NewString(a), nil
		}
		ip, _, err := net.SplitHostPort(r.request.RemoteAddr)
		if err != nil {
			return core.NullValue, err
		}
		return core.NewString(ip), nil
	case "url":
		u := r.request.URL
		u.Host = r.request.Host
		return core.NewObject(&URL{u}), nil
	case "extension":
		return core.NewString(filepath.Ext(r.request.URL.Path)), nil
	case "values":
		var form url.Values
		req := r.request
		if req.Method == "POST" {
			if err := req.ParseMultipartForm(MAX_PARSE_FORM_MEMORY); err != nil {
				if err := req.ParseForm(); err != nil {
					return core.NullValue, err
				}
			}
			form = req.Form
		} else {
			form = req.URL.Query()
		}
		values := make(map[string]core.Value, len(form))
		for k, v := range form {
			values[k] = core.NewString(v[0])
		}
		return core.NewMapValues(values), nil
	case "formValues":
		req := r.request
		if req.Method != "POST" {
			return core.NullValue, nil
		}
		if err := req.ParseMultipartForm(MAX_PARSE_FORM_MEMORY); err != nil {
			if err := req.ParseForm(); err != nil {
				return core.NullValue, err
			}
		}
		form := req.Form
		values := make(map[string]core.Value, len(form))
		for k, v := range form {
			values[k] = core.NewString(v[0])
		}
		return core.NewMapValues(values), nil
	}

	return core.UndefinedValue, nil
}

func (r *request) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "host":
		if v.Type != core.String {
			return fmt.Errorf("invalid type. Expected string")
		}
		r.request.Host = v.ToString()
		return nil
	}
	return ErrReadOnlyOrUndefined
}

func (r *request) GetMethod(name string) core.NativeMethod {
	switch name {
	case "header":
		return r.header
	case "setBasicAuth":
		return r.setBasicAuth
	case "basicAuth":
		return r.basicAuth
	case "setHeader":
		return r.setHeader
	case "execute":
		return r.execute
	case "mustExecute":
		return r.mustExecute
	case "string":
		return r.formString
	case "int":
		return r.formInt
	case "currency":
		return r.formCurrency
	case "float":
		return r.formFloat
	case "bool":
		return r.formBool
	case "date":
		return r.formDate
	case "json":
		return r.formJSON
	case "file":
		return r.file
	case "cookie":
		return r.cookie
	case "addCookie":
		return r.addCookie
	case "addCookieValue":
		return r.addCookieValue
	}
	return nil
}

func (r *request) basicAuth(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	user, pwd, ok := r.request.BasicAuth()
	if !ok {
		return core.NullValue, nil
	}

	m := make(map[string]core.Value)
	m["user"] = core.NewString(user)
	m["password"] = core.NewString(pwd)

	return core.NewMapValues(m), nil
}

func (r *request) setBasicAuth(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String, core.String); err != nil {
		return core.NullValue, err
	}

	user := args[0].ToString()
	pwd := args[1].ToString()
	r.request.SetBasicAuth(user, pwd)

	return core.NullValue, nil
}

func (r *request) execute(args []core.Value, vm *core.VM) (core.Value, error) {
	client := &http.Client{}

	ln := len(args)

	if ln > 0 {
		a := args[0]
		switch a.Type {
		case core.Undefined, core.Null:
		case core.Int, core.Object:
			d, err := ToDuration(a)
			if err != nil {
				return core.NullValue, err
			}

			client.Timeout = d
		default:
			return core.NullValue, fmt.Errorf("expected argument 1 to be duration")
		}
	}

	if ln > 1 {
		tlsc, ok := args[1].ToObjectOrNil().(*tlsConfig)
		if !ok {
			return core.NullValue, fmt.Errorf("expected arg 2 to be TLSConfig")
		}
		client.Transport = &http.Transport{
			TLSClientConfig: tlsc.conf,
		}
	}

	resp, err := client.Do(r.request)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(&response{r: resp}), nil
}

func (r *request) mustExecute(args []core.Value, vm *core.VM) (core.Value, error) {
	client := &http.Client{}

	ln := len(args)

	if ln > 0 {
		a := args[0]
		switch a.Type {
		case core.Undefined, core.Null:
		case core.Int, core.Object:
			d, err := ToDuration(a)
			if err != nil {
				return core.NullValue, err
			}

			client.Timeout = d
		default:
			return core.NullValue, fmt.Errorf("expected argument 1 to be duration")
		}
	}

	if ln > 1 {
		tlsc, ok := args[1].ToObjectOrNil().(*tlsConfig)
		if !ok {
			return core.NullValue, fmt.Errorf("expected arg 2 to be TLSConfig")
		}
		client.Transport = &http.Transport{
			TLSClientConfig: tlsc.conf,
		}
	}

	resp, err := client.Do(r.request)
	if err != nil {
		return core.NullValue, err
	}

	b, err := ioutil.ReadAll(resp.Body)

	resp.Body.Close()

	if err != nil {
		return core.NullValue, err
	}

	if resp.StatusCode != 200 {
		return core.NullValue, fmt.Errorf("hTTP Error %d: %v", resp.StatusCode, string(b))
	}

	return core.NewString(string(b)), nil
}

func (r *request) header(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	key := args[0].ToString()
	v := r.request.Header.Get(key)
	return core.NewString(v), nil
}

func (r *request) setHeader(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String, core.String); err != nil {
		return core.NullValue, err
	}

	key := args[0].ToString()
	value := args[1].ToString()
	r.request.Header.Set(key, value)

	return core.NullValue, nil
}

func (r *request) file(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	key := args[0].ToString()

	req := r.request

	file, header, err := req.FormFile(key)
	if err != nil {
		if err == http.ErrMissingFile {
			return core.NullValue, nil
		}
		return core.NullValue, err
	}

	name := filepath.Base(header.Filename)
	ctype := header.Header.Get("Content-Type")
	return core.NewObject(newFormFile(file, name, ctype, header.Size, vm)), nil
}

func newFormFile(file multipart.File, name string, contentType string, size int64, vm *core.VM) formFile {
	f := formFile{
		file:        file,
		name:        name,
		contentType: contentType,
		size:        size,
	}

	vm.SetGlobalFinalizer(f)
	return f
}

type formFile struct {
	file        multipart.File
	size        int64
	name        string
	contentType string
}

func (f formFile) Type() string {
	return "multipart.File"
}

func (f formFile) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "name":
		return core.NewString(f.name), nil
	case "size":
		return core.NewInt64(f.size), nil
	case "contentType":
		return core.NewString(f.contentType), nil
	}
	return core.UndefinedValue, nil
}

func (f formFile) GetMethod(name string) core.NativeMethod {
	switch name {
	case "close":
		return f.close
	}
	return nil
}

func (f formFile) Read(p []byte) (n int, err error) {
	return f.file.Read(p)
}

func (f formFile) ReadAt(p []byte, off int64) (n int, err error) {
	return f.file.ReadAt(p, off)
}

func (f formFile) Close() error {
	c, ok := f.file.(io.Closer)
	if ok {
		if err := c.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (f formFile) close(args []core.Value, vm *core.VM) (core.Value, error) {
	if c, ok := f.file.(io.Closer); ok {
		if err := c.Close(); err != nil {
			return core.NullValue, err
		}
	}

	return core.NullValue, nil
}

func (r *request) addCookieValue(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String, core.String); err != nil {
		return core.NullValue, err
	}

	k := &http.Cookie{
		Name:  args[0].ToString(),
		Value: args[1].ToString(),
	}

	r.request.AddCookie(k)
	return core.NullValue, nil
}

func (r *request) addCookie(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	c, ok := args[0].ToObject().(*cookie)
	if !ok {
		return core.NullValue, ErrInvalidType
	}

	k := &http.Cookie{
		Domain:   c.domain,
		Path:     c.path,
		Expires:  c.expires,
		Name:     c.name,
		Value:    c.value,
		Secure:   c.secure,
		HttpOnly: c.httpOnly,
	}

	r.request.AddCookie(k)
	return core.NullValue, nil
}

func (r *request) cookie(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	name := args[0].ToString()

	k, err := r.request.Cookie(name)
	if err != nil {
		if err == http.ErrNoCookie {
			return core.NullValue, nil
		}
		return core.NullValue, err
	}

	c := &cookie{
		domain:   k.Domain,
		expires:  k.Expires,
		secure:   k.Secure,
		httpOnly: k.HttpOnly,
		name:     k.Name,
		value:    k.Value,
	}

	return core.NewObject(c), nil
}

func (r *request) formString(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	name := args[0].ToString()

	req := r.request
	if req.Method == "GET" {
		return core.NewString(req.URL.Query().Get(name)), nil
	}

	return core.NewString(req.FormValue(name)), nil
}

func (r *request) formInt(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	name := args[0].ToString()

	var s string
	req := r.request
	if req.Method == "GET" {
		s = req.URL.Query().Get(name)
	} else {
		s = req.FormValue(name)
	}

	if s == "" || s == "undefined" {
		return core.NullValue, nil
	}

	if s == "NaN" {
		return core.NullValue, fmt.Errorf("invalid format: NaN")
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return core.NullValue, core.NewPublicError(fmt.Sprintf("Invalid format: %s", s))
	}

	return core.NewInt(i), nil
}

func (r *request) formFloat(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	name := args[0].ToString()

	var s string
	req := r.request
	if req.Method == "GET" {
		s = req.URL.Query().Get(name)
	} else {
		s = req.FormValue(name)
	}

	if s == "" || s == "undefined" {
		return core.NullValue, nil
	}

	if s == "NaN" {
		return core.NullValue, fmt.Errorf("invalid format: NaN")
	}

	c := GetContext(vm).GetCulture()

	i, err := parseFloat(s, c.culture)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewFloat(i), nil
}

func (r *request) formCurrency(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	name := args[0].ToString()

	var s string
	req := r.request
	if req.Method == "GET" {
		s = req.URL.Query().Get(name)
	} else {
		s = req.FormValue(name)
	}

	if s == "" || s == "undefined" {
		return core.NullValue, nil
	}

	c := GetContext(vm).GetCulture().culture

	s = strings.Replace(s, c.CurrencySymbol, "", 1)

	i, err := parseFloat(s, c)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewFloat(i), nil
}

func (r *request) formBool(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	name := args[0].ToString()

	var s string
	req := r.request
	if req.Method == "GET" {
		s = req.URL.Query().Get(name)
	} else {
		s = req.FormValue(name)
	}

	switch s {
	case "true", "1", "on":
		return core.TrueValue, nil
	default:
		return core.FalseValue, nil
	}
}

func (r *request) formDate(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	name := args[0].ToString()

	var value string
	req := r.request
	if req.Method == "GET" {
		value = req.URL.Query().Get(name)
	} else {
		value = req.FormValue(name)
	}

	if value == "" || value == "undefined" {
		return core.NullValue, nil
	}

	loc := GetContext(vm).GetLocation()

	t, err := parseDate(value, "", loc)
	if err != nil {
		return core.NullValue, err
	}

	return core.NewObject(TimeObj(t)), nil
}

func (r *request) formJSON(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}
	name := args[0].ToString()

	var s string
	req := r.request
	if req.Method == "GET" {
		s = req.URL.Query().Get(name)
	} else {
		s = req.FormValue(name)
	}

	if s == "" || s == "undefined" {
		return core.NullValue, nil
	}

	v, err := unmarshal([]byte(s))
	if err != nil {
		return core.NullValue, err
	}

	return v, nil
}

type URL struct {
	url *url.URL
}

func (*URL) Type() string {
	return "url"
}

func (u *URL) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "path":
		if !vm.HasPermission("trusted") {
			return ErrUnauthorized
		}
		if v.Type != core.String {
			return fmt.Errorf("invalid type. Expected string")
		}
		u.url.Path = v.ToString()
		return nil
	case "scheme":
		if v.Type != core.String {
			return ErrInvalidType
		}
		u.url.Scheme = v.ToString()
		return nil
	case "host":
		if v.Type != core.String {
			return ErrInvalidType
		}
		u.url.Host = v.ToString()
		return nil
	}
	return ErrReadOnlyOrUndefined
}

func (u *URL) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "scheme":
		return core.NewString(u.url.Scheme), nil
	case "host":
		return core.NewString(u.url.Host), nil
	case "hostName":
		// returns the host without the port number if present
		host := u.url.Host
		i := strings.IndexRune(host, ':')
		if i != -1 {
			host = host[:i]
		}
		return core.NewString(host), nil
	case "port":
		// returns the host without the port number if present
		host := u.url.Host
		i := strings.IndexRune(host, ':')
		if i != -1 {
			return core.NewString(host[i+1:]), nil
		}
		return core.NullValue, nil
	case "subdomain":
		return core.NewString(getSubdomain(u.url.Host)), nil
	case "path":
		return core.NewString(u.url.Path), nil
	case "query":
		return core.NewString(u.url.RawQuery), nil
	case "pathAndQuery":
		p := u.url.Path
		q := u.url.RawQuery
		if q != "" {
			p += "?" + q
		}
		return core.NewString(p), nil
	}
	return core.UndefinedValue, nil
}

// return the subdomain if the host has a format subdomain.xxxx.xx
func getSubdomain(host string) string {
	parts := Split(host, ".")
	if len(parts) != 3 {
		return ""
	}
	return parts[0]
}

type responseWriter struct {
	writer  http.ResponseWriter
	request *http.Request
	status  int
	handled bool
}

func (*responseWriter) Type() string {
	return "http.ResponseWriter"
}

func (r *responseWriter) GetProperty(name string, vm *core.VM) (core.Value, error) {
	switch name {
	case "status":
		rc, ok := r.writer.(*httptest.ResponseRecorder)
		if ok {
			return core.NewInt(rc.Code), nil
		}
		return core.NewInt(r.status), nil
	case "handled":
		return core.NewBool(r.handled), nil
	case "body":
		rc, ok := r.writer.(*httptest.ResponseRecorder)
		if !ok {
			return core.NullValue, fmt.Errorf("the response is not a recorder")
		}
		return core.NewObject(Buffer{rc.Body}), nil
	case "headers":
		rc, ok := r.writer.(*httptest.ResponseRecorder)
		if !ok {
			return core.NullValue, fmt.Errorf("the response is not a recorder")
		}
		m := make(map[string]core.Value)
		for k, v := range rc.Header() {
			vs := make([]core.Value, len(v))
			for i, s := range v {
				vs[i] = core.NewString(s)
			}
			m[k] = core.NewArrayValues(vs)
		}
		return core.NewMapValues(m), nil
	}
	return core.UndefinedValue, nil
}

func (r *responseWriter) SetProperty(name string, v core.Value, vm *core.VM) error {
	switch name {
	case "handled":
		if v.Type != core.Bool {
			return fmt.Errorf("invalid type. Expected boolean")
		}
		r.handled = v.ToBool()
		return nil
	}
	return ErrReadOnlyOrUndefined
}

func (r *responseWriter) GetMethod(name string) core.NativeMethod {
	switch name {
	case "addCookie":
		return r.addCookie
	case "cookie":
		return r.cookie
	case "cookies":
		return r.cookies
	case "writeHTML":
		return r.writeHTML
	case "write":
		return r.write
	case "writeJSON":
		return r.writeJSON
	case "writeJSONStatus":
		return r.writeJSONStatus
	case "writeFile":
		return r.writeFile
	case "redirect":
		return r.redirect
	case "setContentType":
		return r.setContentType
	case "setHeader":
		return r.setHeader
	case "setStatus":
		return r.setStatus
	case "writeError":
		return r.writeError
	case "writeJSONError":
		return r.writeJSONError
	}
	return nil
}

func (r *responseWriter) Write(p []byte) (n int, err error) {

	// set 200 by default when anything is written
	if r.status == 0 {
		r.status = 200
	}

	return r.writer.Write(p)
}

func (r *responseWriter) cookies(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args); err != nil {
		return core.NullValue, err
	}

	rc, ok := r.writer.(*httptest.ResponseRecorder)
	if !ok {
		return core.NullValue, fmt.Errorf("the response is not a recorder")
	}

	request := &http.Request{Header: http.Header{"Cookie": rc.Header()["Set-Cookie"]}}

	cookies := request.Cookies()

	v := make([]core.Value, len(cookies))

	for i, k := range cookies {
		c := &cookie{
			domain:   k.Domain,
			expires:  k.Expires,
			secure:   k.Secure,
			httpOnly: k.HttpOnly,
			name:     k.Name,
			value:    k.Value,
		}

		v[i] = core.NewObject(c)
	}

	return core.NewArrayValues(v), nil
}

func (r *responseWriter) cookie(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.String); err != nil {
		return core.NullValue, err
	}

	rc, ok := r.writer.(*httptest.ResponseRecorder)
	if !ok {
		return core.NullValue, fmt.Errorf("the response is not a recorder")
	}

	request := &http.Request{Header: http.Header{"Cookie": rc.Header()["Set-Cookie"]}}

	name := args[0].ToString()

	// Extract the dropped cookie from the request.
	k, err := request.Cookie(name)
	if err != nil {
		return core.NullValue, err
	}

	c := &cookie{
		domain:   k.Domain,
		expires:  k.Expires,
		secure:   k.Secure,
		httpOnly: k.HttpOnly,
		name:     k.Name,
		value:    k.Value,
	}

	return core.NewObject(c), nil
}

func (r *responseWriter) addCookie(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateArgs(args, core.Object); err != nil {
		return core.NullValue, err
	}

	c, ok := args[0].ToObject().(*cookie)
	if !ok {
		return core.NullValue, ErrInvalidType
	}

	path := c.path
	if path == "" {
		path = "/"
	}

	k := &http.Cookie{
		Domain:   c.domain,
		Path:     path,
		Expires:  c.expires,
		Name:     c.name,
		Value:    c.value,
		Secure:   c.secure,
		HttpOnly: c.httpOnly,
	}

	http.SetCookie(r.writer, k)
	return core.NullValue, nil
}

func (r *responseWriter) redirect(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	var a = args[0]
	if a.Type != core.String {
		return core.NullValue, fmt.Errorf("expected a string, got %s", a.TypeName())
	}

	r.status = http.StatusFound

	http.Redirect(r.writer, r.request, a.ToString(), http.StatusFound)

	return core.NullValue, nil
}

func (r *responseWriter) write(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}
	var a = args[0]
	var b []byte

	switch a.Type {
	case core.Null, core.Undefined:
		return core.NullValue, nil
	case core.String, core.Bytes:
		b = a.ToBytes()
	default:
		b = []byte(a.String())
	}

	if err := vm.AddAllocations(len(b)); err != nil {
		return core.NullValue, err
	}

	r.writer.Write(b)

	// set 200 by default when anything is written
	if r.status == 0 {
		r.status = 200
	}

	return core.NullValue, nil
}

func (r *responseWriter) writeHTML(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}
	var a = args[0]
	var d []byte

	switch a.Type {
	case core.Null, core.Undefined:
		return core.NullValue, nil
	case core.String:
		d = []byte(html.EscapeString(a.ToString()))
	case core.Bytes:
		d = a.ToBytes()
	default:
		d = []byte(html.EscapeString(a.String()))
	}

	if err := vm.AddAllocations(len(d)); err != nil {
		return core.NullValue, err
	}

	r.writer.Write(d)

	// set 200 by default when anything is written
	if r.status == 0 {
		r.status = 200
	}

	return core.NullValue, nil
}

func (r *responseWriter) writeError(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateOptionalArgs(args, core.Int, core.String); err != nil {
		return core.NullValue, err
	}
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	code := int(args[0].ToInt())

	r.status = code

	r.writer.WriteHeader(code)

	if l == 2 {
		r.writer.Write(args[1].ToBytes())
	} else {
		switch code {
		case 400:
			r.writer.Write([]byte("Bad Request"))
		case 401:
			r.writer.Write([]byte("Unauthorized"))
		case 403:
			r.writer.Write([]byte("Forbidden"))
		case 404:
			r.writer.Write([]byte("Not Found"))
		default:
			r.writer.Write([]byte("Internal error"))
		}
	}

	r.writer.Write([]byte("\n"))
	return core.NullValue, nil
}

func (r *responseWriter) writeJSONError(args []core.Value, vm *core.VM) (core.Value, error) {
	if err := ValidateOptionalArgs(args, core.Int, core.String); err != nil {
		return core.NullValue, err
	}
	l := len(args)
	if l == 0 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	code := int(args[0].ToInt())

	r.status = code

	var err []byte

	if l == 2 {
		err = args[1].ToBytes()
	} else {
		switch code {
		case 400:
			err = []byte("Bad Request")
		case 401:
			err = []byte("Unauthorized")
		case 404:
			err = []byte("Not Found")
		default:
			err = []byte("Internal error")
		}
	}

	r.writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	r.writer.WriteHeader(code)
	r.writer.Write([]byte(`{"error":"`))
	r.writer.Write(err)
	r.writer.Write([]byte("\"}\n"))
	return core.NullValue, nil
}

func (r *responseWriter) writeJSONStatus(args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l < 2 {
		return core.NullValue, fmt.Errorf("expected 2 arguments, got %d", l)
	}

	a := args[0]
	if a.Type != core.Int {
		return core.NullValue, fmt.Errorf("expected argument 1 status of type int, got %s", a.TypeName())
	}

	return r.doWriteJSON(int(a.ToInt()), args[1:], vm)
}

func (r *responseWriter) writeJSON(args []core.Value, vm *core.VM) (core.Value, error) {
	return r.doWriteJSON(200, args, vm)
}

func (r *responseWriter) doWriteJSON(status int, args []core.Value, vm *core.VM) (core.Value, error) {
	l := len(args)
	if l == 0 || l > 2 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", l)
	}

	v := args[0].Export(0)

	r.status = status

	var skipCacheHeader bool
	if l > 1 {
		i := args[1]
		if i.Type != core.Bool {
			return core.NullValue, fmt.Errorf("expected argument 2 of type boolean, got %s", i.TypeName())
		}
		skipCacheHeader = i.ToBool()
	}

	b, err := json.Marshal(v)

	if err != nil {
		return core.NullValue, err
	}

	if err := vm.AddAllocations(len(b)); err != nil {
		return core.NullValue, err
	}

	w := r.writer
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	r.writer.WriteHeader(status)

	if !skipCacheHeader {
		w.Header().Set("Cache-Breaker", cacheBreaker)
	}

	w.Write(b)
	w.Write([]byte("\n"))
	return core.NullValue, nil
}

var contentDate = time.Date(2016, 3, 4, 0, 0, 0, 0, time.UTC)

func (r *responseWriter) writeFile(args []core.Value, vm *core.VM) (core.Value, error) {
	ln := len(args)
	if ln != 2 {
		return core.NullValue, fmt.Errorf("expected 2 or 3 arguments, got %d", ln)
	}

	if args[0].Type != core.String {
		return core.NullValue, fmt.Errorf("expected argument to be a string, got %v", args[0].TypeName())
	}

	// set 200 by default when anything is written
	if r.status == 0 {
		r.status = 200
	}

	name := args[0].ToString()

	var reader io.ReadSeeker

	switch args[1].Type {
	case core.String:
	case core.Bytes:
		data := []byte(args[1].ToBytes())
		reader = bytes.NewReader(data)
	case core.Object:
		f, ok := args[1].ToObject().(io.ReadSeeker)
		if ok {
			reader = f
		} else {
			fs, ok := args[1].ToObject().(*FileSystemObj)
			if !ok {
				return core.NullValue, ErrInvalidType
			}
			f, err := fs.FS.Open(name)
			if err != nil {
				return core.NullValue, err
			}
			reader = f
			defer f.Close()
		}
	default:
		return core.NullValue, ErrInvalidType
	}

	if strings.Contains(r.request.Header.Get("Accept-Encoding"), "gzip") {
		serveGziped(r.writer, r.request, filepath.Base(name), reader)
	} else {
		http.ServeContent(r.writer, r.request, filepath.Base(name), contentDate, reader)
	}
	return core.NullValue, nil
}

func serveGziped(w http.ResponseWriter, r *http.Request, name string, f io.ReadSeeker) {
	w.Header().Set("Content-Encoding", "gzip")
	gz := gzip.NewWriter(w)
	defer gz.Close()

	gw := &gzipResponseWriter{Writer: gz, ResponseWriter: w}
	http.ServeContent(gw, r, name, contentDate, f)
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	sniffDone bool
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.sniffDone {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(b))
		}
		w.sniffDone = true
	}

	return w.Writer.Write(b)
}

func (r *responseWriter) setContentType(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	var a = args[0]
	if a.Type != core.String {
		return core.NullValue, fmt.Errorf("expected a string, %s", a.TypeName())
	}

	r.writer.Header().Set("Content-Type", a.ToString())
	return core.NullValue, nil
}

func (r *responseWriter) setHeader(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 2 {
		return core.NullValue, fmt.Errorf("expected 2 arguments, got %d", len(args))
	}

	var a = args[0]
	if a.Type != core.String {
		return core.NullValue, fmt.Errorf("expected a string, %s", a.TypeName())
	}

	var b = args[1]
	if b.Type != core.String {
		return core.NullValue, fmt.Errorf("expected a string, %s", a.TypeName())
	}

	r.writer.Header().Set(a.ToString(), b.ToString())
	return core.NullValue, nil
}

func (r *responseWriter) setStatus(args []core.Value, vm *core.VM) (core.Value, error) {
	if len(args) != 1 {
		return core.NullValue, fmt.Errorf("expected 1 argument, got %d", len(args))
	}

	var a = args[0]
	if a.Type != core.Int {
		return core.NullValue, fmt.Errorf("expected a int, %s", a.TypeName())
	}

	s := int(a.ToInt())
	r.writer.WriteHeader(s)
	r.status = s
	return core.NullValue, nil
}
