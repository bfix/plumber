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

/*
 * The code of this package is heavily inspired by the Plan9 implementation
 * of the plumber (see $plan9/src/{cmd/plumb,lib/libplumb,sys/include}).
 * It is not a clean-room re-implementation, but a transformation of the
 * basic approach to Go. Its messages are interoperable.
 */

package lib

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Message exchanged on plumbing ports
type Message struct {
	Src   string
	Dst   string
	Wdir  string
	Type  string
	Attr  map[string]string
	Ndata int
	Data  string
}

// ParseMessage from multi-line string
func ParseMessage(p string) (m *Message, err error) {
	parts := strings.Split(p, "\n")
	if len(parts) != 7 {
		return nil, errors.New("malformed message")
	}
	ndata, _ := strconv.Atoi(parts[5])
	m = &Message{
		Src:   parts[0],
		Dst:   parts[1],
		Wdir:  parts[2],
		Type:  parts[3],
		Ndata: ndata,
		Data:  parts[6],
	}
	m.Attr = m.unpackAttr(parts[4])
	m.Data, err = m.unpackData(m.Data)
	m.Ndata = len(m.Data)
	return
}

// unpack attributes from string
func (m *Message) unpackAttr(s string) map[string]string {
	res := make(map[string]string)
	for _, s := range ParseParts(s) {
		v := strings.SplitN(s, "=", 2)
		res[v[0]] = v[1]
	}
	return res
}

// GetAttr returns the attribute string
func (m *Message) GetAttr() (out string) {
	var list []string
	for k, v := range m.Attr {
		list = append(list, k+"="+Quote(v))
	}
	return strings.Join(list, " ")
}

// unpack encoded data
func (m *Message) unpackData(in string) (string, error) {
	if strings.HasPrefix(in, "base64:") {
		d, err := base64.StdEncoding.DecodeString(in[7:])
		if err != nil {
			return "", err
		}
		return string(d), nil
	}
	return in, nil
}

// pack data (possibly multi-line string)
func (m *Message) packData() string {
	if strings.Contains(m.Data, "\n") {
		return "base64:" + base64.StdEncoding.EncodeToString([]byte(m.Data))
	}
	return m.Data
}

// String returns human-readable message
func (m *Message) String() string {
	buf := new(bytes.Buffer)
	buf.WriteString(m.Src + "\n")
	buf.WriteString(m.Dst + "\n")
	buf.WriteString(m.Wdir + "\n")
	buf.WriteString(m.Type + "\n")
	buf.WriteString(m.GetAttr() + "\n")
	md := m.packData()
	buf.WriteString(fmt.Sprintf("%d\n", len(md)))
	buf.WriteString(md)
	return buf.String()
}

// Get named value
func (m *Message) Get(name string) (string, error) {
	switch name {
	case "src":
		return m.Src, nil
	case "dst":
		return m.Dst, nil
	case "wdir":
		return m.Wdir, nil
	case "type":
		return m.Type, nil
	case "attr":
		return m.GetAttr(), nil
	case "ndata":
		return strconv.Itoa(m.Ndata), nil
	case "data":
		return m.Data, nil
	}
	return "", fmt.Errorf("unknown object '%s'", name)
}

// Set named value
func (m *Message) Set(name, value string) (rc bool) {
	rc = true
	switch name {
	case "src":
		m.Src = value
	case "dst":
		m.Dst = value
	case "wdir":
		m.Wdir = value
	case "type":
		m.Type = value
	case "attr":
		m.Attr = m.unpackAttr(value)
	case "data":
		m.Data = value
	default:
		rc = false
	}
	return
}
