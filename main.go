package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"google.golang.org/api/testing/v1"
	toolresults "google.golang.org/api/toolresults/v1beta3"

	"github.com/bitrise-io/go-steputils/v2/stepconf"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-io/go-utils/v2/env"
	logv2 "github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-steplib/steps-deploy-to-bitrise-io/test/converters/junitxml"
	"github.com/bitrise-steplib/steps-virtual-device-testing-for-ios/output"
	"github.com/ryanuber/go-glob"
)

// ConfigsModel ...
type ConfigsModel struct {
	// api
	APIBaseURL string          `env:"api_base_url,required"`
	APIToken   stepconf.Secret `env:"api_token,required"`
	BuildSlug  string          `env:"BITRISE_BUILD_SLUG,required"`
	AppSlug    string          `env:"BITRISE_APP_SLUG,required"`

	// shared
	ZipPath              string  `env:"zip_path,file"`
	TestDevices          string  `env:"test_devices,required"`
	TestTimeout          float64 `env:"test_timeout,range[0..2700]"`
	DownloadTestResults  bool    `env:"download_test_results,opt[false,true]"`
	NumFlakyTestAttempts int     `env:"num_flaky_test_attempts,range[0..10]"`
}

// UploadURLRequest ...
type UploadURLRequest struct {
	AppURL     string `json:"appUrl"`
	TestAppURL string `json:"testAppUrl"`
}

type VDTForIosStep struct {
	envRepository  env.Repository
	inputParser    stepconf.InputParser
	logger         logv2.Logger
	outputExporter output.Exporter
}

func main() {
	envRepository := env.NewRepository()
	inputParser := stepconf.NewInputParser(envRepository)
	logger := logv2.NewLogger()
	outputExporter := output.NewExporter(output.NewOutputExporter(), junitxml.Converter{}, logger)

	step := NewVDTForIosStep(outputExporter, logger, inputParser, envRepository)
	configs, err := step.ProcessInputs()
	if err != nil {
		failf("Process config: %s", err)
	}
	isTestSuccessful, err := step.Run(configs)
	if err != nil {
		failf("Run step: %s", err)
	}
	if err := step.ExportOutput(configs); err != nil {
		if isTestSuccessful {
			failf("Export output: %s", err)
		} else {
			log.TWarnf("Export output: %s", err)
		}
	}

	if !isTestSuccessful {
		log.TErrorf("Tests failed")
		os.Exit(1)
	}
}

func NewVDTForIosStep(outputExporter output.Exporter, logger logv2.Logger, inputParser stepconf.InputParser, envRepository env.Repository) VDTForIosStep {
	return VDTForIosStep{
		outputExporter: outputExporter,
		logger:         logger,
		inputParser:    inputParser,
		envRepository:  envRepository,
	}
}

func (s VDTForIosStep) ProcessInputs() (ConfigsModel, error) {
	var configs ConfigsModel
	if err := s.inputParser.Parse(&configs); err != nil {
		return ConfigsModel{}, fmt.Errorf("couldn't create step config: %w", err)
	}

	stepconf.Print(configs)
	fmt.Println()

	return configs, nil
}

func (s VDTForIosStep) Run(configs ConfigsModel) (bool, error) {

	log.TInfof("Upload IPAs")
	if err := s.uploadIPAs(configs); err != nil {
		return false, fmt.Errorf("failed to upload IPAs, error: %s", err)
	}
	log.TDonef("=> .xctestrun uploaded")

	fmt.Println()
	log.TInfof("Start test")
	if err := s.startTests(configs); err != nil {
		return false, fmt.Errorf("failed to start tests, error: %s", err)
	}
	log.TDonef("=> Test started")

	fmt.Println()
	log.TInfof("Waiting for test results")
	isTestSuccessful, err := s.waitForTestResult(configs)
	if err != nil {
		return false, fmt.Errorf("failed to wait for test results, error: %s", err)
	}

	return isTestSuccessful, nil
}

