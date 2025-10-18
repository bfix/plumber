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
	"flag"
	"log"
	"os"

	"github.com/bfix/gospel/logger"
	"github.com/bfix/plumber/lib"
)

func main() {
	// handle command-line options
	var rules string
	var foreground bool
	flag.BoolVar(&foreground, "f", false, "run in foreground")
	flag.StringVar(&rules, "p", "", "plumbing file")
	flag.Parse()

	// setup logging
	logger.SetLogLevelFromName("DBG")
	logger.UseFormat(logger.ColorFormat)

	// prepare plumber and load ruleset
	plmb := lib.NewPlumber(PlumbAction)
	f, err := os.Open(rules)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err = plmb.ParseRuleset(f, nil); err != nil {
		log.Fatal(err)
	}

	// build plumber namespace and post/start server
	srv := NamespaceServer(plmb)
	RunService(srv)
}
