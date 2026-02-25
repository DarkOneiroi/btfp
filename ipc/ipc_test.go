package ipc

import (
	"encoding/gob"
	"net"
	"testing"
	"time"
)

func TestIPCPipe(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	testState := PlayerState{
		IsPlaying: true,
		Volume:    0.8,
		Elapsed:   10 * time.Second,
	}

	// 1. Test state broadcast (Mock Server -> Client)
	go func() {
		enc := gob.NewEncoder(c2)
		_ = enc.Encode(testState)
	}()

	dec := gob.NewDecoder(c1)
	var received PlayerState
	err := dec.Decode(&received)
	if err != nil {
		t.Fatalf("failed to decode state: %v", err)
	}

	if received.Volume != 0.8 {
		t.Errorf("expected volume 0.8, got %f", received.Volume)
	}

	// 2. Test command sending (Client -> Mock Server)
	testCmd := Command{Type: CmdPause}
	go func() {
		enc := gob.NewEncoder(c1)
		_ = enc.Encode(testCmd)
	}()

	dec2 := gob.NewDecoder(c2)
	var receivedCmd Command
	err = dec2.Decode(&receivedCmd)
	if err != nil {
		t.Fatalf("failed to decode command: %v", err)
	}

	if receivedCmd.Type != CmdPause {
		t.Errorf("expected CmdPause, got %v", receivedCmd.Type)
	}
}
