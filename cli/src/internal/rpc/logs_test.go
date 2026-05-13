package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1"
	"github.com/jongio/azd-app/cli/src/gen/proto/azdapp/v1/azdappv1connect"
	"github.com/jongio/azd-app/cli/src/internal/dashboard/broadcast"
	"github.com/jongio/azd-app/cli/src/internal/service"
)

// fakeLogsStore implements LogsStore in-memory for handler tests. All
// methods are mutex-guarded so concurrent stream pumps and unary calls
// see consistent state. Subscribe channels are drained via Unsubscribe
// (closes the channel, matching service.LogBuffer semantics).
type fakeLogsStore struct {
	mu sync.Mutex

	// Buffered logs per service. nil means "service does not exist".
	buffers map[string][]service.LogEntry

	// Active subscribers per service. Each entry's pump goroutine reads
	// from the channel; broadcast() pushes to every channel for the
	// service. Unsubscribe closes and removes the channel.
	subs map[string][]chan service.LogEntry

	classifications []service.LogClassification
	classErr        error
	saveClassErr    error

	prefs    []byte
	prefsErr error
	saveErr  error
}

func newFakeStore() *fakeLogsStore {
	return &fakeLogsStore{
		buffers: map[string][]service.LogEntry{},
		subs:    map[string][]chan service.LogEntry{},
	}
}

func (f *fakeLogsStore) addBuffer(name string, entries ...service.LogEntry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.buffers[name] = append(f.buffers[name], entries...)
}

// broadcast pushes entry to every active subscriber for service `name`.
// Non-blocking; mirrors the slow-consumer-protection behaviour of the
// real LogBuffer (drop into the 100-cap channel).
func (f *fakeLogsStore) broadcast(name string, e service.LogEntry) {
	f.mu.Lock()
	subs := append([]chan service.LogEntry(nil), f.subs[name]...)
	f.mu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- e:
		default:
			// Subscriber is full. The Connect handler's ring sits
			// behind this channel and would normally drop-oldest;
			// dropping at the source matches LogBuffer.broadcast
			// behaviour (logbuffer.go:498).
		}
	}
}

func (f *fakeLogsStore) GetRecent(serviceName string, tail int) ([]service.LogEntry, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	entries, ok := f.buffers[serviceName]
	if !ok {
		return nil, false
	}
	if tail >= len(entries) {
		out := make([]service.LogEntry, len(entries))
		copy(out, entries)
		return out, true
	}
	out := make([]service.LogEntry, tail)
	copy(out, entries[len(entries)-tail:])
	return out, true
}

func (f *fakeLogsStore) GetAll(tail int) []service.LogEntry {
	f.mu.Lock()
	defer f.mu.Unlock()
	var all []service.LogEntry
	for _, entries := range f.buffers {
		all = append(all, entries...)
	}
	if tail < len(all) {
		all = all[len(all)-tail:]
	}
	return all
}

func (f *fakeLogsStore) ServiceNames() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, 0, len(f.buffers))
	for k := range f.buffers {
		out = append(out, k)
	}
	return out
}

func (f *fakeLogsStore) Subscribe(serviceName string) (chan service.LogEntry, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.buffers[serviceName]; !ok {
		return nil, false
	}
	ch := make(chan service.LogEntry, 100)
	f.subs[serviceName] = append(f.subs[serviceName], ch)
	return ch, true
}

func (f *fakeLogsStore) Unsubscribe(serviceName string, ch chan service.LogEntry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	subs := f.subs[serviceName]
	for i, s := range subs {
		if s == ch {
			f.subs[serviceName] = append(subs[:i], subs[i+1:]...)
			close(ch)
			return
		}
	}
}

func (f *fakeLogsStore) LoadClassifications() ([]service.LogClassification, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.classErr != nil {
		return nil, f.classErr
	}
	out := make([]service.LogClassification, len(f.classifications))
	copy(out, f.classifications)
	return out, nil
}

func (f *fakeLogsStore) SaveClassifications(c []service.LogClassification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveClassErr != nil {
		return f.saveClassErr
	}
	f.classifications = append([]service.LogClassification(nil), c...)
	return nil
}

func (f *fakeLogsStore) LoadPreferences() ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.prefsErr != nil {
		return nil, f.prefsErr
	}
	if f.prefs == nil {
		return nil, nil
	}
	out := make([]byte, len(f.prefs))
	copy(out, f.prefs)
	return out, nil
}

