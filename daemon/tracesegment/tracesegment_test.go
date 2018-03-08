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
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeflateWithValidInput(t *testing.T) {
	testSegment := GetTestTraceSegment()

	deflatedBytes := testSegment.Deflate()
	rawBytes := *testSegment.Raw

	assert.True(t, len(rawBytes) > len(deflatedBytes), "Deflated bytes should compress raw bytes")

	// Testing reverting compression using zlib
	deflatedBytesBuffer := bytes.NewBuffer(deflatedBytes)
	reader, err := zlib.NewReader(deflatedBytesBuffer)
	if err != nil {
		panic(err)
	}
	var deflatedBytesRecovered = make([]byte, 1000)
	n, err := reader.Read(deflatedBytesRecovered)
	if err != nil && err != io.EOF {
		panic(err)
	}
	deflatedBytesRecovered = deflatedBytesRecovered[:n]

	assert.Equal(t, n, len(deflatedBytesRecovered))
	assert.Equal(t, len(deflatedBytesRecovered), len(rawBytes))
	for index, byteVal := range rawBytes {
		assert.Equal(t, byteVal, deflatedBytesRecovered[index], "Difference in recovered and original bytes")
	}
}

func TestTraceSegmentHeaderIsValid(t *testing.T) {
	header := Header{
		Format:  "json",
		Version: 1,
	}

	valid := header.IsValid()

	assert.True(t, valid)
}

func TestTraceSegmentHeaderIsValidCaseInsensitive(t *testing.T) {
	header := Header{
		Format:  "jSoN",
		Version: 1,
	}

	valid := header.IsValid()

	assert.True(t, valid)
}

func TestTraceSegmentHeaderIsValidWrongVersion(t *testing.T) {
	header := Header{
		Format:  "json",
		Version: 2,
	}

	valid := header.IsValid()

	assert.False(t, valid)
}

func TestTraceSegmentHeaderIsValidWrongFormat(t *testing.T) {
	header := Header{
		Format:  "xml",
		Version: 1,
	}

	valid := header.IsValid()

	assert.False(t, valid)
}

func TestTraceSegmentHeaderIsValidWrongFormatVersion(t *testing.T) {
	header := Header{
		Format:  "xml",
		Version: 2,
	}

	valid := header.IsValid()

	assert.False(t, valid)
}
