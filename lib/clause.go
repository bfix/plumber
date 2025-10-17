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
	// define the grammer of clauses:
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

// Clause to evaluate (a rule is a sequene of clauses)
type Clause struct {
	// Obj of action: arg|attr|data|dst|plumb|src|type|wdir
	Obj string
	// Verb of action: is,matches,set|add,delete|isdir,isfile|to,client,start
	Verb string
	// regexp, list of key/value pairs or literal value
	// (depending on Obj). Can contain variables and may be quoted.
	Data string
}

// String returns a human-readble clause
func (cl *Clause) String() string {
	return cl.Obj + " " + cl.Verb + " " + cl.Data
}

// Action triggered by object "plumb"
type Action func(msg *Message, verb, data string) (bool, bool)

// Kernel is the environment for executing clauses against input data
type Kernel struct {
	Message
	re     *regexp.Regexp
	withFS bool              // if true "isfile" and "isdir" work on the filesystem
	dollar []string          // result of last match
	vars   map[string]string // variables
	plumb  Action            // plumbing action
}

// NewKernel creates a new kernel instance
func NewKernel(a Action) *Kernel {
	return &Kernel{
		Message: Message{
			Attr: make(map[string]string),
		},
		withFS: true,
		dollar: make([]string, 0),
		vars:   make(map[string]string),
		plumb:  a,
	}
}

func (k *Kernel) lookup(name string) string {
	return name
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

// Execute a clause with the given environment in the kernel
func (k *Kernel) Execute(cl *Clause, env map[string]string) (ok, done bool, err error) {
	// currently only text data (maybe encoded ;)
	k.Type = "text"

	// get object and data value
	var obj string
	if obj, err = k.Get(cl.Obj); err != nil {
		return
	}
	data := k.expand(cl.Data, env)

	// handle verbs: the meaning of a verb is independent from the object
	ok = false
	switch cl.Verb {
	case "matches":
		uqd := Unquote(data, k.lookup)
		if k.re, err = regexp.Compile(uqd); err != nil {
			break
		}
		matches := k.re.FindAllStringSubmatch(obj, -1)
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
		ok = k.Set(cl.Obj, data)
	case "add":
		maps.Copy(k.Attr, k.unpackAttr(data))
		ok = true
		k.vars["attr"] = k.GetAttr()
	case "delete":
		delete(k.Attr, data)
		ok = true
		k.vars["attr"] = k.GetAttr()
	case "to", "start", "client":
		ok, done = k.plumb(&k.Message, cl.Verb, data)
	default:
		err = fmt.Errorf("not implemented: '%s'", cl)
	}
	k.Ndata = len(k.Data)
	return
}

// expand $-variables in unquoted string
func (k *Kernel) expand(s string, env map[string]string) string {
	lookup := func(name string) string {
		if i, err := strconv.Atoi(name); err == nil {
			return k.dollar[i]
		}
		if v, err := k.Get(name); err == nil {
			return v
		}
		return env[name]
	}
	out := Unquote(s, lookup)
	return out
}
