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
	"github.com/knusbaum/go9p/fs"
	"github.com/knusbaum/go9p/proto"
)

//----------------------------------------------------------------------

// RuleFile ('/mnt/plumb/rules')
// - Writing new rules file
// - Appending to rules file
// - Reading current rule file
type RulesFile struct {
	fs.BaseFile

	content   map[uint64][]byte // fid-mapped content
	plmb      *Plumber          // reference to plumber instance
	mode      proto.Mode        // open mode: read/write
	syncPorts func()            // sync ports after rule changes
}

// NewRulesFile creates a new filesystem node for rules
func NewRulesFile(s *proto.Stat, plmb *Plumber, sync func()) *RulesFile {
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
	l := uint64(len(f.plmb.File()))
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
	f.content[fid] = f.plmb.File()
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
		return 0, errors.New("illegal offset")
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
		err = f.plmb.ParsePlumbingFromRdr(rdr)
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
	plmb    *Plumber          // reference to plumber instance
}

// NewSendFile creates a new filesystem node for receiving plumb messages
func NewSendFile(s *proto.Stat, plmb *Plumber) *SendFile {
	return &SendFile{
		BaseFile: *fs.NewBaseFile(s),
		content:  make(map[uint64][]byte),
		plmb:     plmb,
	}
}

// Open file for writing only
func (f *SendFile) Open(fid uint64, omode proto.Mode) (err error) {
	f.Lock()
	defer f.Unlock()
	logger.Printf(logger.DBG, "Open{fid:%d,omode=%v}", fid, omode)

	if omode == proto.Owrite {
		f.content[fid] = []byte{}
	} else {
		err = errors.New("permission denied")
	}
	return
}

// Write data to file at given position
func (f *SendFile) Write(fid uint64, ofs uint64, buf []byte) (uint32, error) {
	f.RLock()
	defer f.RUnlock()
	logger.Printf(logger.DBG, "Write{fid:%d,ofs:%d,buf:[%d]}", fid, ofs, len(buf))

	data := f.content[fid]
	flen := uint64(len(data))
	if ofs > flen {
		logger.Printf(logger.WARN, "  write beyond eof: %d > %d", ofs, flen)
		return 0, errors.New("illegal offset")
	}
	f.content[fid] = append(data[:ofs], buf...)
	return uint32(len(buf)), nil
}

// Close file and process content (message)
func (f *SendFile) Close(fid uint64) (err error) {
	logger.Printf(logger.DBG, "Close{fid:%d}", fid)

	data := f.content[fid]
	var msg *lib.Message
	msg, err = lib.ParseMessage(string(data))
	if err == nil && msg != nil {
		var done bool
		if done, err = f.plmb.Process(msg); err == nil && !done && len(msg.Dst) > 0 {
			f.plmb.FeedPort(msg.Dst, msg)
		}
	} else {
		logger.Println(logger.WARN, "received invalid message: "+err.Error())
	}
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
	pending bool        // pending message (don't clear on Open)
	post    chan []byte // channel for posting messages
	watched bool        // is someone reading this?
}

// NewPortFile initializes a new port instance
func NewPortFile(s *proto.Stat) *PortFile {
	return &PortFile{
		BaseFile: *fs.NewBaseFile(s),
		skipped:  0,
		buf:      []byte{},
		pending:  false,
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

// Keep a message for yet un-opened port file
func (f *PortFile) Keep(msg *lib.Message) bool {
	if f.watched {
		return false
	}
	f.Lock()
	defer f.Unlock()

	f.buf = []byte(msg.String())
	f.pending = true
	return true
}

// Open port file for reading
func (f *PortFile) Open(fid uint64, omode proto.Mode) (err error) {
	if f.watched {
		return errors.New("file is in use")
	}
	f.watched = true
	f.Lock()
	defer f.Unlock()
	logger.Printf(logger.DBG, "Open{fid:%d,omode=%v}", fid, omode)

	if omode == proto.Owrite {
		return errors.New("file is read only")
	}
	f.skipped = 0
	if !f.pending {
		f.buf = []byte{}
	}
	f.pending = false
	return
}

// Read data at given position from port file
func (f *PortFile) Read(fid uint64, ofs uint64, count uint64) ([]byte, error) {
	f.RLock()
	defer f.RUnlock()

	if ofs < f.skipped {
		return []byte{}, errors.New("illegal offset")
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
