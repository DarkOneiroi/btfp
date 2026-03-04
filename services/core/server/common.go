// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package server

import (
	"btfp/internal/ipc-shared"
	"btfp/internal/models"
	"encoding/gob"
	"net"
	"time"
)

// Client represents a connected IPC client
type Client struct {
	conn net.Conn
	enc  *gob.Encoder
}

type commandPacket struct {
	cmd ipc.Command
	c   *Client
}

type commandHandler interface {
	Handle(p interface{})
}

func RegisterTypes() {
	gob.Register(time.Duration(0))
	gob.Register(0)
	gob.Register(models.Track{})
	gob.Register([]models.Track{})
	gob.Register(ipc.TrackInfo{})
	gob.Register([]ipc.TrackInfo{})
}
