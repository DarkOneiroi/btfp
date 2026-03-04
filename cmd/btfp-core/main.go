// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package main

import (
	"btfp/services/core/server"
	"flag"
)

func main() {
	session := flag.String("session", "music", "Session name for this instance")
	flag.Parse()

	// btfp-core is now a pure background service
	server.Start(*session)
}
