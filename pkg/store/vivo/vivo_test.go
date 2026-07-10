package vivo

import (
	"encoding/json"
	"testing"

	"github.com/KevinGong2013/apkgo/v3/pkg/store"
)

func TestVivoListing(t *testing.T) {
	valid := func(value int) saleStatusField {
		return saleStatusField{Value: value, Valid: true}
	}
	cases := []struct {
		name string
		code saleStatusField
		want store.ListingState
	}{
		{"missing/invalid", saleStatusField{}, store.ListingUnknown},
		{"not listed", valid(0), store.ListingNotListed},
		{"on shelf", valid(1), store.ListingOnShelf},
		{"off shelf", valid(2), store.ListingOffShelf},
		{"unexpected", valid(9), store.ListingUnknown},
	}
	for _, tc := range cases {
		if got := vivoListing(tc.code); got != tc.want {
			t.Errorf("%s: vivoListing(%v) = %q, want %q", tc.name, tc.code, got, tc.want)
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

// TestVivoAppDetailsDecodeSaleStatus pins that an illegal or missing
// saleStatus (blank string, non-numeric string, or absent key) decodes to
// a listing of unknown rather than being silently coerced into the zero
// value (not_listed) or a real number.
func TestVivoAppDetailsDecodeSaleStatus(t *testing.T) {
	cases := []struct {
		name string
		json string
		want store.ListingState
	}{
		{"empty string", `{"saleStatus":""}`, store.ListingUnknown},
		{"invalid string", `{"saleStatus":"invalid"}`, store.ListingUnknown},
		{"missing", `{}`, store.ListingUnknown},
		{"null", `{"saleStatus":null}`, store.ListingUnknown},
		{"zero", `{"saleStatus":0}`, store.ListingNotListed},
		{"quoted one", `{"saleStatus":"1"}`, store.ListingOnShelf},
		{"unexpected number", `{"saleStatus":9}`, store.ListingUnknown},
		{"unexpected quoted number", `{"saleStatus":"9"}`, store.ListingUnknown},
	}
	for _, tc := range cases {
		var app appDetails
		if err := json.Unmarshal([]byte(tc.json), &app); err != nil {
			t.Fatalf("%s: unmarshal error: %v", tc.name, err)
		}
		if got := vivoListing(app.SaleStatus); got != tc.want {
			t.Errorf("%s: vivoListing = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestApplyVivoFirstListing(t *testing.T) {
	cases := []struct {
		state   store.AuditState
		listing store.ListingState
		want    store.AuditState
	}{
		{store.AuditApproved, store.ListingNotListed, store.AuditApprovedFirst},
		{store.AuditApproved, store.ListingOnShelf, store.AuditApproved},
		{store.AuditApproved, store.ListingUnknown, store.AuditApproved},
		{store.AuditReviewing, store.ListingNotListed, store.AuditReviewing},
		{store.AuditRejected, store.ListingNotListed, store.AuditRejected},
	}
	for _, tc := range cases {
		if got := applyVivoFirstListing(tc.state, tc.listing); got != tc.want {
			t.Errorf("applyVivoFirstListing(%q, %q) = %q, want %q", tc.state, tc.listing, got, tc.want)
		}
	}
}

// TestVivoApprovedWithAbnormalSaleStatusNeverApprovedFirst pins the
// end-to-end combination that audit() drives: for an approved app
// (status=3), an abnormal saleStatus (null / blank / non-numeric /
// missing / an out-of-range enum) must decode to listing unknown, and
// applyVivoFirstListing must never promote that to approved_first —
// approved_first requires a confirmed not_listed signal, not the absence
// of one.
func TestVivoApprovedWithAbnormalSaleStatusNeverApprovedFirst(t *testing.T) {
	cases := []struct {
		name string
		json string
	}{
		{"null", `{"packageName":"com.example.app","status":3,"saleStatus":null}`},
		{"empty string", `{"packageName":"com.example.app","status":3,"saleStatus":""}`},
		{"invalid string", `{"packageName":"com.example.app","status":3,"saleStatus":"invalid"}`},
		{"missing", `{"packageName":"com.example.app","status":3}`},
		{"unexpected enum", `{"packageName":"com.example.app","status":3,"saleStatus":9}`},
	}
	for _, tc := range cases {
		var app appDetails
		if err := json.Unmarshal([]byte(tc.json), &app); err != nil {
			t.Fatalf("%s: unmarshal error: %v", tc.name, err)
		}
		// Same order as audit() in vivo.go.
		state, _ := mapVivoAuditState(int(app.Status), app.UnPassReason)
		listing := vivoListing(app.SaleStatus)
		state = applyVivoFirstListing(state, listing)
		if state != store.AuditApproved {
			t.Errorf("%s: State = %q, want approved (never approved_first)", tc.name, state)
		}
		if listing != store.ListingUnknown {
			t.Errorf("%s: Listing = %q, want unknown", tc.name, listing)
		}
	}
}
