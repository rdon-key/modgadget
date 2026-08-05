package cardputeradv

import (
	"errors"
	"testing"
)

type i2cTransaction struct {
	address uint16
	data    [2]byte
	length  int
}

type fakeI2C struct {
	transactions [32]i2cTransaction
	count        int
	failAt       int
}

func (bus *fakeI2C) Tx(address uint16, write, read []byte) error {
	bus.count++
	if bus.failAt == bus.count {
		return errors.New("I2C failure")
	}
	transaction := &bus.transactions[bus.count-1]
	transaction.address = address
	transaction.length = len(write)
	copy(transaction.data[:], write)
	if len(write) == 1 && len(read) == 1 {
		switch write[0] {
		case 0xfd:
			read[0] = 0x83
		case 0xfe:
			read[0] = 0x11
		}
	}
	return nil
}

func TestCodecDetectionAndPrepareSequence(t *testing.T) {
	bus := &fakeI2C{}
	if err := detectCodec(bus); err != nil {
		t.Fatal(err)
	}
	if err := prepareCodec(bus); err != nil {
		t.Fatal(err)
	}
	if err := unmuteCodec(bus); err != nil {
		t.Fatal(err)
	}
	if bus.count != 2+len(codecPrepareSequence)+1 {
		t.Fatalf("transactions = %d", bus.count)
	}
	for i, transaction := range bus.transactions[:bus.count] {
		if transaction.address != codecAddress {
			t.Fatalf("transaction %d address = 0x%x", i, transaction.address)
		}
	}
	for i, want := range codecPrepareSequence {
		got := bus.transactions[i+2]
		if got.data != [2]byte{want.register, want.value} {
			t.Fatalf("prepare[%d] = %x, want %x", i, got.data, [2]byte{want.register, want.value})
		}
	}
	volumeFound := false
	for _, item := range codecPrepareSequence {
		if item.register == 0x32 {
			volumeFound = true
			if item.value != 0xef {
				t.Fatalf("DAC volume=0x%02x want=0xef", item.value)
			}
		}
	}
	if !volumeFound {
		t.Fatal("DAC volume register was not configured")
	}
	if got := bus.transactions[bus.count-1].data; got != [2]byte{0x31, 0x00} {
		t.Fatalf("unmute = %x", got)
	}
	for _, transaction := range bus.transactions[:bus.count] {
		if transaction.length == 2 && transaction.data[0] >= 0x14 && transaction.data[0] <= 0x1c {
			t.Fatalf("microphone register 0x%02x was configured", transaction.data[0])
		}
	}
}

func TestCodecErrorsStopSequence(t *testing.T) {
	bus := &fakeI2C{failAt: 4}
	if err := prepareCodec(bus); err == nil {
		t.Fatal("prepare succeeded despite I2C error")
	}
	if bus.count != 4 {
		t.Fatalf("transactions continued after error: %d", bus.count)
	}
}

func TestCodecRejectsWrongIdentity(t *testing.T) {
	bus := &wrongIdentityBus{}
	if err := detectCodec(bus); err == nil {
		t.Fatal("wrong codec identity accepted")
	}
}

type wrongIdentityBus struct{}

func (*wrongIdentityBus) Tx(_ uint16, _ []byte, read []byte) error {
	read[0] = 0
	return nil
}
