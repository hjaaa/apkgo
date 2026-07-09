package store

import "context"

// AuditState is the unified review state of a submitted version,
// normalised from each store's own status codes so callers don't have to
// learn every vendor's enum.
type AuditState string

const (
	AuditReviewing AuditState = "reviewing" // 审核中 — submitted, pending review
	AuditApproved  AuditState = "approved"  // 审核通过 — approved (live or pending release)
	AuditRejected  AuditState = "rejected"  // 审核驳回 — not approved
	AuditWithdrawn AuditState = "withdrawn" // 已撤回 — submission cancelled
	AuditUnknown   AuditState = "unknown"   // store returned a state we don't map
	// AuditApprovedFirst 审核通过-首次上架:通过/待发布 且 该应用尚无在架版本
	// (live version is empty). 用于把“首次上架通过”与“更新通过”区分开。
	AuditApprovedFirst AuditState = "approved_first"
	// AuditNeedsFix 待整改:仅在渠道原始标签/状态确实暴露“整改”语义时点亮。
	AuditNeedsFix AuditState = "needs_fix"
)

// Resolved reports whether the review has finished (terminal state).
// `apkgo audit --watch` stops polling once every store is Resolved.
func (s AuditState) Resolved() bool {
	return s == AuditApproved || s == AuditApprovedFirst ||
		s == AuditRejected || s == AuditWithdrawn
}

// ListingState is the visibility state of an app on a store, orthogonal to
// the review workflow state.
type ListingState string

const (
	ListingOnShelf   ListingState = "on_shelf"
	ListingOffShelf  ListingState = "off_shelf"
	ListingNotListed ListingState = "not_listed"
	ListingUnknown   ListingState = "unknown"
)

// AuditQuery identifies the app/version whose review status to look up.
// VersionName/VersionCode are best-effort hints; most stores key off the
// package name and report the latest submission.
type AuditQuery struct {
	Package     string
	VersionName string
	VersionCode int32
	// ExternalID, when set, pins the query to one specific submission (e.g.
	// honor's releaseId, captured from UploadResult.ExternalID at upload
	// time) instead of the store's own notion of "current" state. Stores
	// that support a per-submission audit lookup (honor) use it when
	// present; others ignore it.
	ExternalID string
}

// AuditResult is one store's review status. Error is set when the query
// itself failed (auth / network / not-found); State is meaningful only
// when Error is empty.
type AuditResult struct {
	Store   string       `json:"store"`
	State   AuditState   `json:"state,omitempty"`
	Listing ListingState `json:"listing,omitempty"`
	Detail  string       `json:"detail,omitempty"`
	Error   string       `json:"error,omitempty"`

	// VersionName/VersionCode identify the version the State refers to —
	// the latest submitted/under-review iteration the store reports. They
	// are best-effort: a store with no version field in its audit API
	// leaves them empty (tencent), and an auditor may fall back to the
	// caller's AuditQuery hints (xiaomi, which has no review API and infers
	// state by comparing our submitted version against the live one).
	VersionName string `json:"version_name,omitempty"`
	VersionCode int32  `json:"version_code,omitempty"`
	// LiveVersionName/LiveVersionCode are the version currently published
	// to users, set only by stores that expose the live version separately
	// from an in-review one (huawei's onShelf* fields; xiaomi's queried
	// on-shelf version; tencent best-effort scrapes its public detail page,
	// which carries name only — no LiveVersionCode). Empty when the store
	// reports a single record or the lookup fails.
	LiveVersionName string `json:"live_version_name,omitempty"`
	LiveVersionCode int32  `json:"live_version_code,omitempty"`
}

// AuditFn looks up the current review status for one store from its raw
// config. Like DiagnoseFn it owns its own auth/setup, so it runs on an
// independent context fully decoupled from the upload flow — which is the
// point: upload finishes at "submitted (审核中)" and review progress is
// polled separately, on its own schedule.
type AuditFn func(ctx context.Context, cfg map[string]string, q AuditQuery) AuditResult

var auditors = map[string]AuditFn{}

// RegisterAuditor opts a store into `apkgo audit`. Stores without one are
// reported as unsupported (no review-status API, or not yet wired).
func RegisterAuditor(name string, fn AuditFn) {
	auditors[name] = fn
}

// QueryAudit runs the registered auditor for a store. The second return
// value is false when the store has not registered one.
func QueryAudit(ctx context.Context, name string, cfg map[string]string, q AuditQuery) (AuditResult, bool) {
	fn, ok := auditors[name]
	if !ok {
		return AuditResult{}, false
	}
	return fn(ctx, cfg, q), true
}
