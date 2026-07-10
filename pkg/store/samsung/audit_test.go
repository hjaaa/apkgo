package samsung

import (
	"testing"

	"github.com/KevinGong2013/apkgo/v3/pkg/store"
)

func TestMapSamsungStatus(t *testing.T) {
	cases := map[string]store.AuditState{
		"FOR_SALE":                    store.AuditApproved,
		"READY_FOR_SALE":              store.AuditApproved,
		"READY_FOR_CHANGE":            store.AuditApproved,
		"SUSPENDED":                   store.AuditNeedsFix,
		"PRE_REVIEWS_SUSPENDED":       store.AuditNeedsFix,
		"CONTENT_REVIEW_SUSPENDED":    store.AuditNeedsFix,
		"DEVICE_TEST_SUSPENDED":       store.AuditNeedsFix,
		"TEST_CONFIRMATION_SUSPENDED": store.AuditNeedsFix,
		"UNDER_CONTENT_REVIEW":        store.AuditReviewing,
		"READY_FOR_DEVICE_TEST":       store.AuditReviewing,
		"CONTENT_REVIEW_REJECTED":     store.AuditRejected,
		"CANCELED":                    store.AuditWithdrawn,
		"TERMINATED":                  store.AuditWithdrawn,
		"REGISTERING":                 store.AuditUnknown,
		"WHATEVER":                    store.AuditUnknown,
	}
	for status, want := range cases {
		if got, detail := mapSamsungStatus(status); got != want || detail == "" {
			t.Errorf("mapSamsungStatus(%q) = (%q, %q), want state %q with raw detail", status, got, detail, want)
		}
	}
}

func TestMapSamsungListing(t *testing.T) {
	cases := map[string]store.ListingState{
		"FOR_SALE":                 store.ListingOnShelf,
		"SUSPENDED":                store.ListingOffShelf,
		"TERMINATED":               store.ListingOffShelf,
		"REGISTERING":              store.ListingNotListed,
		"BETA_REGISTERING":         store.ListingNotListed,
		"CONTENT_REVIEW_SUSPENDED": store.ListingOffShelf,
		"UPDATING":                 store.ListingUnknown,
		"READY_FOR_SALE":           store.ListingUnknown,
		"":                         store.ListingUnknown,
		"WHATEVER":                 store.ListingUnknown,
	}
	for status, want := range cases {
		if got := mapSamsungListing(status); got != want {
			t.Errorf("mapSamsungListing(%q) = %q, want %q", status, got, want)
		}
	}
}
