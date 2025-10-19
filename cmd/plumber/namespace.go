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

	"github.com/bfix/gospel/logger"
	"github.com/bfix/plumber/lib"
	"github.com/knusbaum/go9p"
	"github.com/knusbaum/go9p/fs"
	"github.com/knusbaum/go9p/proto"
)

type Service struct {
	srv   go9p.Srv
	fs    *fs.FS
	root  *fs.StaticDir
	plmb  *lib.Plumber
	ports map[string]*PortFile
}

// NamespaceServer for plumber
func NamespaceServer(pl *lib.Plumber) *Service {
	service := &Service{
		plmb:  pl,
		ports: make(map[string]*PortFile),
	}
	plmbFS, root := fs.NewFS("plumb", "plumb", 0775)
	root.AddChild(NewRulesFile(plmbFS.NewStat("rules", "plumb", "plumb", 0666), pl, service.SyncPorts))
	root.AddChild(NewSendFile(plmbFS.NewStat("send", "plumb", "plumb", 0222), pl))
	service.srv = plmbFS.Server()
	service.fs = plmbFS
	service.root = root
	service.SyncPorts()
	return service
}

func (s *Service) SyncPorts() {
	for _, name := range s.plmb.Ports() {
		if _, ok := s.ports[name]; !ok {
			f := NewPortFile(s.fs.NewStat(name, "plumb", "plumb", 0444))
			s.ports[name] = f
			s.root.AddChild(f)
		}
	}
}

func (s *Service) FeedPort(name string, msg *lib.Message) bool {
	f, ok := s.ports[name]
	if !ok {
		return false
	}
	return f.Post(msg)
}

//----------------------------------------------------------------------

type RulesFile struct {
	fs.BaseFile

	content   map[uint64][]byte
	plmb      *lib.Plumber
	mode      proto.Mode
	syncPorts func()
}

func NewRulesFile(s *proto.Stat, plmb *lib.Plumber, sync func()) *RulesFile {
	return &RulesFile{
		BaseFile:  *fs.NewBaseFile(s),
		content:   make(map[uint64][]byte),
		plmb:      plmb,
		syncPorts: sync,
	}
}

func (f *RulesFile) Stat() proto.Stat {
	s := f.BaseFile.Stat()
	l := uint64(len(f.plmb.Rules()))
	//logger.Printf(logger.DBG, "Stat{length: %d -> %d}", s.Length, l)
	s.Length = l
	f.WriteStat(&s)
	return s
}

func (f *RulesFile) Open(fid uint64, omode proto.Mode) error {
	f.Lock()
	defer f.Unlock()
	//logger.Printf(logger.DBG, "Open{fid:%d,omode=%v}", fid, omode)

	f.mode = omode
	f.content[fid] = f.plmb.Rules()
	return nil
}

func (f *RulesFile) Read(fid uint64, ofs uint64, count uint64) ([]byte, error) {
	f.RLock()
	defer f.RUnlock()
	//logger.Printf(logger.DBG, "Read{fid:%d,ofs:%d,cnt:%d}", fid, ofs, count)

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
	//logger.Printf(logger.DBG, "Write{fid:%d,ofs:%d,buf:[%d]}", fid, ofs, len(buf))

	data := f.content[fid]
	flen := uint64(len(data))
	if ofs > flen {
		return 0, errors.New("write beyond eof")
	}
	f.content[fid] = append(data[:ofs], buf...)
	return uint32(len(buf)), nil
}

func (f *RulesFile) Close(fid uint64) (err error) {
	//logger.Printf(logger.DBG, "Close{fid:%d}", fid)
	switch f.mode {
	case proto.Oread:
		// no action
	case proto.Owrite:
		data := f.content[fid]
		rdr := bytes.NewBuffer(data)
		err = f.plmb.ParseRuleset(rdr, nil)
		f.syncPorts()
	case proto.Ordwr:
		data := append(f.content[fid], f.plmb.Rules()...)
		env := f.plmb.Env()
		rdr := bytes.NewBuffer(data)
		err = f.plmb.ParseRuleset(rdr, env)
		f.syncPorts()
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

	if omode == proto.Owrite {
		f.content[fid] = []byte{}
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
	var msg *lib.Message
	msg, err = lib.ParseMessage(string(data))
	f.plmb.Process(msg)
	delete(f.content, fid)
	return
}

//----------------------------------------------------------------------

type Content struct {
	skipped uint64
	buf     []byte
}

func NewContent() *Content {
	return &Content{
		skipped: 0,
		buf:     []byte{},
	}
}

func (c *Content) Get(ofs uint64, count uint64, post chan []byte) (data []byte, err error) {
	flen := uint64(len(c.buf))
	ofs -= c.skipped
	if ofs >= flen {
		c.skipped += flen
		ofs -= flen
		c.buf = <-post
		flen = uint64(len(c.buf))
	}
	last := min(ofs+count, flen)
	return c.buf[ofs:last], nil
}

type PortFile struct {
	fs.BaseFile

	content map[uint64]*Content
	post    chan []byte
}

func NewPortFile(s *proto.Stat) *PortFile {
	return &PortFile{
		BaseFile: *fs.NewBaseFile(s),
		content:  make(map[uint64]*Content),
		post:     make(chan []byte),
	}
}

func (f *PortFile) Post(msg *lib.Message) bool {
	if len(f.content) == 0 {
		return false
	}
	f.post <- []byte(msg.String())
	return true
}

func (f *PortFile) Open(fid uint64, omode proto.Mode) (err error) {
	f.Lock()
	defer f.Unlock()
	logger.Printf(logger.DBG, "Open{fid:%d,omode=%v}", fid, omode)

	if omode == proto.Owrite {
		return errors.New("can't write file")
	}
	f.content[fid] = NewContent()
	return
}

func (f *PortFile) Read(fid uint64, ofs uint64, count uint64) ([]byte, error) {
	f.RLock()
	defer f.RUnlock()

	ct := f.content[fid]
	data, err := ct.Get(ofs, count, f.post)
	logger.Printf(logger.DBG, "Read{fid:%d,ofs:%d,cnt:%d} -> [%d]", fid, ofs, count, len(data))
	return data, err
}

func (f *PortFile) Close(fid uint64) (err error) {
	logger.Printf(logger.DBG, "Close{fid:%d}", fid)

	delete(f.content, fid)
	return
}
