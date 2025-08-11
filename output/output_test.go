package output

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bitrise-steplib/steps-virtual-device-testing-for-ios/mocks"
)

func TestOutput(t *testing.T) {
	_, b, _, _ := runtime.Caller(0)
	outputPackageDir := filepath.Dir(b)
	testDataDir := filepath.Join(outputPackageDir, "testdata")
	mergedTestResultXMLPaths := []string{
		filepath.Join(testDataDir, "iphone8-16.6-en-portrait-test_results_merged.xml"),
		filepath.Join(testDataDir, "iphone13pro-16.6-en-landscape-test_results_merged.xml"),
		filepath.Join(testDataDir, "iphone13pro-16.6-en-portrait-test_results_merged.xml"),
	}
	wantFlakyTestsEnvValue := "- BullsEyeFailingTests.BullsEyeRandomlyFailingTests.testRandomlyFail\n"

	logger := mocks.NewLogger(t)
	mockOutputExporter := mocks.NewOutputExporter(t)

	logger.On("TDonef", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockOutputExporter.On("ExportOutput", "BITRISE_FLAKY_TEST_CASES", wantFlakyTestsEnvValue).Return(nil)

	e := exporter{
		outputExporter: mockOutputExporter,
		logger:         logger,
	}

	err := e.ExportFlakyTestsEnvVar(mergedTestResultXMLPaths)
	require.NoError(t, err)
}

func Test_exporter_exportFlakyTestCasesEnvVar(t *testing.T) {
	longTestSuitName1 := "Suite1"
	longTestCaseName1 := strings.Repeat("a", flakyTestCasesEnvVarSizeLimitInBytes-len(fmt.Sprintf("- %s.\n", longTestSuitName1)))

	tests := []struct {
		name                   string
		testSuites             []TestSuite
		wantFlakyTestsEnvValue string
		expectedWarningLogArgs []any
	}{
		{
			name: "No flaky tests",
			testSuites: []TestSuite{
				{
					Name: "Suite1",
					TestCases: []TestCase{
						{
							Name: "TestCase1",
						},
					},
				},
			},
		},
		{
			name: "One flaky test",
			testSuites: []TestSuite{
				{
					Name: "Suite1",
					TestCases: []TestCase{
						{
							Name:    "TestCase1",
							Failure: &Failure{},
							Flaky:   "true",
						},
					},
				},
			},
			wantFlakyTestsEnvValue: "- Suite1.TestCase1\n",
		},
		{
			name: "Multiple flaky tests",
			testSuites: []TestSuite{
				{
					Name: "Suite1",
					TestCases: []TestCase{
						{
							ClassName: "com.example.TestClass1",
							Name:      "testMethod1",
							Failure:   &Failure{},
							Flaky:     "true",
						},
						{
							ClassName: "com.example.TestClass1",
							Name:      "testMethod2",
							Failure:   &Failure{},
							Flaky:     "true",
						},
						{
							ClassName: "com.example.TestClass2",
							Name:      "testMethod1",
							Failure:   &Failure{},
							Flaky:     "true",
						},
					},
				},
				{
					Name: "Suite2",
					TestCases: []TestCase{
						{
							ClassName: "com.example.TestClass1",
							Name:      "testMethod1",
							Failure:   &Failure{},
							Flaky:     "true",
						},
					},
				},
			},
			wantFlakyTestsEnvValue: `- Suite1.com.example.TestClass1.testMethod1
- Suite1.com.example.TestClass1.testMethod2
- Suite1.com.example.TestClass2.testMethod1
- Suite2.com.example.TestClass1.testMethod1
`,
		},
		{
			name: "Tests with the same Test ID exported only once",
			testSuites: []TestSuite{
				{
					Name: "Suite1",
					TestCases: []TestCase{
						{
							ClassName: "com.example.TestClass1",
							Name:      "testMethod1",
							Failure:   &Failure{},
							Flaky:     "true",
						},
					},
				},
				{
					Name: "Suite1",
					TestCases: []TestCase{
						{
							ClassName: "com.example.TestClass1",
							Name:      "testMethod1",
							Failure:   &Failure{},
							Flaky:     "true",
						},
					},
				},
			},
			wantFlakyTestsEnvValue: "- Suite1.com.example.TestClass1.testMethod1\n",
		},
		{
			name: "Flaky test cases env var size is limited",
			testSuites: []TestSuite{
				{
					Name: longTestSuitName1,
					TestCases: []TestCase{
						{
							Name:    longTestCaseName1,
							Failure: &Failure{},
							Flaky:   "true",
						},
					},
				},
				{
					Name: "Suite2",
					TestCases: []TestCase{
						{
							Name:    "testMethod1",
							Failure: &Failure{},
							Flaky:   "true",
						},
					},
				},
			},
			wantFlakyTestsEnvValue: fmt.Sprintf("- %s.%s\n", longTestSuitName1, longTestCaseName1),
			expectedWarningLogArgs: []any{"%s env var size limit (%d characters) exceeded. Skipping %d test cases.", flakyTestCasesEnvVarKey, flakyTestCasesEnvVarSizeLimitInBytes, 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := mocks.NewLogger(t)
			mockOutputExporter := mocks.NewOutputExporter(t)
			if tt.wantFlakyTestsEnvValue != "" {
				logger.On("TDonef", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockOutputExporter.On("ExportOutput", "BITRISE_FLAKY_TEST_CASES", tt.wantFlakyTestsEnvValue).Return(nil)
			}
			if tt.expectedWarningLogArgs != nil {
				logger.On("TWarnf", tt.expectedWarningLogArgs...).Return(tt.expectedWarningLogArgs...)
			}

			e := exporter{
				outputExporter: mockOutputExporter,
				logger:         logger,
			}
			err := e.exportFlakyTestCasesEnvVar(tt.testSuites)
			require.NoError(t, err)
		})
	}
}
