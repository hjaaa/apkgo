package oppo

import (
	"testing"

	"github.com/KevinGong2013/apkgo/v3/pkg/store"
)

func TestOppoListing(t *testing.T) {
	cases := map[string]store.ListingState{
		"已上线":   store.ListingOnShelf,
		"上架成功":  store.ListingOnShelf,
		"已发布":   store.ListingOnShelf,
		"已下架":   store.ListingOffShelf,
		"已冻结":   store.ListingOffShelf,
		"审核中":   store.ListingUnknown,
		"待上架":   store.ListingUnknown,
		"上架审核中": store.ListingUnknown,
		"":      store.ListingUnknown,
	}
	for name, want := range cases {
		if got := oppoListing(name); got != want {
			t.Errorf("oppoListing(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestMapOppoAuditNeedsFix(t *testing.T) {
	cases := map[string]store.AuditState{
		"待整改": store.AuditNeedsFix,
		"已冻结": store.AuditNeedsFix,
		"已下架": store.AuditWithdrawn,
		"已撤销": store.AuditWithdrawn,
		"已上线": store.AuditApproved,
	}
	for name, want := range cases {
		if got, _ := mapOppoAudit(name, ""); got != want {
			t.Errorf("mapOppoAudit(%q) = %q, want %q", name, got, want)
		}
	}
}
