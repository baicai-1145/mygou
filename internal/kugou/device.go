package kugou

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Device is what KuGou sees as the phone that asks: a GUID, the MID derived
// from it, a developer id, and the dfid the risk service hands out.
type Device struct {
	GUID  string `json:"guid"`
	MID   string `json:"mid"`
	Dev   string `json:"dev"`
	MAC   string `json:"mac"`
	WebGL string `json:"webgl"`
	DFID  string `json:"dfid"`
}

func devicePath(dir string) string { return filepath.Join(dir, "device.json") }

// LoadDevice reads dir/device.json, creating a new device on first run.
// The dfid is fetched later, by RegisterDevice.
func LoadDevice(dir string) (*Device, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(devicePath(dir))
	if err == nil {
		var d Device
		if err := json.Unmarshal(data, &d); err == nil && d.GUID != "" {
			return &d, nil
		}
	}
	d := newDevice()
	if err := d.save(dir); err != nil {
		return nil, err
	}
	return d, nil
}

func newDevice() *Device {
	guid := md5Hex(randomUUID())
	mid := calculateMID(guid)
	return &Device{
		GUID:  guid,
		MID:   mid,
		Dev:   strings.ToUpper(randomString(10)),
		MAC:   "02:00:00:00:00:00",
		WebGL: randomWebGL(),
	}
}

// calculateMID is KuGouMusicApi's calculateMid: the MD5 of the GUID read as
// a base 16 integer, in decimal.
func calculateMID(guid string) string {
	h := md5Hex(guid)
	n, ok := new(big.Int).SetString(h, 16)
	if !ok {
		return "0"
	}
	return n.Text(10)
}

func randomWebGL() string {
	var b [8]byte
	for i := range b {
		b[i] = byte(randIndex(256))
	}
	n := new(big.Int).SetBytes(b[:])
	return n.Text(10)
}

func (d *Device) save(dir string) error {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(devicePath(dir), data, 0o644)
}

// Save writes the device, which keeps a dfid across launches.
func (d *Device) Save(dir string) error { return d.save(dir) }

// RegisterDevice asks the risk service for a dfid, as the concept edition
// does before it plays: /song/url fails without one.
func (c *Client) RegisterDevice(ctx context.Context) error {
	payload := map[string]any{
		"availableRamSize":   4983533568,
		"availableRomSize":   48114719,
		"availableSDSize":    48114717,
		"basebandVer":        "",
		"batteryLevel":       100,
		"batteryStatus":      3,
		"brand":              "Redmi",
		"buildSerial":        "unknown",
		"device":             "marble",
		"imei":               c.Device.GUID,
		"imsi":               "",
		"manufacturer":       "Xiaomi",
		"uuid":               c.Device.GUID,
		"accelerometer":      false,
		"accelerometerValue": "",
		"gravity":            false,
		"gravityValue":       "",
		"gyroscope":          false,
		"gyroscopeValue":     "",
		"light":              false,
		"lightValue":         "",
		"magnetic":           false,
		"magneticValue":      "",
		"orientation":        false,
		"orientationValue":   "",
		"pressure":           false,
		"pressureValue":      "",
		"step_counter":       false,
		"step_counterValue":  "",
		"temperature":        false,
		"temperatureValue":   "",
	}
	plain, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	key, ciphertext, err := playlistAESEncrypt(string(plain))
	if err != nil {
		return err
	}
	portrait, err := json.Marshal(struct {
		AES   string `json:"aes"`
		UID   int64  `json:"uid"`
		Token string `json:"token"`
	}{key, c.UserID, c.Token})
	if err != nil {
		return err
	}
	p, err := rsaPKCS1v15Hex(portrait)
	if err != nil {
		return err
	}

	res, err := c.do(ctx, Request{
		Method:      "POST",
		BaseURL:     "https://userservice.kugou.com",
		Path:        "/risk/v2/r_register_dev",
		Params:      map[string]string{"part": "1", "platid": "1", "p": p},
		Data:        ciphertext,
		EncryptType: "android",
	})
	if err != nil {
		return err
	}

	body := res.Raw
	if plain, derr := playlistAESDecrypt(key, body); derr == nil {
		body = plain
	}
	var out struct {
		Status int64 `json:"status"`
		Data   struct {
			DFID string `json:"dfid"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("kugou: register device: %w: %s", err, snippet(res.Raw))
	}
	if out.Data.DFID == "" {
		return fmt.Errorf("kugou: register device: no dfid: %s", snippet(body))
	}
	c.Device.DFID = out.Data.DFID
	return nil
}

// LoadAccount reads dir/account.json, the login the app saved, into the
// client. A missing file means logged out.
func (c *Client) LoadAccount(dir string) error {
	data, err := os.ReadFile(filepath.Join(dir, "account.json"))
	if err != nil {
		return nil
	}
	var f struct {
		Token  string `json:"token"`
		UserID int64  `json:"userID"`
	}
	if err := json.Unmarshal(data, &f); err != nil {
		return err
	}
	c.Token = f.Token
	c.UserID = f.UserID
	return nil
}

// EnsureDFID registers the device when it has no dfid yet.
func (c *Client) EnsureDFID(ctx context.Context, dir string) error {
	if c.Device.DFID != "" {
		return nil
	}
	if err := c.RegisterDevice(ctx); err != nil {
		return err
	}
	return c.Device.Save(dir)
}

func itoa(n int64) string { return strconv.FormatInt(n, 10) }
