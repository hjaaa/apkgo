package huawei

import (
	"testing"

	"github.com/KevinGong2013/apkgo/v3/pkg/store"
)

// TestMapHuaweiReleaseState locks in the releaseState → unified-state
// mapping (the audit query's only non-trivial logic, and untestable
// end-to-end without real credentials).
func TestMapHuaweiReleaseState(t *testing.T) {
	cases := map[int]store.AuditState{
		4: store.AuditReviewing, 5: store.AuditReviewing, 12: store.AuditReviewing,
		0: store.AuditApproved, 3: store.AuditApproved,
		1: store.AuditRejected, 8: store.AuditRejected, 9: store.AuditRejected, 13: store.AuditRejected,
		2: store.AuditWithdrawn, 10: store.AuditWithdrawn, 11: store.AuditWithdrawn,
		7: store.AuditUnknown, 99: store.AuditUnknown,
	}
	for state, want := range cases {
		if got, _ := mapHuaweiReleaseState(state); got != want {
			t.Errorf("mapHuaweiReleaseState(%d) = %q, want %q", state, got, want)
		}
	}
}

func TestMapHuaweiListing(t *testing.T) {
	cases := []struct {
		state   int
		onShelf int64
		want    store.ListingState
	}{
		{0, 100, store.ListingOnShelf},
		{2, 100, store.ListingOffShelf},
		{6, 100, store.ListingOffShelf},
		{9, 100, store.ListingOnShelf},
		{9, 0, store.ListingUnknown},
		{10, 100, store.ListingOffShelf},
		{11, 100, store.ListingOffShelf},
		{7, 0, store.ListingNotListed},
		{1, 0, store.ListingNotListed},
		{13, 0, store.ListingNotListed},
		{3, 0, store.ListingNotListed},
		{4, 100, store.ListingOnShelf},
		{4, 0, store.ListingNotListed},
		{5, 100, store.ListingOnShelf},
		{8, 100, store.ListingOnShelf},
		{99, 0, store.ListingUnknown},
	}
	for _, tc := range cases {
		if got := mapHuaweiListing(tc.state, tc.onShelf); got != tc.want {
			t.Errorf("mapHuaweiListing(%d, %d) = %q, want %q", tc.state, tc.onShelf, got, tc.want)
		}
	}
}

func TestReviewFromReleaseState(t *testing.T) {
	cases := []struct {
		state   int
		onShelf int64
		want    store.AuditState
	}{
		{0, 0, store.AuditApprovedFirst},
		{3, 0, store.AuditApprovedFirst},
		{0, 120, store.AuditApproved},
		{4, 0, store.AuditReviewing},
		{9, 0, store.AuditRejected},
		{1, 0, store.AuditRejected},
	}
	for _, tc := range cases {
		if got, _ := reviewFromReleaseState(tc.state, tc.onShelf); got != tc.want {
			t.Errorf("reviewFromReleaseState(%d, %d) = %q, want %q", tc.state, tc.onShelf, got, tc.want)
		}
	}
}

func TestAppendHuaweiAuditOpinions(t *testing.T) {
	info := huaweiAuditInfo{
		AuditOpinion:              "隐私政策不完整",
		CopyRightAuditOpinion:     "版权材料缺失",
		CopyRightCodeAuditOpinion: "版号不匹配",
		RecordAuditOpinion:        "备案号无效",
	}
	want := "releaseState=1; auditOpinion=隐私政策不完整; copyRightAuditOpinion=版权材料缺失; copyRightCodeAuditOpinion=版号不匹配; recordAuditOpinion=备案号无效"
	if got := appendHuaweiAuditOpinions(store.AuditRejected, "releaseState=1", info); got != want {
		t.Fatalf("appendHuaweiAuditOpinions() = %q, want %q", got, want)
	}

	empty := huaweiAuditInfo{AuditOpinion: "  隐私政策不完整  "}
	if got := appendHuaweiAuditOpinions(store.AuditRejected, "releaseState=1", empty); got != "releaseState=1; auditOpinion=隐私政策不完整" {
		t.Fatalf("empty opinions were not skipped: %q", got)
	}
	if got := appendHuaweiAuditOpinions(store.AuditReviewing, "releaseState=4", info); got != "releaseState=4" {
		t.Fatalf("non-rejected detail changed: %q", got)
	}
}

// TestClassifyHuawei locks in the "app's packages exceeds the upper limit"
// classification from https://github.com/KevinGong2013/apkgo/issues/31 —
// an AGC-side draft-version package cap, not an apkgo bug, so it must map
// to config_invalid (human fixes the console, not an auto-retry) rather
// than the generic unknown bucket.
func TestClassifyHuawei(t *testing.T) {
	cases := []struct {
		name string
		ret  retInfo
		want store.Category
	}{
		{
			name: "package limit exceeded",
			ret:  retInfo{Code: 204144662, Message: "[cds]add apk failed, additional msg is [the app's packages exceeds the upper limit.]"},
			want: store.CategoryConfigInvalid,
		},
		{
			name: "same code, unrelated message",
			ret:  retInfo{Code: 204144662, Message: "registeredEntity can not be empty"},
			want: store.CategoryUnknown,
		},
		{
			name: "unrelated code",
			ret:  retInfo{Code: 204144660, Message: "package is parsing"},
			want: store.CategoryUnknown,
		},
	}
	for _, tc := range cases {
		if got := classifyHuawei(tc.ret); got != tc.want {
			t.Errorf("%s: classifyHuawei(%+v) = %q, want %q", tc.name, tc.ret, got, tc.want)
		}
	}
}
