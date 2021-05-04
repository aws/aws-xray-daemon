// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package profiler

import (
	"os"
	"runtime/pprof"

	log "github.com/cihub/seelog"
)

// EnableCPUProfile enables CPU profiling.
func EnableCPUProfile(cpuProfile *string) {
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Errorf("error: %v", err)
		}
		pprof.StartCPUProfile(f)
		log.Info("Start CPU Profiling")
	}
}

// MemSnapShot creates memory profile.
func MemSnapShot(memProfile *string) {
	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			log.Errorf("Could not create memory profile: %v", err)
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Errorf("Could not write memory profile: %v", err)
		}
		err = f.Close()
		if err != nil {
			log.Errorf("unable to close file: %v", err)
		}
		log.Info("Finish memory profiling")
		return
	}
}
