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
	"testing"
)

func TestCanonical(t *testing.T) {
	data := [][]string{
		{" A  simple, but\teffective\t way. ", "A simple, but effective way."},
		{"'Our master plan:'  Keep calm and  \t $action!", "'Our master plan:' Keep calm and $action!"},
		{"'You have chosen the '$color' pill...'", "'You have chosen the '$color' pill...'"},
		{"rc -c '''echo % mail '''$0'; mail '$0", ""},
	}
	for _, d := range data {
		e := Canonical(d[0])
		f := d[1]
		if len(f) == 0 {
			f = d[0]
		}
		if e != f {
			t.Log(d[0])
			t.Log(e)
			t.Log(f)
			t.Fatal("mismatch")
		}
	}
}
