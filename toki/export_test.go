package toki

import (
	"time"
)

func ForceUSPacificForTesting() {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.FixedZone("America/Los_Angeles", -8*60*60)
	}
	time.Local = loc
}
