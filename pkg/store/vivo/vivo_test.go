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

func TestMapVivoAuditStateIncludesUnPassReason(t *testing.T) {
	cases := []struct {
		status int
		reason string
		state  store.AuditState
		detail string
	}{
		{4, "隐私政策链接不可访问", store.AuditRejected, "隐私政策链接不可访问"},
		{4, "   ", store.AuditRejected, ""},
		{3, "不应泄漏到通过态", store.AuditApproved, ""},
		{2, "不应泄漏到审核中", store.AuditReviewing, ""},
		{99, "", store.AuditUnknown, "status=99"},
	}
	for _, tc := range cases {
		gotState, gotDetail := mapVivoAuditState(tc.status, tc.reason)
		if gotState != tc.state || gotDetail != tc.detail {
			t.Errorf("mapVivoAuditState(%d, %q) = (%q, %q), want (%q, %q)", tc.status, tc.reason, gotState, gotDetail, tc.state, tc.detail)
		}
	}
}
