# custfind-go

Official Go client for the [Custfind API](https://custfind.com/developers): start lead searches, read scored leads, write and send emails, move deals and receive webhooks. Standard library only; Go 1.23+.

```bash
go get github.com/parevo/custfind-go
```

```go
import custfind "github.com/parevo/custfind-go"

cf, err := custfind.NewClient() // reads CUSTFIND_API_KEY; or custfind.WithAPIKey(...)

created, err := cf.Searches.Create(ctx, custfind.CreateSearchParams{
	Sector:             "Dental clinic",
	Location:           "Kadıköy, Istanbul",
	Limit:              25,        // caps the credits this search reserves
	ExternalCustomerID: "acme-42", // optional: bill your own customer
})

_, err = cf.Searches.WaitForCompletion(ctx, created.JobID, custfind.WaitOptions{})
for lead, err := range cf.Searches.AllLeads(ctx, created.JobID, custfind.LeadListParams{MinScore: custfind.Ptr(70)}) {
	if err != nil {
		return err
	}
	fmt.Println(lead.Name, lead.Email, *lead.Score)
}
```

## Safe retries

Every write is sent with an `Idempotency-Key` that stays the same across retries. Network errors, `429` (respecting `Retry-After`), `5xx` and `409 idempotency_in_progress` are retried up to 3 times (`WithMaxRetries`); a timed-out search is never started twice. Pass your own key to make a retry across process restarts safe too:

```go
cf.Searches.Create(ctx, params, custfind.WithIdempotencyKey("search-"+orderID))
```

Every call takes a `context.Context`; cancelling it stops the call and its retries.

## Errors

Failed calls return `*custfind.Error` with `StatusCode` and the API's `Code`:

```go
var apiErr *custfind.Error
if errors.As(err, &apiErr) && apiErr.Code == custfind.CodeCustomerLimitReached { … }
if custfind.IsPaymentRequired(err) { … } // no credits, customer cap, or plan without API
if custfind.IsNotFound(err) { … }
```

## Webhooks

```go
http.HandleFunc("/webhooks/custfind", func(w http.ResponseWriter, r *http.Request) {
	payload, _ := io.ReadAll(r.Body)
	event, err := custfind.ConstructEvent(payload, r.Header.Get(custfind.SignatureHeader), os.Getenv("CUSTFIND_WEBHOOK_SECRET"))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	switch event.Type {
	case custfind.EventSearchCompleted:
		var d custfind.SearchEventData
		_ = event.Decode(&d)
	case custfind.EventLeadCreated:
		var d custfind.LeadCreatedData
		_ = event.Decode(&d)
	}
	w.WriteHeader(http.StatusOK)
})
```

Deliveries can repeat; use `event.ID` to handle each event once.

## Resources

`Searches`, `Leads`, `Outreach`, `Pipelines`, `Deals`, `Webhooks`, `Customers`, `SenderAccounts`, plus `Me`, `Balance` and `Usage`. See the [API reference](https://custfind.com/developers).

## Contributing

This repository is a read-only mirror of `sdks/go` in Custfind's main repository, updated on every release. Issues are welcome here; pull requests are read and ported by hand, then closed, since the next release overwrites this repository.

## Developing

This module is outside the repo's `go.work`, so it builds on its own:

```bash
GOWORK=off go test ./...
```