func (f *fakeLogsStore) SavePreferences(data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return f.saveErr
	}
	f.prefs = append([]byte(nil), data...)
	return nil
}

// newLogsTestServer wires a LogsHandler around the supplied store
// behind an httptest server so tests exercise the real Connect runtime
// (HTTP/1, header flush timing) rather than the in-process router.
// Streaming tests need the real wire to validate back-pressure paths.
func newLogsTestServer(t *testing.T, store LogsStore) (azdappv1connect.LogsServiceClient, func()) {
	t.Helper()
	mgr := broadcast.New()
	mux := http.NewServeMux()
	Mount(mux, Dependencies{
		Broadcast: mgr,
		Logs:      store,
	})
	srv := httptest.NewServer(mux)
	client := azdappv1connect.NewLogsServiceClient(srv.Client(), srv.URL)
	return client, func() {
		srv.Close()
		mgr.StopAll()
	}
}

// ---- GetLogs ----

func TestGetLogsReturnsEntriesForService(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.addBuffer(
		"api",
		service.LogEntry{Service: "api", Message: "one", Timestamp: now.Add(-2 * time.Second)},
		service.LogEntry{Service: "api", Message: "two", Timestamp: now.Add(-1 * time.Second)},
		service.LogEntry{Service: "api", Message: "three", Timestamp: now},
	)
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	resp, err := client.GetLogs(context.Background(), connect.NewRequest(&v1.GetLogsRequest{
		ServiceName: "api",
		Tail:        10,
	}))
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if got := len(resp.Msg.GetEntries()); got != 3 {
		t.Errorf("entries=%d want 3", got)
	}
	if resp.Msg.GetTailClamped() {
		t.Error("TailClamped=true; want false (tail 10 < requested cap)")
	}
	if resp.Msg.GetEntries()[0].GetMessage() != "one" {
		t.Errorf("first message=%q want one", resp.Msg.GetEntries()[0].GetMessage())
	}
}

func TestGetLogsTailClampedAt10000(t *testing.T) {
	store := newFakeStore()
	store.addBuffer("api", service.LogEntry{Service: "api", Message: "x"})
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	resp, err := client.GetLogs(context.Background(), connect.NewRequest(&v1.GetLogsRequest{
		ServiceName: "api",
		Tail:        50_000,
	}))
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if !resp.Msg.GetTailClamped() {
		t.Error("TailClamped=false; want true for tail=50_000")
	}
}

func TestGetLogsDefaultTailWhenZero(t *testing.T) {
	store := newFakeStore()
	store.addBuffer("api", service.LogEntry{Service: "api", Message: "x"})
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	// tail=0 is the proto default (also handleGetLogs treats 0 as 500).
	// Should not be flagged as clamped because the user didn't ask for
	// something we shrank.
	resp, err := client.GetLogs(context.Background(), connect.NewRequest(&v1.GetLogsRequest{
		ServiceName: "api",
		Tail:        0,
	}))
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if resp.Msg.GetTailClamped() {
		t.Error("TailClamped=true for tail=0; want false (default not a clamp)")
	}
}

func TestGetLogsMergedTailAcrossServices(t *testing.T) {
	store := newFakeStore()
	now := time.Now()
	store.addBuffer("api", service.LogEntry{Service: "api", Message: "a", Timestamp: now.Add(-1 * time.Second)})
	store.addBuffer("web", service.LogEntry{Service: "web", Message: "w", Timestamp: now})
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	resp, err := client.GetLogs(context.Background(), connect.NewRequest(&v1.GetLogsRequest{
		Tail: 10,
	}))
	if err != nil {
		t.Fatalf("GetLogs: %v", err)
	}
	if got := len(resp.Msg.GetEntries()); got != 2 {
		t.Errorf("entries=%d want 2", got)
	}
}

func TestGetLogsUnknownServiceReturnsNotFound(t *testing.T) {
	store := newFakeStore()
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	_, err := client.GetLogs(context.Background(), connect.NewRequest(&v1.GetLogsRequest{
		ServiceName: "ghost",
		Tail:        10,
	}))
	if err == nil {
		t.Fatal("GetLogs returned nil err for unknown service")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Errorf("code=%v want NotFound", got)
	}
}

// ---- Classifications ----

