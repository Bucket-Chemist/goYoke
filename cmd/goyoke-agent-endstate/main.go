package main

import (
	"github.com/Bucket-Chemist/goYoke/defaults"
	"github.com/Bucket-Chemist/goYoke/internal/hooks/agentendstate"
	"github.com/Bucket-Chemist/goYoke/pkg/resolve"
)

func main() {
	resolve.SetDefault(defaults.FS)
	agentendstate.Main()
}
