package custfind

import (
	"context"
	"errors"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ---- Searches ----

// SearchesService starts and follows lead searches.
type SearchesService struct{ c *Client }

// Create starts a search. Set Limit to what you need: it caps the credits
// the search reserves.
func (s *SearchesService) Create(ctx context.Context, params CreateSearchParams, opts ...RequestOption) (*SearchCreated, error) {
	var out SearchCreated
	return &out, s.c.do(ctx, http.MethodPost, "/v1/searches", nil, params, &out, opts...)
}

func (s *SearchesService) Get(ctx context.Context, id string) (*Search, error) {
	var out Search
	return &out, s.c.do(ctx, http.MethodGet, "/v1/searches/"+pathEscape(id), nil, nil, &out)
}

// List returns the latest searches, newest first; limit is at most 100
// (0 means 20).
func (s *SearchesService) List(ctx context.Context, limit int) ([]Search, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var out []Search
	return out, s.c.do(ctx, http.MethodGet, "/v1/searches", q, nil, &out)
}

// Cancel stops a running search; unused credits are refunded.
func (s *SearchesService) Cancel(ctx context.Context, id string, opts ...RequestOption) (*Search, error) {
	var out Search
	return &out, s.c.do(ctx, http.MethodPost, "/v1/searches/"+pathEscape(id)+"/cancel", nil, nil, &out, opts...)
}

// Leads returns one page of a search's leads. params.JobID is ignored.
func (s *SearchesService) Leads(ctx context.Context, id string, params LeadListParams) (*Page[Lead], error) {
	params.JobID = ""
	var out Page[Lead]
	return &out, s.c.do(ctx, http.MethodGet, "/v1/searches/"+pathEscape(id)+"/leads", encodeQuery(params), nil, &out)
}

// AllLeads yields every lead of a search, page by page. Iterate a finished
// search: while one runs, new leads can shift the pages.
func (s *SearchesService) AllLeads(ctx context.Context, id string, params LeadListParams) iter.Seq2[Lead, error] {
	return paginate(params, func(p LeadListParams) (*Page[Lead], error) { return s.Leads(ctx, id, p) })
}

// WaitOptions tune WaitForCompletion.
type WaitOptions struct {
	// Interval between polls. Default 3 s.
	Interval time.Duration
}

// WaitForCompletion polls until the search completes, fails or is
// cancelled, and returns it. Bound the wait with ctx.
func (s *SearchesService) WaitForCompletion(ctx context.Context, id string, opts WaitOptions) (*Search, error) {
	interval := opts.Interval
	if interval <= 0 {
		interval = 3 * time.Second
	}
	for {
		search, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		if search.Status.Done() {
			return search, nil
		}
		if err := sleep(ctx, interval); err != nil {
			return search, err
		}
	}
}

// ---- Leads ----

// LeadsService reads leads and acts on one.
type LeadsService struct{ c *Client }

func (s *LeadsService) List(ctx context.Context, params LeadListParams) (*Page[Lead], error) {
	var out Page[Lead]
	return &out, s.c.do(ctx, http.MethodGet, "/v1/leads", encodeQuery(params), nil, &out)
}

// All yields every lead matching params, page by page.
func (s *LeadsService) All(ctx context.Context, params LeadListParams) iter.Seq2[Lead, error] {
	return paginate(params, func(p LeadListParams) (*Page[Lead], error) { return s.List(ctx, p) })
}

// Get returns a lead with its enrichment, emails and calls.
func (s *LeadsService) Get(ctx context.Context, id string) (*LeadDetail, error) {
	var out LeadDetail
	return &out, s.c.do(ctx, http.MethodGet, "/v1/leads/"+pathEscape(id), nil, nil, &out)
}

// Outreach returns the lead's emails.
func (s *LeadsService) Outreach(ctx context.Context, id string) ([]OutreachMessage, error) {
	var out []OutreachMessage
	return out, s.c.do(ctx, http.MethodGet, "/v1/leads/"+pathEscape(id)+"/outreach", nil, nil, &out)
}

// WriteDraft writes the lead's email draft if it has none. With regenerate
// it rewrites the draft, which spends one AI edit.
func (s *LeadsService) WriteDraft(ctx context.Context, id string, regenerate bool, opts ...RequestOption) ([]OutreachMessage, error) {
	var out []OutreachMessage
	body := map[string]bool{"regenerate": regenerate}
	return out, s.c.do(ctx, http.MethodPost, "/v1/leads/"+pathEscape(id)+"/outreach", nil, body, &out, opts...)
}

// CRM returns the lead's deal, notes, tasks, tags, emails and meetings.
func (s *LeadsService) CRM(ctx context.Context, id string) (*LeadCRM, error) {
	var out LeadCRM
	return &out, s.c.do(ctx, http.MethodGet, "/v1/leads/"+pathEscape(id)+"/crm", nil, nil, &out)
}

func (s *LeadsService) AddNote(ctx context.Context, id, body string, opts ...RequestOption) (*Note, error) {
	var out Note
	return &out, s.c.do(ctx, http.MethodPost, "/v1/leads/"+pathEscape(id)+"/notes", nil, map[string]string{"body": body}, &out, opts...)
}

// ---- Outreach ----

// OutreachService edits and sends emails.
type OutreachService struct{ c *Client }

func (s *OutreachService) List(ctx context.Context, params OutreachListParams) (*OutreachPage, error) {
	var out OutreachPage
	return &out, s.c.do(ctx, http.MethodGet, "/v1/outreach", encodeQuery(params), nil, &out)
}

func (s *OutreachService) Get(ctx context.Context, id string) (*OutreachMessage, error) {
	var out OutreachMessage
	return &out, s.c.do(ctx, http.MethodGet, "/v1/outreach/"+pathEscape(id), nil, nil, &out)
}

func (s *OutreachService) Update(ctx context.Context, id string, params UpdateOutreachParams, opts ...RequestOption) (*OutreachMessage, error) {
	var out OutreachMessage
	return &out, s.c.do(ctx, http.MethodPatch, "/v1/outreach/"+pathEscape(id), nil, params, &out, opts...)
}

// Rewrite rewrites the email with AI by tone or instruction. It spends one
// AI edit, refunded if the rewrite fails.
func (s *OutreachService) Rewrite(ctx context.Context, id string, params RewriteOutreachParams, opts ...RequestOption) (*OutreachMessage, error) {
	if params.Tone == "" && params.Instruction == "" {
		return nil, errors.New("custfind: Rewrite needs a Tone or an Instruction")
	}
	var out OutreachMessage
	return &out, s.c.do(ctx, http.MethodPost, "/v1/outreach/"+pathEscape(id)+"/rewrite", nil, params, &out, opts...)
}

// Send queues the email now, or for scheduledAt when it is not zero.
func (s *OutreachService) Send(ctx context.Context, id string, scheduledAt time.Time, opts ...RequestOption) (*OutreachMessage, error) {
	body := map[string]string{}
	if !scheduledAt.IsZero() {
		body["scheduled_at"] = scheduledAt.UTC().Format(time.RFC3339)
	}
	var out OutreachMessage
	return &out, s.c.do(ctx, http.MethodPost, "/v1/outreach/"+pathEscape(id)+"/send", nil, body, &out, opts...)
}

// ---- CRM ----

type PipelinesService struct{ c *Client }

// List returns the pipelines with their stages in order.
func (s *PipelinesService) List(ctx context.Context) ([]Pipeline, error) {
	var out []Pipeline
	return out, s.c.do(ctx, http.MethodGet, "/v1/pipelines", nil, nil, &out)
}

type DealsService struct{ c *Client }

func (s *DealsService) Create(ctx context.Context, params CreateDealParams, opts ...RequestOption) (*Deal, error) {
	var out Deal
	return &out, s.c.do(ctx, http.MethodPost, "/v1/deals", nil, params, &out, opts...)
}

// Update changes the fields that are set, e.g. moves a deal to another stage.
func (s *DealsService) Update(ctx context.Context, id string, params UpdateDealParams, opts ...RequestOption) (*Deal, error) {
	var out Deal
	return &out, s.c.do(ctx, http.MethodPatch, "/v1/deals/"+pathEscape(id), nil, params, &out, opts...)
}

// ---- Webhooks ----

type WebhooksService struct{ c *Client }

// Create registers an endpoint. The returned Secret is shown only now; keep
// it to verify deliveries with ConstructEvent.
func (s *WebhooksService) Create(ctx context.Context, params CreateWebhookParams, opts ...RequestOption) (*WebhookEndpointCreated, error) {
	var out WebhookEndpointCreated
	return &out, s.c.do(ctx, http.MethodPost, "/v1/webhooks", nil, params, &out, opts...)
}

func (s *WebhooksService) List(ctx context.Context) ([]WebhookEndpoint, error) {
	var out []WebhookEndpoint
	return out, s.c.do(ctx, http.MethodGet, "/v1/webhooks", nil, nil, &out)
}

func (s *WebhooksService) Update(ctx context.Context, id string, params UpdateWebhookParams, opts ...RequestOption) (*WebhookEndpoint, error) {
	var out WebhookEndpoint
	return &out, s.c.do(ctx, http.MethodPatch, "/v1/webhooks/"+pathEscape(id), nil, params, &out, opts...)
}

func (s *WebhooksService) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	return s.c.do(ctx, http.MethodDelete, "/v1/webhooks/"+pathEscape(id), nil, nil, nil, opts...)
}

