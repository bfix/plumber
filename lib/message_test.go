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
	"slices"
	"strings"
	"testing"
)

var msg = `lola
outerspace
/home/glenda
test
cat=url type=web
16
https://p9f.org/`

func TestMessageParse(t *testing.T) {
	m, err := ParseMessage(msg)
	if err != nil {
		t.Log(msg)
		t.Fatal(err)
	}
	o := m.String()

	inP := strings.Split(msg, "\n")
	outP := strings.Split(o, "\n")
	ok := true
	for i, in := range inP {
		if i != 4 {
			if in != outP[i] {
				ok = false
				break
			}
		} else {
			inA := strings.Split(in, " ")
			slices.Sort(inA)
			outA := strings.Split(outP[4], " ")
			slices.Sort(outA)
			if !slices.Equal(inA, outA) {
				ok = false
				break
			}
		}
	}
	if !ok {
		t.Log(msg)
		t.Log(o)
		t.Fatal("mismatch")
	}
}

func TestMessageMultilineData(t *testing.T) {
	m := &Message{
		Data: msg,
	}
	out := m.packData()
	exp := "base64:bG9sYQpvdXRlcnNwYWNlCi9ob21lL2dsZW5kYQp0ZXN0CmNhdD11cmwgdHlwZT13ZWIKMTYKaHR0cHM6Ly9wOWYub3JnLw=="
	if out != exp {
		t.Log(out)
		t.Log(exp)
		t.Fatal("mismatch")
	}
}
