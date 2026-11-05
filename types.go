package custfind

import (
	"encoding/json"
	"time"
)

// SearchStatus is where a search is in its life.
type SearchStatus string

const (
	SearchQueued     SearchStatus = "queued"
	SearchScraping   SearchStatus = "scraping"
	SearchProcessing SearchStatus = "processing"
	SearchCompleted  SearchStatus = "completed"
	SearchFailed     SearchStatus = "failed"
	SearchCancelled  SearchStatus = "cancelled"
)

// Done reports whether the search has stopped (completed, failed or cancelled).
func (s SearchStatus) Done() bool {
	return s == SearchCompleted || s == SearchFailed || s == SearchCancelled
}

// CreateSearchParams starts a search. Set Limit to what you need: it caps
// the credits the search reserves.
type CreateSearchParams struct {
	Sector   string `json:"sector"`
	Location string `json:"location,omitempty"`
	// Limit is the most leads to collect; 0 means the plan's maximum.
	Limit int `json:"limit,omitempty"`
	// SearchMode is "point" (default) or "country".
	SearchMode  string   `json:"search_mode,omitempty"`
	CountryCode string   `json:"country_code,omitempty"`
	Latitude    *float64 `json:"latitude,omitempty"`
	Longitude   *float64 `json:"longitude,omitempty"`
	RadiusKm    *float64 `json:"radius_km,omitempty"`
	Language    string   `json:"language,omitempty"`
	// SellerProfileID picks the offer the leads are pitched with; empty
	// uses the workspace default.
	SellerProfileID string `json:"seller_profile_id,omitempty"`
	// ExternalCustomerID is your own id for the customer this search is
	// for; usage and monthly caps are tracked per customer.
	ExternalCustomerID string `json:"external_customer_id,omitempty"`
}

type SearchCreated struct {
	ProjectID string       `json:"project_id"`
	JobID     string       `json:"job_id"`
	Status    SearchStatus `json:"status"`
}

