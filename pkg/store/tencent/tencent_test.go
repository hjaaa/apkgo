package tencent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/KevinGong2013/apkgo/v3/pkg/store"
)

// nextDataFixture mirrors the shape of 应用宝's __NEXT_DATA__ blob: the props
// tree embeds the target app plus unrelated recommendation records, and a stub
// reference to the same package that carries no version_name.
const nextDataFixture = `{
  "props": {
    "pageProps": {
      "appInfo": {
        "pkg_name": "com.example.app",
        "version_name": "10.12.2",
        "app_id": "52463570"
      },
      "recommends": [
        {"pkg_name": "com.example.other", "version_name": "9.2.3"},
        {"pkg_name": "com.example.app", "version_name": ""}
      ]
    }
  }
}`

const nextDataNotListedFixture = `{
  "props": {
    "pageProps": {
      "appInfo": {
        "pkg_name": "com.example.app",
        "version_name": "",
        "app_id": "52463570"
      }
    }
  }
}`

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestFindVersionName(t *testing.T) {
	var data any
	if err := json.Unmarshal([]byte(nextDataFixture), &data); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	if got := findVersionName(data, "com.example.app"); got != "10.12.2" {
		t.Errorf("com.example.app: got %q, want %q (stub with empty version_name must be skipped)", got, "10.12.2")
	}
	if got := findVersionName(data, "com.example.other"); got != "9.2.3" {
		t.Errorf("com.example.other: got %q, want %q", got, "9.2.3")
	}
	if got := findVersionName(data, "com.absent.pkg"); got != "" {
		t.Errorf("absent package: got %q, want empty", got)
	}
}

func TestNextDataRe(t *testing.T) {
	html := `<html><head></head><body>` +
		`<script id="__NEXT_DATA__" type="application/json" crossorigin="anonymous">{"a":1}</script>` +
		`</body></html>`
	m := nextDataRe.FindStringSubmatch(html)
	if m == nil {
		t.Fatal("regex did not match the __NEXT_DATA__ script tag")
	}
	if m[1] != `{"a":1}` {
		t.Errorf("captured %q, want %q", m[1], `{"a":1}`)
	}
}

func TestTencentListing(t *testing.T) {
	if got := tencentListing("1.2.0", true); got != store.ListingOnShelf {
		t.Errorf("tencentListing(found live) = %q, want on_shelf", got)
	}
	if got := tencentListing("", true); got != store.ListingNotListed {
		t.Errorf("tencentListing(confirmed absent) = %q, want not_listed", got)
	}
	if got := tencentListing("", false); got != store.ListingUnknown {
		t.Errorf("tencentListing(scrape unknown) = %q, want unknown", got)
	}
}

func TestApplyTencentFirstListing(t *testing.T) {
	if got := applyTencentFirstListing(store.AuditApproved, store.ListingNotListed); got != store.AuditApprovedFirst {
		t.Errorf("apply(approved, not_listed) = %q, want approved_first", got)
	}
	if got := applyTencentFirstListing(store.AuditApproved, store.ListingOnShelf); got != store.AuditApproved {
		t.Errorf("apply(approved, on_shelf) = %q, want approved", got)
	}
	if got := applyTencentFirstListing(store.AuditApproved, store.ListingUnknown); got != store.AuditApproved {
		t.Errorf("apply(approved, unknown) = %q, want approved", got)
	}
	if got := applyTencentFirstListing(store.AuditReviewing, store.ListingNotListed); got != store.AuditReviewing {
		t.Errorf("apply(reviewing, not_listed) = %q, want reviewing", got)
	}
}

func TestLiveVersionFromStorePage(t *testing.T) {
	t.Run("found live version marks on_shelf", func(t *testing.T) {
		restore := withStorePageClient(serverBackedStorePageClient(t, http.StatusOK, nextDataFixture))
		defer restore()

		gotVersion, gotListing := liveVersionFromStorePage(t.Context(), "com.example.app")
		if gotVersion != "10.12.2" {
			t.Fatalf("liveVersionFromStorePage() version = %q, want %q", gotVersion, "10.12.2")
		}
		if gotListing != store.ListingOnShelf {
			t.Fatalf("liveVersionFromStorePage() listing = %q, want %q", gotListing, store.ListingOnShelf)
		}
	})

	t.Run("parsed page without live version marks not_listed", func(t *testing.T) {
		restore := withStorePageClient(serverBackedStorePageClient(t, http.StatusOK, nextDataNotListedFixture))
		defer restore()

		gotVersion, gotListing := liveVersionFromStorePage(t.Context(), "com.example.app")
		if gotVersion != "" {
			t.Fatalf("liveVersionFromStorePage() version = %q, want empty", gotVersion)
		}
		if gotListing != store.ListingNotListed {
			t.Fatalf("liveVersionFromStorePage() listing = %q, want %q", gotListing, store.ListingNotListed)
		}
	})

	t.Run("non-200 keeps listing unknown", func(t *testing.T) {
		restore := withStorePageClient(serverBackedStorePageClient(t, http.StatusBadGateway, "bad gateway"))
		defer restore()

		gotVersion, gotListing := liveVersionFromStorePage(t.Context(), "com.example.app")
		if gotVersion != "" {
			t.Fatalf("liveVersionFromStorePage() version = %q, want empty", gotVersion)
		}
		if gotListing != store.ListingUnknown {
			t.Fatalf("liveVersionFromStorePage() listing = %q, want %q", gotListing, store.ListingUnknown)
		}
	})

	t.Run("network error keeps listing unknown", func(t *testing.T) {
		restore := withStorePageClient(&http.Client{
			Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, errors.New("dial tcp: i/o timeout")
			}),
		})
		defer restore()

		gotVersion, gotListing := liveVersionFromStorePage(t.Context(), "com.example.app")
		if gotVersion != "" {
			t.Fatalf("liveVersionFromStorePage() version = %q, want empty", gotVersion)
		}
		if gotListing != store.ListingUnknown {
			t.Fatalf("liveVersionFromStorePage() listing = %q, want %q", gotListing, store.ListingUnknown)
		}
	})
}

func withStorePageClient(client *http.Client) func() {
	prev := storePageClient
	storePageClient = client
	return func() {
		storePageClient = prev
	}
}

func serverBackedStorePageClient(t *testing.T, status int, body string) *http.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/appdetail/com.example.app" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, htmlWithNextData(body))
	}))
	t.Cleanup(srv.Close)

	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			target := srv.URL + req.URL.Path
			cloned, err := http.NewRequestWithContext(req.Context(), req.Method, target, nil)
			if err != nil {
				return nil, err
			}
			cloned.Header = req.Header.Clone()
			return srv.Client().Transport.RoundTrip(cloned)
		}),
	}
}

func htmlWithNextData(payload string) string {
	return fmt.Sprintf(`<html><body><script id="__NEXT_DATA__" type="application/json">%s</script></body></html>`, payload)
}
