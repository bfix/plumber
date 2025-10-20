//----------------------------------------------------------------------
// This file is part of plumber.
// Copyright (C) 2024-present Bernd Fix   >Y<
//
// plumber is free software: you can redistribute it and/or modify it
// under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// plumber is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
// Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.
//
// SPDX-License-Identifier: AGPL3.0-or-later
//----------------------------------------------------------------------

/*
 * The code of this package is heavily inspired by the Plan9 implementation
 * of the plumber (see $plan9/src/{cmd/plumb,lib/libplumb,sys/include}).
 * It is not a clean-room re-implementation, but a transformation of the
 * basic approach to Go. Its messages are interoperable.
 */

package lib

import (
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/bfix/gospel/logger"
)

// Grammer contains a list of (valid) verbs for an object
type Grammer map[string][]string

// Valid return true if an object can have a certain verb
func (g Grammer) Valid(obj, verb string) bool {
	verbs, ok := g[obj]
	if !ok {
		return false
	}
	return slices.Contains(verbs, verb)
}

var (
	// define the grammer of rules:
	// object: { verb1, verb2, ...}
	grammer = Grammer{
		"arg":   {"isdir", "isfile"},
		"attr":  {"add", "delete"},
		"data":  {"is", "set", "matches"},
		"dst":   {"is", "set", "matches"},
		"plumb": {"client", "start", "to"},
		"src":   {"is", "set", "matches"},
		"type":  {"is"},
		"wdir":  {"is", "set", "matches"},
	}
)

// Rule to evaluate
type Rule struct {
	// Obj of action: arg|attr|data|dst|plumb|src|type|wdir
	Obj string
	// Verb of action: is,matches,set|add,delete|isdir,isfile|to,client,start
	Verb string
	// regexp, list of key/value pairs or literal value
	// (depending on Obj). Can contain variables and may be quoted.
	Data string
}

// String returns a human-readble rule
func (r *Rule) String() string {
	return r.Obj + " " + r.Verb + " " + r.Data
}

// Kernel is the environment for executing rules against input data
type Kernel struct {
	Message
	re     *regexp.Regexp
	withFS bool              // if true "isfile" and "isdir" work on the filesystem
	dollar []string          // result of last match
	vars   map[string]string // variables
	worker Action            // performs plumbing action
}

// NewKernel creates a new kernel instance
func NewKernel(w Action) *Kernel {
	return &Kernel{
		Message: Message{
			Attr: make(map[string]string),
		},
		withFS: true,
		dollar: []string{},
		vars:   make(map[string]string),
		worker: w,
	}
}

// Get a variable value from kernel
func (k *Kernel) Get(name string) (string, error) {
	switch name {
	case "arg", "plumb":
		// ignore value-less "variables"
		return "", nil
	}
	return k.Message.Get(name)
}

// Execute a rule with the given environment in the kernel
func (k *Kernel) Execute(r *Rule, env map[string]string) (ok bool, done bool, err error) {
	// currently only text data (maybe encoded ;)
	k.Type = "text"

	// get object and data value
	var obj string
	if obj, err = k.Get(r.Obj); err != nil {
		return
	}
	data := k.expand(r.Data, env)

	// handle verbs: the meaning of a verb is independent from the object
	ok = false
	switch r.Verb {
	case "matches":
		if k.re, err = regexp.Compile(data); err != nil {
			break
		}
		matches := k.re.FindAllStringSubmatch(obj, -1)
		logger.Printf(logger.DBG, "| match '%s' against '%s' => %v", obj, data, matches)
		if ok = (matches != nil && (obj == matches[0][0])); ok {
			k.dollar = matches[0]
		}
	case "is":
		ok = (obj == data)
	case "isdir":
		if k.withFS {
			fn := data
			if !strings.HasPrefix(fn, "/") {
				fn = k.Wdir + "/" + fn
			}
			var fi os.FileInfo
			if fi, err = os.Stat(fn); err == nil {
				ok = fi.IsDir()
			} else {
				err = nil
			}
		} else {
			ok = true
		}
		if ok {
			k.vars["dir"] = data
		}
	case "isfile":
		if k.withFS {
			fn := data
			if !strings.HasPrefix(fn, "/") {
				fn = k.Wdir + "/" + fn
			}
			var fi os.FileInfo
			if fi, err = os.Stat(fn); err == nil {
				ok = !fi.IsDir()
			} else {
				err = nil
			}
		} else {
			ok = true
		}
		if ok {
			k.vars["file"] = data
		}
	case "set":
		ok = k.Set(r.Obj, data)
	case "add":
		maps.Copy(k.Attr, k.unpackAttr(data))
		ok = true
		k.vars["attr"] = k.GetAttr()
	case "delete":
		delete(k.Attr, data)
		ok = true
		k.vars["attr"] = k.GetAttr()
	case "to", "start", "client":
		ok = true
		if k.worker != nil {
			ok, done = k.worker(&k.Message, r.Verb, data)
		}
	default:
		err = fmt.Errorf("not implemented: '%s'", r)
	}
	k.Ndata = len(k.Data)
	return
}

// expand $-variables in unquoted string
func (k *Kernel) expand(s string, env map[string]string) string {
	lookup := func(name string) string {
		if i, err := strconv.Atoi(name); err == nil {
			if i >= 0 && i < len(k.dollar) {
				return k.dollar[i]
			}
		}
		if v, err := k.Get(name); err == nil {
			return v
		}
		return env[name]
	}
	out := Unquote(s, lookup)
	return out
}
