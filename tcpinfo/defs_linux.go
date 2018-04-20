// Copyright 2016 Mikio Hara. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package tcpinfo

/*
#include <linux/tcp.h>
*/
import "C"

const (
	TCP_INFO      = C.TCP_INFO
	SizeofTCPInfo = C.sizeof_struct_tcp_info
)

type TCPInfo C.struct_tcp_info
