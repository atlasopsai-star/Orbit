package main

import (
	"embed"

	"github.com/atlasopsai-star/Orbit/src/cmd"
)

var (
	//go:embed src/orbit_config/*
	content embed.FS
)

func main() {
	cmd.Run(content)
}
