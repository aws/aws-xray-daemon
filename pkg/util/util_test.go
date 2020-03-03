package util

import (
	"strings"
	"testing"
	"github.com/aws/aws-xray-daemon/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestSplitHeaderBodyWithSeparatorExists(t *testing.T) {
	str := "Header\nBody"
	separator := "\n"
	buf := []byte(str)
	separatorArray := []byte(separator)
	result := make([][]byte, 2)

	returnResult := SplitHeaderBody(&buf, &separatorArray, &result)

	assert.EqualValues(t, len(result), 2)
	assert.EqualValues(t, string(result[0]), "Header")
	assert.EqualValues(t, string(result[1]), "Body")
	assert.EqualValues(t, string(returnResult[0]), "Header")
	assert.EqualValues(t, string(returnResult[1]), "Body")
	assert.EqualValues(t, string(buf), str)
	assert.EqualValues(t, string(separatorArray), separator)
}

func TestSplitHeaderBodyWithSeparatorDoesNotExist(t *testing.T) {
	str := "Header"
	separator := "\n"
	buf := []byte(str)
	separatorArray := []byte(separator)
	result := make([][]byte, 2)

	returnResult := SplitHeaderBody(&buf, &separatorArray, &result)

	assert.EqualValues(t, len(result), 2)
	assert.EqualValues(t, string(result[0]), "Header")
	assert.EqualValues(t, string(result[1]), "")
	assert.EqualValues(t, string(returnResult[0]), "Header")
	assert.EqualValues(t, string(returnResult[1]), "")
	assert.EqualValues(t, string(buf), str)
	assert.EqualValues(t, string(separatorArray), separator)
}

func TestSplitHeaderBodyNilBuf(t *testing.T) {
	log := test.LogSetup()
	separator := "\n"
	separatorArray := []byte(separator)
	result := make([][]byte, 2)
	SplitHeaderBody(nil, &separatorArray, &result)

	assert.True(t, strings.Contains(log.Logs[0], "Buf to split passed nil"))
}

func TestSplitHeaderBodyNilSeparator(t *testing.T) {
	log := test.LogSetup()
	str := "Test String"
	buf := []byte(str)
	result := make([][]byte, 2)

	SplitHeaderBody(&buf, nil, &result)

	assert.True(t, strings.Contains(log.Logs[0], "Separator used to split passed nil"))
}

func TestSplitHeaderBodyNilResult(t *testing.T) {
	log := test.LogSetup()
	str := "Test String"
	buf := []byte(str)
	separator := "\n"
	separatorArray := []byte(separator)
	SplitHeaderBody(&buf, &separatorArray, nil)

	assert.True(t, strings.Contains(log.Logs[0], "Return Buf to be used to store split passed nil"))
}

func TestGetMinIntValue(t *testing.T) {
	assert.Equal(t, GetMinIntValue(1, 1), 1, "Return value should be 1")
	assert.Equal(t, GetMinIntValue(0, 1), 0, "Return value should be 0")
	assert.Equal(t, GetMinIntValue(1, 0), 0, "Return value should be 0")
}
