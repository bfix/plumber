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
	"errors"
	"io"
	"os"
)

// Action triggered by object "plumb"
type Action func(msg *Message, verb, data string) (ok bool, done bool)

// NewAction returns a new 'plumb' function
type NewAction func() Action

// Plumber
type Plumber struct {
	rl     *RuleList
	worker NewAction
}

// NewPlumber creates a new plumber instance
func NewPlumber(worker NewAction) *Plumber {
	return &Plumber{
		worker: worker,
	}
}

// ParsePlumbingFromRdr reads rulesets from a reader
func (p *Plumber) ParsePlumbingFromRdr(rdr io.Reader) (err error) {
	p.rl, err = ParsePlumbingFromRdr(rdr)
	p.rl.Exec = p.worker
	return
}

// ParsePlumbingFile with a fallback if the initial read fails
func (p *Plumber) ParsePlumbingFile(fname, fallback string) error {
	err := p.parsePlumbingFile(fname)
	if err != nil {
		err = p.parsePlumbingFile(fallback)
	}
	return err
}

// ParsePlumbingFile reads rules from a file.
// If the file does not exist, the
func (p *Plumber) parsePlumbingFile(fname string) error {
	if len(fname) == 0 {
		return errors.New("no filename")
	}
	f, err := os.Open(fname)
	if err != nil {
		return err
	}
	defer f.Close()
	return p.ParsePlumbingFromRdr(f)
}

// Ports returns a list of all ports referenced in the current list of rules
func (p *Plumber) Ports() (list []string) {
	return p.rl.Ports()
}

// File returns the current rules as a byte array
func (p *Plumber) File() []byte {
	return p.rl.File()
}

// Env returns the current environment from the rules file
func (p *Plumber) Env() map[string]string {
	return p.rl.Env
}

// Eval runs evaluation of data based on defined rules
func (p *Plumber) Eval(data, src, dst, wdir string) (bool, error) {
	msg := &Message{
		Src:   src,
		Dst:   dst,
		Wdir:  wdir,
		Attr:  make(map[string]string),
		Ndata: len(data),
		Data:  data,
	}
	out, _, err := p.rl.Evaluate(msg, false)
	return out != nil, err
}

// Process a plumbing message
func (p *Plumber) Process(msg *Message) (bool, error) {
	out, _, err := p.rl.Evaluate(msg, false)
	return out != nil, err
}
