package plans

import "travel-agent/internal/domain"

// UserPlanDTO is the JSON shape returned to /me/plans clients. It excludes
// archive content; consumers fetch /me/plans/:id/conversation if they need it.
type UserPlanDTO struct {
	ID                 string             `json:"plan_id"`
	UserID             string             `json:"user_id"`
	TaskID             string             `json:"task_id,omitempty"`
	SourcePublicPlanID string             `json:"source_public_plan_id,omitempty"`
	Title              string             `json:"title"`
	Note               string             `json:"note,omitempty"`
	Summary            string             `json:"summary,omitempty"`
	Tags               []string           `json:"tags"`
	Plan               *domain.TravelPlan `json:"plan,omitempty"`
	DestinationCity    string             `json:"destination_city"`
	Days               int                `json:"days"`
	Visibility         string             `json:"visibility"`
	PublishStatus      string             `json:"publish_status"`
	CreatedAt          string             `json:"created_at"`
	UpdatedAt          string             `json:"updated_at"`
}

func ToUserPlanDTO(plan UserPlan, includePlan bool) UserPlanDTO {
	dto := UserPlanDTO{
		ID:                 plan.ID,
		UserID:             plan.UserID,
		TaskID:             plan.TaskID,
		SourcePublicPlanID: plan.SourcePublicPlanID,
		Title:              plan.Title,
		Note:               plan.Note,
		Summary:            plan.Summary,
		Tags:               nonNilTags(plan.Tags),
		DestinationCity:    plan.DestinationCity,
		Days:               plan.Days,
		Visibility:         plan.Visibility,
		PublishStatus:      plan.PublishStatus,
		CreatedAt:          plan.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:          plan.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if includePlan {
		dto.Plan = plan.Plan
	}
	return dto
}

// PublicPlanDTO is the response shape for /public/plans*. It deliberately
// omits user_id and never includes private fields like note or archives.
type PublicPlanDTO struct {
	PublicPlanID    string                  `json:"public_plan_id"`
	Title           string                  `json:"title"`
	Summary         string                  `json:"summary,omitempty"`
	Tags            []string                `json:"tags"`
	Plan            *domain.TravelPlan      `json:"plan,omitempty"`
	DestinationCity string                  `json:"destination_city"`
	Days            int                     `json:"days"`
	Author          PublicPlanAuthorDTO     `json:"author"`
	HotScore        int64                   `json:"hot_score"`
	ViewCount       int64                   `json:"view_count"`
	SaveCount       int64                   `json:"save_count"`
	PublishedAt     string                  `json:"published_at"`
	UpdatedAt       string                  `json:"updated_at,omitempty"`
}

type PublicPlanAuthorDTO struct {
	DisplayName string `json:"display_name"`
}

func ToPublicPlanDTO(pub PublicPlan, includePlan bool) PublicPlanDTO {
	dto := PublicPlanDTO{
		PublicPlanID:    pub.ID,
		Title:           pub.Title,
		Summary:         pub.Summary,
		Tags:            nonNilTags(pub.Tags),
		DestinationCity: pub.DestinationCity,
		Days:            pub.Days,
		Author:          PublicPlanAuthorDTO{DisplayName: pub.AuthorName},
		HotScore:        pub.HotScore,
		ViewCount:       pub.ViewCount,
		SaveCount:       pub.SaveCount,
		PublishedAt:     pub.PublishedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       pub.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
	if includePlan {
		dto.Plan = pub.Plan
	}
	return dto
}

// ConversationArchiveDTO is the response of /me/plans/:id/conversation.
type ConversationArchiveDTO struct {
	PlanID    string                       `json:"plan_id"`
	TaskID    string                       `json:"task_id,omitempty"`
	Brief     *domain.TravelRequest        `json:"brief,omitempty"`
	Messages  []ArchivedMessage            `json:"messages"`
	Events    []ArchivedEvent              `json:"events"`
	CreatedAt string                       `json:"created_at"`
}

func ToArchiveDTO(archive ConversationArchive) ConversationArchiveDTO {
	return ConversationArchiveDTO{
		PlanID:    archive.PlanID,
		TaskID:    archive.TaskID,
		Brief:     archive.Brief,
		Messages:  nonNilMessages(archive.Messages),
		Events:    nonNilEvents(archive.Events),
		CreatedAt: archive.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func nonNilTags(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilMessages(values []ArchivedMessage) []ArchivedMessage {
	if values == nil {
		return []ArchivedMessage{}
	}
	return values
}

func nonNilEvents(values []ArchivedEvent) []ArchivedEvent {
	if values == nil {
		return []ArchivedEvent{}
	}
	return values
}
