package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-plist"
	"github.com/bitrise-io/go-steputils/v2/testquarantine"
	"github.com/bitrise-io/go-utils/v2/command"
	"github.com/bitrise-io/go-utils/v2/env"
	"github.com/bitrise-io/go-utils/v2/pathutil"
)

/*
parseQuarantinedTests converts the Bitrise quarantined tests JSON input ($BITRISE_QUARANTINED_TESTS_JSON)
to xctestrun file's SkipTestIdentifiers format: TestClass/TestMethod (`()` suffix removed) mapped by TestTargets.
*/
func parseQuarantinedTests(quarantinedTestsInput string) (map[string][]string, error) {
	quarantinedTests, err := testquarantine.ParseQuarantinedTests(quarantinedTestsInput)
	if err != nil {
		return nil, fmt.Errorf("failed to parse quarantined tests input: %w", err)
	}

	skippedTestsByTarget := map[string][]string{}
	for _, qt := range quarantinedTests {
		if len(qt.TestSuiteName) == 0 || qt.TestSuiteName[0] == "" || qt.ClassName == "" || qt.TestCaseName == "" {
			continue
		}

		testTarget := qt.TestSuiteName[0]
		testClass := qt.ClassName
		testMethod := strings.TrimSuffix(qt.TestCaseName, "()")

		skippedTests := skippedTestsByTarget[testTarget]
		skippedTests = append(skippedTests, fmt.Sprintf("%s/%s", testClass, testMethod))
		skippedTestsByTarget[testTarget] = skippedTests
	}

	return skippedTestsByTarget, nil
}

func addQuarantinedTestsToTestBundle(testBundleZipPth string, skippedTestByTarget map[string][]string) (string, error) {
	tmpTestBundlePth, err := unzipTestBundle(testBundleZipPth)
	if err != nil {
		return "", err
	}

	entries, err := os.ReadDir(tmpTestBundlePth)
	if err != nil {
		return "", fmt.Errorf("failed to read unzipped test bundle dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".xctestrun" {
			continue
		}

		xctestrunPth := filepath.Join(tmpTestBundlePth, entry.Name())
		if err := addQuarantinedTestsToXctestrun(xctestrunPth, skippedTestByTarget); err != nil {
			return "", fmt.Errorf("failed to add quarantined tests to xctestrun file (%s): %w", xctestrunPth, err)
		}
	}

	updatedTestBundleZipPath, err := zipTestBundle(tmpTestBundlePth, 6)
	if err != nil {
		return "", err
	}

	return updatedTestBundleZipPath, nil
}

func unzipTestBundle(testBundleZipPth string) (string, error) {
	pathProvider := pathutil.NewPathProvider()
	cmdFactory := command.NewFactory(env.NewRepository())

	tmpDir, err := pathProvider.CreateTempDir("test_bundle")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir for unzipping test bundle: %w", err)
	}

	cmd := cmdFactory.Create("unzip", []string{"-o", testBundleZipPth}, &command.Opts{Dir: tmpDir})
	cmdOut, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		if cmdOut != "" {
			printLastLines(cmdOut)
		}
		return "", err
	}

	return tmpDir, nil
}

func zipTestBundle(testBundlePth string, compressionLevel int) (string, error) {
	pathProvider := pathutil.NewPathProvider()
	cmdFactory := command.NewFactory(env.NewRepository())

	tmpDir, err := pathProvider.CreateTempDir("test_bundle_zip")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir for unzipping test bundle: %w", err)
	}

	testBundleZipPth := filepath.Join(tmpDir, "testbundle.zip")

	args := []string{"-r", fmt.Sprintf("-%d", compressionLevel), testBundleZipPth}

	entries, err := os.ReadDir(testBundlePth)
	if err != nil {
		return "", fmt.Errorf("failed to read unzipped test bundle dir: %w", err)
	}

	// add all build folders and xctestrun files to zip file:
	// - Debug-iphonesimulator/
	// - Debug-watchsimulator/
	// - BullsEye_UITests_iphonesimulator18.5-arm64.xctestrun
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) == ".xctestrun" {
			args = append(args, entry.Name())
		}
	}

	cmd := cmdFactory.Create("zip", args, &command.Opts{
		Dir: testBundlePth,
	})
	cmdOut, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		if cmdOut != "" {
			printLastLines(cmdOut)
		}
		return "", err
	}

	return testBundleZipPth, nil
}

