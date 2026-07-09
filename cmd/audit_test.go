package cmd

import (
	"testing"

	"github.com/KevinGong2013/apkgo/v3/pkg/apkgo"
	"github.com/KevinGong2013/apkgo/v3/pkg/store"
)

func TestAuditOneLiner(t *testing.T) {
	cases := []struct {
		name string
		row  apkgo.AuditStoreResult
		want string
	}{
		{
			name: "on shelf approved first",
			row: apkgo.AuditStoreResult{
				Supported: true,
				AuditResult: store.AuditResult{
					State:   store.AuditApprovedFirst,
					Listing: store.ListingOnShelf,
				},
			},
			want: "🟢在架  🎉 approved_first",
		},
		{
			name: "unknown listing needs fix",
			row: apkgo.AuditStoreResult{
				Supported: true,
				AuditResult: store.AuditResult{
					State:   store.AuditNeedsFix,
					Listing: store.ListingUnknown,
					Detail:  "metadata pending",
				},
			},
			want: "❔未知  🛠 needs_fix (metadata pending)",
		},
		{
			name: "no listing column",
			row: apkgo.AuditStoreResult{
				Supported: true,
				AuditResult: store.AuditResult{
					State: store.AuditApproved,
				},
			},
			want: "✅ approved",
		},
		{
			name: "error branch",
			row: apkgo.AuditStoreResult{
				Supported: true,
				AuditResult: store.AuditResult{
					Error: "boom",
				},
			},
			want: "⚠ boom",
		},
		{
			name: "unsupported branch",
			row: apkgo.AuditStoreResult{
				Supported: false,
			},
			want: "   audit not supported",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := auditOneLiner(tc.row, false); got != tc.want {
				t.Fatalf("auditOneLiner() = %q, want %q", got, tc.want)
			}
		})
	}
}
