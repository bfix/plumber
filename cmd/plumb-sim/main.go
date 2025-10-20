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
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/bfix/plumber/lib"
)

func main() {
	var rules string
	flag.StringVar(&rules, "r", "", "name of rules file")
	flag.Parse()

	exec := func(msg *lib.Message, verb, data string) (ok, done bool) {
		log.Printf("==> %s %s", verb, lib.Quote(data))
		log.Printf("    Attr: %s", msg.GetAttr())
		ok = true
		return
	}
	worker := func() lib.Action {
		return exec
	}

	plmb := lib.NewPlumber(worker)
	loadRules := func(name string) error {
		f, err := os.Open(name)
		if err != nil {
			log.Fatal(err)
		}
		err = plmb.ParseRulesFile(f, nil)
		f.Close()
		return err
	}
	loadRules(rules)

	rdr := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Enter text to plumb:")
		data, _, err := rdr.ReadLine()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			log.Fatal(err)
		}
		line := string(data)

		parts := lib.ParseParts(line)
		switch parts[0] {
		case ".reload":
			if err = loadRules(rules); err != nil {
				log.Fatal(err)
			}
			continue

		case ".load":
			if err = loadRules(parts[1]); err != nil {
				log.Fatal(err)
			}
			continue

		default:
			log.Printf("<== %s", line)
			if err = plmb.Eval(line, "", "", ""); err != nil {
				log.Fatal(err)
			}
		}
	}
}
