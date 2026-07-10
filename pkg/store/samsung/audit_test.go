package samsung

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-resty/resty/v2"

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
		"FOR_SALE":         store.ListingOnShelf,
		"SUSPENDED":        store.ListingOffShelf,
		"TERMINATED":       store.ListingOffShelf,
		"REGISTERING":      store.ListingNotListed,
		"BETA_REGISTERING": store.ListingNotListed,
		// 审核/注册阶段的 *_SUSPENDED（官方映射表 appStatus=REGISTRATION）可能
		// 发生在从未上架的首次提交上，不能断言 off_shelf，降级 unknown。
		"PRE_REVIEWS_SUSPENDED":       store.ListingUnknown,
		"CONTENT_REVIEW_SUSPENDED":    store.ListingUnknown,
		"DEVICE_TEST_SUSPENDED":       store.ListingUnknown,
		"TEST_CONFIRMATION_SUSPENDED": store.ListingUnknown,
		"BETA_SUSPENDED":              store.ListingUnknown,
		"UPDATING":                    store.ListingUnknown,
		"READY_FOR_SALE":              store.ListingUnknown,
		"":                            store.ListingUnknown,
		"WHATEVER":                    store.ListingUnknown,
	}
	for status, want := range cases {
		if got := mapSamsungListing(status); got != want {
			t.Errorf("mapSamsungListing(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestHasSamsungSaleVersion(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		wantSale  bool
		wantError bool
	}{
		{"sale exists", http.StatusOK, `{"resultCode":"0000","resultMessage":"Ok","data":{"binaries":[]}}`, true, false},
		{"sale missing", http.StatusBadRequest, `{"resultCode":"3100","resultMessage":"Not found for contentId"}`, false, false},
		{"unexpected failure", http.StatusInternalServerError, `{"resultCode":"9000","resultMessage":"Internal server error"}`, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/seller/v2/content/stagedRolloutBinary" {
					t.Fatalf("path = %q", r.URL.Path)
				}
				if r.URL.Query().Get("contentId") != "000007654321" || r.URL.Query().Get("appStatus") != "SALE" {
					t.Fatalf("query = %v", r.URL.Query())
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()

			s := &Store{client: resty.New().SetBaseURL(srv.URL), contentID: "000007654321"}
			got, err := s.hasSamsungSaleVersion(context.Background())
			if got != tc.wantSale || (err != nil) != tc.wantError {
				t.Fatalf("hasSamsungSaleVersion() = (%v, %v), want (%v, error=%v)", got, err, tc.wantSale, tc.wantError)
			}
		})
	}
}

// TestHasSamsungSaleVersionWithErrorHook guards the ordering in
// hasSamsungSaleVersion: it must read resp.ResultCode before checking err,
// because New() installs an OnAfterResponse hook that turns any non-2xx
// response (samsung returns 3100 with HTTP 400) into a non-nil err, while
// resty's own body parsing already populated resp by then. If the checks
// were reordered (err first), this test would see the 3100 case reported
// as a probe error instead of (false, nil).
func TestHasSamsungSaleVersionWithErrorHook(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"resultCode":"3100","resultMessage":"Not found for contentId"}`)
	}))
	defer srv.Close()

	client := resty.New().SetBaseURL(srv.URL)
	client.OnAfterResponse(func(_ *resty.Client, r *resty.Response) error {
		if r.IsError() {
			body := strings.TrimSpace(r.String())
			if len(body) > 500 {
				body = body[:500]
			}
			return fmt.Errorf("http %d: %s", r.StatusCode(), body)
		}
		return nil
	})

	s := &Store{client: client, contentID: "000007654321"}
	got, err := s.hasSamsungSaleVersion(context.Background())
	if got != false || err != nil {
		t.Fatalf("hasSamsungSaleVersion() = (%v, %v), want (false, nil)", got, err)
	}
}

func TestApplySamsungSaleProbe(t *testing.T) {
	probeErr := fmt.Errorf("probe failed")
	cases := []struct {
		state     store.AuditState
		listing   store.ListingState
		hasSale   bool
		err       error
		wantState store.AuditState
		wantList  store.ListingState
	}{
		{store.AuditApproved, store.ListingUnknown, false, nil, store.AuditApprovedFirst, store.ListingNotListed},
		{store.AuditApproved, store.ListingUnknown, true, nil, store.AuditApproved, store.ListingOnShelf},
		{store.AuditApproved, store.ListingUnknown, false, probeErr, store.AuditApproved, store.ListingUnknown},
		{store.AuditReviewing, store.ListingUnknown, false, nil, store.AuditReviewing, store.ListingUnknown},
	}
	for _, tc := range cases {
		gotState, gotList := applySamsungSaleProbe(tc.state, tc.listing, tc.hasSale, tc.err)
		if gotState != tc.wantState || gotList != tc.wantList {
			t.Errorf("applySamsungSaleProbe() = (%q, %q), want (%q, %q)", gotState, gotList, tc.wantState, tc.wantList)
		}
	}
}
