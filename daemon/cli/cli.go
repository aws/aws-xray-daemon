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
	"flag"
	"fmt"
	"os"
)

// Flag is used for cli parameters.
type Flag struct {
	// A set of flags used for cli configuration.
	fs *flag.FlagSet

	// String array used to display flag information on cli.
	cliStrings []string
}

// NewFlag returns a new flag with provided flag name.
func NewFlag(name string) *Flag {
	flag := &Flag{
		cliStrings: make([]string, 0, 19),
		fs:         flag.NewFlagSet(name, flag.ExitOnError),
	}
	return flag
}

// IntVarF defines 2 int flags for specified name and shortName with default value, and usage string.
// The argument ptr points to an int variable in which to store the value of the flag.
func (f *Flag) IntVarF(ptr *int, name string, shortName string, value int, usage string) {
	f.fs.IntVar(ptr, name, value, usage)
	f.fs.IntVar(ptr, shortName, value, usage)
	s := fmt.Sprintf("\t-%v\t--%v\t%v", shortName, name, usage)
	f.cliStrings = append(f.cliStrings, s)
}

// StringVarF defines 2 string flags for specified name and shortName, default value, and usage string.
// The argument ptr points to a string variable in which to store the value of the flag.
func (f *Flag) StringVarF(ptr *string, name string, shortName string, value string, usage string) {
	f.fs.StringVar(ptr, name, value, usage)
	f.fs.StringVar(ptr, shortName, value, usage)
	var s string
	if len(name) <= 4 {
		s = fmt.Sprintf("\t-%v\t--%v\t\t%v", shortName, name, usage)
	} else {
		s = fmt.Sprintf("\t-%v\t--%v\t%v", shortName, name, usage)
	}
	f.cliStrings = append(f.cliStrings, s)
}

// BoolVarF defines 2 bool flags with specified name and shortName, default value, and usage string.
// The argument ptr points to a bool variable in which to store the value of the flag.
func (f *Flag) BoolVarF(ptr *bool, name string, shortName string, value bool, usage string) {
	f.fs.BoolVar(ptr, name, value, usage)
	f.fs.BoolVar(ptr, shortName, value, usage)
	s := fmt.Sprintf("\t-%v\t--%v\t%v", shortName, name, usage)
	f.cliStrings = append(f.cliStrings, s)
}

// Format function formats Flag f for cli display.
func (f *Flag) Format() []string {
	var cliDisplay = make([]string, 0, 20)
	s := fmt.Sprint("Usage: X-Ray [options]")
	cliDisplay = append(cliDisplay, s)
	for val := range f.cliStrings {
		cliDisplay = append(cliDisplay, f.cliStrings[val])
	}
	s = fmt.Sprint("\t-h\t--help\t\tShow this screen")
	cliDisplay = append(cliDisplay, s)
	return cliDisplay
}

// ParseFlags parses flag definitions from the command line, which should not
// include the command name. Must be called after all flags in the FlagSet
// are defined and before flags are accessed by the program.
// The return value will be ErrHelp if -help or -h were set but not defined.
func (f *Flag) ParseFlags() {
	f.fs.Usage = func() {
		display := f.Format()
		for val := range display {
			fmt.Println(display[val])
		}
	}
	f.fs.Parse(os.Args[1:])
}
