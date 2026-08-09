# Connected Provider Routing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent a disconnected historical provider from masking a connected provider for the same model.

**Architecture:** Treat relay connectivity as an eligibility input in the OpenAI handler before cost calculation and `router.Select`. Preserve the existing post-selection connectivity check as protection against disconnect races.

**Tech Stack:** Go, `net/http`, `httptest`, `coder/websocket`, Myference relay hub and router.

---

### Task 1: Reproduce connected-provider fallthrough

**Files:**
- Test: `server/internal/api/openai_integration_test.go`

- [ ] **Step 1: Write the failing test**

Add `TestOpenAIRoutesPastDisconnectedCandidate` using a real relay hub with `machine-live` connected. Return two equally eligible Qwen candidates in ranking order: `machine-disconnected` first and `machine-live` second. Have the live provider accept the job and finish with one output chunk. Assert the API returns `200`, the response contains the live output, and the reservation records `machine-live`.

```go
func TestOpenAIRoutesPastDisconnectedCandidate(t *testing.T) {
	hub := relay.NewHub(func(context.Context, string) (string, error) { return "machine-live", nil }, relay.Options{HeartbeatTimeout: time.Second})
	relayServer := httptest.NewTLSServer(hub)
	defer relayServer.Close()
	relayClient := relayServer.Client()
	relayClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}
	provider, _, err := websocket.Dial(context.Background(), "wss"+strings.TrimPrefix(relayServer.URL, "https"), &websocket.DialOptions{HTTPClient: relayClient, HTTPHeader: http.Header{"Authorization": {"Bearer machine-token"}}})
	if err != nil { t.Fatal(err) }
	defer provider.Close(websocket.StatusNormalClosure, "")
	requireRelayConnection(t, hub, "machine-live")

	reserved := make(chan Reservation, 1)
	handler := NewOpenAI(Dependencies{
		Hub: hub,
		Authorize: func(context.Context, string, string, string, uint64) (Principal, error) {
			return Principal{AccountID: "account", SessionID: "session", SessionBalance: 1_000_000}, nil
		},
		Candidates: func(context.Context, string) ([]router.Candidate, error) {
			candidate := router.Candidate{OfferID: "offer", Model: "qwen", Capabilities: []string{"text", "stream"}, ConfirmedBond: true, Healthy: true, Capacity: 1, PriceVersion: 1, InputPerMillion: "1", OutputPerMillion: "1", ComputePerSecond: "1"}
			disconnected, connected := candidate, candidate
			disconnected.MachineID = "machine-disconnected"
			connected.MachineID = "machine-live"
			return []router.Candidate{disconnected, connected}, nil
		},
		Reserve: func(_ context.Context, reservation Reservation) error { reserved <- reservation; return nil },
		Transition: func(context.Context, string, string) error { return nil },
		Abort: func(context.Context, string, string) error { return nil },
		Persist: func(context.Context, Proposal) error { return nil },
	})

	go func() {
		_, payload, readErr := provider.Read(context.Background())
		if readErr != nil { return }
		var envelope v1.Envelope
		if json.Unmarshal(payload, &envelope) != nil { return }
		var offer v1.JobOffer
		if envelope.DecodeBody(&offer) != nil { return }
		writeProvider(t, provider, "accept-live", v1.MessageJobAccept, &v1.JobAccept{RequestID: offer.RequestID})
		writeProvider(t, provider, "output-live", v1.MessageOutputChunk, &v1.OutputChunk{RequestID: offer.RequestID, Sequence: 1, Data: "live", Done: true, InputTokens: 1, OutputTokens: 1, ComputeMilliseconds: 1})
	}()

	request := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"qwen","stream":true,"max_completion_tokens":1,"messages":[{"role":"user","content":"hello"}]}`))
	request.Header.Set("Authorization", "Bearer api-token")
	request.Header.Set("X-Myference-Max-Spend", "1000000")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"content":"live"`) {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
	if reservation := <-reserved; reservation.MachineID != "machine-live" {
		t.Fatalf("reservation=%+v", reservation)
	}
}
```

- [ ] **Step 2: Run the regression and verify red**

Run:

```bash
go test ./server/internal/api -run '^TestOpenAIRoutesPastDisconnectedCandidate$' -count=1 -v
```

Expected: `FAIL`; the current handler selects `machine-disconnected` and returns `503 no eligible provider`.

### Task 2: Filter disconnected candidates before ranking

**Files:**
- Modify: `server/internal/api/openai.go`
- Test: `server/internal/api/openai_integration_test.go`

- [ ] **Step 1: Implement the minimal eligibility change**

At the start of the existing candidate cost loop, add:

```go
if !h.dependencies.Hub.Connected(candidates[index].MachineID) {
    candidates[index].Capacity = 0
    continue
}
```

This lets the existing router reject disconnected candidates without changing ranking or weakening any other predicate.

- [ ] **Step 2: Run the regression and verify green**

Run:

```bash
go test ./server/internal/api -run '^TestOpenAIRoutesPastDisconnectedCandidate$' -count=1 -v
```

Expected: `PASS`; the reservation and output come from `machine-live`.

- [ ] **Step 3: Run focused package coverage**

Run:

```bash
go test ./server/internal/api ./server/internal/router -count=1
```

Expected: both packages pass.

### Task 3: Verify and deploy

**Files:**
- No additional code files.

- [ ] **Step 1: Run repository verification**

Run:

```bash
make verify
```

Expected: Go tests/vet/build, contract tests, web tests/lint/build, and script validation all pass.

- [ ] **Step 2: Commit and push**

```bash
git add server/internal/api/openai.go server/internal/api/openai_integration_test.go docs/superpowers/plans/2026-08-10-connected-provider-routing.md
git commit -m "fix: route past disconnected providers"
git push origin main
```

- [ ] **Step 3: Validate deployment**

Wait for the GitHub `verify` workflow and Render deployment. Confirm the public Qwen model still reports the live laptop, then retry the playground request and verify it no longer returns `no eligible provider`.
