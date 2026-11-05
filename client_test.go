package custfind

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type seen struct {
	Method, Path, Query, Body string
	Header                    http.Header
}

type reply struct {
	status  int
	body    string
	headers map[string]string
}

func testServer(t *testing.T, handle func(n int, r seen) reply) (*Client, *[]seen) {
	t.Helper()
	var mu sync.Mutex
	var reqs []seen
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		s := seen{r.Method, r.URL.EscapedPath(), r.URL.RawQuery, string(b), r.Header.Clone()}
		mu.Lock()
		reqs = append(reqs, s)
		n := len(reqs)
		mu.Unlock()
		out := handle(n, s)
		for k, v := range out.headers {
			w.Header().Set(k, v)
		}
		if out.body != "" && w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json")
		}
		w.WriteHeader(out.status)
		_, _ = io.WriteString(w, out.body)
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient(WithAPIKey("cf_live_test"), WithBaseURL(srv.URL), WithMaxRetries(3))
	if err != nil {
		t.Fatal(err)
	}
	return c, &reqs
}

func TestNewClientNeedsKey(t *testing.T) {
	t.Setenv("CUSTFIND_API_KEY", "")
	if _, err := NewClient(); err == nil {
		t.Fatal("want an error without a key")
	}
	t.Setenv("CUSTFIND_API_KEY", "cf_live_env")
	c, err := NewClient()
	if err != nil || c.apiKey != "cf_live_env" {
		t.Fatalf("env key not used: %v", err)
	}
}

