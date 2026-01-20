package core

// This file handles asset embedding. 
// For a production build, place your gzipped binaries (node.gz, payload.gz) 
// and JS bridge (client.js) in the assets/ directory.

import "embed"

// No-op file to ensure the package exists and can be imported.
// The actual embedding is handled in loader.go via //go:embed assets/*
