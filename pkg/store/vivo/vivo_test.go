package vivo

import (
	"testing"

	"github.com/KevinGong2013/apkgo/v3/pkg/store"
)

func TestVivoListingFallsBackToUnknownWhenOnlineStateIsUnverified(t *testing.T) {
	cases := []int{0, 1, 2, 9}
	for _, code := range cases {
		if got := vivoListing(code); got != store.ListingUnknown {
			t.Fatalf("vivoListing(%d) = %q, want %q", code, got, store.ListingUnknown)
		}
	}
}
