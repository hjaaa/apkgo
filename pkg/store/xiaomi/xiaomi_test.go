package xiaomi

import (
	"testing"

	"github.com/KevinGong2013/apkgo/v3/pkg/store"
)

func TestXiaomiListing(t *testing.T) {
	if got := xiaomiListing(true); got != store.ListingOnShelf {
		t.Errorf("xiaomiListing(true) = %q, want on_shelf", got)
	}
	if got := xiaomiListing(false); got != store.ListingNotListed {
		t.Errorf("xiaomiListing(false) = %q, want not_listed", got)
	}
}
