package output

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters/junitxml"
)

const (
	flakyTestCasesEnvVarKey              = "BITRISE_FLAKY_TEST_CASES"
	flakyTestCasesEnvVarSizeLimitInBytes = 1024
)

type Exporter interface {
	ExportTestResultsDir(dir string) error
	ExportFlakyTestsEnvVar(mergedTestResultXmlPths []string) error
}

type exporter struct {
	outputExporter OutputExporter
	converter      junitxml.Converter
	logger         log.Logger
}

func NewExporter(outputExporter OutputExporter, converter junitxml.Converter, logger log.Logger) Exporter {
	return &exporter{
		outputExporter: outputExporter,
		converter:      converter,
		logger:         logger,
	}
}

func (e exporter) ExportTestResultsDir(dir string) error {
	if err := e.outputExporter.ExportOutput("VDTESTING_DOWNLOADED_FILES_DIR", dir); err != nil {
		return err
	}
	e.logger.Donef("The downloaded test assets path (%s) is exported to the VDTESTING_DOWNLOADED_FILES_DIR environment variable.", dir)
	return nil
}

func (e exporter) ExportFlakyTestsEnvVar(mergedTestResultXmlPths []string) error {
	var flakyTestSuites []TestSuite
	for _, testResultXMLPth := range mergedTestResultXmlPths {
		testReport, err := e.convertTestReport(testResultXMLPth)
		if err != nil {
			return fmt.Errorf("failed to convert test report (%s): %w", testResultXMLPth, err)
		}

		testSuites := e.getFlakyTestSuites(testReport)
		flakyTestSuites = append(flakyTestSuites, testSuites...)
	}

	if err := e.exportFlakyTestCasesEnvVar(flakyTestSuites); err != nil {
		return fmt.Errorf("failed to export flaky test cases env var: %w", err)
	}

	return nil
}

func (e exporter) convertTestReport(pth string) (TestReport, error) {
	data, err := os.ReadFile(pth)
	if err != nil {
		return TestReport{}, err
	}

	var testReport TestReport
	if err := xml.Unmarshal(data, &testReport); err == nil {
		return testReport, nil
	}

	return testReport, nil
}

func (e exporter) getFlakyTestSuites(testReport TestReport) []TestSuite {
	var flakyTestSuites []TestSuite

	var flakyTests []TestCase
	for _, testCase := range testReport.TestSuite.TestCases {
		if testCase.Flaky == "true" {
			flakyTests = append(flakyTests, testCase)
		}
	}

	if len(flakyTests) > 0 {
		flakyTestSuites = append(flakyTestSuites, TestSuite{
			XMLName:   testReport.TestSuite.XMLName,
			Name:      testReport.TestSuite.Name,
			TestCases: flakyTests,
		})
	}

	return flakyTestSuites
}

func (e exporter) exportFlakyTestCasesEnvVar(flakyTestSuites []TestSuite) error {
	if len(flakyTestSuites) == 0 {
		return nil
	}

	storedFlakyTestCases := map[string]bool{}
	var flakyTestCases []string

	for _, testSuite := range flakyTestSuites {
		for _, testCase := range testSuite.TestCases {
			testCaseName := testCase.Name
			if len(testCase.ClassName) > 0 {
				testCaseName = fmt.Sprintf("%s.%s", testCase.ClassName, testCase.Name)
			}

			if len(testSuite.Name) > 0 {
				testCaseName = testSuite.Name + "." + testCaseName
			}

			if _, stored := storedFlakyTestCases[testCaseName]; !stored {
				storedFlakyTestCases[testCaseName] = true
				flakyTestCases = append(flakyTestCases, testCaseName)
			}
		}
	}

	if len(flakyTestCases) > 0 {
		e.logger.TDonef("%d flaky test case(s) detected, exporting %s env var", len(flakyTestCases), flakyTestCasesEnvVarKey)
	}

	var flakyTestCasesMessage string
	for i, flakyTestCase := range flakyTestCases {
		flakyTestCasesMessageLine := fmt.Sprintf("- %s\n", flakyTestCase)

		if len(flakyTestCasesMessage)+len(flakyTestCasesMessageLine) > flakyTestCasesEnvVarSizeLimitInBytes {
			e.logger.TWarnf("%s env var size limit (%d characters) exceeded. Skipping %d test cases.", flakyTestCasesEnvVarKey, flakyTestCasesEnvVarSizeLimitInBytes, len(flakyTestCases)-i)
			break
		}

		flakyTestCasesMessage += flakyTestCasesMessageLine
	}

	if err := e.outputExporter.ExportOutput(flakyTestCasesEnvVarKey, flakyTestCasesMessage); err != nil {
		return fmt.Errorf("failed to export %s: %w", flakyTestCasesEnvVarKey, err)
	}

	return nil
}
