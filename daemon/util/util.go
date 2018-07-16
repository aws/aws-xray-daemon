package util

import (
	"bytes"

	log "github.com/cihub/seelog"
)

// SplitHeaderBody separates header and body of buf using provided separator sep, and stores in returnByte.
func SplitHeaderBody(buf, sep *[]byte, returnByte *[][]byte) [][]byte {
	if buf == nil {
		log.Error("Buf to split passed nil")
		return nil
	}
	if sep == nil {
		log.Error("Separator used to split passed nil")
		return nil
	}
	if returnByte == nil {
		log.Error("Return Buf to be used to store split passed nil")
		return nil
	}

	separator := *sep
	bufVal := *buf
	lenSeparator := len(separator)
	var header, body []byte
	header = *buf
	for i := 0; i < len(bufVal); i++ {
		if bytes.Equal(bufVal[i:i+lenSeparator], separator) {
			header = bufVal[0:i]
			body = bufVal[i+lenSeparator:]
			break
		}
		if i == len(bufVal)-1 {
			log.Warnf("Missing header: %s", header)
		}
	}
	returnByteVal := *returnByte
	return append(returnByteVal[:0], header, body)
}

// GetMinIntValue returns minimum between a and b.
func GetMinIntValue(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Bool(b bool) *bool {
	return &b
}
