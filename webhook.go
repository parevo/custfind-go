package custfind

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

// SignatureHeader is the header every webhook delivery is signed in:
// "t=<unix seconds>,v1=<hex HMAC-SHA256 of "<t>.<body>">".
const SignatureHeader = "Custfind-Signature"

// DefaultWebhookTolerance is how old a signed timestamp ConstructEvent accepts.
const DefaultWebhookTolerance = 5 * time.Minute

// Webhook verification errors.
var (
	ErrWebhookMalformedSignature = errors.New("custfind: malformed Custfind-Signature header")
	ErrWebhookTimestamp          = errors.New("custfind: webhook timestamp outside tolerance")
	ErrWebhookSignatureMismatch  = errors.New("custfind: webhook signature mismatch")
)

// VerifyWebhookSignature checks a Custfind-Signature header against the raw
// request body and your endpoint's whsec_ secret. tolerance 0 means
// DefaultWebhookTolerance.
func VerifyWebhookSignature(payload []byte, header, secret string, tolerance time.Duration) error {
	return verifyAt(payload, header, secret, tolerance, time.Now())
}

func verifyAt(payload []byte, header, secret string, tolerance time.Duration, now time.Time) error {
	if tolerance <= 0 {
		tolerance = DefaultWebhookTolerance
	}
	var ts int64
	var sigs []string
	for _, part := range strings.Split(header, ",") {
		k, v, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch k {
		case "t":
			ts, _ = strconv.ParseInt(v, 10, 64)
		case "v1":
			sigs = append(sigs, v)
		}
	}
	if ts == 0 || len(sigs) == 0 {
		return ErrWebhookMalformedSignature
	}
	if d := now.Sub(time.Unix(ts, 0)); d > tolerance || d < -tolerance {
		return ErrWebhookTimestamp
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10) + "."))
	mac.Write(payload)
	want := []byte(hex.EncodeToString(mac.Sum(nil)))
	for _, s := range sigs {
		if hmac.Equal([]byte(s), want) {
			return nil
		}
	}
	return ErrWebhookSignatureMismatch
}

// ConstructEvent verifies a webhook delivery and returns its event. Pass the
// raw body exactly as received, before any JSON parsing:
//
//	payload, _ := io.ReadAll(r.Body)
//	event, err := custfind.ConstructEvent(payload, r.Header.Get(custfind.SignatureHeader), secret)
//	if err != nil { w.WriteHeader(400); return }
//	if event.Type == custfind.EventSearchCompleted {
//		var d custfind.SearchEventData
//		_ = event.Decode(&d)
//	}
//
// Deliveries can repeat; use event.ID to handle each event once.
func ConstructEvent(payload []byte, header, secret string) (*WebhookEvent, error) {
	if err := VerifyWebhookSignature(payload, header, secret, 0); err != nil {
		return nil, err
	}
	var ev WebhookEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return nil, err
	}
	return &ev, nil
}