type Search struct {
	ID                 string       `json:"id"`
	OrganizationID     string       `json:"organization_id"`
	ProjectID          string       `json:"project_id"`
	CreatedBy          string       `json:"created_by"`
	Sector             string       `json:"sector"`
	Location           string       `json:"location"`
	Limit              int          `json:"limit"`
	Latitude           *float64     `json:"latitude,omitempty"`
	Longitude          *float64     `json:"longitude,omitempty"`
	RadiusKm           *float64     `json:"radius_km,omitempty"`
	SearchMode         string       `json:"search_mode,omitempty"`
	CountryCode        string       `json:"country_code,omitempty"`
	Status             SearchStatus `json:"status"`
	Progress           int          `json:"progress"`
	LeadsFound         int          `json:"leads_found"`
	LeadsProcessed     int          `json:"leads_processed"`
	CreditsReserved    int          `json:"credits_reserved,omitempty"`
	CreditsConsumed    int          `json:"credits_consumed,omitempty"`
	CreditsRefunded    int          `json:"credits_refunded,omitempty"`
	ErrorMessage       string       `json:"error_message,omitempty"`
	Source             string       `json:"source,omitempty"`
	SellerProfileID    string       `json:"seller_profile_id,omitempty"`
	APIKeyID           string       `json:"api_key_id,omitempty"`
	ExternalCustomerID string       `json:"external_customer_id,omitempty"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
}

type Lead struct {
	ID        string   `json:"id"`
	ProjectID string   `json:"project_id"`
	Name      string   `json:"name"`
	Phone     string   `json:"phone,omitempty"`
	Website   string   `json:"website,omitempty"`
	Email     string   `json:"email,omitempty"`
	Address   string   `json:"address,omitempty"`
	Category  string   `json:"category,omitempty"`
	Rating    *float64 `json:"rating,omitempty"`
	Score     *int     `json:"score,omitempty"`
	// ContactLocked is true when the plan's contact-reveal allowance hides
	// this lead's email and phone.
	ContactLocked     bool       `json:"contact_locked,omitempty"`
	Source            string     `json:"source,omitempty"`
	RegionCode        string     `json:"region_code,omitempty"`
	RegionName        string     `json:"region_name,omitempty"`
	Language          string     `json:"language,omitempty"`
	CreatedAt         *time.Time `json:"created_at,omitempty"`
	UnsubscribedAt    *time.Time `json:"unsubscribed_at,omitempty"`
	SellerProfileID   string     `json:"seller_profile_id,omitempty"`
	SellerProfileName string     `json:"seller_profile_name,omitempty"`
	DealID            string     `json:"deal_id,omitempty"`
	DealStageID       string     `json:"deal_stage_id,omitempty"`
	DealStageName     string     `json:"deal_stage_name,omitempty"`
	AssignedUserID    string     `json:"assigned_user_id,omitempty"`
	AssignedName      string     `json:"assigned_name,omitempty"`
}

type LeadEnrichment struct {
	ID              string         `json:"id"`
	LeadID          string         `json:"lead_id"`
	Email           string         `json:"email,omitempty"`
	EmailStatus     string         `json:"email_status"`
	EmailVerifiedAt *time.Time     `json:"email_verified_at,omitempty"`
	Socials         []string       `json:"socials"`
	Metadata        map[string]any `json:"metadata"`
	CreatedAt       time.Time      `json:"created_at"`
}

type Call struct {
	ID              string    `json:"id"`
	LeadID          string    `json:"lead_id"`
	Status          string    `json:"status"`
	RecordingURL    string    `json:"recording_url,omitempty"`
	Transcript      string    `json:"transcript,omitempty"`
	Summary         string    `json:"summary,omitempty"`
	DurationSeconds int       `json:"duration_seconds"`
	MinutesCharged  int       `json:"minutes_charged,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// LeadDetail is a lead with its enrichment, emails and calls.
type LeadDetail struct {
	Lead       Lead              `json:"lead"`
	Enrichment *LeadEnrichment   `json:"enrichment,omitempty"`
	Outreach   []OutreachMessage `json:"outreach,omitempty"`
	Calls      []Call            `json:"calls,omitempty"`
}

// Page is one page of a list.
type Page[T any] struct {
	Items    []T `json:"items"`
	Total    int `json:"total"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// LeadListParams filters GET /v1/leads. Zero values are left out.
type LeadListParams struct {
	Page     int `url:"page"`
	PageSize int `url:"page_size"` // at most 100
	// Q searches name, email, phone and website.
	Q        string `url:"q"`
	JobID    string `url:"job_id"`
	Category string `url:"category"`
	// Contact is "email", "phone", "both", "website" or "none".
	Contact   string   `url:"contact"`
	MinScore  *int     `url:"min_score"`
	MaxScore  *int     `url:"max_score"`
	MinRating *float64 `url:"min_rating"`
	MaxRating *float64 `url:"max_rating"`
	Region    string   `url:"region"`
	Language  string   `url:"language"`
	// Source is "scrape", "manual", "import", "linkedin" or "manual_import".
	Source string `url:"source"`
	// OutreachStatus is "none", "pending", "sent", "opened" or "failed".
	OutreachStatus string `url:"outreach_status"`
	// Unsubscribed is "yes" or "no".
	Unsubscribed string `url:"unsubscribed"`
	StageID      string `url:"stage_id"`
	TagID        string `url:"tag_id"`
	// Sort is e.g. "score_desc", "rating_desc", "created_asc", "name".
	// Empty means newest first.
	Sort string `url:"sort"`
}

// OutreachStatus filters emails by state.
type OutreachStatus string

const (
	OutreachDraft     OutreachStatus = "draft"
	OutreachScheduled OutreachStatus = "scheduled"
	OutreachSent      OutreachStatus = "sent"
	OutreachFailed    OutreachStatus = "failed"
)

type OutreachMessage struct {
	ID                  string     `json:"id"`
	LeadID              string     `json:"lead_id"`
	LeadName            string     `json:"lead_name,omitempty"`
	LeadEmail           string     `json:"lead_email,omitempty"`
	LeadEmailStatus     string     `json:"lead_email_status,omitempty"`
	LeadEmailSuppressed bool       `json:"lead_email_suppressed,omitempty"`
	Channel             string     `json:"channel"`
	EmailSubject        string     `json:"email_subject,omitempty"`
	Message             string     `json:"message"`
	ScheduledAt         *time.Time `json:"scheduled_at,omitempty"`
	SentAt              *time.Time `json:"sent_at,omitempty"`
	SenderAccountID     *string    `json:"sender_account_id,omitempty"`
	ErrorMessage        *string    `json:"error_message,omitempty"`
	OpenedAt            *time.Time `json:"opened_at,omitempty"`
	OpenCount           int        `json:"open_count,omitempty"`
	LastOpenedAt        *time.Time `json:"last_opened_at,omitempty"`
	ClickCount          int        `json:"click_count,omitempty"`
	FirstClickedAt      *time.Time `json:"first_clicked_at,omitempty"`
	LastClickedAt       *time.Time `json:"last_clicked_at,omitempty"`
	EngagementScore     int        `json:"engagement_score,omitempty"`
	StepPosition        *int       `json:"step_position,omitempty"`
	SequenceStepName    string     `json:"sequence_step_name,omitempty"`
	SequenceTotalSteps  int        `json:"sequence_total_steps,omitempty"`
	// QualityIssues lists the automatic checks an AI draft failed; such a
	// draft needs a human look before it is sent.
	QualityIssues []string  `json:"quality_issues,omitempty"`
	PromptVersion string    `json:"prompt_version,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type OutreachListParams struct {
	Page     int            `url:"page"`
	PageSize int            `url:"page_size"`
	JobID    string         `url:"job_id"`
	Status   OutreachStatus `url:"status"`
	Channel  string         `url:"channel"`
	Q        string         `url:"q"`
	// Sort is "created_at_asc", "sent_at_desc", "sent_at_asc",
	// "lead_name_asc", "lead_name_desc" or "engagement_score_desc".
	// Empty means newest first.
	Sort string `url:"sort"`
}

// OutreachStats counts the emails matching an outreach list's filter.
type OutreachStats struct {
	Total          int `json:"total"`
	Draft          int `json:"draft"`
	NeedsReview    int `json:"needs_review"`
	Scheduled      int `json:"scheduled"`
	Sent           int `json:"sent"`
	Failed         int `json:"failed"`
	Opened         int `json:"opened"`
	OpenRate       int `json:"open_rate"`
	AvgEngagement  int `json:"avg_engagement"`
	EngagementRate int `json:"engagement_rate"`
}

type OutreachPage struct {
	Page[OutreachMessage]
	Stats OutreachStats `json:"stats"`
}

type UpdateOutreachParams struct {
	Message      string  `json:"message"`
	EmailSubject *string `json:"email_subject,omitempty"`
}

// Rewrite tones.
const (
	ToneProfessional = "professional"
	ToneShort        = "short"
	ToneCasual       = "casual"
)

// RewriteOutreachParams needs a Tone, an Instruction or both.
type RewriteOutreachParams struct {
	Tone        string `json:"tone,omitempty"`
	Instruction string `json:"instruction,omitempty"`
}

type Stage struct {
	ID         string    `json:"id"`
	PipelineID string    `json:"pipeline_id"`
	Name       string    `json:"name"`
	Position   int       `json:"position"`
	Color      string    `json:"color"`
	IsWon      bool      `json:"is_won"`
	IsLost     bool      `json:"is_lost"`
	CreatedAt  time.Time `json:"created_at"`
}

// Pipeline is a deal pipeline with its stages in order.
type Pipeline struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IsDefault bool      `json:"is_default"`
	CreatedAt time.Time `json:"created_at"`
	Stages    []Stage   `json:"stages"`
}

