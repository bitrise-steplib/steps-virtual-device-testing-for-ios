package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/go-utils/sliceutil"
	"github.com/bitrise-tools/go-steputils/input"
	"github.com/bitrise-tools/go-steputils/tools"
	testing "google.golang.org/api/testing/v1"
	toolresults "google.golang.org/api/toolresults/v1beta3"
)

// ConfigsModel ...
type ConfigsModel struct {
	// api
	APIBaseURL string
	BuildSlug  string
	AppSlug    string
	APIToken   string

	// shared
	TestDevices          string
	TestTimeout          string
	DownloadTestResults  string
	DirectoriesToPull    string
	EnvironmentVariables string
	ZipPath              string

	// loop
	LoopScenarios      string
	LoopScenarioLabels string
}

// UploadURLRequest ...
type UploadURLRequest struct {
	AppURL     string `json:"appUrl"`
	TestAppURL string `json:"testAppUrl"`
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		// api
		APIBaseURL: os.Getenv("api_base_url"),
		BuildSlug:  os.Getenv("BITRISE_BUILD_SLUG"),
		AppSlug:    os.Getenv("BITRISE_APP_SLUG"),
		APIToken:   os.Getenv("api_token"),

		ZipPath: os.Getenv("zip_path"),

		// loop
		LoopScenarios:      os.Getenv("loop_scenarios"),
		LoopScenarioLabels: os.Getenv("loop_scenario_labels"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")

	log.Printf("- TestTimeout: %s", configs.TestTimeout)
	log.Printf("- DirectoriesToPull: %s", configs.DirectoriesToPull)
	log.Printf("- EnvironmentVariables: %s", configs.EnvironmentVariables)
	log.Printf("- TestDevices:\n---")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Print(w, "Model\tAPI Level\tLocale\tOrientation\t")
	scanner := bufio.NewScanner(strings.NewReader(configs.TestDevices))
	for scanner.Scan() {
		device := scanner.Text()
		device = strings.TrimSpace(device)
		if device == "" {
			continue
		}

		deviceParams := strings.Split(device, ",")

		if len(deviceParams) != 4 {
			continue
		}

		fmt.Print(w, fmt.Sprintf("%s\t%s\t%s\t%s\t", deviceParams[0], deviceParams[1], deviceParams[3], deviceParams[2]))
	}
	if err := w.Flush(); err != nil {
		log.Errorf("Failed to flush writer, error: %s", err)
	}
}

func (configs ConfigsModel) validate() error {
	if err := input.ValidateIfNotEmpty(configs.APIBaseURL); err != nil {
		if _, set := os.LookupEnv("BITRISE_IO"); !set {
			log.Warnf("Warning: please make sure that Virtual Device Testing add-on is turned on under your app's settings tab.")
		}
		return fmt.Errorf("Issue with APIBaseURL: %s", err)
	}
	if err := input.ValidateIfNotEmpty(configs.APIToken); err != nil {
		return fmt.Errorf("Issue with APIToken: %s", err)
	}
	if err := input.ValidateIfNotEmpty(configs.BuildSlug); err != nil {
		return fmt.Errorf("Issue with BuildSlug: %s", err)
	}
	if err := input.ValidateIfNotEmpty(configs.AppSlug); err != nil {
		return fmt.Errorf("Issue with AppSlug: %s", err)
	}

	return nil
}

func failf(f string, v ...interface{}) {
	log.Errorf(f, v...)
	os.Exit(1)
}

func main() {
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		failf("%s", err)
	}

	fmt.Println()

	successful := true

	log.Infof("Upload XCTestrun")
	{
		url := configs.APIBaseURL + "/assets/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + configs.APIToken

		req, err := http.NewRequest("POST", url, nil)
		if err != nil {
			failf("Failed to create http request, error: %s", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			failf("Failed to get http response, error: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				failf("Failed to read response body, error: %s", err)
			}
			failf("Failed to start test: %d, error: %s", resp.StatusCode, string(body))
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			failf("Failed to read response body, error: %s", err)
		}

		responseModel := &UploadURLRequest{}

		err = json.Unmarshal(body, responseModel)
		if err != nil {
			failf("Failed to unmarshal response body, error: %s", err)
		}

		err = uploadFile(responseModel.AppURL, configs.ZipPath)
		if err != nil {
			failf("Failed to upload file(%s) to (%s), error: %s", configs.ZipPath, responseModel.AppURL, err)
		}

		log.Donef("=> XCTestrun uploaded")
	}

	fmt.Println()
	log.Infof("Start test")
	{
		url := configs.APIBaseURL + "/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + configs.APIToken

		testModel := &testing.TestMatrix{}
		testModel.EnvironmentMatrix = &testing.EnvironmentMatrix{IosDeviceList: &testing.IosDeviceList{}}
		testModel.EnvironmentMatrix.IosDeviceList.IosDevices = []*testing.IosDevice{}

		scanner := bufio.NewScanner(strings.NewReader(configs.TestDevices))
		for scanner.Scan() {
			device := scanner.Text()
			device = strings.TrimSpace(device)
			if device == "" {
				continue
			}

			deviceParams := strings.Split(device, ",")
			if len(deviceParams) != 4 {
				failf("Invalid test device configuration: %s", device)
			}

			newDevice := testing.IosDevice{
				IosModelId:   deviceParams[0],
				IosVersionId: deviceParams[1],
			}

			testModel.EnvironmentMatrix.IosDeviceList.IosDevices = append(testModel.EnvironmentMatrix.IosDeviceList.IosDevices, &newDevice)
		}

		log.Warnf("Devides done;")

		// parse directories to pull
		scanner = bufio.NewScanner(strings.NewReader(configs.DirectoriesToPull))
		directoriesToPull := []string{}
		for scanner.Scan() {
			path := scanner.Text()
			path = strings.TrimSpace(path)
			if path == "" {
				continue
			}
			directoriesToPull = append(directoriesToPull, path)
		}

		// parse environment variables
		scanner = bufio.NewScanner(strings.NewReader(configs.EnvironmentVariables))
		envs := []*testing.EnvironmentVariable{}
		for scanner.Scan() {
			envStr := scanner.Text()

			if envStr == "" {
				continue
			}

			if !strings.Contains(envStr, "=") {
				continue
			}

			envStrSplit := strings.Split(envStr, "=")
			envKey := envStrSplit[0]
			envValue := strings.Join(envStrSplit[1:], "=")

			envs = append(envs, &testing.EnvironmentVariable{Key: envKey, Value: envValue})
		}

		testModel.TestSpecification = &testing.TestSpecification{
			TestTimeout: fmt.Sprintf("%ss", configs.TestTimeout),
			TestSetup: &testing.TestSetup{
				EnvironmentVariables: envs,
				DirectoriesToPull:    directoriesToPull,
			},
		}

		jsonByte, err := json.Marshal(testModel)
		if err != nil {
			failf("Failed to marshal test model, error: %s", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonByte))
		if err != nil {
			failf("Failed to create http request, error: %s", err)
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			failf("Failed to get http response, error: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				failf("Failed to read response body, error: %s", err)
			}
			failf("Failed to start test: %d, error: %s", resp.StatusCode, string(body))
		}

		log.Donef("=> Test started")
	}

	fmt.Println()
	log.Infof("Waiting for test results")
	{
		finished := false
		printedLogs := []string{}
		for !finished {
			url := configs.APIBaseURL + "/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + configs.APIToken

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				failf("Failed to create http request, error: %s", err)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if resp.StatusCode != http.StatusOK || err != nil {
				resp, err = client.Do(req)
				if err != nil {
					failf("Failed to get http response, error: %s", err)
				}
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				failf("Failed to read response body, error: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				failf("Failed to get test status, error: %s", string(body))
			}

			responseModel := &toolresults.ListStepsResponse{}

			err = json.Unmarshal(body, responseModel)
			if err != nil {
				failf("Failed to unmarshal response body, error: %s, body: %s", err, string(body))
			}

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
				msg = fmt.Sprintf("- Validating")
			} else {
				msg = fmt.Sprintf("- (%d/%d) running", testsRunning, len(responseModel.Steps))
			}

			if !sliceutil.IsStringInSlice(msg, printedLogs) {
				log.Printf(msg)
				printedLogs = append(printedLogs, msg)
			}

			if finished {
				log.Donef("=> Test finished")
				fmt.Println()

				log.Infof("Test results:")
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Print(w, "Model\tAPI Level\tLocale\tOrientation\tOutcome\t")

				for _, step := range responseModel.Steps {
					dimensions := map[string]string{}
					for _, dimension := range step.DimensionValue {
						dimensions[dimension.Key] = dimension.Value
					}

					outcome := step.Outcome.Summary

					switch outcome {
					case "success":
						outcome = colorstring.Green(outcome)
					case "failure":
						successful = false
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
						successful = false
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
						successful = false
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

					fmt.Print(w, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t", dimensions["Model"], dimensions["Version"], dimensions["Locale"], dimensions["Orientation"], outcome))
				}
				if err := w.Flush(); err != nil {
					log.Errorf("Failed to flush writer, error: %s", err)
				}
			}
			if !finished {
				time.Sleep(5 * time.Second)
			}
		}
	}

	if configs.DownloadTestResults == "true" {
		fmt.Println()
		log.Infof("Downloading test assets")
		{
			url := configs.APIBaseURL + "/assets/" + configs.AppSlug + "/" + configs.BuildSlug + "/" + configs.APIToken

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				failf("Failed to create http request, error: %s", err)
			}

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				failf("Failed to get http response, error: %s", err)
			}

			if resp.StatusCode != http.StatusOK {
				failf("Failed to get http response, status code: %d", resp.StatusCode)
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				failf("Failed to read response body, error: %s", err)
			}

			responseModel := map[string]string{}

			err = json.Unmarshal(body, &responseModel)
			if err != nil {
				failf("Failed to unmarshal response body, error: %s", err)
			}

			tempDir, err := pathutil.NormalizedOSTempDirPath("vdtesting_test_assets")
			if err != nil {
				failf("Failed to create temp dir, error: %s", err)
			}

			for fileName, fileURL := range responseModel {
				err := downloadFile(fileURL, filepath.Join(tempDir, fileName))
				if err != nil {
					failf("Failed to download file, error: %s", err)
				}
			}

			log.Donef("=> Assets downloaded")
			if err := tools.ExportEnvironmentWithEnvman("VDTESTING_DOWNLOADED_FILES_DIR", tempDir); err != nil {
				log.Warnf("Failed to export environment (VDTESTING_DOWNLOADED_FILES_DIR), error: %s", err)
			} else {
				log.Printf("The downloaded test assets path (%s) is exported to the VDTESTING_DOWNLOADED_FILES_DIR environment variable.", tempDir)
			}
		}
	}

	if !successful {
		os.Exit(1)
	}
}

func downloadFile(url string, localPath string) error {
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

	_, err = io.Copy(out, resp.Body)
	if err != nil {
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

	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("Failed to read response: %s", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("Failed to upload file, response code was: %d", resp.StatusCode)
	}

	return nil
}
