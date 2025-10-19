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
	"io"
)

// Plumber
type Plumber struct {
	rs    *Ruleset
	exec  Action
	rules []byte
}

func NewPlumber(call Action) *Plumber {
	return &Plumber{
		exec:  call,
		rules: []byte{},
	}
}

// ParseRuleset from a reader
func (p *Plumber) ParseRuleset(rdr io.Reader, env map[string]string) (err error) {
	p.rs, err = ParseRuleset(rdr, env)
	p.rs.Exec = p.exec
	p.rules = []byte(p.rs.String())
	return
}

func (p *Plumber) Ports() (list []string) {
	for _, r := range p.rs.Rules {
		for _, c := range r.Stmts {
			if c.Obj == "plumb" && c.Verb == "to" {
				list = append(list, c.Data)
			}
		}
	}
	return
}

func (p *Plumber) Rules() []byte {
	return p.rules
}

func (p *Plumber) Env() map[string]string {
	return p.rs.Env
}

// Eval runs evaluation of data based on defined rules
func (p *Plumber) Eval(data, src, dst, wdir string) error {
	msg := &Message{
		Src:  src,
		Dst:  dst,
		Wdir: wdir,
		Attr: make(map[string]string),
		Data: data,
	}
	_, _, err := p.rs.Evaluate(msg, false)
	return err
}

func (p *Plumber) Process(msg *Message) error {
	_, _, err := p.rs.Evaluate(msg, false)
	return err
}

func (p *Plumber) ReadRules(ofs uint64, num uint32) ([]byte, error) {
	count := uint64(len(p.rules))
	if ofs > count-1 {
		return []byte{}, nil
	}
	n := min(count, ofs+uint64(num))
	return p.rules[ofs:n], nil
}

func (p *Plumber) WriteRules(ofs uint64, data []byte) (uint32, error) {
	return uint32(len(data)), nil
}
