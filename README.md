# plumber: Plan9 plumber implementation in Go

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
the messages, and acts according to rules defined in a rules file.

### Rules file

The rules for plumbing are loaded from a file at start-up. The format of the
rules file is compatible with the Plan9 native format - any Plan9 plumbing
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

In a `plumber` rules file you can write this as:

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

Nested blocks can contain further nested blocks; nested blocks usually appear
at the end of blocks.

#### Additional variables

Using rule branching often requires additional state to be keep for decision
making. `plumber` rulesets therefore provide additional objects named `v_<name>`
for this purpose. `name` is a user-defined identifier.

#### Testing rule files

The `plumb-sim` program can be used to test rules interactively. It is started with

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

* Reading from this file returns the current list of rules.

* Writing to this file creates a new list of rules.

* Appending to this file adds new text to the rules file. Appending invalid
data may break the `plumber` logic.

#### `/mnt/plumb/send`

This file is write-only; processes can send a `plumb message` to the `plumber`
to be analyzed and executed upon. The message format is defined by the Plan9 plumber.

#### Ports `/mnt/plumb/<portname>`

For each port referenced in the list of rules a corresponding port file is created
with the name of the port. A port cannot be named `rules`or `send`; these
files are maintained by the plumber directly.

Processes can read from port files to be informed about new messages.

## Use with Linux

### Starting the service and mounting the filesystem

Although `plumber` is part of Plan9 (and Plan9 **is not Unix**), it can be
used with Linux-like environments easily if `/mnt/plumb` is a directory
with full access by the current user:

```bash
plumber -f rules/default &
PLUMBER_PID=?!
9pfuse 127.0.0.1:3124 /mnt/plumb
```

### Sending plumb messages

A little script (`plumb.sh`) in `$PATH` can be used to send plumbing messages
to the plumber for processing:

```bash
#!/bin/bash

data="$*"
cat > /mnt/plumb/send <<EOF
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
fusermount -u /mnt/plumb
kill $PLUMBER_ID
```
