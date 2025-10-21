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
	"fmt"
	"io"
	"strings"

	"github.com/bfix/gospel/data"
	"github.com/bfix/gospel/logger"
)

// RuleList is a list of rules and environment variables
type RuleList struct {
	file     []byte            // plumbing file
	Rulesets []*RuleSet        // list of rules
	Env      map[string]string // environment variables
	Exec     NewAction         // plumbing action
}

// Evaluate incoming message against all rulesets.
// If msg is not null, rid points to the matching ruleset
func (rl *RuleList) Evaluate(in *Message, withFS bool) (out *Message, rid int, err error) {
	rid = -1
	for i, r := range rl.Rulesets {
		if out, err = r.Evaluate(in, rl.Env, withFS, rl.Exec); err != nil {
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

// String returns the active rules as string
func (rl *RuleList) String() string {
	return string(rl.file)
}

// Ports returns all ports referenced in in list
func (rl *RuleList) Ports() (list []string) {
	for _, r := range rl.Rulesets {
		list = append(list, r.Ports()...)
	}
	return
}

// ParsePlumbingFile reads a list of rules and environment settings from a reader
func ParsePlumbingFile(in io.Reader, env map[string]string) (rs *RuleList, err error) {
	if env == nil {
		env = make(map[string]string)
	}
	rs = &RuleList{
		file:     []byte{},
		Rulesets: []*RuleSet{},
		Env:      env,
	}

	// parse rules
	parseRuleSet := func(r string) {
		var rule *RuleSet
		if rule, err = ParseRuleSet(r); err != nil {
			return
		}
		rs.Rulesets = append(rs.Rulesets, rule)
	}

	// read rules as a list of multi-line strings
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
		rs.file = append(rs.file, s...)
		rs.file = append(rs.file, '\n')

		// skip comments
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
				parseRuleSet(buf)
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
		parseRuleSet(buf)
	}
	return
}

//----------------------------------------------------------------------

// RuleSet is a list of rules that are evaluated against an input
type RuleSet struct {
	Rules []any // can be *Rule or *RuleSet
}

// ParseRuleSet parses a single ruleset from a multi-line string
// Rulesets can be nested.
func ParseRuleSet(s string) (r *RuleSet, err error) {
	var curr []any
	st := data.NewStack()
	for line := range strings.SplitSeq(s, "\n") {
		// skip comments
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		// handle nesting
		if line[0] == '{' {
			st.Push(curr)
			curr = []any{}
			continue
		} else if line[0] == '}' {
			last := st.Pop().([]any)
			last = append(last, curr)
			curr = last
			continue
		}
		// parse rule
		line = Canonical(line)
		logger.Println(logger.DBG, line)
		words := strings.SplitN(line, " ", 3)
		if !grammer.Valid(words[0], words[1]) {
			err = fmt.Errorf("invalid rule: '%s'", line)
			break
		}
		rule := &Rule{
			Obj:  words[0],
			Verb: words[1],
			Data: words[2],
		}
		curr = append(curr, rule)
	}
	return &RuleSet{
		Rules: curr,
	}, nil
}

// String returns a human-readble representation of a rule
func (r *RuleSet) String() string {
	return strings.Join(r.lines(""), "\n")
}

// return the ruleset as a list of rule lines (correctly indented)
func (r *RuleSet) lines(indent string) (list []string) {
	for _, c := range r.Rules {
		switch x := c.(type) {
		case *Rule:
			list = append(list, x.String())
		case []any:
			list = append(list, indent+"{")
			s := (&RuleSet{x}).lines(indent + "  ")
			list = append(list, s...)
			list = append(list, indent+"}")
		}
	}
	return
}

// Ports returns a list of referenced ports in the ruleset
func (r *RuleSet) Ports() (list []string) {
	for _, c := range r.Rules {
		switch x := c.(type) {
		case *Rule:
			if x.Obj == "plumb" && x.Verb == "to" {
				list = append(list, x.Data)
			}
		case []any:
			p := (&RuleSet{x}).Ports()
			list = append(list, p...)
		}
	}
	return
}

// Evaluate a rule against input
func (r *RuleSet) Evaluate(in *Message, env map[string]string, withFS bool, worker NewAction) (out *Message, err error,
) {
	k := NewKernel(worker())
	k.Message = *(in.Clone())
	k.withFS = withFS

	st := data.NewStack()
	var eval func([]any) (*Message, error)
	eval = func(rules []any) (*Message, error) {
		for _, rule := range rules {
			switch x := rule.(type) {
			case *Rule:
				ok, done, err := k.Execute(x, env)
				logger.Printf(logger.DBG, "! %s -> ok=%v, done=%v", x.String(), ok, done)
				if err != nil {
					return nil, err
				}
				if !ok {
					return nil, nil
				}
				if done {
					return &k.Message, nil
				}
			case []any:
				st.Push(k.Clone())
				logger.Println(logger.DBG, "! branch down")
				out, err := eval(x)
				logger.Printf(logger.DBG, "! branch up -> out=%v, err=%v", out != nil, err)
				k = st.Pop().(*Kernel)
				if err != nil || out != nil {
					return out, err
				}
			}
		}
		return nil, nil
	}
	return eval(r.Rules)
}
