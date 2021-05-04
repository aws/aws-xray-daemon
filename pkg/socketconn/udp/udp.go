// Copyright 2018-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package udp

import (
	"net"
	"os"

	"github.com/aws/aws-xray-daemon/pkg/socketconn"
	log "github.com/cihub/seelog"
)

// UDP defines UDP socket connection.
type UDP struct {
	socket *net.UDPConn
}

// New returns new instance of UDP.
func New(udpAddress string) socketconn.SocketConn {
	log.Debugf("Listening on UDP %v", udpAddress)
	addr, err := net.ResolveUDPAddr("udp", udpAddress)
	if err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}
	sock, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Errorf("%v", err)
		os.Exit(1)
	}
	return UDP{
		socket: sock,
	}
}

// Read returns number of bytes read from the UDP connection.
func (conn UDP) Read(b []byte) (int, error) {
	rlen, _, err := conn.socket.ReadFromUDP(b)
	return rlen, err
}

// Close closes current UDP connection.
func (conn UDP) Close() {
	err := conn.socket.Close()
	if err != nil {
		log.Errorf("unable to close the UDP connection: %v", err)
	}
}
