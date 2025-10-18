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
	"bytes"
	"errors"

	"github.com/bfix/plumber/lib"
	"github.com/knusbaum/go9p"
	"github.com/knusbaum/go9p/fs"
	"github.com/knusbaum/go9p/proto"
)

// NamespaceServer for plumber
func NamespaceServer(pl *lib.Plumber) go9p.Srv {

	plmbFS, root := fs.NewFS("plumb", "plumb", 0775)
	root.AddChild(NewRulesFile(plmbFS.NewStat("rules", "plumb", "plumb", 0666), pl))
	root.AddChild(NewSendFile(plmbFS.NewStat("send", "plumb", "plumb", 0222), pl))
	return plmbFS.Server()
}

//----------------------------------------------------------------------

type RulesFile struct {
	fs.BaseFile

	content map[uint64][]byte
	plmb    *lib.Plumber
	mode    proto.Mode
}

func NewRulesFile(s *proto.Stat, plmb *lib.Plumber) *RulesFile {
	return &RulesFile{
		BaseFile: *fs.NewBaseFile(s),
		content:  make(map[uint64][]byte),
		plmb:     plmb,
	}
}

func (f *RulesFile) Open(fid uint64, omode proto.Mode) error {
	f.Lock()
	defer f.Unlock()

	f.mode = omode
	switch omode {
	case proto.Oread, proto.Ordwr:
		f.content[fid] = f.plmb.Rules()
	case proto.Owrite:
		f.content[fid] = []byte{}
	}
	return nil
}

func (f *RulesFile) Read(fid uint64, ofs uint64, count uint64) ([]byte, error) {
	f.RLock()
	defer f.RUnlock()

	data := f.content[fid]
	flen := uint64(len(data))
	if ofs >= flen {
		return []byte{}, nil
	}
	if ofs+count > flen {
		count = flen - ofs
	}
	return data[ofs : ofs+count], nil
}

func (f *RulesFile) Write(fid uint64, ofs uint64, buf []byte) (uint32, error) {
	f.RLock()
	defer f.RUnlock()

	data := f.content[fid]
	flen := uint64(len(data))
	if ofs >= flen {
		return 0, errors.New("write beyond eof")
	}
	f.content[fid] = append(data[:ofs], buf...)
	return uint32(len(buf)), nil
}

func (f *RulesFile) Close(fid uint64) (err error) {
	switch f.mode {
	case proto.Oread:
		// no action
	case proto.Owrite:
		data := f.content[fid]
		rdr := bytes.NewBuffer(data)
		err = f.plmb.ParseRuleset(rdr)
	case proto.Ordwr:
		data := append(f.content[fid], f.plmb.Rules()...)
		rdr := bytes.NewBuffer(data)
		err = f.plmb.ParseRuleset(rdr)
	}
	delete(f.content, fid)
	return
}

//----------------------------------------------------------------------

type SendFile struct {
	fs.BaseFile

	content map[uint64][]byte
	plmb    *lib.Plumber
}

func NewSendFile(s *proto.Stat, plmb *lib.Plumber) *SendFile {
	return &SendFile{
		BaseFile: *fs.NewBaseFile(s),
		content:  make(map[uint64][]byte),
	}
}

func (f *SendFile) Open(fid uint64, omode proto.Mode) (err error) {
	f.Lock()
	defer f.Unlock()

	switch omode {
	case proto.Owrite, proto.Ordwr:
		f.content[fid] = []byte{}
	default:
		err = errors.New("can't write file")
	}
	return
}

func (f *SendFile) Write(fid uint64, ofs uint64, buf []byte) (uint32, error) {
	f.RLock()
	defer f.RUnlock()

	data := f.content[fid]
	flen := uint64(len(data))
	if ofs >= flen {
		return 0, errors.New("write beyond eof")
	}
	f.content[fid] = append(data[:ofs], buf...)
	return uint32(len(buf)), nil

}

func (f *SendFile) Close(fid uint64) (err error) {
	data := f.content[fid]
	f.plmb.Eval(string(data), "", "", "")
	delete(f.content, fid)
	return
}