func parseXctestrun(xctestrunPth string) (map[string]any, int, error) {
	xctestrunContent, err := os.ReadFile(xctestrunPth)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read xctestrun file: %w", err)
	}

	var xctestrun map[string]any
	format, err := plist.Unmarshal(xctestrunContent, &xctestrun)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal xctestrun plist: %w", err)
	}

	return xctestrun, format, nil
}

func writeXctestrun(xctestrunPth string, xctestrun map[string]any, format int) error {
	updatedXctestrunContent, err := plist.Marshal(xctestrun, format)
	if err != nil {
		return fmt.Errorf("failed to marshal xctestrun plist: %w", err)
	}

	if err := os.WriteFile(xctestrunPth, updatedXctestrunContent, 0644); err != nil {
		return fmt.Errorf("failed to write updated xctestrun file: %w", err)
	}

	return nil
}

func addQuarantinedTestsToXctestrun(xctestrunPth string, skippedTestByTarget map[string][]string) error {
	xctestrun, plistFormat, err := parseXctestrun(xctestrunPth)
	if err != nil {
		return err
	}

	updatedXctestrun, err := addSkippedTestsToXctestrun(xctestrun, skippedTestByTarget)
	if err != nil {
		return err
	}

	if err := writeXctestrun(xctestrunPth, updatedXctestrun, plistFormat); err != nil {
		return err
	}

	return nil
}

func addSkippedTestsToXctestrun(xctestrun map[string]any, skippedTestByTarget map[string][]string) (map[string]any, error) {
	testConfigurationsRaw, ok := xctestrun["TestConfigurations"]
	if !ok {
		return nil, fmt.Errorf("TestConfigurations not found in xctestrun")
	}

	testConfigurationsSlice, ok := testConfigurationsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid TestConfigurations format in xctestrun")
	}

	for testConfigurationIdx, testConfigurationRaw := range testConfigurationsSlice {
		testConfiguration, ok := testConfigurationRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid test configuration format in xctestrun")
		}

		testTargetsRaw, ok := testConfiguration["TestTargets"]
		if !ok {
			return nil, fmt.Errorf("TestTargets not found in test configuration")
		}

		testTargetsSlice, ok := testTargetsRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid TestTargets format in test configuration")
		}

		for testTargetIdx, testTargetRaw := range testTargetsSlice {
			testTarget, ok := testTargetRaw.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("invalid test target format in test configuration")
			}

			blueprintNameRaw, ok := testTarget["BlueprintName"]
			if !ok {
				return nil, fmt.Errorf("BlueprintName not found in test target")
			}

			blueprintName, ok := blueprintNameRaw.(string)
			if !ok {
				return nil, fmt.Errorf("invalid BlueprintName format in test target")
			}

			skippedTestsToAdd, ok := skippedTestByTarget[blueprintName]
			if !ok {
				continue
			}

			var skipTestIdentifiers []interface{}
			skipTestIdentifiersRaw, ok := testTarget["SkipTestIdentifiers"]
			if ok {
				skipTestIdentifiers, ok = skipTestIdentifiersRaw.([]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid SkipTestIdentifiers format in test target")
				}
			}

			for _, skippedTestsToAddItem := range skippedTestsToAdd {
				skipTestIdentifiers = append(skipTestIdentifiers, skippedTestsToAddItem)
			}

			testTarget["SkipTestIdentifiers"] = skipTestIdentifiers
			testTargetsSlice[testTargetIdx] = testTarget
		}

		testConfiguration["TestTargets"] = testTargetsSlice
		testConfigurationsSlice[testConfigurationIdx] = testConfiguration
	}

	xctestrun["TestConfigurations"] = testConfigurationsSlice

	return xctestrun, nil
}

func printLastLines(cmdOut string) {
	cmdOutSplit := strings.Split(cmdOut, "\n")
	var lastCmdOutLines string
	if len(cmdOutSplit) > 10 {
		lastCmdOutLines = strings.Join(cmdOutSplit[len(cmdOutSplit)-10:], "\n")
	} else {
		lastCmdOutLines = strings.Join(cmdOutSplit, "\n")
	}
	fmt.Printf("Last line of unzip output:\n%s", lastCmdOutLines)
}
