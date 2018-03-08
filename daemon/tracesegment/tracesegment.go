// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package tracesegment

import (
	"bytes"
	"compress/zlib"
	"strings"
)

// Header stores header of trace segment.
type Header struct {
	Format  string `json:"format"`
	Version int    `json:"version"`
}

// IsValid validates Header.
func (t Header) IsValid() bool {
	return strings.EqualFold(t.Format, "json") && t.Version == 1
}

// TraceSegment stores raw segment.
type TraceSegment struct {
	Raw     *[]byte
	PoolBuf *[]byte
}

// Deflate converts TraceSegment to bytes
func (r *TraceSegment) Deflate() []byte {
	var b bytes.Buffer

	w := zlib.NewWriter(&b)
	rawBytes := *r.Raw
	w.Write(rawBytes)
	w.Close()

	return b.Bytes()
}