func TestListClassificationsEmpty(t *testing.T) {
	store := newFakeStore()
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	resp, err := client.ListClassifications(context.Background(),
		connect.NewRequest(&v1.ListClassificationsRequest{}))
	if err != nil {
		t.Fatalf("ListClassifications: %v", err)
	}
	if got := len(resp.Msg.GetClassifications()); got != 0 {
		t.Errorf("classifications=%d want 0", got)
	}
}

func TestListClassificationsPopulated(t *testing.T) {
	store := newFakeStore()
	store.classifications = []service.LogClassification{
		{Text: "Connection refused", Level: "error"},
		{Text: "cache miss", Level: "info"},
	}
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	resp, err := client.ListClassifications(context.Background(),
		connect.NewRequest(&v1.ListClassificationsRequest{}))
	if err != nil {
		t.Fatalf("ListClassifications: %v", err)
	}
	got := resp.Msg.GetClassifications()
	if len(got) != 2 {
		t.Fatalf("classifications=%d want 2", len(got))
	}
	if got[0].GetText() != "Connection refused" || got[0].GetLevel() != v1.LogLevel_LOG_LEVEL_ERROR {
		t.Errorf("got[0]=%+v", got[0])
	}
	if got[1].GetLevel() != v1.LogLevel_LOG_LEVEL_INFO {
		t.Errorf("got[1].level=%v want info", got[1].GetLevel())
	}
}

func TestListClassificationsLoadErrorReturnsInternal(t *testing.T) {
	store := newFakeStore()
	store.classErr = errors.New("yaml parse failure")
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	_, err := client.ListClassifications(context.Background(),
		connect.NewRequest(&v1.ListClassificationsRequest{}))
	if err == nil {
		t.Fatal("ListClassifications: nil err want Internal")
	}
	if got := connect.CodeOf(err); got != connect.CodeInternal {
		t.Errorf("code=%v want Internal", got)
	}
}

func TestAddClassificationAppendsNew(t *testing.T) {
	store := newFakeStore()
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	resp, err := client.AddClassification(context.Background(),
		connect.NewRequest(&v1.AddClassificationRequest{
			Classification: &v1.Classification{
				Text:  "new error",
				Level: v1.LogLevel_LOG_LEVEL_ERROR,
			},
		}))
	if err != nil {
		t.Fatalf("AddClassification: %v", err)
	}
	if resp.Msg.GetClassification().GetText() != "new error" {
		t.Errorf("text=%q", resp.Msg.GetClassification().GetText())
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.classifications) != 1 {
		t.Errorf("stored=%d want 1", len(store.classifications))
	}
	if store.classifications[0].Level != "error" {
		t.Errorf("stored level=%q want error", store.classifications[0].Level)
	}
}

func TestAddClassificationUpdatesExistingCaseInsensitive(t *testing.T) {
	store := newFakeStore()
	store.classifications = []service.LogClassification{
		{Text: "Connection Refused", Level: "info"},
	}
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	// Send same text with different casing; level must update in-place.
	resp, err := client.AddClassification(context.Background(),
		connect.NewRequest(&v1.AddClassificationRequest{
			Classification: &v1.Classification{
				Text:  "connection refused",
				Level: v1.LogLevel_LOG_LEVEL_ERROR,
			},
		}))
	if err != nil {
		t.Fatalf("AddClassification: %v", err)
	}
	// Response keeps the ORIGINAL stored text casing (per
	// handleCreateClassification: only the level is updated).
	if got := resp.Msg.GetClassification().GetText(); got != "Connection Refused" {
		t.Errorf("response text=%q want preserved 'Connection Refused'", got)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.classifications) != 1 {
		t.Errorf("stored=%d want 1 (in-place update, not append)", len(store.classifications))
	}
	if store.classifications[0].Level != "error" {
		t.Errorf("level=%q want error", store.classifications[0].Level)
	}
}

func TestAddClassificationRejectsEmptyText(t *testing.T) {
	store := newFakeStore()
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	_, err := client.AddClassification(context.Background(),
		connect.NewRequest(&v1.AddClassificationRequest{
			Classification: &v1.Classification{Text: "   ", Level: v1.LogLevel_LOG_LEVEL_ERROR},
		}))
	if err == nil {
		t.Fatal("nil err for blank text")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("code=%v want InvalidArgument", got)
	}
}

