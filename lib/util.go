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

import "strings"

// ParseParts splits a string into parts separated
// by whitespaces. N.B.: If the input string is quoted,
// the parts may contain whitespaces.
func ParseParts(line string) (parts []string) {
	// parse command
	state := 0
	spaced := true
	escaped := false
	var quote rune
	var part []rune
	for _, ch := range line {
		switch state {
		case 0: // outside quoted string
			switch ch {
			case ' ', '\t':
				if !spaced {
					spaced = true
					if len(part) > 0 {
						parts = append(parts, string(part))
						part = []rune{}
					}
				}
				continue
			case '"', '\'':
				state = 1
				quote = ch
				fallthrough
			default:
				spaced = false
			}
		case 1: // inside quoted string
			switch ch {
			case '\\':
				escaped = !escaped
			case '"', '\'':
				if !escaped && quote == ch {
					state = 0
				}
				fallthrough
			default:
				escaped = false
			}
		}
		part = append(part, ch)
	}
	if len(part) > 0 {
		parts = append(parts, string(part))
	}
	return
}

// Canonical transforms a string to unified format (segments separated by
// a single space). An unquoted segment is a sequence of non-whitespace
// characters; a quoted segment may contain whitespaces.
func Canonical(v string) string {
	return strings.Join(ParseParts(v), " ")
}