// SendTest queues a webhook.test delivery to the endpoint.
func (s *WebhooksService) SendTest(ctx context.Context, id string, opts ...RequestOption) (*WebhookDelivery, error) {
	var out WebhookDelivery
	return &out, s.c.do(ctx, http.MethodPost, "/v1/webhooks/"+pathEscape(id)+"/test", nil, nil, &out, opts...)
}

// Deliveries returns the endpoint's latest deliveries (limit 0 = default).
func (s *WebhooksService) Deliveries(ctx context.Context, id string, limit int) ([]WebhookDelivery, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	var out []WebhookDelivery
	return out, s.c.do(ctx, http.MethodGet, "/v1/webhooks/"+pathEscape(id)+"/deliveries", q, nil, &out)
}

// ---- Customers ----

// CustomersService manages monthly credit caps for your own customers
// (the external_customer_id you pass when starting searches).
type CustomersService struct{ c *Client }

func (s *CustomersService) List(ctx context.Context) ([]Customer, error) {
	var out []Customer
	return out, s.c.do(ctx, http.MethodGet, "/v1/customers", nil, nil, &out)
}

func (s *CustomersService) Get(ctx context.Context, externalCustomerID string) (*Customer, error) {
	var out Customer
	return &out, s.c.do(ctx, http.MethodGet, "/v1/customers/"+pathEscape(externalCustomerID), nil, nil, &out)
}

