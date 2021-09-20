package maintenance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/pkg/errors"
)

func TestDeviceList(t *testing.T) {
	signedIn, err := checkAccounts()
	if err != nil {
		t.Errorf("%s", err)
	}

	if !signedIn {
		if err := signIn(); err != nil {
			t.Errorf("%s", err)
		}
	}

	if err := checkDeviceList(); err != nil {
		t.Error(err)
	}
}

func checkDeviceList() error {
	cmd := command.New("gcloud", "firebase", "test", "ios", "models", "list", "--format", "text")

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return errors.Wrap(err, out)
	}

	if out == deviceList {
		return nil
	}

	cmd = command.New("gcloud", "firebase", "test", "ios", "models", "list")

	outFormatted, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return errors.Wrap(err, out)
	}

	fmt.Println("Fresh devices list to use in this maintenance test:")
	fmt.Println(out)
	fmt.Println()
	fmt.Println("Fresh device list to use in the step's descriptor:")
	fmt.Println(outFormatted)

	return fmt.Errorf("device list has changed, update the corresponding step descriptor blocks")
}

func signIn() error {
	tmpDir, err := pathutil.NormalizedOSTempDirPath("_serv_acc_")
	if err != nil {
		return err
	}

	servAccFileContent := os.Getenv("SERVICE_ACCOUNT_JSON")
	if servAccFileContent == "" {
		return fmt.Errorf("$SERVICE_ACCOUNT_JSON is not set")
	}

	servAccFilePAth := filepath.Join(tmpDir, "serv-acc.json")
	if err := fileutil.WriteStringToFile(servAccFilePAth, servAccFileContent); err != nil {
		return err
	}

	var servAcc struct {
		ProjectID string `json:"project_id"`
	}

	if err := json.NewDecoder(strings.NewReader(servAccFileContent)).Decode(&servAcc); err != nil {
		return err
	}

	if servAcc.ProjectID == "" {
		return fmt.Errorf("invalid service account json, no project_id found")
	}

	cmd := command.New("gcloud",
		"auth",
		"activate-service-account",
		fmt.Sprintf("--key-file=%s", servAccFilePAth),
		"--project", servAcc.ProjectID)

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()

	return errors.Wrap(err, out)
}

func checkAccounts() (bool, error) {
	cmd := command.New("gcloud", "auth", "list", "--format", "json")

	out, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return false, err
	}

	var accounts []interface{}
	if err := json.NewDecoder(strings.NewReader(out)).Decode(&accounts); err != nil {
		return false, err
	}

	return len(accounts) > 0, nil
}

