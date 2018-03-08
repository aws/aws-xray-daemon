// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cli

import (
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

type CLIArgs struct {
	StorageShort      []string // store the shorthand flag
	StorageLong       []string // store the flag name
	StorageUsage      []string // store the flag usage
	StorageFlagInt    []int    // store the flag int value
	StorageFlagString []string // store the flag string value
	StorageFlagBool   []bool   // store the flag bool value
}

// generate the random string for given length
func RandStr(strSize int) string {
	alphaNum := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, strSize)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphaNum[b%byte(len(alphaNum))]
	}
	return string(bytes)
}

// store the given number into an variable
func InitialVar(paras []int) []int {
	passLen := make([]int, 0, len(paras))
	for i := 0; i < len(paras); i++ {
		passLen = append(passLen, paras[i])
	}
	return passLen
}

// mock commandline input
func SetUpInputs(args []string, f *Flag) {
	a := os.Args[1:]
	if args != nil {
		a = args
	}
	f.fs.Parse(a)
}

func (cli *CLIArgs) DefineFlagsArray(arrayLen int, strSize []int, strSizeFlag []int) *CLIArgs {
	cli.StorageShort = make([]string, 0, arrayLen)
	cli.StorageLong = make([]string, 0, arrayLen)
	cli.StorageUsage = make([]string, 0, arrayLen)
	cli.StorageFlagInt = make([]int, 0, arrayLen)
	cli.StorageFlagString = make([]string, 0, arrayLen)
	cli.StorageFlagBool = make([]bool, 0, arrayLen)
	mShort := make(map[string]bool, arrayLen)
	mLong := make(map[string]bool, arrayLen)
	mUsage := make(map[string]bool, arrayLen)
	for i := 0; i < len(strSize); i++ {
		for j := 0; j < arrayLen; j++ {
			if strSize[i] == strSizeFlag[0] {
				for {
					s := RandStr(strSize[i])
					_, ok := mShort[s]
					if !ok {
						mShort[s] = true
						break
					}
				}
			}
			if strSize[i] == strSizeFlag[1] {
				for {
					s := RandStr(strSize[i])
					_, ok := mLong[s]
					if !ok {
						mLong[s] = true
						break
					}
				}
			}
			if strSize[i] == strSizeFlag[2] {
				for {
					s := RandStr(strSize[i])
					_, ok := mUsage[s]
					if !ok {
						mUsage[s] = true
						break
					}
				}
			}
		}
	}
	for k := range mShort {
		cli.StorageShort = append(cli.StorageShort, k)
	}
	for k := range mLong {
		cli.StorageLong = append(cli.StorageLong, k)
	}
	for k := range mUsage {
		cli.StorageUsage = append(cli.StorageUsage, k)
	}
	for i := 0; i < arrayLen; i++ {
		cli.StorageFlagInt = append(cli.StorageFlagInt, 0)
	}
	for i := 0; i < arrayLen; i++ {
		cli.StorageFlagString = append(cli.StorageFlagString, "&")
	}
	for i := 0; i < arrayLen; i++ {
		cli.StorageFlagBool = append(cli.StorageFlagBool, true)
	}
	return cli
}

func (cli *CLIArgs) InitialFlags(f *Flag) *CLIArgs {
	for i := 0; i < 10; i++ {
		f.IntVarF(&cli.StorageFlagInt[i], cli.StorageLong[i], cli.StorageShort[i], -1, cli.StorageUsage[i])
	}
	for i := 10; i < 20; i++ {
		f.StringVarF(&cli.StorageFlagString[i-10], cli.StorageLong[i], cli.StorageShort[i], "*", cli.StorageUsage[i])
	}
	for i := 20; i < 30; i++ {
		f.BoolVarF(&cli.StorageFlagBool[i-20], cli.StorageLong[i], cli.StorageShort[i], false, cli.StorageUsage[i])
	}

	return cli
}

func TestSettingsFromFlags(t *testing.T) {
	f := NewFlag("Test Flag")
	paras := []int{1, 5, 10} // generate the random string, the length are 1, 5, 10
	varSize := InitialVar(paras)
	c := CLIArgs{}
	cli := c.DefineFlagsArray(30, paras, varSize)
	cli = c.InitialFlags(f)

	var num [10]string
	var str [10]string
	var bo [10]string
	input := make([]string, 0, 60)
	inputFlags := make([]string, 0, 30)
	inputFlagsValue := make([]string, 0, 30)

	// generate the commandline input
	for i := 0; i < 10; i++ {
		num[i] = strconv.Itoa(rand.Intn(100))
		str[i] = RandStr(rand.Intn(5) + 1)
		bo[i] = strconv.FormatBool(true)
	}
	for i := 0; i < 30; i++ {
		if i < 10 {
			marked := "-" + cli.StorageShort[i]
			input = append(input, marked)
			inputFlags = append(inputFlags, marked)
			input = append(input, num[i])
			inputFlagsValue = append(inputFlagsValue, num[i])
		}
		if i >= 10 && i < 20 {
			marked := "-" + cli.StorageShort[i]
			input = append(input, marked)
			inputFlags = append(inputFlags, marked)
			input = append(input, str[i-10])
			inputFlagsValue = append(inputFlagsValue, str[i-10])

		}
		if i >= 20 && i < 30 {
			inputFlags = append(inputFlags, "-"+cli.StorageShort[i])
			marked := "-" + cli.StorageShort[i] + "=" + bo[i-20]
			input = append(input, marked)
			inputFlagsValue = append(inputFlagsValue, bo[i-20])
		}
	}

	// test the default value
	SetUpInputs([]string{""}, f)

	for i := 0; i < 30; i++ {
		if i < 10 {
			assert.Equal(t, -1, cli.StorageFlagInt[i], "Failed to get the default value")
		}
		if i >= 10 && i < 20 {
			assert.Equal(t, "*", cli.StorageFlagString[i-10], "Failed to get the default value")
		}
		if i >= 20 && i < 30 {
			assert.Equal(t, false, cli.StorageFlagBool[i-20], "Failed to get the default value")
		}
	}

	// test commandline parse value
	SetUpInputs(input, f)

	for i := 0; i < 30; i++ {
		if i < 10 {
			assert.Equal(t, inputFlagsValue[i], strconv.Itoa(cli.StorageFlagInt[i]), "Failed to parse the value")
		}
		if i >= 10 && i < 20 {
			assert.Equal(t, inputFlagsValue[i], cli.StorageFlagString[i-10], "Failed to parse the value")
		}
		if i >= 20 && i < 30 {
			assert.Equal(t, inputFlagsValue[i], strconv.FormatBool(cli.StorageFlagBool[i-20]), "Failed to parse the value")
		}
	}

	// test flag usage
	for i := 0; i < 30; i++ {
		assert.Equal(t, cli.StorageUsage[i], f.fs.Lookup(cli.StorageShort[i]).Usage, "Failed to give the usage of the flag")
	}

	// test the display of usage
	s := f.Format()
	for i := 0; i < 30; i++ {
		assert.Equal(t, f.cliStrings[i], s[i+1], "Failed to match the format")
	}
}
