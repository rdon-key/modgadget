package cardputeradv

import "testing"

func TestI2SClockConfiguration(t *testing.T) {
	if got := configuredSampleRate(); got != SampleRate {
		t.Fatalf("sample rate = %d, want %d", got, SampleRate)
	}
	moduleNumerator := uint64(i2sSourceClockHz) * uint64(i2sClockDivDen)
	moduleDenominator := uint64(i2sClockDivInt*i2sClockDivDen + i2sClockDivNum)
	if moduleNumerator%moduleDenominator != 0 {
		t.Fatal("module clock division is not exact")
	}
	if got := moduleNumerator / moduleDenominator; got != 12_288_000 {
		t.Fatalf("module clock = %d", got)
	}
}
