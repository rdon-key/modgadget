//go:build !tinygo || !esp32s3

package cardputeradv

import "errors"

func audioDebugText(string)            {}
func audioDebugToneQueued(int, uint32) {}

type unsupportedDevice struct{}

func newPCMDevice() pcmDevice { return unsupportedDevice{} }

func (unsupportedDevice) Configure() error {
	return errors.New("Cardputer ADV audio requires TinyGo on ESP32-S3")
}
func (unsupportedDevice) Ready() bool           { return true }
func (unsupportedDevice) WritePCM([]byte) error { return nil }
func (unsupportedDevice) Stop() error           { return nil }