// SetLimit caps what the customer's searches may spend each month.
func (s *CustomersService) SetLimit(ctx context.Context, externalCustomerID string, monthlyCredits int, opts ...RequestOption) (*Customer, error) {
	var out Customer
	body := map[string]int{"monthly_credits": monthlyCredits}
	return &out, s.c.do(ctx, http.MethodPut, "/v1/customers/"+pathEscape(externalCustomerID)+"/limit", nil, body, &out, opts...)
}

func (s *CustomersService) RemoveLimit(ctx context.Context, externalCustomerID string, opts ...RequestOption) error {
	return s.c.do(ctx, http.MethodDelete, "/v1/customers/"+pathEscape(externalCustomerID)+"/limit", nil, nil, nil, opts...)
}

// ---- Sender accounts ----

// SenderAccountsService manages the mailboxes emails go out from.
type SenderAccountsService struct{ c *Client }

// List returns the mailboxes; a non-empty externalCustomerID returns only
// that customer's.
func (s *SenderAccountsService) List(ctx context.Context, externalCustomerID string) ([]SenderAccount, error) {
	q := url.Values{}
	if externalCustomerID != "" {
		q.Set("external_customer_id", externalCustomerID)
	}
	var out []SenderAccount
	return out, s.c.do(ctx, http.MethodGet, "/v1/sender-accounts", q, nil, &out)
}

// Create connects a mailbox; the SMTP login is tested before it is saved.
func (s *SenderAccountsService) Create(ctx context.Context, params CreateSenderAccountParams, opts ...RequestOption) (*SenderAccount, error) {
	var out SenderAccount
	return &out, s.c.do(ctx, http.MethodPost, "/v1/sender-accounts", nil, params, &out, opts...)
}

func (s *SenderAccountsService) Delete(ctx context.Context, id string, opts ...RequestOption) error {
	return s.c.do(ctx, http.MethodDelete, "/v1/sender-accounts/"+pathEscape(id), nil, nil, nil, opts...)
}

// ---- pagination ----

func paginate(params LeadListParams, fetch func(LeadListParams) (*Page[Lead], error)) iter.Seq2[Lead, error] {
	return func(yield func(Lead, error) bool) {
		if params.Page < 1 {
			params.Page = 1
		}
		if params.PageSize < 1 {
			params.PageSize = 100
		}
		for {
			page, err := fetch(params)
			if err != nil {
				yield(Lead{}, err)
				return
			}
			for _, lead := range page.Items {
				if !yield(lead, nil) {
					return
				}
			}
			if len(page.Items) == 0 || params.Page*params.PageSize >= page.Total {
				return
			}
			params.Page++
		}
	}
}