func TestSendsKeyBodyAndIdempotencyKey(t *testing.T) {
	c, reqs := testServer(t, func(n int, r seen) reply {
		if n == 1 {
			return reply{202, `{"project_id":"p1","job_id":"j1","status":"queued"}`, nil}
		}
		return reply{200, `{"items":[{"id":"l1","project_id":"p1","name":"Dent","score":81}],"total":1,"page":1,"page_size":25}`, nil}
	})
	ctx := context.Background()
	res, err := c.Searches.Create(ctx, CreateSearchParams{Sector: "Dental clinic", Location: "Istanbul", Limit: 25, ExternalCustomerID: "acme"})
	if err != nil {
		t.Fatal(err)
	}
	if res.JobID != "j1" || res.Status != SearchQueued {
		t.Fatalf("got %+v", res)
	}
	r := (*reqs)[0]
	if r.Method != "POST" || r.Path != "/v1/searches" {
		t.Fatalf("got %s %s", r.Method, r.Path)
	}
	if r.Header.Get("Authorization") != "Bearer cf_live_test" {
		t.Fatalf("auth header %q", r.Header.Get("Authorization"))
	}
	if r.Header.Get("Idempotency-Key") == "" {
		t.Fatal("writes must carry an Idempotency-Key")
	}
	var body map[string]any
	_ = json.Unmarshal([]byte(r.Body), &body)
	want := map[string]any{"sector": "Dental clinic", "location": "Istanbul", "limit": float64(25), "external_customer_id": "acme"}
	if len(body) != len(want) {
		// The API rejects unknown fields, so nothing unset may be sent.
		t.Fatalf("body %v, want only %v", body, want)
	}
	for k, v := range want {
		if body[k] != v {
			t.Fatalf("body[%s] = %v, want %v", k, body[k], v)
		}
	}

	page, err := c.Leads.List(ctx, LeadListParams{MinScore: Ptr(70), Contact: "email", Sort: "score_desc"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || *page.Items[0].Score != 81 {
		t.Fatalf("got %+v", page)
	}
	r = (*reqs)[1]
	if r.Query != "contact=email&min_score=70&sort=score_desc" {
		t.Fatalf("query %q", r.Query)
	}
	if r.Header.Get("Idempotency-Key") != "" {
		t.Fatal("reads must not carry an Idempotency-Key")
	}
}

func TestOwnIdempotencyKey(t *testing.T) {
	c, reqs := testServer(t, func(int, seen) reply { return reply{202, `{"job_id":"j"}`, nil} })
	if _, err := c.Searches.Create(context.Background(), CreateSearchParams{Sector: "x"}, WithIdempotencyKey("order-42")); err != nil {
		t.Fatal(err)
	}
	if got := (*reqs)[0].Header.Get("Idempotency-Key"); got != "order-42" {
		t.Fatalf("key %q", got)
	}
}

func TestRetries5xx429AndInProgressWithSameKey(t *testing.T) {
	c, reqs := testServer(t, func(n int, _ seen) reply {
		switch n {
		case 1:
			return reply{503, `{"error":"x"}`, nil}
		case 2:
			return reply{429, `{"error":"rate_limited"}`, map[string]string{"Retry-After": "0"}}
		case 3:
			return reply{409, `{"error":"idempotency_in_progress"}`, map[string]string{"Retry-After": "0"}}
		}
		return reply{202, `{"job_id":"j2"}`, nil}
	})
	c.maxRetries = 4
	res, err := c.Searches.Create(context.Background(), CreateSearchParams{Sector: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if res.JobID != "j2" || len(*reqs) != 4 {
		t.Fatalf("job %q after %d requests", res.JobID, len(*reqs))
	}
	key := (*reqs)[0].Header.Get("Idempotency-Key")
	for _, r := range *reqs {
		if r.Header.Get("Idempotency-Key") != key {
			t.Fatal("the key must stay the same across retries")
		}
		if r.Body != (*reqs)[0].Body {
			t.Fatal("the body must be resent unchanged")
		}
	}
}

func TestFinalConflictIsNotRetried(t *testing.T) {
	c, reqs := testServer(t, func(int, seen) reply { return reply{409, `{"error":"invalid job state"}`, nil} })
	_, err := c.Searches.Cancel(context.Background(), "j1")
	if !HasCode(err, "invalid job state") {
		t.Fatalf("err %v", err)
	}
	if len(*reqs) != 1 {
		t.Fatalf("a final 409 was retried: %d requests", len(*reqs))
	}
}

func TestErrors(t *testing.T) {
	c, reqs := testServer(t, func(n int, _ seen) reply {
		if n == 1 {
			return reply{402, `{"error":"customer_limit_reached"}`, nil}
		}
		return reply{400, "invalid json body\n", map[string]string{"Content-Type": "text/plain"}}
	})
	_, err := c.Searches.Create(context.Background(), CreateSearchParams{Sector: "x", ExternalCustomerID: "acme"})
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 402 || apiErr.Code != CodeCustomerLimitReached {
		t.Fatalf("err %#v", err)
	}
	if !IsPaymentRequired(err) || !HasCode(err, CodeCustomerLimitReached) {
		t.Fatal("helpers disagree")
	}
	_, err = c.Deals.Create(context.Background(), CreateDealParams{LeadID: "l"})
	if !errors.As(err, &apiErr) || apiErr.Code != "http_400" || apiErr.Message != "invalid json body" {
		t.Fatalf("err %#v", err)
	}
	if len(*reqs) != 2 {
		t.Fatalf("4xx must not be retried: %d requests", len(*reqs))
	}
}

func TestNetworkErrorIsRetriedThenReported(t *testing.T) {
	c, err := NewClient(WithAPIKey("k"), WithBaseURL("http://127.0.0.1:1"), WithMaxRetries(1))
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Me(context.Background())
	var apiErr *Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 0 || apiErr.Code != "network_error" {
		t.Fatalf("err %#v", err)
	}
}

func TestContextCancelStopsRetries(t *testing.T) {
	c, _ := testServer(t, func(int, seen) reply { return reply{503, `{"error":"x"}`, map[string]string{"Retry-After": "30"}} })
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := c.Balance(ctx)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 2*time.Second {
		t.Fatalf("err %v after %v", err, time.Since(start))
	}
}

func TestAllLeadsPaginates(t *testing.T) {
	c, reqs := testServer(t, func(n int, r seen) reply {
		if n == 1 {
			return reply{200, `{"items":[{"id":"a"},{"id":"b"}],"total":3,"page":1,"page_size":2}`, nil}
		}
		return reply{200, `{"items":[{"id":"c"}],"total":3,"page":2,"page_size":2}`, nil}
	})
	var ids []string
	for lead, err := range c.Searches.AllLeads(context.Background(), "job/1", LeadListParams{PageSize: 2}) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, lead.ID)
	}
	if strings.Join(ids, ",") != "a,b,c" || len(*reqs) != 2 {
		t.Fatalf("ids %v after %d requests", ids, len(*reqs))
	}
	if (*reqs)[0].Path != "/v1/searches/job%2F1/leads" {
		t.Fatalf("path %q", (*reqs)[0].Path)
	}
	if (*reqs)[1].Query != "page=2&page_size=2" {
		t.Fatalf("query %q", (*reqs)[1].Query)
	}
}

func TestWaitForCompletion(t *testing.T) {
	c, reqs := testServer(t, func(n int, _ seen) reply {
		if n < 3 {
			return reply{200, `{"id":"j","status":"scraping"}`, nil}
		}
		return reply{200, `{"id":"j","status":"completed","leads_found":9}`, nil}
	})
	s, err := c.Searches.WaitForCompletion(context.Background(), "j", WaitOptions{Interval: time.Millisecond})
	if err != nil || s.Status != SearchCompleted || s.LeadsFound != 9 || len(*reqs) != 3 {
		t.Fatalf("search %+v err %v after %d polls", s, err, len(*reqs))
	}
}

func TestOutreachListKeepsStatsAndSendSchedules(t *testing.T) {
	c, reqs := testServer(t, func(n int, _ seen) reply {
		if n == 1 {
			return reply{200, `{"items":[{"id":"m1","lead_id":"l1","channel":"email","message":"hi","created_at":"2026-01-02T03:04:05Z"}],"total":1,"page":1,"page_size":25,"stats":{"total":1,"draft":1}}`, nil}
		}
		return reply{202, `{"id":"m1","lead_id":"l1","channel":"email","message":"hi","created_at":"2026-01-02T03:04:05Z"}`, nil}
	})
	ctx := context.Background()
	page, err := c.Outreach.List(ctx, OutreachListParams{Status: OutreachDraft})
	if err != nil || len(page.Items) != 1 || page.Stats.Draft != 1 {
		t.Fatalf("page %+v err %v", page, err)
	}
	at := time.Date(2026, 3, 1, 9, 0, 0, 0, time.FixedZone("TRT", 3*3600))
	if _, err := c.Outreach.Send(ctx, "m1", at); err != nil {
		t.Fatal(err)
	}
	if got := (*reqs)[1].Body; got != `{"scheduled_at":"2026-03-01T06:00:00Z"}` {
		t.Fatalf("body %s", got)
	}
}

func TestCustomerIDIsPathEscaped(t *testing.T) {
	c, reqs := testServer(t, func(int, seen) reply { return reply{204, "", nil} })
	if err := c.Customers.RemoveLimit(context.Background(), "team/a b"); err != nil {
		t.Fatal(err)
	}
	if got := (*reqs)[0].Path; got != "/v1/customers/team%2Fa%20b/limit" {
		t.Fatalf("path %q", got)
	}
}
