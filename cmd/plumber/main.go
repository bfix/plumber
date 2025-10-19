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
	"time"

	"github.com/bfix/gospel/logger"
	"github.com/bfix/plumber/lib"
)

func main() {
	// handle command-line options
	flag.Bool("f", false, "run in foreground")
	rules := flag.String("p", "", "plumbing file")
	flag.Parse()

	// setup logging
	logger.SetLogLevelFromName("DBG")
	logger.UseFormat(logger.ColorFormat)

	// prepare plumber and load rules file
	action := new(PlumbAction)
	plmb := lib.NewPlumber(action.Process)
	f, err := os.Open(*rules)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err = plmb.ParseRulesFile(f, nil); err != nil {
		log.Fatal(err)
	}

	// build plumber namespace and post/start server
	ns := NamespaceService(plmb)

	go func() {
		msg := lib.NewMessage("plumber", "outerspace", "/home/user", "", "https://9p.sdf.org")
		tick := time.NewTicker(30 * time.Second)
		count := 0
		for range tick.C {
			if count++; count > 6 {
				break
			}
			ns.FeedPort("outerspace", msg)
		}
	}()

	action.srv = ns
	RunService(ns.srv)
}
