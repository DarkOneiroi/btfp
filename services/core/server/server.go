// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package server

// Start initializes and runs the IPC server based on the session type
func Start(session string) {
	RegisterTypes()

	s := NewMusicServer(session)
	s.Start()
}
