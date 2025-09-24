package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_addQuarantinedTestsToTestBundle(t *testing.T) {
	// Given
	xctestrunPth := filepath.Join("testdata", "BullsEye_RandomlyFailingTests_iphoneos18.2-arm64.xctestrun")

	tests := []struct {
		name                        string
		skippedTestIdentifiersToAdd map[string][]string
		wantOriginalSkippedTests    []string
		wantUpdatedSkippedTests     []string
	}{
		{
			name:                        "Add new skipped test, no existing skipped tests",
			skippedTestIdentifiersToAdd: map[string][]string{"BullsEyeUITests": {"BullsEyeUITests2/testGameStyleSwitch"}},
			wantOriginalSkippedTests:    []string(nil),
			wantUpdatedSkippedTests:     []string{"BullsEyeUITests2/testGameStyleSwitch"},
		},
		{
			name:                        "Add new skipped test to existing skipped tests",
			skippedTestIdentifiersToAdd: map[string][]string{"BullsEyeFailingTests": {"BullsEyeRandomlyFailingTests/testRandomlyFail"}},
			wantOriginalSkippedTests: []string{
				"BullsEyeEventuallyFailingInMemoryTests",
				"BullsEyeEventuallyFailingInMemoryTests/testFailIfNoSuccessesRemain",
				"BullsEyeEventuallyFailingTests",
				"BullsEyeEventuallyFailingTests/testFailIfNoSuccessesRemain",
				"BullsEyeEventuallySucceedingTests",
				"BullsEyeEventuallySucceedingTests/testPassIfNoFailuresRemain",
				"BullsEyeFailingTests",
				"BullsEyeFailingTests/testApiCallCompletes",
			},
			wantUpdatedSkippedTests: []string{
				"BullsEyeEventuallyFailingInMemoryTests",
				"BullsEyeEventuallyFailingInMemoryTests/testFailIfNoSuccessesRemain",
				"BullsEyeEventuallyFailingTests",
				"BullsEyeEventuallyFailingTests/testFailIfNoSuccessesRemain",
				"BullsEyeEventuallySucceedingTests",
				"BullsEyeEventuallySucceedingTests/testPassIfNoFailuresRemain",
				"BullsEyeFailingTests",
				"BullsEyeFailingTests/testApiCallCompletes",
				// New
				"BullsEyeRandomlyFailingTests/testRandomlyFail",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			xctestrun, _, err := parseXctestrun(xctestrunPth)
			require.NoError(t, err)

			var target string
			for k := range tt.skippedTestIdentifiersToAdd {
				target = k
				break
			}

			originalSkippedTests := getSkippedTests(xctestrun, target, t)
			require.Equal(t, tt.wantOriginalSkippedTests, originalSkippedTests)

			// When
			updatedXctestrun, err := addSkippedTestsToXctestrun(xctestrun, tt.skippedTestIdentifiersToAdd)

			// Then
			require.NoError(t, err)
			updatedSkippedTests := getSkippedTests(updatedXctestrun, target, t)
			require.Equal(t, tt.wantUpdatedSkippedTests, updatedSkippedTests)
		})
	}
}

func getSkippedTests(xctestrun map[string]any, target string, t *testing.T) []string {
	testConfigurationsRaw, ok := xctestrun["TestConfigurations"]
	require.True(t, ok)

	testConfigurationsSlice, ok := testConfigurationsRaw.([]interface{})
	require.True(t, ok)

	for _, testConfigurationRaw := range testConfigurationsSlice {
		testConfiguration, ok := testConfigurationRaw.(map[string]interface{})
		require.True(t, ok)

		testTargetsRaw, ok := testConfiguration["TestTargets"]
		require.True(t, ok)

		testTargetsSlice, ok := testTargetsRaw.([]interface{})
		require.True(t, ok)

		for _, testTargetRaw := range testTargetsSlice {
			testTarget, ok := testTargetRaw.(map[string]interface{})
			require.True(t, ok)

			blueprintNameRaw, ok := testTarget["BlueprintName"]
			require.True(t, ok)

			blueprintName, ok := blueprintNameRaw.(string)
			require.True(t, ok)

			if blueprintName != target {
				continue
			}

			var skipTestIdentifiers []string
			skipTestIdentifiersRaw, ok := testTarget["SkipTestIdentifiers"]
			if ok {
				skipTestIdentifiersList, ok := skipTestIdentifiersRaw.([]interface{})
				require.True(t, ok)

				for _, skipTestIdentifiersListItem := range skipTestIdentifiersList {
					skipTestIdentifier, ok := skipTestIdentifiersListItem.(string)
					require.True(t, ok)

					skipTestIdentifiers = append(skipTestIdentifiers, skipTestIdentifier)
				}
			}

			return skipTestIdentifiers
		}
	}

	return nil
}
