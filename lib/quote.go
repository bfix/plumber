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
	"strings"
	"unicode"
)

// flag for quoted/unquoted string segments
var kind = map[bool]rune{
	true:  '+',
	false: '-',
}

// Quote a string with special characters
func Quote(v string) string {
	if strings.Contains(v, " '=\t") {
		return "'" + strings.ReplaceAll(v, "'", "''") + "'"
	}
	return v
}

// Lookup function for variables
type Lookup func(string) string

// Unquote and expand a string with variables:
// Variable lookup only happens in unquoted segments and is done via
// the lookup function.
func Unquote(in string, look Lookup) (out string) {
	inQuote := false
	segm := make([][]rune, 1)
	segm[0] = []rune{kind[inQuote]}
	idx := 0

	var last rune
	for _, ch := range in {
		if ch == '\'' {
			inQuote = !inQuote
			idx++
			segm = append(segm, []rune{kind[inQuote]})
			if inQuote && last == ch {
				segm = segm[:idx-1]
				idx -= 2
				segm[idx] = append(segm[idx], ch)
				ch = 0
			}
			last = ch
			continue
		}
		last = ch
		segm[idx] = append(segm[idx], ch)
	}
	for _, s := range segm {
		if len(s) == 1 {
			continue
		}
		p := string(s[1:])
		if s[0] == kind[false] {
			out += expand(p, look)
		} else {
			out += Quote(p)
		}
	}
	return
}

// expand unquoted string with variables.
// Variables will be unquoted (and possibly expanded).
func expand(in string, look Lookup) (out string) {
	if look == nil {
		return in
	}
	var key string
	var skip int
	for {
		i := strings.IndexRune(in, '$')
		if i == -1 || i+1 > len(in)-1 {
			out += in
			break
		}
		if j := int(in[i+1] - '0'); j >= 0 && j < 10 {
			key = in[i+1 : i+2]
			skip = 1
		} else {
			n := strings.IndexFunc(in[i+1:], func(r rune) bool {
				return !unicode.IsLetter(r)
			})
			if n == -1 {
				n = len(in[i+1:])
			}
			key = in[i+1 : i+1+n]
			skip = n
		}
		out += in[:i]
		if len(key) > 0 {
			out += Unquote(look(key), look)
		}
		in = in[i+1+skip:]
	}
	return
}