func (s VDTForIosStep) ExportOutput(configs ConfigsModel) error {
	if configs.DownloadTestResults {
		fmt.Println()
		log.TInfof("Downloading test assets")
		{
			url := configs.APIBaseURL + "/assets/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + string(configs.APIToken)

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return fmt.Errorf("failed to create http request, error: %s", err)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("failed to get http response, error: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to get http response, status code: %d", resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response body, error: %s", err)
			}

			responseModel := map[string]string{}

			if err := json.Unmarshal(body, &responseModel); err != nil {
				return fmt.Errorf("failed to unmarshal response body, error: %s", err)
			}

			tempDir, err := pathutil.NormalizedOSTempDirPath("vdtesting_test_assets")
			if err != nil {
				return fmt.Errorf("failed to create temp dir, error: %s", err)
			}

			mergedTestResultXmlPth := ""
			singleTestResultXmlPth := ""
			for fileName, fileURL := range responseModel {
				pth := filepath.Join(tempDir, fileName)
				if err := downloadFile(fileURL, pth); err != nil {
					return fmt.Errorf("failed to download file, error: %s", err)
				}

				// per test run results: iphone13pro-16.6-en-portrait_test_result_0.xml
				if glob.Glob("*test_result_*.xml", pth) {
					singleTestResultXmlPth = pth
				}

				// merged result: iphone13pro-16.6-en-portrait-test_results_merged.xml
				if strings.HasSuffix(fileName, "test_results_merged.xml") {
					if mergedTestResultXmlPth != "" {
						log.TWarnf("Multiple merged test results XML files found, using the last one: %s", pth)
					} else {
						log.TPrintf("Merged test results XML found: %s", pth)
					}
					mergedTestResultXmlPth = pth
				}
			}

			testResultXmlPth := mergedTestResultXmlPth
			if testResultXmlPth == "" && singleTestResultXmlPth != "" {
				log.TWarnf("No merged test results XML found, using the latest single test result XML: %s", singleTestResultXmlPth)
				testResultXmlPth = singleTestResultXmlPth
			}

			log.TPrintf("Test results XML: %s", testResultXmlPth)
			log.TDonef("=> %d Test Assets downloaded", len(responseModel))

			if err := s.outputExporter.ExportTestResultsDir(tempDir); err != nil {
				log.TWarnf("Failed to export test assets: %s", err)
			} else if testResultXmlPth != "" {
				if err := s.outputExporter.ExportFlakyTestsEnvVar(testResultXmlPth); err != nil {
					log.TWarnf("Failed to export flaky tests env var: %s", err)
				}
			}
		}
	}
	return nil
}

func (s VDTForIosStep) uploadIPAs(configs ConfigsModel) error {
	url := configs.APIBaseURL + "/assets/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + string(configs.APIToken)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request, error: %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get http response, error: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body, error: %s", err)
		}
		return fmt.Errorf("failed to start test: %d, error: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body, error: %s", err)
	}

	responseModel := &UploadURLRequest{}

	if err := json.Unmarshal(body, responseModel); err != nil {
		return fmt.Errorf("failed to unmarshal response body, error: %s", err)
	}

	if err := uploadFile(responseModel.AppURL, configs.ZipPath); err != nil {
		return fmt.Errorf("failed to upload file(%s) to (%s), error: %s", configs.ZipPath, responseModel.AppURL, err)
	}

	return nil
}