type Deal struct {
	ID             string     `json:"id"`
	LeadID         string     `json:"lead_id"`
	PipelineID     string     `json:"pipeline_id"`
	StageID        string     `json:"stage_id"`
	Title          string     `json:"title"`
	ValueCents     *int64     `json:"value_cents,omitempty"`
	Currency       string     `json:"currency"`
	AssignedUserID *string    `json:"assigned_user_id,omitempty"`
	AssignedName   string     `json:"assigned_name,omitempty"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	LeadName       string     `json:"lead_name,omitempty"`
	LeadEmail      string     `json:"lead_email,omitempty"`
	LeadScore      *int       `json:"lead_score,omitempty"`
}

type CreateDealParams struct {
	LeadID string `json:"lead_id"`
	Title  string `json:"title,omitempty"`
	// StageID defaults to the first stage of the default pipeline.
	StageID        string `json:"stage_id,omitempty"`
	ValueCents     *int64 `json:"value_cents,omitempty"`
	Currency       string `json:"currency,omitempty"`
	AssignedUserID string `json:"assigned_user_id,omitempty"`
}

// UpdateDealParams changes only the fields that are set.
type UpdateDealParams struct {
	Title          *string `json:"title,omitempty"`
	PipelineID     *string `json:"pipeline_id,omitempty"`
	StageID        *string `json:"stage_id,omitempty"`
	ValueCents     *int64  `json:"value_cents,omitempty"`
	Currency       *string `json:"currency,omitempty"`
	AssignedUserID *string `json:"assigned_user_id,omitempty"`
}

type Note struct {
	ID         string    `json:"id"`
	LeadID     string    `json:"lead_id"`
	AuthorName string    `json:"author_name,omitempty"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Task struct {
	ID           string     `json:"id"`
	LeadID       string     `json:"lead_id"`
	DealID       *string    `json:"deal_id,omitempty"`
	AssignedName string     `json:"assigned_name,omitempty"`
	Title        string     `json:"title"`
	DueAt        *time.Time `json:"due_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Meeting struct {
	ID            string    `json:"id"`
	LeadID        string    `json:"lead_id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	StartsAt      time.Time `json:"starts_at"`
	EndsAt        time.Time `json:"ends_at"`
	MeetLink      string    `json:"meet_link,omitempty"`
	AttendeeEmail string    `json:"attendee_email,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// LeadCRM is everything the CRM holds about a lead.
type LeadCRM struct {
	Deal     *Deal             `json:"deal,omitempty"`
	Notes    []Note            `json:"notes"`
	Tasks    []Task            `json:"tasks"`
	Tags     []Tag             `json:"tags"`
	Outreach []OutreachMessage `json:"outreach"`
	Meetings []Meeting         `json:"meetings"`
}

// Webhook event types.
const (
	EventSearchCompleted  = "search.completed"
	EventSearchFailed     = "search.failed"
	EventSearchCancelled  = "search.cancelled"
	EventOutreachSent     = "outreach.sent"
	EventOutreachReplied  = "outreach.replied"
	EventDealStageChanged = "deal.stage_changed"
	EventLeadCreated      = "lead.created"
	// EventWebhookTest is sent only by Webhooks.SendTest.
	EventWebhookTest = "webhook.test"
)

type WebhookEndpoint struct {
	ID                  string     `json:"id"`
	URL                 string     `json:"url"`
	Description         string     `json:"description"`
	Events              []string   `json:"events"`
	CreatedAt           time.Time  `json:"created_at"`
	DisabledAt          *time.Time `json:"disabled_at,omitempty"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
}

// WebhookEndpointCreated carries the signing secret, shown only once.
type WebhookEndpointCreated struct {
	Endpoint WebhookEndpoint `json:"endpoint"`
	Secret   string          `json:"secret"`
}

type CreateWebhookParams struct {
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Description string   `json:"description,omitempty"`
}

// UpdateWebhookParams changes only the fields that are set.
type UpdateWebhookParams struct {
	URL         *string  `json:"url,omitempty"`
	Description *string  `json:"description,omitempty"`
	Events      []string `json:"events,omitempty"`
	Enabled     *bool    `json:"enabled,omitempty"`
}

type WebhookDelivery struct {
	ID             string     `json:"id"`
	EventID        string     `json:"event_id"`
	EventType      string     `json:"event_type"`
	Status         string     `json:"status"` // pending, delivered or failed
	Attempts       int        `json:"attempts"`
	LastStatusCode *int       `json:"last_status_code,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	NextAttemptAt  *time.Time `json:"next_attempt_at,omitempty"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// WebhookEvent is the body of a webhook delivery. Decode Data with
// Event.Decode into the struct for its Type (SearchEventData, …).
type WebhookEvent struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	CreatedAt      time.Time       `json:"created_at"`
	OrganizationID string          `json:"organization_id"`
	Data           json.RawMessage `json:"data"`
}

// Decode unmarshals the event's data into v.
func (e *WebhookEvent) Decode(v any) error { return json.Unmarshal(e.Data, v) }

// SearchEventData is the data of search.completed, search.failed and
// search.cancelled.
type SearchEventData struct {
	ID                 string       `json:"id"`
	Status             SearchStatus `json:"status"`
	Sector             string       `json:"sector"`
	Location           string       `json:"location"`
	LeadsFound         int          `json:"leads_found"`
	CreditsReserved    int          `json:"credits_reserved"`
	CreditsConsumed    int          `json:"credits_consumed"`
	Source             string       `json:"source"`
	APIKeyID           *string      `json:"api_key_id"`
	ExternalCustomerID *string      `json:"external_customer_id"`
	ErrorMessage       *string      `json:"error_message"`
}

// OutreachSentData is the data of outreach.sent.
type OutreachSentData struct {
	ID           string    `json:"id"`
	LeadID       string    `json:"lead_id"`
	Channel      string    `json:"channel"`
	EmailSubject *string   `json:"email_subject"`
	SentAt       time.Time `json:"sent_at"`
}

// OutreachRepliedData is the data of outreach.replied.
type OutreachRepliedData struct {
	ID                string    `json:"id"`
	LeadID            *string   `json:"lead_id"`
	OutreachMessageID *string   `json:"outreach_message_id"`
	ThreadID          string    `json:"thread_id"`
	FromEmail         string    `json:"from_email"`
	FromName          *string   `json:"from_name"`
	Subject           *string   `json:"subject"`
	Text              *string   `json:"text"` // the first 2000 characters
	ReceivedAt        time.Time `json:"received_at"`
}

// DealStageChangedData is the data of deal.stage_changed. PreviousStageID is
// nil when the deal was just created.
type DealStageChangedData struct {
	ID              string  `json:"id"`
	LeadID          string  `json:"lead_id"`
	PipelineID      string  `json:"pipeline_id"`
	StageID         string  `json:"stage_id"`
	StageName       string  `json:"stage_name"`
	IsWon           bool    `json:"is_won"`
	IsLost          bool    `json:"is_lost"`
	PreviousStageID *string `json:"previous_stage_id"`
}

// LeadCreatedData is the data of lead.created.
type LeadCreatedData struct {
	ID                 string   `json:"id"`
	SearchID           *string  `json:"search_id"`
	ExternalCustomerID *string  `json:"external_customer_id"`
	Name               string   `json:"name"`
	Phone              *string  `json:"phone"`
	Website            *string  `json:"website"`
	Address            *string  `json:"address"`
	Category           *string  `json:"category"`
	Rating             *float64 `json:"rating"`
	RegionName         *string  `json:"region_name"`
	Language           *string  `json:"language"`
	Source             string   `json:"source"`
}

// Customer is one of your customers (external_customer_id) and its monthly
// credit cap. MonthlyCreditLimit and Remaining are nil when uncapped.
type Customer struct {
	ExternalCustomerID string    `json:"external_customer_id"`
	MonthlyCreditLimit *int      `json:"monthly_credit_limit"`
	SpentThisMonth     int       `json:"spent_this_month"`
	Remaining          *int      `json:"remaining"`
	PeriodStart        time.Time `json:"period_start"`
}

type SenderAccount struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	Email              string     `json:"email"`
	SMTPHost           string     `json:"smtp_host"`
	SMTPPort           int        `json:"smtp_port"`
	SMTPUser           string     `json:"smtp_user"`
	DailyLimit         int        `json:"daily_limit"`
	ExternalCustomerID string     `json:"external_customer_id,omitempty"`
	IMAPSyncStatus     string     `json:"imap_sync_status,omitempty"`
	IMAPSyncedAt       *time.Time `json:"imap_synced_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// CreateSenderAccountParams connects a mailbox. The password is write-only
// and tested against the server before the account is saved.
type CreateSenderAccountParams struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`
	SMTPUser string `json:"smtp_user"`
	SMTPPass string `json:"smtp_pass"`
	// ExternalCustomerID reserves the mailbox for one of your customers:
	// that customer's leads are then emailed only from its own mailboxes.
	ExternalCustomerID string `json:"external_customer_id,omitempty"`
}

