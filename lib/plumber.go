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
	rs    *RuleList
	exec  Action
	rules []byte
}

// NewPlumber creates a new plumber instance
func NewPlumber(call Action) *Plumber {
	return &Plumber{
		exec:  call,
		rules: []byte{},
	}
}

// ParseRulesFile from a reader
func (p *Plumber) ParseRulesFile(rdr io.Reader, env map[string]string) (err error) {
	p.rs, err = ParseRulesFile(rdr, env)
	p.rs.Exec = p.exec
	p.rules = []byte(p.rs.String())
	return
}

// Ports returns a list of all ports referenced in the current list of rules
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

// Rules returns the current rules as a byte array
func (p *Plumber) Rules() []byte {
	return p.rules
}

// Env returns the current environment from the rules file
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

// Process a plumbing message
func (p *Plumber) Process(msg *Message) error {
	_, _, err := p.rs.Evaluate(msg, false)
	return err
}
