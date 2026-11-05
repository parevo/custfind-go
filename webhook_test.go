package custfind

import (
	"errors"
	"testing"
	"time"
)

// Signed by the API's own shared.WebhookSignatureValue.
const (
	vectorBody   = `{"id":"evt_1","type":"search.completed","created_at":"2023-11-14T22:13:20Z","organization_id":"org_1","data":{"id":"job_1","status":"completed","leads_found":12}}`
	vectorHeader = "t=1700000000,v1=09714fde5144a52b4b8218ce4fbd186b9cb1fce94a75f5396bfbc54f6fbee486"
	vectorSecret = "whsec_test"
)

var vectorTime = time.Unix(1700000000, 0)

func TestWebhookVerifiesServerSignature(t *testing.T) {
	if err := verifyAt([]byte(vectorBody), vectorHeader, vectorSecret, 0, vectorTime.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	// A second v1 (secret rotation) is accepted when either matches.
	if err := verifyAt([]byte(vectorBody), "t=1700000000,v1=deadbeef, "+vectorHeader[13:], vectorSecret, 0, vectorTime); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookRejects(t *testing.T) {
	cases := map[string]struct {
		body, header, secret string
		now                  time.Time
		want                 error
	}{
		"tampered body": {vectorBody[:len(vectorBody)-2] + "3}}", vectorHeader, vectorSecret, vectorTime, ErrWebhookSignatureMismatch},
		"wrong secret":  {vectorBody, vectorHeader, "whsec_other", vectorTime, ErrWebhookSignatureMismatch},
		"too old":       {vectorBody, vectorHeader, vectorSecret, vectorTime.Add(6 * time.Minute), ErrWebhookTimestamp},
		"from future":   {vectorBody, vectorHeader, vectorSecret, vectorTime.Add(-6 * time.Minute), ErrWebhookTimestamp},
		"no header":     {vectorBody, "", vectorSecret, vectorTime, ErrWebhookMalformedSignature},
		"no v1":         {vectorBody, "t=1700000000", vectorSecret, vectorTime, ErrWebhookMalformedSignature},
	}
	for name, tc := range cases {
		if err := verifyAt([]byte(tc.body), tc.header, tc.secret, 0, tc.now); !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", name, err, tc.want)
		}
	}
}

func TestEventDecode(t *testing.T) {
	ev := WebhookEvent{}
	if err := ev.Decode(nil); err == nil {
		t.Fatal("empty data must not decode")
	}
	ev.Data = []byte(`{"id":"job_1","status":"completed","leads_found":12,"external_customer_id":null}`)
	var d SearchEventData
	if err := ev.Decode(&d); err != nil || d.LeadsFound != 12 || d.Status != SearchCompleted || d.ExternalCustomerID != nil {
		t.Fatalf("%+v %v", d, err)
	}
}
