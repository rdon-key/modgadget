//go:build !tinygo

package cardputeradv

import "testing"

func TestConfigureFailureReturnsNilPlayer(t *testing.T) {
	player, err := Configure()
	if err == nil {
		t.Fatal("Configure succeeded outside TinyGo")
	}
	if player != nil {
		t.Fatalf("Configure player=%v, want nil after error", player)
	}
}
