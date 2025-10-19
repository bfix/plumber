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

import "github.com/bfix/plumber/lib"

// PlumbAction to process rules with "plumb" clauses
type PlumbAction struct {
	srv *Service // reference to namespace service
}

// Process a message according to verb.
// 'ok' is true if the clause executes without failure/mismatch
// 'done' is true if this action terminates the rule
func (a *PlumbAction) Process(msg *lib.Message, verb, data string) (ok, done bool) {
	if verb == "to" {
		ok = true
		done = a.srv.FeedPort(data, msg)
		return
	}
	return
}
