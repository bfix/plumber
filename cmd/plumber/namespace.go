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

// Service handling the plumbing namespace
type Service struct {
	srv   go9p.Srv             // 9P server
	fs    *fs.FS               // synth. filesystem
	root  *fs.StaticDir        // root folder
	plmb  *lib.Plumber         // reference to plumber
	ports map[string]*PortFile // list of plumbing ports
}

// NamespaceService returns a service instance
func NamespaceService(pl *lib.Plumber) *Service {
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

// SyncPorts after rule changes. New ports are created, but unused ports
// are not removed from the filesystem.
func (s *Service) SyncPorts() {
	for _, name := range s.plmb.Ports() {
		if _, ok := s.ports[name]; !ok {
			f := NewPortFile(s.fs.NewStat(name, "plumb", "plumb", 0444))
			s.ports[name] = f
			s.root.AddChild(f)
		}
	}
}

// FeedPort post a message on the specified port.
func (s *Service) FeedPort(name string, msg *lib.Message) bool {
	f, ok := s.ports[name]
	if !ok {
		return false
	}
	return f.Post(msg)
}

//----------------------------------------------------------------------

// RuleFile ('/mnt/plumb/rules')
// - Writing new rules file
// - Appending to rules file
// - Reading current rule file
type RulesFile struct {
	fs.BaseFile

	content   map[uint64][]byte // fid-mapped content
	plmb      *lib.Plumber      // reference to plumber instance
	mode      proto.Mode        // open mode: read/write
	syncPorts func()            // sync ports after rule changes
}

// NewRulesFile creates a new filesystem node for rules
func NewRulesFile(s *proto.Stat, plmb *lib.Plumber, sync func()) *RulesFile {
	return &RulesFile{
		BaseFile:  *fs.NewBaseFile(s),
		content:   make(map[uint64][]byte),
		plmb:      plmb,
		syncPorts: sync,
	}
}

// Stat returns the current file stats
func (f *RulesFile) Stat() proto.Stat {
	s := f.BaseFile.Stat()
	l := uint64(len(f.plmb.Rules()))
	//logger.Printf(logger.DBG, "Stat{length: %d -> %d}", s.Length, l)
	s.Length = l // adjust file size to content
	f.WriteStat(&s)
	return s
}

// Open file with given mode
func (f *RulesFile) Open(fid uint64, omode proto.Mode) error {
	f.Lock()
	defer f.Unlock()
	//logger.Printf(logger.DBG, "Open{fid:%d,omode=%v}", fid, omode)

	f.mode = omode
	f.content[fid] = f.plmb.Rules()
	return nil
}

// Read specified range from file
func (f *RulesFile) Read(fid uint64, ofs uint64, count uint64) ([]byte, error) {
	f.RLock()
	defer f.RUnlock()
	//logger.Printf(logger.DBG, "Read{fid:%d,ofs:%d,cnt:%d}", fid, ofs, count)

	data := f.content[fid]
	flen := uint64(len(data))
	if ofs >= flen {
		// no (more) content
		return []byte{}, nil
	}
	last := min(ofs+count, flen)
	return data[ofs:last], nil
}

// Write data to file at given position
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

// Close file and parse written content
func (f *RulesFile) Close(fid uint64) (err error) {
	//logger.Printf(logger.DBG, "Close{fid:%d}", fid)
	switch f.mode {
	case proto.Oread:
		// no action
	case proto.Owrite:
		data := f.content[fid]
		rdr := bytes.NewBuffer(data)
		err = f.plmb.ParseRulesFile(rdr, nil)
		f.syncPorts()
	}
	delete(f.content, fid)
	return
}

//----------------------------------------------------------------------

// SendFile ('/mnt/plumb/send') is a write-only file that receives plumbing
// messages to be processed by this plumber.
type SendFile struct {
	fs.BaseFile

	content map[uint64][]byte // fid-mapped content
	plmb    *lib.Plumber      // reference to plumber instance
}

// NewSendFile creates a new filesystem node for receiving plumb messages
func NewSendFile(s *proto.Stat, plmb *lib.Plumber) *SendFile {
	return &SendFile{
		BaseFile: *fs.NewBaseFile(s),
		content:  make(map[uint64][]byte),
	}
}

// Open file for writing only
func (f *SendFile) Open(fid uint64, omode proto.Mode) (err error) {
	f.Lock()
	defer f.Unlock()

	if omode == proto.Owrite {
		f.content[fid] = []byte{}
	} else {
		err = errors.New("file is write-only")
	}
	return
}

// Write data to file at given position
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

// Close file and process content (message)
func (f *SendFile) Close(fid uint64) (err error) {
	data := f.content[fid]
	var msg *lib.Message
	msg, err = lib.ParseMessage(string(data))
	f.plmb.Process(msg)
	delete(f.content, fid)
	return
}

//----------------------------------------------------------------------

// PortFile ('/mnt/plumb/<portname>') is a read-only file where the plumber
// publishes messages to a single reader.
type PortFile struct {
	fs.BaseFile

	skipped uint64      // fid-mapped skips
	buf     []byte      // current content
	post    chan []byte // channel for posting messages
	watched bool        // is someone reading this?
}

// NewPortFile initializes a new port instance
func NewPortFile(s *proto.Stat) *PortFile {
	return &PortFile{
		BaseFile: *fs.NewBaseFile(s),
		skipped:  0,
		buf:      []byte{},
		post:     make(chan []byte),
		watched:  false,
	}
}

// Post a message on the port (only if we have readers)
func (f *PortFile) Post(msg *lib.Message) bool {
	if !f.watched {
		return false
	}
	f.post <- []byte(msg.String())
	return true
}

// Open port file for reading
func (f *PortFile) Open(fid uint64, omode proto.Mode) (err error) {
	if f.watched {
		return errors.New("file already in use")
	}
	f.watched = true
	f.Lock()
	defer f.Unlock()
	logger.Printf(logger.DBG, "Open{fid:%d,omode=%v}", fid, omode)

	if omode == proto.Owrite {
		return errors.New("can't write file")
	}
	f.skipped = 0
	f.buf = []byte{}
	return
}

// Read data at given position from port file
func (f *PortFile) Read(fid uint64, ofs uint64, count uint64) ([]byte, error) {
	f.RLock()
	defer f.RUnlock()

	if ofs < f.skipped {
		return []byte{}, errors.New("read out of sync")
	}
	ofs -= f.skipped

	flen := uint64(len(f.buf))
	if ofs >= flen {
		f.skipped += flen
		ofs -= flen
		f.buf = <-f.post
		flen = uint64(len(f.buf))
	}
	last := min(ofs+count, flen)
	data := f.buf[ofs:last]
	logger.Printf(logger.DBG, "Read{fid:%d,ofs:%d,cnt:%d} -> [%d]", fid, ofs, count, len(data))
	return data, nil
}

// Close port file
func (f *PortFile) Close(fid uint64) (err error) {
	logger.Printf(logger.DBG, "Close{fid:%d}", fid)
	f.watched = false
	return
}
