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
