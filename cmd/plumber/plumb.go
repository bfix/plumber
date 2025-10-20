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
)

// PlumbAction to process rules with "plumb" rules
type PlumbAction struct {
	Srv  *Service // reference to namespace service
	port string   // name of plumbing port (if specified)
	Dry  bool     // dry run (on exec)
}

// NewWorker returns a new worker instance
func (a *PlumbAction) NewWorker() lib.Action {
	w := &PlumbAction{
		Srv:  a.Srv,
		port: "",
	}
	return w.process
}

// process a message according to verb.
// 'ok' is true if the rule executes without failure/mismatch
// 'done' is true if this action terminates the ruleset
func (a *PlumbAction) process(msg *lib.Message, verb, data string) (ok, done bool) {
	logger.Printf(logger.INFO, ">> plumb %s %s", verb, data)
	switch verb {
	case "to":
		ok = true
		done = a.Srv.FeedPort(data, msg)
	case "client":
		if ok = a.Srv.KeepMsg(data, msg); !ok {
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
	if !a.Dry {
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
