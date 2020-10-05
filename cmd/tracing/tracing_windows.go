// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

// +build windows

package main

import (
	"time"

	"golang.org/x/sys/windows/svc"
)

const serviceName = "AmazonX-RayDaemon"

func main() {
	svc.Run(serviceName, &TracingDaemonService{})
}

// Structure for X-Ray daemon as a service.
type TracingDaemonService struct{}

// Execute xray as Windows service. Implement golang.org/x/sys/windows/svc#Handler.
func (a *TracingDaemonService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {

	// notify service controller status is now StartPending
	s <- svc.Status{State: svc.StartPending}

	// start service
	d := initDaemon(config)
	// Start a routine to monitor all channels/routines initiated are closed
	// This is required for windows as windows daemon wait for process to finish using infinite for loop below
	go d.close()
	runDaemon(d)
	// update service status to Running
	const acceptCmds = svc.AcceptStop | svc.AcceptShutdown
	s <- svc.Status{State: svc.Running, Accepts: acceptCmds}
loop:
	// using an infinite loop to wait for ChangeRequests
	for {
		// block and wait for ChangeRequests
		c := <-r

		// handle ChangeRequest, svc.Pause is not supported
		switch c.Cmd {
		case svc.Interrogate:
			s <- c.CurrentStatus
			// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
			time.Sleep(100 * time.Millisecond)
			s <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			break loop
		default:
			continue loop
		}
	}
	s <- svc.Status{State: svc.StopPending}
	d.stop()
	return false, 0
}