func (s VDTForIosStep) startTests(configs ConfigsModel) error {
	url := configs.APIBaseURL + "/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + string(configs.APIToken)

	testModel := &testing.TestMatrix{}
	testModel.EnvironmentMatrix = &testing.EnvironmentMatrix{IosDeviceList: &testing.IosDeviceList{}}
	testModel.EnvironmentMatrix.IosDeviceList.IosDevices = []*testing.IosDevice{}
	testModel.FlakyTestAttempts = int64(configs.NumFlakyTestAttempts)

	scanner := bufio.NewScanner(strings.NewReader(configs.TestDevices))
	for scanner.Scan() {
		device := scanner.Text()
		device = strings.TrimSpace(device)
		if device == "" {
			continue
		}

		deviceParams := strings.Split(device, ",")
		if len(deviceParams) != 4 {
			return fmt.Errorf("invalid test device configuration: %s", device)
		}

		newDevice := testing.IosDevice{
			IosModelId:   deviceParams[0],
			IosVersionId: deviceParams[1],
			Locale:       deviceParams[2],
			Orientation:  deviceParams[3],
		}

		testModel.EnvironmentMatrix.IosDeviceList.IosDevices = append(testModel.EnvironmentMatrix.IosDeviceList.IosDevices, &newDevice)
	}

	testModel.TestSpecification = &testing.TestSpecification{
		TestTimeout: fmt.Sprintf("%fs", configs.TestTimeout),
	}

	testModel.TestSpecification.IosXcTest = &testing.IosXcTest{}

	jsonByte, err := json.Marshal(testModel)
	if err != nil {
		return fmt.Errorf("failed to marshal test model, error: %s", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonByte))
	if err != nil {
		return fmt.Errorf("failed to create http request, error: %s", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get http response, error: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body, error: %s", err)
		}
		return fmt.Errorf("failed to start test: %d, error: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s VDTForIosStep) waitForTestResult(configs ConfigsModel) (bool, error) {
	finished := false
	printedLogs := []string{}

	stepIDToStepStates := map[string]stepStates{}
	dimensionToStatus := map[string]bool{}

	for !finished {
		url := configs.APIBaseURL + "/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + string(configs.APIToken)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return false, fmt.Errorf("failed to create http request, error: %s", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil || (resp != nil && resp.StatusCode != http.StatusOK) {
			resp, err = client.Do(req)
			if err != nil {
				return false, fmt.Errorf("failed to get http response, error: %s", err)
			}
		}

		if resp == nil || resp.Body == nil {
			return false, fmt.Errorf("failed to get http response, response body is nil")
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, fmt.Errorf("failed to read response body, error: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("failed to get test status, error: %s", string(body))
		}

		responseModel := &toolresults.ListStepsResponse{}

		if err := json.Unmarshal(body, responseModel); err != nil {
			return false, fmt.Errorf("failed to unmarshal response body, error: %s, body: %s", err, string(body))
		}

		updateStepsStates(stepIDToStepStates, *responseModel)

		finished = true
		testsRunning := 0
		for _, step := range responseModel.Steps {
			if step.State != "complete" {
				finished = false
				testsRunning++
			}
		}

		msg := ""
		if len(responseModel.Steps) == 0 {
			finished = false
			msg = "- Validating"
		} else {
			msg = fmt.Sprintf("- (%d/%d) running", testsRunning, len(responseModel.Steps))
		}

		if !sliceutil.IsStringInSlice(msg, printedLogs) {
			log.Printf(msg)
			printedLogs = append(printedLogs, msg)
		}

		if finished {
			log.TDonef("=> Test finished")
			fmt.Println()

			printStepsStates(stepIDToStepStates, time.Now(), os.Stdout)
			fmt.Println()

			log.TInfof("Test results:")
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			if _, err := fmt.Fprintln(w, "Model\tOS version\tOrientation\tLocale\tOutcome\t"); err != nil {
				return false, fmt.Errorf("failed to write in writer")
			}

			for _, step := range responseModel.Steps {
				dimensions := createDimensions(*step)
				outcome := step.Outcome.Summary

				dimensionID := fmt.Sprintf("%s.%s.%s.%s", dimensions["Model"], dimensions["Version"], dimensions["Orientation"], dimensions["Locale"])
				dimensionToStatus[dimensionID] = true

				fmt.Printf("%s: %s\n", dimensionID, outcome)

				isSuccess := true
				if outcome == "failure" || outcome == "inconclusive" || outcome == "skipped" {
					isSuccess = false
				}

				_, exists := dimensionToStatus[dimensionID]
				if exists {
					if isSuccess {
						// Mark the dimension as successful if at least one step (test run) was successful.
						dimensionToStatus[dimensionID] = true
					}
				} else {
					dimensionToStatus[dimensionID] = isSuccess
				}

				switch outcome {
				case "success":
					dimensionToStatus[dimensionID] = true
					outcome = colorstring.Green(outcome)
				case "failure":
					dimensionToStatus[dimensionID] = false
					if step.Outcome.FailureDetail != nil {
						if step.Outcome.FailureDetail.Crashed {
							outcome += "(Crashed)"
						}
						if step.Outcome.FailureDetail.NotInstalled {
							outcome += "(NotInstalled)"
						}
						if step.Outcome.FailureDetail.OtherNativeCrash {
							outcome += "(OtherNativeCrash)"
						}
						if step.Outcome.FailureDetail.TimedOut {
							outcome += "(TimedOut)"
						}
						if step.Outcome.FailureDetail.UnableToCrawl {
							outcome += "(UnableToCrawl)"
						}
					}
					outcome = colorstring.Red(outcome)
				case "inconclusive":
					dimensionToStatus[dimensionID] = false
					if step.Outcome.InconclusiveDetail != nil {
						if step.Outcome.InconclusiveDetail.AbortedByUser {
							outcome += "(AbortedByUser)"
						}
						if step.Outcome.InconclusiveDetail.InfrastructureFailure {
							outcome += "(InfrastructureFailure)"
						}
					}
					outcome = colorstring.Yellow(outcome)
				case "skipped":
					dimensionToStatus[dimensionID] = false
					if step.Outcome.SkippedDetail != nil {
						if step.Outcome.SkippedDetail.IncompatibleAppVersion {
							outcome += "(IncompatibleAppVersion)"
						}
						if step.Outcome.SkippedDetail.IncompatibleArchitecture {
							outcome += "(IncompatibleArchitecture)"
						}
						if step.Outcome.SkippedDetail.IncompatibleDevice {
							outcome += "(IncompatibleDevice)"
						}
					}
					outcome = colorstring.Blue(outcome)
				}

				if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t\n", dimensions["Model"], dimensions["Version"], dimensions["Orientation"], dimensions["Locale"], outcome); err != nil {
					return false, fmt.Errorf("failed to write in writer")
				}
			}
			if err := w.Flush(); err != nil {
				log.Errorf("Failed to flush writer, error: %s", err)
			}
		}
		if !finished {
			time.Sleep(10 * time.Second)
		}
	}

	testRunSuccessful := true
	var failingDimensions []string
	for dimension, isSuccess := range dimensionToStatus {
		if !isSuccess {
			testRunSuccessful = false
			failingDimensions = append(failingDimensions, dimension)
		}
	}

	fmt.Println("Failing dimensions:", strings.Join(failingDimensions, "\n"))

	return testRunSuccessful, nil
}

func downloadFile(url string, localPath string) error {
	// on HFS file system the max file name length: 255 UTF-16 encoding units
	base := filepath.Base(localPath)
	if len(base) > 255 {
		log.Warnf("too long filename: %s", base)
		base = base[len(base)-255:]
		log.Warnf("trimming to: %s", base)
		localPath = filepath.Join(filepath.Dir(localPath), base)
	}

	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("Failed to open the local cache file for write: %s", err)
	}
	defer func() {
		if err := out.Close(); err != nil {
			log.Printf("Failed to close Archive download file (%s): %s", localPath, err)
		}
	}()

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Failed to create cache download request: %s", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Failed to close Archive download response body: %s", err)
		}
	}()

	if resp.StatusCode != 200 {
		return fmt.Errorf("Failed to download archive - non success response code: %d", resp.StatusCode)
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("Failed to save cache content into file: %s", err)
	}

	return nil
}

