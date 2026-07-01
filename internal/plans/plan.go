package plans

import (
	"time"

	"travel-agent/internal/domain"
)

const (
	VisibilityPrivate = "private"
	VisibilityPublic  = "public"

	PublishStatusDraft       = "draft"
	PublishStatusPublished   = "published"
	PublishStatusUnpublished = "unpublished"

	PublicPlanStatusPublished   = "published"
	PublicPlanStatusUnpublished = "unpublished"
	PublicPlanStatusRemoved     = "removed"
)

// UserPlan is the saved travel plan owned by a user. plan_json holds the
// snapshot of the underlying domain.TravelPlan at save time so later edits in
// other surfaces don't mutate the user's library entry.
type UserPlan struct {
	ID                 string
	UserID             string
	TaskID             string
	SourcePublicPlanID string
	Title              string
	Note               string
	Summary            string
	Tags               []string
	Plan               *domain.TravelPlan
	DestinationCity    string
	Days               int
	Visibility         string
	PublishStatus      string
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}

func (p UserPlan) IsDeleted() bool {
	return p.DeletedAt != nil && !p.DeletedAt.IsZero()
}

// PublicPlan is the publish-side snapshot. It deliberately does not contain
// the user's private note or conversation archive.
type PublicPlan struct {
	ID              string
	PlanID          string
	UserID          string
	AuthorName      string
	Title           string
	Summary         string
	Tags            []string
	Plan            *domain.TravelPlan
	DestinationCity string
	Days            int
	Status          string
	ViewCount       int64
	SaveCount       int64
	CopyCount       int64
	HotScore        int64
	PublishedAt     time.Time
	UpdatedAt       time.Time
}

// PublicViewer is the dedup/analytics identity for public plan reads. UserID
// wins when present; anonymous clients use a hashed client fingerprint.
type PublicViewer struct {
	UserID     string
	ClientHash string
}

func (v PublicViewer) DedupKey() string {
	if v.UserID != "" {
		return "user:" + v.UserID
	}
	if v.ClientHash != "" {
		return "anon:" + v.ClientHash
	}
	return ""
}

// PublicPlanEvent is an audit row for counted public plan interactions.
type PublicPlanEvent struct {
	PublicPlanID string
	UserID       string
	EventType    string
	ClientHash   string
	CreatedAt    time.Time
}

// ConversationArchive captures the brief and chat that produced this plan. It
// is private to the owner and never leaks into public details.
type ConversationArchive struct {
	ID        string
	PlanID    string
	UserID    string
	TaskID    string
	Brief     *domain.TravelRequest
	Messages  []ArchivedMessage
	Events    []ArchivedEvent
	CreatedAt time.Time
}

type ArchivedMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type ArchivedEvent struct {
	Type     string `json:"type"`
	NodeName string `json:"node_name,omitempty"`
	Message  string `json:"message,omitempty"`
}

// TaskSnapshot is the view of a generation task that the plan service needs to
// save it. Defined in this package so we don't take a hard dependency on
// internal/travel; the server wiring layer adapts the travel store into this.
type TaskSnapshot struct {
	TaskID  string
	UserID  string
	Status  string
	Plan    *domain.TravelPlan
	Request domain.TravelRequest
}

const (
	TaskStatusSucceeded = "succeeded"
)

// ListFilter is the union of /me/plans query parameters.
type ListFilter struct {
	Query           string
	Visibility      string
	PublishStatus   string
	DestinationCity string
	Page            int
	PageSize        int
}

// PublicListFilter drives /public/plans search & ranking.
type PublicListFilter struct {
	Query           string
	DestinationCity string
	Days            int
	Interest        string
	Sort            string // "hot" | "latest"
	Page            int
	PageSize        int
}