func TestAddClassificationRejectsInvalidLevel(t *testing.T) {
	store := newFakeStore()
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	// LOG_LEVEL_DEBUG is a valid LogEntry level but NOT a valid
	// classification level. Server must reject.
	_, err := client.AddClassification(context.Background(),
		connect.NewRequest(&v1.AddClassificationRequest{
			Classification: &v1.Classification{Text: "x", Level: v1.LogLevel_LOG_LEVEL_DEBUG},
		}))
	if err == nil {
		t.Fatal("nil err for debug-level classification")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("code=%v want InvalidArgument", got)
	}
}

func TestDeleteClassificationRemovesByIndex(t *testing.T) {
	store := newFakeStore()
	store.classifications = []service.LogClassification{
		{Text: "first", Level: "info"},
		{Text: "second", Level: "error"},
	}
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	_, err := client.DeleteClassification(context.Background(),
		connect.NewRequest(&v1.DeleteClassificationRequest{Index: 0}))
	if err != nil {
		t.Fatalf("DeleteClassification: %v", err)
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.classifications) != 1 || store.classifications[0].Text != "second" {
		t.Errorf("after delete: %+v", store.classifications)
	}
}

func TestDeleteClassificationNegativeIndexInvalidArgument(t *testing.T) {
	store := newFakeStore()
	store.classifications = []service.LogClassification{{Text: "x", Level: "info"}}
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	_, err := client.DeleteClassification(context.Background(),
		connect.NewRequest(&v1.DeleteClassificationRequest{Index: -1}))
	if err == nil {
		t.Fatal("nil err for index=-1")
	}
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("code=%v want InvalidArgument", got)
	}
}

func TestDeleteClassificationOutOfRangeReturnsNotFound(t *testing.T) {
	store := newFakeStore()
	store.classifications = []service.LogClassification{{Text: "x", Level: "info"}}
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	_, err := client.DeleteClassification(context.Background(),
		connect.NewRequest(&v1.DeleteClassificationRequest{Index: 5}))
	if err == nil {
		t.Fatal("nil err for out-of-range index")
	}
	if got := connect.CodeOf(err); got != connect.CodeNotFound {
		t.Errorf("code=%v want NotFound", got)
	}
}

// ---- Preferences ----

func TestGetPreferencesReturnsDefaultsWhenEmpty(t *testing.T) {
	store := newFakeStore()
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	resp, err := client.GetPreferences(context.Background(),
		connect.NewRequest(&v1.GetPreferencesRequest{}))
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}
	prefs := resp.Msg.GetPreferences()
	if prefs.GetVersion() != "1.0" {
		t.Errorf("version=%q want 1.0", prefs.GetVersion())
	}
	if prefs.GetUi().GetGridColumns() != 2 {
		t.Errorf("grid_columns=%d want 2", prefs.GetUi().GetGridColumns())
	}
	if !prefs.GetBehavior().GetAutoScroll() {
		t.Error("auto_scroll=false; default should be true")
	}
	if prefs.GetCopy().GetDefaultFormat() != "plaintext" {
		t.Errorf("default_format=%q want plaintext", prefs.GetCopy().GetDefaultFormat())
	}
}

func TestGetPreferencesReadsStoredBlob(t *testing.T) {
	store := newFakeStore()
	// Hand-write a JSON blob in the protojson camelCase shape so we can
	// assert GetPreferences decodes it correctly.
	store.prefs = []byte(`{
		"version": "1.0",
		"theme": "dark",
		"ui": {"gridColumns": 4, "gridAutoFit": true, "viewMode": "unified", "selectedServices": ["api"]},
		"behavior": {"autoScroll": false, "pauseOnScroll": true, "timestampFormat": "iso"},
		"copy": {"defaultFormat": "json", "includeTimestamp": false, "includeService": true}
	}`)
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	resp, err := client.GetPreferences(context.Background(),
		connect.NewRequest(&v1.GetPreferencesRequest{}))
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}
	prefs := resp.Msg.GetPreferences()
	if prefs.GetTheme() != "dark" {
		t.Errorf("theme=%q want dark", prefs.GetTheme())
	}
	if !prefs.GetUi().GetGridAutoFit() {
		t.Error("grid_auto_fit=false; want true (round-trip)")
	}
	if prefs.GetUi().GetGridColumns() != 4 {
		t.Errorf("grid_columns=%d want 4", prefs.GetUi().GetGridColumns())
	}
	if prefs.GetBehavior().GetAutoScroll() {
		t.Error("auto_scroll=true; want false")
	}
	if prefs.GetCopy().GetDefaultFormat() != "json" {
		t.Errorf("default_format=%q want json", prefs.GetCopy().GetDefaultFormat())
	}
}

