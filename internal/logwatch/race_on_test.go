//go:build race

package logwatch

// raceEnabled reports whether the race detector is on. Its instrumentation
// allocates, so allocation-count assertions are skipped under -race.
const raceEnabled = true
