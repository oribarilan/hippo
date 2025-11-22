package main

// Version is set via ldflags during build: -X main.Version=v0.3.0
var Version = "dev"

// defaultLoadLimit is the number of items to load per request
const defaultLoadLimit = 40
