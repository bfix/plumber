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

package main

import (
	"os/exec"

	"github.com/bfix/gospel/logger"
	"github.com/bfix/plumber/lib"
	"github.com/knusbaum/go9p"
	"github.com/knusbaum/go9p/fs"
)

// Plumber with namespace handling
type Plumber struct {
	lib.Plumber // base plumber logic

	srv   go9p.Srv             // 9P server
	fs    *fs.FS               // synth. filesystem
	root  *fs.StaticDir        // root folder
	ports map[string]*PortFile // list of plumbing ports
	Dry   bool                 // dry run (on exec)
}

// NewPlumber
func NewPlumber() *Plumber {
	return &Plumber{
		ports: make(map[string]*PortFile),
	}
}

// NamespaceService returns a service instance
func (p *Plumber) NamespaceService() {
	p.ports = make(map[string]*PortFile)

	p.fs, p.root = fs.NewFS("plumb", "plumb", 0775)
	p.root.AddChild(NewRulesFile(p.fs.NewStat("rules", "plumb", "plumb", 0666), p, p.SyncPorts))
	p.root.AddChild(NewSendFile(p.fs.NewStat("send", "plumb", "plumb", 0222), p))
	p.srv = p.fs.Server()
	p.SyncPorts()
}

// SyncPorts after rule changes. New ports are created, but unused ports
// are not removed from the filesystem.
func (p *Plumber) SyncPorts() {
	for _, name := range p.Ports() {
		if _, ok := p.ports[name]; !ok {
			f := NewPortFile(p.fs.NewStat(name, "plumb", "plumb", 0444))
			p.ports[name] = f
			p.root.AddChild(f)
		}
	}
}

// FeedPort post a message on the specified port.
func (p *Plumber) FeedPort(name string, msg *lib.Message) bool {
	f, ok := p.ports[name]
	if !ok {
		return false
	}
	return f.Post(msg)
}

// KeepMsg for un-opened port file
func (p *Plumber) KeepMsg(name string, msg *lib.Message) bool {
	f, ok := p.ports[name]
	if !ok {
		return false
	}
	return f.Keep(msg)
}

// PlumbAction to process rules with "plumb" rules
type PlumbAction struct {
	plmb *Plumber // back-reference to plumber
	port string   // name of plumbing port (if specified)
	dry  bool     // dry run
}

// NewWorker returns a new worker instance
func (p *Plumber) NewWorker() lib.Action {
	return (&PlumbAction{
		plmb: p,
		port: "",
		dry:  p.Dry,
	}).process
}

// process a message according to verb.
// 'ok' is true if the rule executes without failure/mismatch
// 'done' is true if this action terminates the ruleset
func (a *PlumbAction) process(msg *lib.Message, verb, data string) (ok, done bool) {
	logger.Printf(logger.INFO, ">> plumb %s %s", verb, data)
	switch verb {
	case "to":
		ok = true
		done = a.plmb.FeedPort(data, msg)
	case "client":
		if ok = a.plmb.KeepMsg(data, msg); !ok {
			done = true
			return
		}
		fallthrough
	case "start":
		a.Exec(data)
		ok = true
		done = true
	}
	logger.Printf(logger.INFO, "<< ok=%v, done=%v", ok, done)
	return
}

// Exec plumbing request
func (a *PlumbAction) Exec(data string) {
	if !a.dry {
		go func() {
			parts := lib.ParseParts(data)
			cmd := exec.Command(parts[0], parts[1:]...)
			logger.Println(logger.DBG, cmd.String())
			stdout, err := cmd.Output()
			if err != nil {
				logger.Println(logger.ERROR, err.Error())
				return
			}
			// Print the output
			logger.Println(logger.DBG, string(stdout))
			logger.Flush()
		}()
	} else {
		logger.Printf(logger.INFO, ">> EXEC '%s'", data)
	}
}
