package main

import (
	"bytes"
	"testing"
	"time"
)

func testRefTime() time.Time {
	return time.Date(2022, 1, 1, 12, 30, 0, 0, time.UTC)
}

func Test_printStepsStates(t *testing.T) {
	tests := []struct {
		name               string
		stepIDToStepStates map[string]stepStates
		currentTime        time.Time
	}{
		{
			name: "",
			stepIDToStepStates: map[string]stepStates{
				"ID_1": {
					name: "iOS Tests",
					stateToStartTime: map[string]time.Time{
						"pending":    testRefTime(),
						"inProgress": testRefTime().Add(60 * time.Second),
						"complete":   testRefTime().Add(90 * time.Second),
					},
				},
				"ID_2": {
					name: "iOS Unit Tests",
					stateToStartTime: map[string]time.Time{
						"pending":    testRefTime(),
						"inProgress": testRefTime().Add(40 * time.Second),
						"complete":   testRefTime().Add(90 * time.Second),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			printStepsStates(tt.stepIDToStepStates, testRefTime().Add(90*time.Second), &b)

			actual := b.String()
			expected := `iOS Tests
- time spent in pending state: ~1m0s
- time spent in inProgress state: ~30s
- time spent in complete state: ~0s
iOS Unit Tests
- time spent in pending state: ~40s
- time spent in inProgress state: ~50s
- time spent in complete state: ~0s
`
			if actual != expected {
				t.Fatalf("%s != %s", actual, expected)
			}
		})
	}
}
