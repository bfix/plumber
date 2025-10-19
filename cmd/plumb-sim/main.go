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
	plmb := lib.NewPlumber(exec)
	f, err := os.Open(rules)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err = plmb.ParseRulesFile(f, nil); err != nil {
		log.Fatal(err)
	}

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

		log.Printf("<== %s", line)
		if err = plmb.Eval(line, "", "", ""); err != nil {
			log.Fatal(err)
		}
	}
}
