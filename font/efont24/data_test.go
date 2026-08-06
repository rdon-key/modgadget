package efont24

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestEmbeddedFont(t *testing.T) {
	if len(data) != 2706726 {
		t.Fatalf("data size=%d", len(data))
	}
	if hash := fmt.Sprintf("%x", sha256.Sum256([]byte(data))); hash != "d87645e7b45cbf9e9758349a9a337bd38ef832d781e96c19d0596f113ca8f4a7" {
		t.Fatalf("SHA-256=%s", hash)
	}
	if !Font.Valid() || Font.Metrics().LineHeight() != 24 {
		t.Fatalf("valid=%v metrics=%+v", Font.Valid(), Font.Metrics())
	}
	for _, r := range []rune{' ', 'A', '\\', 'あ', 'ア', '日', '￥'} {
		if !Font.HasGlyph(r) {
			t.Fatalf("missing U+%04X", r)
		}
	}
}

func TestAccessAllocations(t *testing.T) {
	if got := testing.AllocsPerRun(100, func() { _ = Font.HasGlyph('あ'); _ = Font.Metrics() }); got != 0 {
		t.Fatalf("allocations=%v", got)
	}
}
