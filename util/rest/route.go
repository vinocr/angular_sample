//Copyright 2017 Huawei Technologies Co., Ltd
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.
package rest

import (
	"errors"
	"fmt"
	"github.com/ServiceComb/service-center/util"
	"net/http"
	"net/url"
	"strings"
)

type URLPattern struct {
	Method string
	Path   string
}

type urlPatternHandler struct {
	Path string
	http.Handler
}

type Route struct {
	// Method is one of the following: GET,PUT,POST,DELETE
	Method string
	// Path contains a path pattern
	Path string
	// rest callback function for the specified Method and Path
	Func func(w http.ResponseWriter, r *http.Request)
}

// HTTP request multiplexer
// Attention:
//   1. not thread-safe, must be initialized completely before serve http request
//   2. redirect not supported
type ROAServerHandler struct {
	handlers map[string][]*urlPatternHandler
}

func NewROAServerHander() *ROAServerHandler {
	return &ROAServerHandler{
		handlers: make(map[string][]*urlPatternHandler)}
}

func (this *ROAServerHandler) addRoute(route *Route) (err error) {
	method := strings.ToUpper(route.Method)
	if !isValidMethod(method) || !strings.HasPrefix(route.Path, "/") || route.Func == nil {
		message := fmt.Sprintf("Invalid route parameters(method: %s, path: %s)", method, route.Path)
		util.Logger().Errorf(nil, message)
		return errors.New(message)
	}

	this.handlers[method] = append(this.handlers[method], &urlPatternHandler{route.Path, http.HandlerFunc(route.Func)})
	util.Logger().Infof("register route %s(%s).", route.Path, method)

	return nil
}

func (this *ROAServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, ph := range this.handlers[r.Method] {
		if params, ok := ph.try(r.URL.Path); ok {
			if len(params) > 0 {
				r.URL.RawQuery = url.Values(params).Encode() + "&" + r.URL.RawQuery
			}

			ph.ServeHTTP(w, r)
			return
		}
	}

	allowed := make([]string, 0, len(this.handlers))
	for meth, handlers := range this.handlers {
		if meth == r.Method {
			continue
		}

		for _, ph := range handlers {
			if _, ok := ph.try(r.URL.Path); ok {
				allowed = append(allowed, meth)
			}
		}
	}

	if len(allowed) == 0 {
		http.NotFound(w, r)
		return
	}

	w.Header().Add("Allow", util.StringJoin(allowed, ", "))
	http.Error(w, "Method Not Allowed", 405)
}

func (this *urlPatternHandler) try(path string) (p map[string][]string, _ bool) {
	var i, j int
	for i < len(path) {
		switch {
		case j >= len(this.Path):
			if this.Path != "/" && len(this.Path) > 0 && this.Path[len(this.Path)-1] == '/' {
				return p, true
			}
			return nil, false
		case this.Path[j] == ':':
			var val string
			var nextc byte
			o := j
			_, nextc, j = match(this.Path, isAlnum, 0, j+1)
			val, _, i = match(path, matchParticial, nextc, i)

			if p == nil {
				p = make(map[string][]string)
			}
			p[this.Path[o:j]] = []string{val}
		case path[i] == this.Path[j]:
			i++
			j++
		default:
			return nil, false
		}
	}
	if j != len(this.Path) {
		return nil, false
	}
	return p, true
}

func match(s string, f func(c byte) bool, exclude byte, i int) (matched string, next byte, j int) {
	j = i
	for j < len(s) && f(s[j]) && s[j] != exclude {
		j++
	}

	if j < len(s) {
		next = s[j]
	}
	return s[i:j], next, j
}

func matchParticial(c byte) bool {
	return c != '/'
}

func isAlpha(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isAlnum(ch byte) bool {
	return isAlpha(ch) || isDigit(ch)
}