func TestGetPreferencesDiscardsUnknownFields(t *testing.T) {
	store := newFakeStore()
	// Blob from a "future" schema version with extra keys plus extra
	// nested keys. DiscardUnknown=true on the unmarshal MUST keep the
	// known fields and silently drop the unknown ones rather than
	// failing the read.
	store.prefs = []byte(`{
		"version": "2.0",
		"theme": "light",
		"futureField": "ignored",
		"ui": {"gridColumns": 3, "futureUiField": 99},
		"behavior": {"autoScroll": true, "newKnob": "ignored"},
		"copy": {"defaultFormat": "csv"}
	}`)
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	resp, err := client.GetPreferences(context.Background(),
		connect.NewRequest(&v1.GetPreferencesRequest{}))
	if err != nil {
		t.Fatalf("GetPreferences (with unknown fields) failed: %v", err)
	}
	prefs := resp.Msg.GetPreferences()
	if prefs.GetVersion() != "2.0" {
		t.Errorf("version=%q want 2.0", prefs.GetVersion())
	}
	if prefs.GetUi().GetGridColumns() != 3 {
		t.Errorf("grid_columns=%d want 3", prefs.GetUi().GetGridColumns())
	}
	if prefs.GetCopy().GetDefaultFormat() != "csv" {
		t.Errorf("default_format=%q want csv", prefs.GetCopy().GetDefaultFormat())
	}
}

func TestGetPreferencesCorruptBlobFallsBackToDefaults(t *testing.T) {
	store := newFakeStore()
	store.prefs = []byte(`{"this is": not valid json`)
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	resp, err := client.GetPreferences(context.Background(),
		connect.NewRequest(&v1.GetPreferencesRequest{}))
	if err != nil {
		t.Fatalf("GetPreferences: %v", err)
	}
	if got := resp.Msg.GetPreferences().GetVersion(); got != "1.0" {
		t.Errorf("version=%q want 1.0 (defaults)", got)
	}
}

func TestSavePreferencesRoundTripsThemeAndGridAutoFit(t *testing.T) {
	store := newFakeStore()
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	in := &v1.Preferences{
		Version: "1.0",
		Theme:   "dark",
		Ui: &v1.UIPreferences{
			GridColumns:      6,
			GridAutoFit:      true,
			ViewMode:         "grid",
			SelectedServices: []string{"api", "web"},
		},
		Behavior: &v1.BehaviorPreferences{AutoScroll: true, TimestampFormat: "hh:mm"},
		Copy:     &v1.CopyPreferences{DefaultFormat: "markdown", IncludeTimestamp: true},
	}

	resp, err := client.SavePreferences(context.Background(),
		connect.NewRequest(&v1.SavePreferencesRequest{Preferences: in}))
	if err != nil {
		t.Fatalf("SavePreferences: %v", err)
	}
	if resp.Msg.GetPreferences().GetTheme() != "dark" {
		t.Errorf("response theme=%q", resp.Msg.GetPreferences().GetTheme())
	}

	// Verify the persisted blob retained the theme + gridAutoFit fields
	// (this is the exact regression the proto-level rewrite exists to
	// fix - the legacy Go struct silently dropped both).
	store.mu.Lock()
	stored := append([]byte(nil), store.prefs...)
	store.mu.Unlock()

	var raw map[string]any
	if err := json.Unmarshal(stored, &raw); err != nil {
		t.Fatalf("stored blob not valid JSON: %v\nblob=%s", err, stored)
	}
	if got := raw["theme"]; got != "dark" {
		t.Errorf("stored theme=%v want dark", got)
	}
	uiMap, ok := raw["ui"].(map[string]any)
	if !ok {
		t.Fatalf("ui key missing or wrong type: %T", raw["ui"])
	}
	if got := uiMap["gridAutoFit"]; got != true {
		t.Errorf("stored gridAutoFit=%v want true", got)
	}
	// Verify it round-trips: a follow-up Get returns the same theme.
	resp2, err := client.GetPreferences(context.Background(),
		connect.NewRequest(&v1.GetPreferencesRequest{}))
	if err != nil {
		t.Fatalf("GetPreferences after save: %v", err)
	}
	if resp2.Msg.GetPreferences().GetTheme() != "dark" {
		t.Errorf("round-trip theme=%q want dark", resp2.Msg.GetPreferences().GetTheme())
	}
	if !resp2.Msg.GetPreferences().GetUi().GetGridAutoFit() {
		t.Error("round-trip grid_auto_fit=false; want true")
	}
}

