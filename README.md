# iOS Device Testing

[![Step changelog](https://shields.io/github/v/release/bitrise-steplib/steps-virtual-device-testing-for-ios?include_prereleases&label=changelog&color=blueviolet)](https://github.com/bitrise-steplib/steps-virtual-device-testing-for-ios/releases)

Run iOS XCUITests on devices

<details>
<summary>Description</summary>

Run iOS XCUItests on physical devices with Google's Firebase Test Lab. You do not have to set up and register your own devices and you don't need your own Firebase account.

We'll go over the most important configuration information for the Step. For more information, check out our [detailed guide](https://devcenter.bitrise.io/en/testing/device-testing-for-ios.html).

### Configuring the Step

To use the Step, you need to build an IPA file with Xcode's `build-for-testing` action. You can use our dedicated Step:

1. Add the **Xcode Build for testing for iOS** Step to your Workflow.

   The Step exports a .zip file that contains your test directory (by default, it’s `Debug-iphoneos`) and the xctestrun file.
1. Add the **iOS Device Testing** Step to the Workflow.
1. In the **Test devices** input field, specify the devices on which you want to test the app. The input's description lists the devices and OS versions that were available on 2026-07-24 — see **Checking the current device list** below if you need an up-to-date list.
1. Optionally, you can set a test timeout and configure file download in the **Debug** input group. The path to the downloaded files will be exported as an Environment Variable and it can be used by subsequent Steps.
1. Make sure you have the **Deploy to Bitrise.io** Step in your Workflow, with version 1.4.1 or newer. With the older versions of the Step, you won’t be able to check your results on the Test Reports page!

Please note you should not change the value of the **API token** and the **Test API's base URL** input.

### Checking the current device list

The device tables in the **Test devices** input description were generated on **2026-07-24**. Google regularly adds and retires Firebase Test Lab devices, so they can go out of date. Three ways to check the current catalog:

- **Firebase console** — open [Test Lab](https://console.firebase.google.com/project/_/testlab) and start a test run; the available devices are listed in the **Select dimensions** step.
- **Google APIs Explorer** — no CLI and no Firebase project needed: call [`testEnvironmentCatalog.get` with `environmentType=IOS`](https://developers.google.com/apis-explorer/#p/testing/v1/testing.testEnvironmentCatalog.get?environmentType=IOS).
- **gcloud CLI** — install the [Google Cloud CLI](https://cloud.google.com/sdk/docs/install) (version 417.0.0 or newer), run `gcloud auth login`, set a Google Cloud project, then:

  ```
  gcloud firebase test ios models list
  gcloud firebase test ios list-device-capacities
  ```

### Troubleshooting

If you get the **Build already exists** error, it is because you have more than one instance of the Step in your Workflow. This doesn't work as Bitrise sends the build slug to Firebase and having the Step more than once in the same Workflow results in sending the same build slug multiple times.

### Useful links

[Device testing for iOS](https://devcenter.bitrise.io/en/testing/device-testing-for-ios.html)

### Related Steps

[Xcode Build for testing for iOS](https://www.bitrise.io/integrations/steps/xcode-build-for-test)
[Xcode Test for iOS](https://www.bitrise.io/integrations/steps/xcode-test)
</details>

## 🧩 Get started

Add this step directly to your workflow in the [Bitrise Workflow Editor](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/steps/adding-steps-to-a-workflow.html).

You can also run this step directly with [Bitrise CLI](https://github.com/bitrise-io/bitrise).

## ⚙️ Configuration

<details>
<summary>Inputs</summary>

| Key | Description | Flags | Default |
| --- | --- | --- | --- |
| `zip_path` | Open finder, and navigate to the directory you designated for Derived Data output. Open the folder for your project, then the Build/Products folders inside it. You should see a folder Debug-iphoneos and PROJECT_NAME_iphoneos_DEVELOPMENT_TARGET-arm64.xctestrun. Select them both, then right-click on one of them and select Compress 2 items.  | required | `$BITRISE_TEST_BUNDLE_ZIP_PATH` |
| `test_devices` | Format: One device configuration per line and the parameters are separated with `,` in the order of: `deviceID,version,language,orientation` For example: `iphone8,16.6,en,portrait` `iphone8,16.6,en,landscape` Available devices, OS versions and their capacity (generated on 2026-07-27): ``` ┌─────────────┬────────────────────────┬───────────────┬─────────────────┬─────────┐ │   MODEL_ID  │       MODEL_NAME       │ OS_VERSION_ID │ DEVICE_CAPACITY │   TAGS  │ ├─────────────┼────────────────────────┼───────────────┼─────────────────┼─────────┤ │ ipad10      │ iPad (10th generation) │ 16.6          │ Medium          │         │ │ iphone11pro │ iPhone 11 Pro          │ 16.6          │ Medium          │         │ │ iphone14pro │ iPhone 14 Pro          │ 16.6          │ Medium          │ default │ │ iphone16pro │ iPhone 16 Pro          │ 18.3          │ Medium          │         │ │ iphone8     │ iPhone 8               │ 16.6          │ Medium          │         │ │ iphonese3   │ iPhone SE 3            │ 18.4          │ Medium          │         │ │ iphonese3   │ iPhone SE 3            │ 26.3          │ Medium          │         │ └─────────────┴────────────────────────┴───────────────┴─────────────────┴─────────┘ ```  **Video recording:** Firebase Test Lab does not return videos for iOS 18 and newer, so runs on `iphone16pro,18.3`, `iphonese3,18.4` and `iphonese3,26.3` produce screenshots but no video. The 16.6 devices above are currently the only ones that return video. This is a Firebase limitation, not a Step one — see [Available devices in Test Lab](https://firebase.google.com/docs/test-lab/ios/available-testing-devices).  Google regularly adds and retires Firebase Test Lab devices, so the table above can go out of date. There are three ways to check the current catalog:  - **Firebase console** (no CLI needed): open [Test Lab](https://console.firebase.google.com/project/_/testlab) and start a test run — the available devices are shown in the **Select dimensions** step. - **Google APIs Explorer** (no CLI and no Firebase project needed): call [`testEnvironmentCatalog.get` with `environmentType=IOS`](https://developers.google.com/apis-explorer/#p/testing/v1/testing.testEnvironmentCatalog.get?environmentType=IOS). - **gcloud CLI** locally, version 417.0.0 or newer, signed in with `gcloud auth login` and a Google Cloud project set:   ```   gcloud firebase test ios models list   gcloud firebase test ios list-device-capacities   ```  | required | `iphone16pro,18.3,en,portrait` |
| `num_flaky_test_attempts` | Specifies the number of times a test execution should be reattempted if one or more of its test cases fail for any reason.  An execution that initially fails but succeeds on any reattempt is reported as FLAKY. The maximum number of reruns allowed is 10. (Default: 0, which implies no reruns.) | required | `0` |
| `test_timeout` | Max time a test execution is allowed to run before it is automatically canceled. The default value is 900 (15 min).  Duration in seconds with up to nine fractional digits. Example: "3.5".  |  | `900` |
| `download_test_results` | If this input is set to `true` all files generated in the test run will be downloaded. Otherwise, no any file will be downloaded.  | required | `false` |
| `api_base_url` | The URL where test API is accessible.  | required | `https://vdt.bitrise.io/test` |
| `api_token` | The token required to authenticate with the API.  | required, sensitive | `$ADDON_VDTESTING_API_TOKEN` |
| `quarantined_tests` | JSON list of tests added to quarantine on Bitrise.io, quarantined tests are excluded from test runs. |  | `$BITRISE_QUARANTINED_TESTS_JSON` |
</details>

<details>
<summary>Outputs</summary>

| Environment Variable | Description |
| --- | --- |
| `VDTESTING_DOWNLOADED_FILES_DIR` | The directory containing the downloaded files if you have set `download_test_results` inputs above. |
| `BITRISE_FLAKY_TEST_CASES` | A list of flaky test cases. A test case is considered flaky if it has failed at least once, but passed at least once as well.  The list contains the test cases in the following format: ``` - TestSuit_1.TestClass_1.TestName_1 - TestSuit_1.TestClass_1.TestName_2 - TestSuit_1.TestClass_2.TestName_1 - TestSuit_2.TestClass_1.TestName_1 ... ```  To export `BITRISE_FLAKY_TEST_CASES` Step Output `download_test_results` Step Input should be set to `true`. |
</details>

## 🙋 Contributing

We welcome [pull requests](https://github.com/bitrise-steplib/steps-virtual-device-testing-for-ios/pulls) and [issues](https://github.com/bitrise-steplib/steps-virtual-device-testing-for-ios/issues) against this repository.

For pull requests, work on your changes in a forked repository and use the Bitrise CLI to [run step tests locally](https://docs.bitrise.io/en/bitrise-ci/bitrise-cli/running-your-first-local-build-with-the-cli.html).

Learn more about developing steps:

- [Create your own step](https://docs.bitrise.io/en/bitrise-ci/workflows-and-pipelines/developing-your-own-bitrise-step/developing-a-new-step.html)
