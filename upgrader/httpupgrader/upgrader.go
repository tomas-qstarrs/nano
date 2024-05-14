// Copyright (c) nano Authors. All Rights Reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package httpupgrader

import (
	"bufio"
	"fmt"
	"net"
	"net/http"

	"github.com/tomas-qstarrs/nano/upgrader"
)

type Upgrader struct{}

func NewUpgrader() *Upgrader {
	return &Upgrader{}
}

var defaultUpgrader = NewUpgrader()

func Default() upgrader.Upgrader {
	return defaultUpgrader
}

func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request, params map[string]string) (net.Conn, error) {
	h, ok := w.(http.Hijacker)
	if !ok {
		return nil, fmt.Errorf("wrong type of response writer")
	}
	var brw *bufio.ReadWriter
	conn, brw, err := h.Hijack()
	if err != nil {
		return nil, err
	}

	c := NewConn(w, r, conn, brw, params)
	return c, nil
}