// Balance is the plan and what is left to spend. Credits are remaining meter
// balances ("lead", "ai_edit", "call_minute"); MonthlyCredits what the plan
// grants each month.
type Balance struct {
	Plan               string         `json:"plan"`
	SubscriptionStatus string         `json:"subscription_status"`
	PeriodEnd          *time.Time     `json:"period_end,omitempty"`
	Credits            map[string]int `json:"credits"`
	MonthlyCredits     map[string]int `json:"monthly_credits"`
	FairUseSoftReached bool           `json:"fair_use_soft_reached"`
	FairUseHardReached bool           `json:"fair_use_hard_reached"`
}

type UsageParams struct {
	From string `url:"from"` // YYYY-MM-DD or RFC 3339
	To   string `url:"to"`
	// GroupBy is "customer", "api_key" or "none".
	GroupBy            string `url:"group_by"`
	ExternalCustomerID string `url:"external_customer_id"`
}

type UsageRow struct {
	Group       string `json:"group"`
	Searches    int    `json:"searches"`
	LeadsFound  int    `json:"leads_found"`
	CreditsUsed int    `json:"credits_used"`
	CreditsHeld int    `json:"credits_held"`
}

type Usage struct {
	From    time.Time  `json:"from"`
	To      time.Time  `json:"to"`
	GroupBy string     `json:"group_by"`
	Rows    []UsageRow `json:"rows"`
	Total   UsageRow   `json:"total"`
}
