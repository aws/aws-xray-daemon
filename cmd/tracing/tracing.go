// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

// +build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/cihub/seelog"
)

func (d *Daemon) blockSignalReceived() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Kill)
	s := <-sigs
	log.Debugf("Shutdown Initiated. Current epoch in nanoseconds: %v", time.Now().UnixNano())
	log.Infof("Got shutdown signal: %v", s)
	d.stop()
}

func main() {
	d := initDaemon(config)
	defer d.close()
	go func() {
		d.blockSignalReceived()
	}()
	runDaemon(d)
}
