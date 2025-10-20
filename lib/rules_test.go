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
	"os"
	"testing"
)

func getRuleList(fname string) (rs *RuleList, err error) {
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ParseRulesFile(f, nil)
}

func TestRulesInOut(t *testing.T) {
	rs, err := getRuleList("../rules/plan9")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%d rules read", len(rs.Rulesets))
}

func TestRulesEval(t *testing.T) {
	data := [][]string{
		{"+49 30 12345678", "", ""},
		{"rfc9498", "", ""},
		{"https://9p.sdf.org", "", ""},
		{"/usr/glenda/docs/readme.rtf", "", ""},
		{"docs/paper.doc", "", "msword"},
		{"user@domain.com", "", ""},
		{"music/tune.mp3", "", ""},
		{"images/cover.png", "", ""},
		{"docs/paper.pdf!123", "", ""},
		{"docs/paper.pdf", "", ""},
		{"plumb.h:23", "", ""},
		{"fortress.m:42", "", ""},
		{"/mail/fs/mbox/35", "", ""},
		{"intro(1)", "", ""},
		{"src/main.go:87", "", ""},
		{"Local date", "", ""},
	}
	rs, err := getRuleList("../rules/plan9")
	if err != nil {
		t.Fatal(err)
	}

	for i, d := range data {
		msg := &Message{
			Src:  d[1],
			Dst:  d[2],
			Wdir: "",
			Attr: make(map[string]string),
		}
		_, rid, err := rs.Evaluate(msg, false)
		if err != nil {
			t.Fatal(err)
		}
		if i != rid {
			t.Log(d[0])
			t.Fatalf("rule mismatch: %d != %d", i, rid)
		}
	}
}
