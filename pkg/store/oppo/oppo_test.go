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

func TestAppendOppoAuditExtra(t *testing.T) {
	rejected := oppoAuditExtra{
		RefuseAdvice:         "补充隐私政策",
		BusinessRefuseReason: "资质不完整",
		RefuseFile:           "https://example.com/refuse.png",
		FreezeReason:         "不应出现在驳回态",
		FreezeAdvice:         "不应出现在驳回态",
	}
	wantRejected := "审核不通过: 主理由; refuse_advice=补充隐私政策; business_refuse_reason=资质不完整; refuse_file=https://example.com/refuse.png"
	if got := appendOppoAuditExtra(store.AuditRejected, "审核不通过: 主理由", rejected); got != wantRejected {
		t.Fatalf("rejected detail = %q, want %q", got, wantRejected)
	}

	frozen := oppoAuditExtra{FreezeReason: "违规冻结", FreezeAdvice: "完成整改后申诉", RefuseAdvice: "不应出现"}
	wantFrozen := "已冻结; freeze_reason=违规冻结; freeze_advice=完成整改后申诉"
	if got := appendOppoAuditExtra(store.AuditNeedsFix, "已冻结", frozen); got != wantFrozen {
		t.Fatalf("freeze detail = %q, want %q", got, wantFrozen)
	}

	if got := appendOppoAuditExtra(store.AuditApproved, "已上线", rejected); got != "已上线" {
		t.Fatalf("approved detail changed: %q", got)
	}
}
