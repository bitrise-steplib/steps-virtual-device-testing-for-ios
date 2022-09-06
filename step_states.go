package main

import (
	"fmt"
	"io"
	"sort"
	"time"

	toolresults "google.golang.org/api/toolresults/v1beta3"
)

type stepStates struct {
	name             string
	stateToStartTime map[string]time.Time
}

func newStepStates(step toolresults.Step) stepStates {
	return stepStates{
		name:             createStepNameWithDimensions(step),
		stateToStartTime: map[string]time.Time{},
	}
}

func (s *stepStates) saveState(state string, startTime time.Time) {
	if _, ok := s.stateToStartTime[state]; ok {
		return
	}

	// Haven't seen this state yet -> set state start time
	s.stateToStartTime[state] = startTime
}

func (s *stepStates) print(currentTime time.Time, w io.Writer) {
	if _, err := fmt.Fprintln(w, s.name); err != nil {
		fmt.Printf("Failed to print step status durations: %s", err)
		return
	}

	var states []string
	for state := range s.stateToStartTime {
		states = append(states, state)
	}

	sort.Slice(states, func(i, j int) bool {
		stateI, stateJ := states[i], states[j]
		startTimeI, startTimeJ := s.stateToStartTime[stateI], s.stateToStartTime[stateJ]

		return startTimeI.Before(startTimeJ)
	})

	for i, state := range states {
		startTime := s.stateToStartTime[state]

		var endTime time.Time
		if i == len(states)-1 {
			endTime = currentTime
		} else {
			nextState := states[i+1]
			endTime = s.stateToStartTime[nextState]
		}

		duration := endTime.Sub(startTime)

		if _, err := fmt.Fprintf(w, "- time spent in %s state: ~%s\n", state, duration.Round(time.Second).String()); err != nil {
			fmt.Printf("Failed to print step status durations: %s", err)
			return
		}
	}
}

func createStepNameWithDimensions(step toolresults.Step) string {
	dimensions := createDimensions(step)
	return fmt.Sprintf("%s (%s %s %s %s)", step.Name, dimensions["Model"], dimensions["Version"], dimensions["Orientation"], dimensions["Locale"])
}

func updateStepsStates(stepIDtoStates map[string]stepStates, response toolresults.ListStepsResponse) {
	for _, step := range response.Steps {
		stepStates, ok := stepIDtoStates[step.StepId]
		if !ok {
			stepStates = newStepStates(*step)
			stepIDtoStates[step.StepId] = stepStates
		}

		stepStates.saveState(step.State, time.Now())
	}
}

func printStepsStates(stepIDtoStates map[string]stepStates, currentTime time.Time, w io.Writer) {
	var stepIDs []string
	for stepID := range stepIDtoStates {
		stepIDs = append(stepIDs, stepID)
	}

	sort.Strings(stepIDs)

	for _, stepID := range stepIDs {
		stepState := stepIDtoStates[stepID]
		stepState.print(currentTime, w)
	}
}
