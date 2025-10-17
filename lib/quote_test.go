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

package lib

import (
	"testing"
)

type params map[string]string

func (p params) lookup(name string) string {
	return p[name]
}

func runExpand(t *testing.T, look Lookup, in, exp string) {
	t.Helper()
	out := expand(in, look)
	if out != exp {
		t.Log(out)
		t.Log(exp)
		t.Fatal("mismatch")
	}
}

func runUnquote(t *testing.T, look Lookup, in, exp string) {
	t.Helper()
	out := Unquote(in, look)
	if out != exp {
		t.Log(out)
		t.Log(exp)
		t.Fatal("mismatch")
	}
}

func TestExpand1(t *testing.T) {
	vars := params{
		"var": "good",
	}
	in := "this is a $var thing!"
	exp := "this is a good thing!"
	runExpand(t, vars.lookup, in, exp)
}

func TestExpand2(t *testing.T) {
	vars := params{
		"mood": "very",
		"var":  "$mood good",
	}
	in := "this is a $var thing!"
	exp := "this is a very good thing!"
	runExpand(t, vars.lookup, in, exp)
}

func TestUnquote1(t *testing.T) {
	vars := params{
		"addrelem": `'((#?[0-9]+)|(/[A-Za-z0-9_\^]+/?)|[.$])'`,
		"addr":     `($addrelem([,;+\-]$addrelem)*)`,
	}
	in := `'([a-zA-Z¡-￿0-9]+\.h)('$addr')?'`
	exp := `([a-zA-Z¡-￿0-9]+\.h)((((#?[0-9]+)|(/[A-Za-z0-9_\^]+/?)|[.$])([,;+\-]((#?[0-9]+)|(/[A-Za-z0-9_\^]+/?)|[.$]))*))?`
	runUnquote(t, vars.lookup, in, exp)
}

func TestUnquote2(t *testing.T) {
	data := [][]string{
		{`'It''s so simple, isn''t it?'`, "It's so simple, isn't it?"},
		{"''''", "'"},
	}
	for _, d := range data {
		runUnquote(t, nil, d[0], d[1])
	}
}

func TestUnquote3(t *testing.T) {
	vars := params{
		"addr": `':(#?[0-9]+)'`,
	}
	in := `'([a-zA-Z¡-￿0-9]+\.h)('$addr')?'`
	exp := `([a-zA-Z¡-￿0-9]+\.h)(:(#?[0-9]+))?`
	runUnquote(t, vars.lookup, in, exp)
}

func TestUnquote4(t *testing.T) {
	vars := params{
		"0": "user@example.org",
	}
	in := "rc -c '''echo % mail '''$0'; mail '$0"
	exp := "rc -c 'echo % mail 'user@example.org; mail user@example.org"
	runUnquote(t, vars.lookup, in, exp)
}
