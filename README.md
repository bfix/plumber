# plumber: Plan9 plumber implementation in Go

Copyright (C) 2014-present, Bernd Fix  >Y<

plumber is free software: you can redistribute it and/or modify it
under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License,
or (at your option) any later version.

plumber is distributed in the hope that it will be useful, but
WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.

SPDX-License-Identifier: AGPL3.0-or-later

## Caveat

THIS IS WORK-IN-PROGRESS AT A VERY EARLY STATE. DON'T EXPECT ANY COMPLETE
DOCUMENTATION OR COMPILABLE, RUNNABLE OR EVEN OPERATIONAL SOURCE CODE.

## TL;DR

Go v1.23+ is required to compile the code.

```bash
git clone https://github.com/bfix/plumber
cd plumber
go mod tidy
go install ./...
```

The binaries are stored in `${GOPATH}/bin`.

## Plumber

`plumber` is a program originating from the [Plan9](https://p9f.org) operating
system. It receives `plumb message`s from other programs, analyzes the data in
the messages, and acts according to rules defined in a ruleset.

The rules for plumbing are loaded from a file at start-up. The format of the
file is compatible with the Plan9 native format; differences are outlined in
the `Rules`section.

`plumber` works by handling files in a (mounted) filesystem. Assume the
plumbing service is mounted at `/mnt/plumb`, the plumber will *watch* two
files (for other files managed by `plumber` see section `Ports`):

### `/mnt/plumb/rules`

#### Reading from file

Reading from this file returns the current ruleset.

#### Writing to file

Writing to this file creates a new ruleset.

### Appending to file

Appending to this file adds the data to the ruleset. Appending invalid data
breaks the `plumber` logic.

### `/mnt/plumb/send`

This file is write only. Programs can send a `plumb message` to the plumber
to be analyzed and executed upon.

## Use with Linux

Although `plumber` is part of Plan9 (and Plan9 **is not Unix**), it can be
used with Linux-like environments easily if `/mnt/plumb` is a directory
with full access by the current user:

```bash
$ plumber -f rules/default &
$ PLUMBER_PID=?!
$ 9pfuse 127.0.0.1:3124 /mnt/plumb
```

If done with the service, tear it down with

```bash
$ fusermount -u /mnt/plumb
$ kill $PLUMBER_ID
```

