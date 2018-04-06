// Copyright 2016 Mikio Hara. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package tcpinfo

/*
#include <netinet/tcp.h>
*/
import "C"

const (
	TCP_CONNECTION_INFO     = C.TCP_CONNECTION_INFO
	SizeofTCPConnectionInfo = C.sizeof_struct_tcp_connection_info
)

type TCPConnectionInfo C.struct_tcp_connection_info