// ---- StreamLocalLogs ----

func TestStreamLocalLogsReceivesEntries(t *testing.T) {
	store := newFakeStore()
	store.addBuffer("api")
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Goroutine-driven stimulus per lifecycle_test.go pattern: HTTP/1
	// won't flush headers until the first Send, so we drive emits from
	// a goroutine that waits for the server to subscribe.
	go func() {
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			store.mu.Lock()
			n := len(store.subs["api"])
			store.mu.Unlock()
			if n > 0 {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		store.broadcast("api", service.LogEntry{
			Service:   "api",
			Message:   "hello",
			Level:     service.LogLevelInfo,
			Timestamp: time.Now(),
		})
	}()

	stream, err := client.StreamLocalLogs(ctx, connect.NewRequest(&v1.StreamLocalLogsRequest{
		ServiceName: "api",
	}))
	if err != nil {
		t.Fatalf("StreamLocalLogs: %v", err)
	}

	if !stream.Receive() {
		t.Fatalf("Receive: %v", stream.Err())
	}
	got := stream.Msg().GetEntry()
	if got == nil {
		t.Fatalf("expected entry, got %+v", stream.Msg())
	}
	if got.GetMessage() != "hello" {
		t.Errorf("message=%q want hello", got.GetMessage())
	}
}

func TestStreamLocalLogsEmitsBackfill(t *testing.T) {
	store := newFakeStore()
	store.addBuffer(
		"api",
		service.LogEntry{Service: "api", Message: "old1", Timestamp: time.Now().Add(-2 * time.Second)},
		service.LogEntry{Service: "api", Message: "old2", Timestamp: time.Now().Add(-1 * time.Second)},
	)
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.StreamLocalLogs(ctx, connect.NewRequest(&v1.StreamLocalLogsRequest{
		ServiceName: "api",
		Backfill:    10,
	}))
	if err != nil {
		t.Fatalf("StreamLocalLogs: %v", err)
	}

	// Two backfill entries should arrive without any goroutine stimulus
	// because they're sent before the main loop blocks.
	for i, want := range []string{"old1", "old2"} {
		if !stream.Receive() {
			t.Fatalf("Receive[%d]: %v", i, stream.Err())
		}
		if got := stream.Msg().GetEntry().GetMessage(); got != want {
			t.Errorf("backfill[%d]=%q want %q", i, got, want)
		}
	}
}

func TestStreamLocalLogsUnknownServiceReturnsNotFound(t *testing.T) {
	store := newFakeStore()
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	stream, err := client.StreamLocalLogs(context.Background(),
		connect.NewRequest(&v1.StreamLocalLogsRequest{ServiceName: "ghost"}))
	if err != nil {
		// Some Connect transports surface errors before stream open;
		// either path is acceptable.
		if got := connect.CodeOf(err); got != connect.CodeNotFound {
			t.Errorf("code=%v want NotFound", got)
		}
		return
	}
	if stream.Receive() {
		t.Fatal("Receive returned true on unknown-service stream")
	}
	rerr := stream.Err()
	if rerr == nil {
		t.Fatal("stream.Err() nil; want NotFound")
	}
	if got := connect.CodeOf(rerr); got != connect.CodeNotFound {
		t.Errorf("code=%v want NotFound; err=%v", got, rerr)
	}
}

func TestStreamLocalLogsClientCancelExitsCleanly(t *testing.T) {
	store := newFakeStore()
	store.addBuffer("api")
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		// Wait for subscribe, then cancel.
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			store.mu.Lock()
			n := len(store.subs["api"])
			store.mu.Unlock()
			if n > 0 {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	}()

	stream, err := client.StreamLocalLogs(ctx, connect.NewRequest(&v1.StreamLocalLogsRequest{
		ServiceName: "api",
	}))
	if err != nil {
		// Cancel before headers flushed is OK.
		return
	}
	if stream.Receive() {
		_ = stream // drain a frame if one arrives before cancel takes effect
	}
	rerr := stream.Err()
	if rerr != nil &&
		!strings.Contains(strings.ToLower(rerr.Error()), "cancel") &&
		connect.CodeOf(rerr) != connect.CodeCanceled {
		t.Logf("stream err on cancel (acceptable): %v", rerr)
	}

	// Verify subscriber was released.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		store.mu.Lock()
		n := len(store.subs["api"])
		store.mu.Unlock()
		if n == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Error("subscriber not released after cancel")
}

// ---- localLogRing unit tests (drop-oldest semantics) ----

func TestRingDropsOldestWhenFull(t *testing.T) {
	r := newLocalLogRing(3)
	r.push(&v1.LogEntry{Message: "1"})
	r.push(&v1.LogEntry{Message: "2"})
	r.push(&v1.LogEntry{Message: "3"})
	r.push(&v1.LogEntry{Message: "4"}) // drops "1"
	r.push(&v1.LogEntry{Message: "5"}) // drops "2"

	got, dropped := r.drain()
	if dropped != 2 {
		t.Errorf("dropped=%d want 2", dropped)
	}
	if len(got) != 3 {
		t.Fatalf("len=%d want 3", len(got))
	}
	for i, want := range []string{"3", "4", "5"} {
		if got[i].GetMessage() != want {
			t.Errorf("[%d]=%q want %q", i, got[i].GetMessage(), want)
		}
	}
}

func TestRingNotifyCoalesces(t *testing.T) {
	r := newLocalLogRing(10)
	for i := 0; i < 100; i++ {
		r.push(&v1.LogEntry{Message: "x"})
	}
	// Exactly one signal should be pending despite 100 pushes.
	select {
	case <-r.notify:
	default:
		t.Fatal("expected one pending notify")
	}
	select {
	case <-r.notify:
		t.Fatal("expected at most one notify (coalesced)")
	default:
	}
}

func TestRingDrainEmpty(t *testing.T) {
	r := newLocalLogRing(5)
	got, dropped := r.drain()
	if got != nil || dropped != 0 {
		t.Errorf("empty drain: got=%v dropped=%d", got, dropped)
	}
}

// ---- Preferences round-trip via dashboard.UserPreferences shape ----

// TestSavePreferencesRoundTripsViaDashboardCompatibleJSON guards the
// contract that the Connect handler writes JSON the legacy REST handler
// could still partially read (drops theme + gridAutoFit per current
// dashboard.UserPreferences struct, but doesn't fail). This keeps the
// parallel-stack window safe: a Connect write does not break a REST
// read of unrelated keys.
func TestSavePreferencesRoundTripsViaDashboardCompatibleJSON(t *testing.T) {
	store := newFakeStore()
	client, cleanup := newLogsTestServer(t, store)
	defer cleanup()

	in := &v1.Preferences{
		Version: "1.0",
		Theme:   "system",
		Ui:      &v1.UIPreferences{GridColumns: 3, ViewMode: "grid"},
		Behavior: &v1.BehaviorPreferences{
			AutoScroll: true, TimestampFormat: "hh:mm:ss.sss",
		},
		Copy: &v1.CopyPreferences{DefaultFormat: "plaintext"},
	}
	if _, err := client.SavePreferences(context.Background(),
		connect.NewRequest(&v1.SavePreferencesRequest{Preferences: in})); err != nil {
		t.Fatalf("SavePreferences: %v", err)
	}

	store.mu.Lock()
	stored := append([]byte(nil), store.prefs...)
	store.mu.Unlock()

	// The legacy Go struct uses the same camelCase JSON tags
	// (logs_config.go) so a json.Unmarshal into UserPreferences should
	// succeed and pick up the keys it knows about.
	type uiCheck struct {
		GridColumns int    `json:"gridColumns"`
		ViewMode    string `json:"viewMode"`
	}
	type prefsCheck struct {
		Version string  `json:"version"`
		UI      uiCheck `json:"ui"`
	}
	var got prefsCheck
	if err := json.Unmarshal(stored, &got); err != nil {
		t.Fatalf("legacy decoder rejected Connect-written blob: %v\nblob=%s", err, stored)
	}
	if got.Version != "1.0" {
		t.Errorf("version=%q want 1.0", got.Version)
	}
	if got.UI.GridColumns != 3 {
		t.Errorf("ui.gridColumns=%d want 3", got.UI.GridColumns)
	}
	if got.UI.ViewMode != "grid" {
		t.Errorf("ui.viewMode=%q want grid", got.UI.ViewMode)
	}
}