func uploadFile(uploadURL string, archiveFilePath string) error {
	archFile, err := os.Open(archiveFilePath)
	if err != nil {
		return fmt.Errorf("Failed to open archive file for upload (%s): %s", archiveFilePath, err)
	}
	isFileCloseRequired := true
	defer func() {
		if !isFileCloseRequired {
			return
		}
		if err := archFile.Close(); err != nil {
			log.Printf(" (!) Failed to close archive file (%s): %s", archiveFilePath, err)
		}
	}()

	fileInfo, err := archFile.Stat()
	if err != nil {
		return fmt.Errorf("Failed to get File Stats of the Archive file (%s): %s", archiveFilePath, err)
	}
	fileSize := fileInfo.Size()

	req, err := http.NewRequest("PUT", uploadURL, archFile)
	if err != nil {
		return fmt.Errorf("Failed to create upload request: %s", err)
	}

	req.Header.Add("Content-Length", strconv.FormatInt(fileSize, 10))
	req.ContentLength = fileSize

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Failed to upload: %s", err)
	}
	isFileCloseRequired = false
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf(" [!] Failed to close response body: %s", err)
		}
	}()

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to read response: %s", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Failed to upload file, response code was: %d", resp.StatusCode)
	}

	return nil
}

func createDimensions(step toolresults.Step) map[string]string {
	dimensions := map[string]string{}
	for _, dimension := range step.DimensionValue {
		dimensions[dimension.Key] = dimension.Value
	}
	return dimensions
}

func failf(f string, v ...interface{}) {
	log.Errorf(f, v...)
	os.Exit(1)
}
