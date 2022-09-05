package main

import (
	"bytes"
	"testing"
	"time"
)

func testRefTime() time.Time {
	return time.Date(2022, 1, 1, 12, 30, 0, 0, time.UTC)
}

func Test_printStepsStatesToStartTime(t *testing.T) {
	tests := []struct {
		name                   string
		stepsStatesToStartTime map[string]map[string]time.Time
		stepsToNames           map[string]string
		currentTime            time.Time
	}{
		{
			name: "",
			stepsStatesToStartTime: map[string]map[string]time.Time{
				"TestID": {
					"pending":    testRefTime(),
					"inProgress": testRefTime().Add(60 * time.Second),
					"complete":   testRefTime().Add(90 * time.Second),
				},
			},
			stepsToNames: map[string]string{
				"TestID": "iOS Tests",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			printStepsStatesToStartTime(tt.stepsStatesToStartTime, tt.stepsToNames, testRefTime().Add(90*time.Second), &b)

			actual := b.String()
			expected := `iOS Tests
- time spent in pending state: ~60s
- time spent in inProgress state: ~30s
- time spent in complete state: ~0s
`
			if actual != expected {
				t.Fatalf("%s != %s", actual, expected)
			}
		})
	}
}
