//go:build linux

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
	"os"
	"os/signal"
	"syscall"

	"github.com/bfix/gospel/logger"
	"github.com/knusbaum/go9p"
)

// RunService (on Linux)
func (p *Plumber) Run() {
	go func() {
		if err := go9p.Serve("0.0.0.0:3124", p.srv); err != nil {
			logger.Println(logger.CRITICAL, "can't start service: "+err.Error())
		}
	}()

	// handle OS signals
	sigCh := make(chan os.Signal, 5)
	signal.Notify(sigCh)
loop:
	for sig := range sigCh {
		switch sig {
		case syscall.SIGKILL, syscall.SIGINT, syscall.SIGTERM:
			logger.Printf(logger.INFO, "Terminating service (on signal '%s')\n", sig)
			break loop
		case syscall.SIGHUP:
			logger.Println(logger.INFO, "SIGHUP")
		case syscall.SIGURG:
			// TODO: https://github.com/golang/go/issues/37942
		default:
			logger.Println(logger.INFO, "Unhandled signal: "+sig.String())
		}
	}
}
