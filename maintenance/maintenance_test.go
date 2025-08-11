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
		return fmt.Errorf("out: %s, err: %w", out, err)
	}

	if out == deviceList {
		return nil
	}

	cmd = command.New("gcloud", "firebase", "test", "ios", "models", "list")

	outFormatted, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("out: %s, err: %w", out, err)
	}

	// Your gcloud sdk version must be 417.0.0 or greater for this command to succeed.
	cmd = command.New("gcloud", "firebase", "test", "ios", "list-device-capacities")

	capacityFormatted, err := cmd.RunAndReturnTrimmedCombinedOutput()
	if err != nil {
		return fmt.Errorf("out: %s, err: %w", out, err)
	}

	fmt.Println("Fresh devices list to use in this maintenance test:")
	fmt.Println(out)
	fmt.Println()
	fmt.Println("Fresh device list to use in the step's descriptor:")
	fmt.Println("Available devices and its versions:")
	fmt.Println(outFormatted)
	fmt.Println()
	fmt.Println("Device and OS Capacity:")
	fmt.Println(capacityFormatted)

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
	if err != nil {
		return fmt.Errorf("out: %s, err: %w", out, err)
	}

	return nil
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
deviceCapabilities[0]:            accelerometer
deviceCapabilities[1]:            arm64
deviceCapabilities[2]:            armv6
deviceCapabilities[3]:            armv7
deviceCapabilities[4]:            auto-focus-camera
deviceCapabilities[5]:            bluetooth-le
deviceCapabilities[6]:            front-facing-camera
deviceCapabilities[7]:            gamekit
deviceCapabilities[8]:            gyroscope
deviceCapabilities[9]:            location-services
deviceCapabilities[10]:           magnetometer
deviceCapabilities[11]:           metal
deviceCapabilities[12]:           microphone
deviceCapabilities[13]:           opengles-1
deviceCapabilities[14]:           opengles-2
deviceCapabilities[15]:           opengles-3
deviceCapabilities[16]:           peer-peer
deviceCapabilities[17]:           still-camera
deviceCapabilities[18]:           video-camera
deviceCapabilities[19]:           wifi
deviceCapabilities[20]:           arkit
formFactor:                       TABLET
id:                               ipad10
name:                             iPad (10th generation)
perVersionInfo[0].deviceCapacity: DEVICE_CAPACITY_LOW
perVersionInfo[0].versionId:      16.6
screenDensity:                    264
screenX:                          1640
screenY:                          2360
supportedVersionIds[0]:           16.6
---
deviceCapabilities[0]:            accelerometer
deviceCapabilities[1]:            arm64
deviceCapabilities[2]:            armv6
deviceCapabilities[3]:            armv7
deviceCapabilities[4]:            auto-focus-camera
deviceCapabilities[5]:            bluetooth-le
deviceCapabilities[6]:            front-facing-camera
deviceCapabilities[7]:            gamekit
deviceCapabilities[8]:            gyroscope
deviceCapabilities[9]:            location-services
deviceCapabilities[10]:           magnetometer
deviceCapabilities[11]:           metal
deviceCapabilities[12]:           microphone
deviceCapabilities[13]:           opengles-1
deviceCapabilities[14]:           opengles-2
deviceCapabilities[15]:           opengles-3
deviceCapabilities[16]:           peer-peer
deviceCapabilities[17]:           still-camera
deviceCapabilities[18]:           video-camera
deviceCapabilities[19]:           wifi
deviceCapabilities[20]:           arkit
deviceCapabilities[21]:           camera-flash
deviceCapabilities[22]:           gps
deviceCapabilities[23]:           healthkit
deviceCapabilities[24]:           nfc
deviceCapabilities[25]:           sms
deviceCapabilities[26]:           telephony
formFactor:                       PHONE
id:                               iphone11pro
name:                             iPhone 11 Pro
perVersionInfo[0].deviceCapacity: DEVICE_CAPACITY_HIGH
perVersionInfo[0].versionId:      16.6
screenDensity:                    458
screenX:                          1125
screenY:                          2436
supportedVersionIds[0]:           16.6
---
deviceCapabilities[0]:            accelerometer
deviceCapabilities[1]:            arm64
deviceCapabilities[2]:            armv6
deviceCapabilities[3]:            armv7
deviceCapabilities[4]:            auto-focus-camera
deviceCapabilities[5]:            bluetooth-le
deviceCapabilities[6]:            front-facing-camera
deviceCapabilities[7]:            gamekit
deviceCapabilities[8]:            gyroscope
deviceCapabilities[9]:            location-services
deviceCapabilities[10]:           magnetometer
deviceCapabilities[11]:           metal
deviceCapabilities[12]:           microphone
deviceCapabilities[13]:           opengles-1
deviceCapabilities[14]:           opengles-2
deviceCapabilities[15]:           opengles-3
deviceCapabilities[16]:           peer-peer
deviceCapabilities[17]:           still-camera
deviceCapabilities[18]:           video-camera
deviceCapabilities[19]:           wifi
deviceCapabilities[20]:           arkit
deviceCapabilities[21]:           camera-flash
deviceCapabilities[22]:           gps
deviceCapabilities[23]:           healthkit
deviceCapabilities[24]:           nfc
deviceCapabilities[25]:           sms
deviceCapabilities[26]:           telephony
formFactor:                       PHONE
id:                               iphone13pro
name:                             iPhone 13 Pro
perVersionInfo[0].deviceCapacity: DEVICE_CAPACITY_HIGH
perVersionInfo[0].versionId:      15.7
perVersionInfo[1].deviceCapacity: DEVICE_CAPACITY_HIGH
perVersionInfo[1].versionId:      16.6
screenDensity:                    460
screenX:                          1170
screenY:                          2532
supportedVersionIds[0]:           15.7
supportedVersionIds[1]:           16.6
tags[0]:                          default
---
deviceCapabilities[0]:            accelerometer
deviceCapabilities[1]:            arm64
deviceCapabilities[2]:            armv6
deviceCapabilities[3]:            armv7
deviceCapabilities[4]:            auto-focus-camera
deviceCapabilities[5]:            bluetooth-le
deviceCapabilities[6]:            front-facing-camera
deviceCapabilities[7]:            gamekit
deviceCapabilities[8]:            gyroscope
deviceCapabilities[9]:            location-services
deviceCapabilities[10]:           magnetometer
deviceCapabilities[11]:           metal
deviceCapabilities[12]:           microphone
deviceCapabilities[13]:           opengles-1
deviceCapabilities[14]:           opengles-2
deviceCapabilities[15]:           opengles-3
deviceCapabilities[16]:           peer-peer
deviceCapabilities[17]:           still-camera
deviceCapabilities[18]:           video-camera
deviceCapabilities[19]:           wifi
deviceCapabilities[20]:           arkit
deviceCapabilities[21]:           camera-flash
deviceCapabilities[22]:           gps
deviceCapabilities[23]:           healthkit
deviceCapabilities[24]:           nfc
deviceCapabilities[25]:           sms
deviceCapabilities[26]:           telephony
formFactor:                       PHONE
id:                               iphone14pro
name:                             iPhone 14 Pro
perVersionInfo[0].deviceCapacity: DEVICE_CAPACITY_HIGH
perVersionInfo[0].versionId:      16.6
screenDensity:                    460
screenX:                          1179
screenY:                          2556
supportedVersionIds[0]:           16.6
---
deviceCapabilities[0]:            accelerometer
deviceCapabilities[1]:            arm64
deviceCapabilities[2]:            armv6
deviceCapabilities[3]:            armv7
deviceCapabilities[4]:            auto-focus-camera
deviceCapabilities[5]:            bluetooth-le
deviceCapabilities[6]:            front-facing-camera
deviceCapabilities[7]:            gamekit
deviceCapabilities[8]:            gyroscope
deviceCapabilities[9]:            location-services
deviceCapabilities[10]:           magnetometer
deviceCapabilities[11]:           metal
deviceCapabilities[12]:           microphone
deviceCapabilities[13]:           opengles-1
deviceCapabilities[14]:           opengles-2
deviceCapabilities[15]:           opengles-3
deviceCapabilities[16]:           peer-peer
deviceCapabilities[17]:           still-camera
deviceCapabilities[18]:           video-camera
deviceCapabilities[19]:           wifi
deviceCapabilities[20]:           arkit
deviceCapabilities[21]:           camera-flash
deviceCapabilities[22]:           gps
deviceCapabilities[23]:           healthkit
deviceCapabilities[24]:           nfc
deviceCapabilities[25]:           sms
deviceCapabilities[26]:           telephony
formFactor:                       PHONE
id:                               iphone15
name:                             iPhone 15
perVersionInfo[0].deviceCapacity: DEVICE_CAPACITY_MEDIUM
perVersionInfo[0].versionId:      18.0
screenDensity:                    460
screenX:                          1170
screenY:                          2532
supportedVersionIds[0]:           18.0
tags[0]:                          preview=18.0
---
deviceCapabilities[0]:            accelerometer
deviceCapabilities[1]:            arm64
deviceCapabilities[2]:            armv6
deviceCapabilities[3]:            armv7
deviceCapabilities[4]:            auto-focus-camera
deviceCapabilities[5]:            bluetooth-le
deviceCapabilities[6]:            front-facing-camera
deviceCapabilities[7]:            gamekit
deviceCapabilities[8]:            gyroscope
deviceCapabilities[9]:            location-services
deviceCapabilities[10]:           magnetometer
deviceCapabilities[11]:           metal
deviceCapabilities[12]:           microphone
deviceCapabilities[13]:           opengles-1
deviceCapabilities[14]:           opengles-2
deviceCapabilities[15]:           opengles-3
deviceCapabilities[16]:           peer-peer
deviceCapabilities[17]:           still-camera
deviceCapabilities[18]:           video-camera
deviceCapabilities[19]:           wifi
deviceCapabilities[20]:           arkit
deviceCapabilities[21]:           camera-flash
deviceCapabilities[22]:           gps
deviceCapabilities[23]:           healthkit
deviceCapabilities[24]:           nfc
deviceCapabilities[25]:           sms
deviceCapabilities[26]:           telephony
formFactor:                       PHONE
id:                               iphone15pro
name:                             iPhone 15 Pro
perVersionInfo[0].deviceCapacity: DEVICE_CAPACITY_MEDIUM
perVersionInfo[0].versionId:      18.0
screenDensity:                    460
screenX:                          1179
screenY:                          2556
supportedVersionIds[0]:           18.0
tags[0]:                          preview=18.0
---
deviceCapabilities[0]:            accelerometer
deviceCapabilities[1]:            arm64
deviceCapabilities[2]:            armv6
deviceCapabilities[3]:            armv7
deviceCapabilities[4]:            auto-focus-camera
deviceCapabilities[5]:            bluetooth-le
deviceCapabilities[6]:            front-facing-camera
deviceCapabilities[7]:            gamekit
deviceCapabilities[8]:            gyroscope
deviceCapabilities[9]:            location-services
deviceCapabilities[10]:           magnetometer
deviceCapabilities[11]:           metal
deviceCapabilities[12]:           microphone
deviceCapabilities[13]:           opengles-1
deviceCapabilities[14]:           opengles-2
deviceCapabilities[15]:           opengles-3
deviceCapabilities[16]:           peer-peer
deviceCapabilities[17]:           still-camera
deviceCapabilities[18]:           video-camera
deviceCapabilities[19]:           wifi
deviceCapabilities[20]:           arkit
deviceCapabilities[21]:           camera-flash
deviceCapabilities[22]:           gps
deviceCapabilities[23]:           healthkit
deviceCapabilities[24]:           nfc
deviceCapabilities[25]:           sms
deviceCapabilities[26]:           telephony
formFactor:                       PHONE
id:                               iphone8
name:                             iPhone 8
perVersionInfo[0].deviceCapacity: DEVICE_CAPACITY_MEDIUM
perVersionInfo[0].versionId:      15.7
perVersionInfo[1].deviceCapacity: DEVICE_CAPACITY_HIGH
perVersionInfo[1].versionId:      16.6
screenDensity:                    326
screenX:                          750
screenY:                          1334
supportedVersionIds[0]:           15.7
supportedVersionIds[1]:           16.6`
