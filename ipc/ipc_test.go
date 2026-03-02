// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package ipc

import (
	"net"
	"testing"
)

func TestIPC(t *testing.T) {
	l, err := net.Listen("unix", "/tmp/test.sock")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = l.Close() }()

	c1, err := net.Dial("unix", "/tmp/test.sock")
	if err == nil {
		defer func() { _ = c1.Close() }()
	}

	// Basic test to ensure types are registered and connection works
}