const deviceList = `---
deviceCapabilities[0]:  accelerometer
deviceCapabilities[1]:  arm64
deviceCapabilities[2]:  armv6
deviceCapabilities[3]:  armv7
deviceCapabilities[4]:  auto-focus-camera
deviceCapabilities[5]:  bluetooth-le
deviceCapabilities[6]:  front-facing-camera
deviceCapabilities[7]:  gamekit
deviceCapabilities[8]:  gyroscope
deviceCapabilities[9]:  location-services
deviceCapabilities[10]: magnetometer
deviceCapabilities[11]: metal
deviceCapabilities[12]: microphone
deviceCapabilities[13]: opengles-1
deviceCapabilities[14]: opengles-2
deviceCapabilities[15]: opengles-3
deviceCapabilities[16]: peer-peer
deviceCapabilities[17]: still-camera
deviceCapabilities[18]: video-camera
deviceCapabilities[19]: wifi
deviceCapabilities[20]: arkit
formFactor:             TABLET
id:                     ipad5
name:                   iPad (5th generation)
screenDensity:          264
screenX:                1536
screenY:                2048
supportedVersionIds[0]: 14.1
tags[0]:                deprecated=14.1
---
deviceCapabilities[0]:  accelerometer
deviceCapabilities[1]:  arm64
deviceCapabilities[2]:  armv6
deviceCapabilities[3]:  armv7
deviceCapabilities[4]:  auto-focus-camera
deviceCapabilities[5]:  bluetooth-le
deviceCapabilities[6]:  front-facing-camera
deviceCapabilities[7]:  gamekit
deviceCapabilities[8]:  gyroscope
deviceCapabilities[9]:  location-services
deviceCapabilities[10]: magnetometer
deviceCapabilities[11]: metal
deviceCapabilities[12]: microphone
deviceCapabilities[13]: opengles-1
deviceCapabilities[14]: opengles-2
deviceCapabilities[15]: opengles-3
deviceCapabilities[16]: peer-peer
deviceCapabilities[17]: still-camera
deviceCapabilities[18]: video-camera
deviceCapabilities[19]: wifi
formFactor:             TABLET
id:                     ipadmini4
name:                   iPad mini 4
screenDensity:          326
screenX:                1536
screenY:                2048
supportedVersionIds[0]: 14.1
tags[0]:                deprecated=14.1
---
deviceCapabilities[0]:  accelerometer
deviceCapabilities[1]:  arm64
deviceCapabilities[2]:  armv6
deviceCapabilities[3]:  armv7
deviceCapabilities[4]:  auto-focus-camera
deviceCapabilities[5]:  bluetooth-le
deviceCapabilities[6]:  front-facing-camera
deviceCapabilities[7]:  gamekit
deviceCapabilities[8]:  gyroscope
deviceCapabilities[9]:  location-services
deviceCapabilities[10]: magnetometer
deviceCapabilities[11]: metal
deviceCapabilities[12]: microphone
deviceCapabilities[13]: opengles-1
deviceCapabilities[14]: opengles-2
deviceCapabilities[15]: opengles-3
deviceCapabilities[16]: peer-peer
deviceCapabilities[17]: still-camera
deviceCapabilities[18]: video-camera
deviceCapabilities[19]: wifi
deviceCapabilities[20]: arkit
deviceCapabilities[21]: camera-flash
deviceCapabilities[22]: gps
deviceCapabilities[23]: healthkit
deviceCapabilities[24]: nfc
deviceCapabilities[25]: sms
deviceCapabilities[26]: telephony
formFactor:             PHONE
id:                     iphone11
name:                   iPhone 11
screenDensity:          326
screenX:                828
screenY:                1792
supportedVersionIds[0]: 13.3
supportedVersionIds[1]: 13.6
---
deviceCapabilities[0]:  accelerometer
deviceCapabilities[1]:  arm64
deviceCapabilities[2]:  armv6
deviceCapabilities[3]:  armv7
deviceCapabilities[4]:  auto-focus-camera
deviceCapabilities[5]:  bluetooth-le
deviceCapabilities[6]:  front-facing-camera
deviceCapabilities[7]:  gamekit
deviceCapabilities[8]:  gyroscope
deviceCapabilities[9]:  location-services
deviceCapabilities[10]: magnetometer
deviceCapabilities[11]: metal
deviceCapabilities[12]: microphone
deviceCapabilities[13]: opengles-1
deviceCapabilities[14]: opengles-2
deviceCapabilities[15]: opengles-3
deviceCapabilities[16]: peer-peer
deviceCapabilities[17]: still-camera
deviceCapabilities[18]: video-camera
deviceCapabilities[19]: wifi
deviceCapabilities[20]: arkit
deviceCapabilities[21]: camera-flash
deviceCapabilities[22]: gps
deviceCapabilities[23]: healthkit
deviceCapabilities[24]: nfc
deviceCapabilities[25]: sms
deviceCapabilities[26]: telephony
formFactor:             PHONE
id:                     iphone11pro
name:                   iPhone 11 Pro
screenDensity:          458
screenX:                1125
screenY:                2436
supportedVersionIds[0]: 13.3
supportedVersionIds[1]: 14.1
supportedVersionIds[2]: 14.7
tags[0]:                deprecated=14.1
---
deviceCapabilities[0]:  accelerometer
deviceCapabilities[1]:  arm64
deviceCapabilities[2]:  armv6
deviceCapabilities[3]:  armv7
deviceCapabilities[4]:  auto-focus-camera
deviceCapabilities[5]:  bluetooth-le
deviceCapabilities[6]:  front-facing-camera
deviceCapabilities[7]:  gamekit
deviceCapabilities[8]:  gyroscope
deviceCapabilities[9]:  location-services
deviceCapabilities[10]: magnetometer
deviceCapabilities[11]: metal
deviceCapabilities[12]: microphone
deviceCapabilities[13]: opengles-1
deviceCapabilities[14]: opengles-2
deviceCapabilities[15]: opengles-3
deviceCapabilities[16]: peer-peer
deviceCapabilities[17]: still-camera
deviceCapabilities[18]: video-camera
deviceCapabilities[19]: wifi
deviceCapabilities[20]: arkit
deviceCapabilities[21]: camera-flash
deviceCapabilities[22]: gps
deviceCapabilities[23]: healthkit
deviceCapabilities[24]: nfc
deviceCapabilities[25]: sms
deviceCapabilities[26]: telephony
formFactor:             PHONE
id:                     iphone7plus
name:                   iPhone 7 Plus
screenDensity:          401
screenX:                1080
screenY:                1920
supportedVersionIds[0]: 14.1
tags[0]:                deprecated=14.1
---
deviceCapabilities[0]:  accelerometer
deviceCapabilities[1]:  arm64
deviceCapabilities[2]:  armv6
deviceCapabilities[3]:  armv7
deviceCapabilities[4]:  auto-focus-camera
deviceCapabilities[5]:  bluetooth-le
deviceCapabilities[6]:  front-facing-camera
deviceCapabilities[7]:  gamekit
deviceCapabilities[8]:  gyroscope
deviceCapabilities[9]:  location-services
deviceCapabilities[10]: magnetometer
deviceCapabilities[11]: metal
deviceCapabilities[12]: microphone
deviceCapabilities[13]: opengles-1
deviceCapabilities[14]: opengles-2
deviceCapabilities[15]: opengles-3
deviceCapabilities[16]: peer-peer
deviceCapabilities[17]: still-camera
deviceCapabilities[18]: video-camera
deviceCapabilities[19]: wifi
deviceCapabilities[20]: arkit
deviceCapabilities[21]: camera-flash
deviceCapabilities[22]: gps
deviceCapabilities[23]: healthkit
deviceCapabilities[24]: nfc
deviceCapabilities[25]: sms
deviceCapabilities[26]: telephony
formFactor:             PHONE
id:                     iphone8
name:                   iPhone 8
screenDensity:          326
screenX:                750
screenY:                1334
supportedVersionIds[0]: 11.4
supportedVersionIds[1]: 12.0
supportedVersionIds[2]: 12.4
supportedVersionIds[3]: 13.6
supportedVersionIds[4]: 14.1
supportedVersionIds[5]: 14.7
tags[0]:                deprecated=14.1
tags[1]:                default
---
deviceCapabilities[0]:  accelerometer
deviceCapabilities[1]:  arm64
deviceCapabilities[2]:  armv6
deviceCapabilities[3]:  armv7
deviceCapabilities[4]:  auto-focus-camera
deviceCapabilities[5]:  bluetooth-le
deviceCapabilities[6]:  front-facing-camera
deviceCapabilities[7]:  gamekit
deviceCapabilities[8]:  gyroscope
deviceCapabilities[9]:  location-services
deviceCapabilities[10]: magnetometer
deviceCapabilities[11]: metal
deviceCapabilities[12]: microphone
deviceCapabilities[13]: opengles-1
deviceCapabilities[14]: opengles-2
deviceCapabilities[15]: opengles-3
deviceCapabilities[16]: peer-peer
deviceCapabilities[17]: still-camera
deviceCapabilities[18]: video-camera
deviceCapabilities[19]: wifi
deviceCapabilities[20]: arkit
deviceCapabilities[21]: camera-flash
deviceCapabilities[22]: gps
deviceCapabilities[23]: healthkit
deviceCapabilities[24]: nfc
deviceCapabilities[25]: sms
deviceCapabilities[26]: telephony
formFactor:             PHONE
id:                     iphone8plus
name:                   iPhone 8 Plus
screenDensity:          401
screenX:                1080
screenY:                1920
supportedVersionIds[0]: 12.0
---
deviceCapabilities[0]:  accelerometer
deviceCapabilities[1]:  arm64
deviceCapabilities[2]:  armv6
deviceCapabilities[3]:  armv7
deviceCapabilities[4]:  auto-focus-camera
deviceCapabilities[5]:  bluetooth-le
deviceCapabilities[6]:  front-facing-camera
deviceCapabilities[7]:  gamekit
deviceCapabilities[8]:  gyroscope
deviceCapabilities[9]:  location-services
deviceCapabilities[10]: magnetometer
deviceCapabilities[11]: metal
deviceCapabilities[12]: microphone
deviceCapabilities[13]: opengles-1
deviceCapabilities[14]: opengles-2
deviceCapabilities[15]: opengles-3
deviceCapabilities[16]: peer-peer
deviceCapabilities[17]: still-camera
deviceCapabilities[18]: video-camera
deviceCapabilities[19]: wifi
deviceCapabilities[20]: arkit
deviceCapabilities[21]: camera-flash
deviceCapabilities[22]: gps
deviceCapabilities[23]: healthkit
deviceCapabilities[24]: nfc
deviceCapabilities[25]: sms
deviceCapabilities[26]: telephony
formFactor:             PHONE
id:                     iphonexr
name:                   iPhone XR
screenDensity:          326
screenX:                828
screenY:                1792
supportedVersionIds[0]: 12.4
supportedVersionIds[1]: 13.2
---
deviceCapabilities[0]:  accelerometer
deviceCapabilities[1]:  arm64
deviceCapabilities[2]:  armv6
deviceCapabilities[3]:  armv7
deviceCapabilities[4]:  auto-focus-camera
deviceCapabilities[5]:  bluetooth-le
deviceCapabilities[6]:  front-facing-camera
deviceCapabilities[7]:  gamekit
deviceCapabilities[8]:  gyroscope
deviceCapabilities[9]:  location-services
deviceCapabilities[10]: magnetometer
deviceCapabilities[11]: metal
deviceCapabilities[12]: microphone
deviceCapabilities[13]: opengles-1
deviceCapabilities[14]: opengles-2
deviceCapabilities[15]: opengles-3
deviceCapabilities[16]: peer-peer
deviceCapabilities[17]: still-camera
deviceCapabilities[18]: video-camera
deviceCapabilities[19]: wifi
deviceCapabilities[20]: arkit
deviceCapabilities[21]: camera-flash
deviceCapabilities[22]: gps
deviceCapabilities[23]: healthkit
deviceCapabilities[24]: nfc
deviceCapabilities[25]: sms
deviceCapabilities[26]: telephony
formFactor:             PHONE
id:                     iphonexsmax
name:                   iPhone XS Max
screenDensity:          458
screenX:                1242
screenY:                2688
supportedVersionIds[0]: 12.1`
