/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package http

import (
	"net/http"

	"github.com/api7/ext-plugin-proto/go/A6"
	hrc "github.com/api7/ext-plugin-proto/go/A6/HTTPReqCall"
	flatbuffers "github.com/google/flatbuffers/go"
)

type ReadHeader interface {
	HeadersLength() int
	Headers(*A6.TextEntry, int) bool
}

type Header struct {
	hdr    http.Header
	rawHdr http.Header

	deleteField []string
}

func newHeader(r ReadHeader) *Header {
	hh := http.Header{}
	size := r.HeadersLength()
	obj := A6.TextEntry{}
	for i := 0; i < size; i++ {
		if r.Headers(&obj, i) {
			hh.Add(string(obj.Name()), string(obj.Value()))
		}
	}

	return &Header{
		hdr:    http.Header{},
		rawHdr: hh,

		deleteField: make([]string, 0),
	}
}

func (h *Header) Set(key, value string) {
	h.hdr.Set(key, value)
}

func (h *Header) Del(key string) {
	h.hdr.Del(key)

	if h.rawHdr.Get(key) != "" {
		h.deleteField = append(h.deleteField, key)
	}
}

func (h *Header) Get(key string) string {
	if v := h.hdr.Get(key); v != "" {
		return v
	}

	return h.rawHdr.Get(key)
}

// View
//Deprecated: refactoring
func (h *Header) View() http.Header {
	return h.hdr
}

func (h *Header) Build(builder *flatbuffers.Builder) flatbuffers.UOffsetT {
	var hdrs []flatbuffers.UOffsetT

	// deleted
	for _, d := range h.deleteField {
		name := builder.CreateString(d)
		A6.TextEntryStart(builder)
		A6.TextEntryAddName(builder, name)
		te := A6.TextEntryEnd(builder)
		hdrs = append(hdrs, te)
	}

	// set
	for hKey, hVal := range h.hdr {
		if raw, ok := h.rawHdr[hKey]; !ok || raw[0] != hVal[0] {
			name := builder.CreateString(hKey)
			value := builder.CreateString(hVal[0])
			A6.TextEntryStart(builder)
			A6.TextEntryAddName(builder, name)
			A6.TextEntryAddValue(builder, value)
			te := A6.TextEntryEnd(builder)
			hdrs = append(hdrs, te)
		}
	}

	size := len(hdrs)
	hrc.RewriteStartHeadersVector(builder, size)
	for i := size - 1; i >= 0; i-- {
		te := hdrs[i]
		builder.PrependUOffsetT(te)
	}

	return builder.EndVector(size)
}
