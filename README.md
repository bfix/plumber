# plumber: Plan9-inspired implementation in Go

Copyright (C) 2024-present, Bernd Fix  >Y<

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

TO BE USEFUL, A GOOD UNDERSTANDING OF PLAN9 (N.B.: PLAN9 IS **NOT** UNIX)
AND THE PLUMBING ON PLAN9 IS MANDATORY.

## TL;DR

Go v1.25+ is required to compile the code.

```bash
git clone https://github.com/bfix/plumber
cd plumber
go mod tidy
go install ./...
```

The binaries are stored in `${GOPATH}/bin`. They can be compiled for Plan9
with `GOOS=plan9` and for Linux with `GOOS=linux`. Other operating systems
might work - or not...

## Plumber

`plumber` is a [program](https://www.plan9foundation.org/magic/man2html/4/plumber)
from the [Plan9](https://p9f.org) operating system. It receives
[`plumb messages`](https://www.plan9foundation.org/magic/man2html/4/plumber)
from other programs, analyzes the data in the messages, and acts according to
rules defined in a plumbing file.

### Plumbing file

The rules for plumbing are loaded from a file at start-up. The format of the
plumbing file is compatible with the Plan9 native format - any Plan9 plumbing
file can be used, but `plumber` has the following extensions:

#### `matches` regular expressions

All Plan9 regular expressions in `matches` rules are supported, but `plumber`
also supports the full RE2 syntax. Plumbing files with RE2 expressions are
not backward-compatible.

#### Rule branching

Often rulesets look similar in their structure like:

```bash
type is text
data matches $filename
arg isfile $0
data matches $filename'\.rtf'
plumb to msword
plumb start wdoc2txt $file

type is text
data matches $filename
arg isfile $0
data matches $filename'\.pdf'
plumb start page $file

type is text
data matches $filename
arg isfile $0
data matches $filename'\.'$audio
plumb to audio
plumb start window -scroll play $file

:
```

In a `plumber` rules file you can use nested blocks and write the rulesets as:

```bash
type    is      text
data    matches $filename
arg     isfile  $0
{
  data    matches $filename'\.'$document
  v_type  set     $1
  {
    v_type  matches '(?i)pdf'
    plumb   start   page $file
  }
  {
    v_type  matches '(?i)rtf'
    plumb   to      msword
    plumb   start   wdoc2txt $file
  }
}
{
  data    matches $filename'\.'$audio
  plumb   to      audio
  plumb   start   window -scroll play $file
}
:
```

Nested blocks can contain further nested blocks; nested blocks usually
appear at the end of blocks. Plumbing files with nested blocks are not
backward-compatible.

#### Additional variables

Using nested rules often requires additional state to be keep for decision
making in deeper branches. `plumber` rulesets therefore provide additional
objects named `v_<name>` where `name` is a user-defined identifier.

#### Testing plumbing files

The `plumb-sim` program can be used to test rules interactively.
It is started with

```bash
./plumb-sim -p <rules file>
```

and prompts for data to plumb. Debug messages are shown for all `matches`
rules encountered. If a ruleset matches the plumbing actions are also
shown.

If the input is a command (starting with a dot), it is executed. The following
commands are defined:

* `.reload` reloads the start-up rules file (after editing)
* `.load <rules file>` loads a new rules file
* `.show` displays the current rules

### Plumbing filesystem

`plumber` works by handling files in a filesystem. Assume the plumbing service
is mounted at `/mnt/plumb`, these files are:

#### `/mnt/plumb/rules`

* Reading from this file returns the current plumbing file.

* Writing to this file replaces the current plumbing file.

* Appending to this file adds new text to the current plumbing file.
Appending invalid data may break the `plumber` logic.

Changes only happen inside the plumber; no external files are modified.

#### `/mnt/plumb/send`

This file is write-only; processes can send a `plumb message` to the `plumber`
to be analyzed and executed upon.

#### Ports `/mnt/plumb/<portname>`

For each port referenced in the plumbing file a corresponding port file is
created with the name of the port. A port cannot be named `rules`or `send`;
these files are maintained by the plumber directly.

Processes can read from port files to be informed about new messages.
Port files only allow a single reader.

## Use with Linux

### Starting the service and mounting the filesystem

Although `plumber` is part of Plan9 (and Plan9 **is not Unix**), it can be
used with Linux-like environments easily if `$PLUMBER_MNT` refers to a
directory owned by the current user (or at least with full access):

```bash
plumber -f rules/default &
PLUMBER_PID=?!
mkdir -f $PLUMBER_MNT
9pfuse 127.0.0.1:3124 $PLUMBER_MNT
```

### Sending plumb messages

A little script (`plumb.sh`) in `$PATH` can be used to send plumbing messages
to the plumber for processing:

```bash
#!/bin/bash

data="$*"
cat > $PLUMBER_MNT/send <<EOF
$USER
plumber
$HOME
text

${#data}
$data

EOF
```

To send a message, run `plumb.sh "<text>"`. The text will be analyzed and
acted upon by the plumber service. For convenience you can use the script
with a clipboard manager that triggers the plumbing from a context menu.

### Unmounting the filesystem and terminating the service

If done with the service, tear it down with

```bash
fusermount -u $PLUMBER_MNT
kill $PLUMBER_ID
```
