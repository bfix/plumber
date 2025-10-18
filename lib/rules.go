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
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// Ruleset is a list of rules and environment variables
type Ruleset struct {
	Rules []*Rule           // list of rules
	Env   map[string]string // environment variables
	Exec  Action            // plumbing action
}

// Evaluate data,src,dst,wdir against all rules in set.
// If msg is not null, rid points to the matching rule
func (rs *Ruleset) Evaluate(in *Message, withFS bool) (out *Message, rid int, err error) {
	rid = -1
	for i, r := range rs.Rules {
		if out, err = r.Evaluate(in, rs.Env, withFS, rs.Exec); err != nil {
			return
		}
		if out == nil {
			continue
		}
		rid = i
		break
	}
	return
}

// String returns the active ruleset as string
func (rs *Ruleset) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString("# active plumbing ruleset\n\n")
	for k, v := range rs.Env {
		buf.WriteString(fmt.Sprintf("%s = %s\n", k, v))
	}
	buf.WriteString("\n# rules\n\n")
	for _, r := range rs.Rules {
		buf.WriteString(r.String() + "\n\n")
	}
	return buf.String()
}

// ParseRuleset reads a list of rules and environment settings from a reader
func ParseRuleset(in io.Reader, env map[string]string) (rs *Ruleset, err error) {
	if env == nil {
		env = make(map[string]string)
	}
	rs = &Ruleset{
		Rules: make([]*Rule, 0),
		Env:   env,
	}
	// read rules as a list of multi-line strings
	var list []string
	rdr := bufio.NewReader(in)
	buf := ""
	for {
		// read next line
		var s []byte
		if s, _, err = rdr.ReadLine(); err != nil {
			if err == io.EOF {
				break
			}
			return
		}
		if len(s) > 0 && s[0] == '#' {
			continue
		}
		line := Canonical(string(s))

		// check for enviroment setting
		t := Canonical(strings.Replace(line, "=", " = ", 1))
		if parts := strings.SplitN(t, " ", 3); len(parts) == 3 && parts[1] == "=" {
			rs.Env[parts[0]] = parts[2]
			continue
		}

		// handle possible rule
		if len(line) == 0 {
			if len(buf) > 0 {
				list = append(list, buf)
			}
			buf = ""
			continue
		}
		if len(buf) > 0 {
			buf += "\n"
		}
		buf += Canonical(line)
	}
	if len(buf) > 0 {
		list = append(list, buf)
	}
	// parse rules
	for _, r := range list {
		var rule *Rule
		if rule, err = ParseRule(r); err != nil {
			return
		}
		rs.Rules = append(rs.Rules, rule)
	}
	return
}

//----------------------------------------------------------------------

// Rule is a list of clause that are evaluated against an input
type Rule struct {
	Stmts []*Clause
}

// ParseRule parses a single rule from a multi-line string
func ParseRule(s string) (r *Rule, err error) {
	r = &Rule{
		Stmts: make([]*Clause, 0),
	}
	for _, line := range strings.Split(s, "\n") {
		if len(line) == 0 {
			continue
		}
		line = Canonical(line)
		words := strings.SplitN(line, " ", 3)
		if !grammer.Valid(words[0], words[1]) {
			err = fmt.Errorf("invalid clause: '%s'", line)
			break
		}
		cl := &Clause{
			Obj:  words[0],
			Verb: words[1],
			Data: words[2],
		}
		r.Stmts = append(r.Stmts, cl)
	}
	return
}

// String returns a human-readble representation of a rule
func (r *Rule) String() string {
	var list []string
	for _, c := range r.Stmts {
		list = append(list, c.String())
	}
	return strings.Join(list, "\n")
}

// Evaluate a rule against input
func (r *Rule) Evaluate(in *Message, env map[string]string, withFS bool, action Action) (out *Message, err error,
) {
	k := NewKernel(action)
	k.Message = *in
	k.withFS = withFS

	for _, cl := range r.Stmts {
		var ok, done bool
		if ok, done, err = k.Execute(cl, env); err != nil {
			return
		}
		if !ok {
			return nil, nil
		}
		if done {
			break
		}
	}
	return &k.Message, nil
}
