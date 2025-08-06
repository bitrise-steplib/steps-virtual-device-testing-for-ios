package output

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/testreport"
	"github.com/bitrise-steplib/steps-virtual-device-testing-for-ios/mocks"
)

func Test_exporter_ExportFlakyTestsEnvVar(t *testing.T) {
	longTestSuitName1 := "Suite1"
	longTestCaseName1 := strings.Repeat("a", flakyTestCasesEnvVarSizeLimitInBytes-len(fmt.Sprintf("- %s.\n", longTestSuitName1)))

	tests := []struct {
		name                   string
		testReport             testreport.TestReport
		wantFlakyTestsEnvValue string
		expectedWarningLogArgs []any
	}{
		{
			name: "No flaky tests",
			testReport: testreport.TestReport{
				TestSuites: []testreport.TestSuite{
					{
						Name: "Suite1",
						TestCases: []testreport.TestCase{
							{
								Name: "TestCase1",
							},
						},
					},
				},
			},
		},
		{
			name: "One flaky test",
			testReport: testreport.TestReport{
				TestSuites: []testreport.TestSuite{
					{
						Name: "Suite1",
						TestCases: []testreport.TestCase{
							{
								Name:    "TestCase1",
								Failure: &testreport.Failure{},
							},
							{
								Name: "TestCase1",
							},
						},
					},
				},
			},
			wantFlakyTestsEnvValue: "- Suite1.TestCase1\n",
		},
		{
			name: "Multiple flaky tests",
			testReport: testreport.TestReport{
				TestSuites: []testreport.TestSuite{
					{
						Name: "Suite1",
						TestCases: []testreport.TestCase{
							{
								ClassName: "com.example.TestClass1",
								Name:      "testMethod1",
								Failure:   &testreport.Failure{},
							},
							{
								ClassName: "com.example.TestClass1",
								Name:      "testMethod1",
							},
							{
								ClassName: "com.example.TestClass1",
								Name:      "testMethod2",
								Failure:   &testreport.Failure{},
							},
							{
								ClassName: "com.example.TestClass1",
								Name:      "testMethod2",
							},
							{
								ClassName: "com.example.TestClass2",
								Name:      "testMethod1",
								Failure:   &testreport.Failure{},
							},
							{
								ClassName: "com.example.TestClass2",
								Name:      "testMethod1",
							},
						},
					},
					{
						Name: "Suite2",
						TestCases: []testreport.TestCase{
							{
								ClassName: "com.example.TestClass1",
								Name:      "testMethod1",
								Failure:   &testreport.Failure{},
							},
							{
								ClassName: "com.example.TestClass1",
								Name:      "testMethod1",
							},
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
			testReport: testreport.TestReport{
				TestSuites: []testreport.TestSuite{
					{
						Name: "Suite1",
						TestCases: []testreport.TestCase{
							{
								ClassName: "com.example.TestClass1",
								Name:      "testMethod1",
								Failure:   &testreport.Failure{},
							},
							{
								ClassName: "com.example.TestClass1",
								Name:      "testMethod1",
							},
						},
					},
					{
						Name: "Suite1",
						TestCases: []testreport.TestCase{
							{
								ClassName: "com.example.TestClass1",
								Name:      "testMethod1",
								Failure:   &testreport.Failure{},
							},
							{
								ClassName: "com.example.TestClass1",
								Name:      "testMethod1",
							},
						},
					},
				},
			},
			wantFlakyTestsEnvValue: "- Suite1.com.example.TestClass1.testMethod1\n",
		},
		{
			name: "Flaky test cases env var size is limited",
			testReport: testreport.TestReport{
				TestSuites: []testreport.TestSuite{
					{
						Name: longTestSuitName1,
						TestCases: []testreport.TestCase{
							{
								Name:    longTestCaseName1,
								Failure: &testreport.Failure{},
							},
							{
								Name: longTestCaseName1,
							},
						},
					},
					{
						Name: "Suite2",
						TestCases: []testreport.TestCase{
							{
								Name:    "testMethod1",
								Failure: &testreport.Failure{},
							},
							{
								Name: "testMethod1",
							},
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
			flakyTestSuites := e.getFlakyTestSuites(tt.testReport)
			err := e.exportFlakyTestCasesEnvVar(flakyTestSuites)

			require.NoError(t, err)
		})
	}
}
