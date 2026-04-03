package pipeline

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"byteflow-studio/internal/session"
	"byteflow-studio/internal/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmitter captures emitted events for assertions in tests.
type mockEmitter struct {
	mu     sync.Mutex
	events []struct{ name string; data any }
}

func (m *mockEmitter) Emit(name string, data ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var d any
	if len(data) > 0 {
		d = data[0]
	}
	m.events = append(m.events, struct{ name string; data any }{name, d})
}

func (m *mockEmitter) findEvent(name, msgSubstr string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ev := range m.events {
		if ev.name != name {
			continue
		}
		if m2, ok := ev.data.(map[string]any); ok {
			if msg, ok2 := m2["message"].(string); ok2 && strings.Contains(msg, msgSubstr) {
				return true
			}
		}
	}
	return false
}

// stubInputBlock emits n DataChunks then blocks until context is cancelled.
type stubInputBlock struct {
	id     string
	chunks []DataChunk
}

func (s *stubInputBlock) ID() string                                { return s.id }
func (s *stubInputBlock) Type() string                              { return "stub-input" }
func (s *stubInputBlock) Category() BlockCategory                   { return CategoryInput }
func (s *stubInputBlock) Configure(map[string]any) error            { return nil }
func (s *stubInputBlock) InputPorts() []Port                        { return nil }
func (s *stubInputBlock) OutputPorts() []Port {
	return []Port{{ID: "out", Direction: PortDirOutput, DataType: DataTypeNum}}
}
func (s *stubInputBlock) Run(ctx context.Context, _ map[string]<-chan DataChunk, outputs map[string]chan<- DataChunk, _ chan<- BlockError) error {
	out := outputs["out"]
	for _, chunk := range s.chunks {
		select {
		case out <- chunk:
		case <-ctx.Done():
			return nil
		}
	}
	<-ctx.Done()
	return nil
}

// stubAnalysisBlock receives chunks and records them (mutex-protected for race safety).
type stubAnalysisBlock struct {
	id       string
	mu       sync.Mutex
	received []DataChunk
}

func (s *stubAnalysisBlock) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.received)
}

func (s *stubAnalysisBlock) ID() string                                { return s.id }
func (s *stubAnalysisBlock) Type() string                              { return "stub-analysis" }
func (s *stubAnalysisBlock) Category() BlockCategory                   { return CategoryAnalysis }
func (s *stubAnalysisBlock) Configure(map[string]any) error            { return nil }
func (s *stubAnalysisBlock) OutputPorts() []Port                       { return nil }
func (s *stubAnalysisBlock) InputPorts() []Port {
	return []Port{{ID: "in", Direction: PortDirInput, DataType: DataTypeNum}}
}
func (s *stubAnalysisBlock) Run(ctx context.Context, inputs map[string]<-chan DataChunk, _ map[string]chan<- DataChunk, _ chan<- BlockError) error {
	in := inputs["in"]
	for {
		select {
		case chunk, ok := <-in:
			if !ok {
				return nil
			}
			s.mu.Lock()
			s.received = append(s.received, chunk)
			s.mu.Unlock()
		case <-ctx.Done():
			return nil
		}
	}
}

// stubProcessingBlock passes chunks through unchanged.
type stubProcessingBlock struct {
	id string
}

func (s *stubProcessingBlock) ID() string                                { return s.id }
func (s *stubProcessingBlock) Type() string                              { return "stub-processing" }
func (s *stubProcessingBlock) Category() BlockCategory                   { return CategoryProcessing }
func (s *stubProcessingBlock) Configure(map[string]any) error            { return nil }
func (s *stubProcessingBlock) InputPorts() []Port {
	return []Port{{ID: "in", Direction: PortDirInput, DataType: DataTypeNum}}
}
func (s *stubProcessingBlock) OutputPorts() []Port {
	return []Port{{ID: "out", Direction: PortDirOutput, DataType: DataTypeNum}}
}
func (s *stubProcessingBlock) Run(ctx context.Context, inputs map[string]<-chan DataChunk, outputs map[string]chan<- DataChunk, _ chan<- BlockError) error {
	in := inputs["in"]
	out := outputs["out"]
	for {
		select {
		case chunk, ok := <-in:
			if !ok {
				return nil
			}
			select {
			case out <- chunk:
			case <-ctx.Done():
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// noopRecorder implements SessionRecorder but does nothing.
type noopRecorder struct{}

func (n *noopRecorder) AppendRaw(_ string, _ session.RawDataRecord) error      { return nil }
func (n *noopRecorder) AppendProcessed(_ string, _ session.ProcessedDataRecord) error { return nil }

func makeTestWorkflow(blocks []workflow.BlockDef, conns []workflow.ConnectionDef) workflow.Workflow {
	return workflow.Workflow{ID: "wf-test", Blocks: blocks, Connections: conns}
}

func TestEngine2BlockRun(t *testing.T) {
	engine := NewEngine(nil)

	chunks := []DataChunk{
		{Timestamp: 1, SourceID: "in1", Values: []float64{1}},
		{Timestamp: 2, SourceID: "in1", Values: []float64{2}},
		{Timestamp: 3, SourceID: "in1", Values: []float64{3}},
		{Timestamp: 4, SourceID: "in1", Values: []float64{4}},
		{Timestamp: 5, SourceID: "in1", Values: []float64{5}},
	}
	inBlock := &stubInputBlock{id: "in1", chunks: chunks}
	anBlock := &stubAnalysisBlock{id: "an1"}

	wf := makeTestWorkflow(
		[]workflow.BlockDef{
			{ID: "in1", Type: "stub-input", Category: "input"},
			{ID: "an1", Type: "stub-analysis", Category: "analysis"},
		},
		[]workflow.ConnectionDef{
			{ID: "c1", FromBlockID: "in1", FromPortID: "out", ToBlockID: "an1", ToPortID: "in"},
		},
	)
	portTypes := map[string]map[string]DataType{
		"in1": {"out": DataTypeNum},
		"an1": {"in": DataTypeNum},
	}
	blocks := map[string]Block{"in1": inBlock, "an1": anBlock}

	err := engine.Start(wf, "sess-1", portTypes, blocks, &noopRecorder{})
	require.NoError(t, err)
	assert.Equal(t, FlowStateRunning, engine.GetState())

	// Wait for all 5 chunks to arrive
	deadline := time.After(500 * time.Millisecond)
	for {
		if anBlock.count() >= 5 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("only received %d chunks before timeout", anBlock.count())
		case <-time.After(10 * time.Millisecond):
		}
	}
	assert.Equal(t, 5, anBlock.count())

	err = engine.Stop()
	require.NoError(t, err)
	assert.Equal(t, FlowStateIdle, engine.GetState())
}

func TestEngineFanOut(t *testing.T) {
	engine := NewEngine(nil)

	const numChunks = 5
	chunks := make([]DataChunk, numChunks)
	for i := range chunks {
		chunks[i] = DataChunk{Timestamp: int64(i + 1), SourceID: "in1", Values: []float64{float64(i + 1)}}
	}

	inBlock := &stubInputBlock{id: "in1", chunks: chunks}
	procBlock := &stubProcessingBlock{id: "proc1"}
	an1 := &stubAnalysisBlock{id: "an1"}
	an2 := &stubAnalysisBlock{id: "an2"}

	wf := makeTestWorkflow(
		[]workflow.BlockDef{
			{ID: "in1", Type: "stub-input", Category: "input"},
			{ID: "proc1", Type: "stub-processing", Category: "processing"},
			{ID: "an1", Type: "stub-analysis", Category: "analysis"},
			{ID: "an2", Type: "stub-analysis", Category: "analysis"},
		},
		[]workflow.ConnectionDef{
			{ID: "c1", FromBlockID: "in1", FromPortID: "out", ToBlockID: "proc1", ToPortID: "in"},
			{ID: "c2", FromBlockID: "proc1", FromPortID: "out", ToBlockID: "an1", ToPortID: "in"},
			{ID: "c3", FromBlockID: "proc1", FromPortID: "out", ToBlockID: "an2", ToPortID: "in"},
		},
	)
	portTypes := map[string]map[string]DataType{
		"in1":   {"out": DataTypeNum},
		"proc1": {"in": DataTypeNum, "out": DataTypeNum},
		"an1":   {"in": DataTypeNum},
		"an2":   {"in": DataTypeNum},
	}
	blocks := map[string]Block{"in1": inBlock, "proc1": procBlock, "an1": an1, "an2": an2}

	err := engine.Start(wf, "sess-fanout", portTypes, blocks, &noopRecorder{})
	require.NoError(t, err)

	deadline := time.After(500 * time.Millisecond)
	for {
		if an1.count() >= numChunks && an2.count() >= numChunks {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("fan-out: an1 received %d, an2 received %d (want %d each)", an1.count(), an2.count(), numChunks)
		case <-time.After(10 * time.Millisecond):
		}
	}

	assert.Equal(t, numChunks, an1.count(), "an1 should receive all chunks")
	assert.Equal(t, numChunks, an2.count(), "an2 should receive all chunks")

	require.NoError(t, engine.Stop())
}

func TestEngineFanIn(t *testing.T) {
	engine := NewEngine(nil)

	const chunksPerInput = 3
	makeChunks := func(sourceID string) []DataChunk {
		chunks := make([]DataChunk, chunksPerInput)
		for i := range chunks {
			chunks[i] = DataChunk{Timestamp: int64(i + 1), SourceID: sourceID, Values: []float64{float64(i + 1)}}
		}
		return chunks
	}

	in1 := &stubInputBlock{id: "in1", chunks: makeChunks("in1")}
	in2 := &stubInputBlock{id: "in2", chunks: makeChunks("in2")}
	proc := &stubProcessingBlock{id: "proc1"}
	an1 := &stubAnalysisBlock{id: "an1"}

	wf := makeTestWorkflow(
		[]workflow.BlockDef{
			{ID: "in1", Type: "stub-input", Category: "input"},
			{ID: "in2", Type: "stub-input", Category: "input"},
			{ID: "proc1", Type: "stub-processing", Category: "processing"},
			{ID: "an1", Type: "stub-analysis", Category: "analysis"},
		},
		[]workflow.ConnectionDef{
			{ID: "c1", FromBlockID: "in1", FromPortID: "out", ToBlockID: "proc1", ToPortID: "in"},
			{ID: "c2", FromBlockID: "in2", FromPortID: "out", ToBlockID: "proc1", ToPortID: "in"},
			{ID: "c3", FromBlockID: "proc1", FromPortID: "out", ToBlockID: "an1", ToPortID: "in"},
		},
	)
	portTypes := map[string]map[string]DataType{
		"in1":   {"out": DataTypeNum},
		"in2":   {"out": DataTypeNum},
		"proc1": {"in": DataTypeNum, "out": DataTypeNum},
		"an1":   {"in": DataTypeNum},
	}
	blocks := map[string]Block{"in1": in1, "in2": in2, "proc1": proc, "an1": an1}

	err := engine.Start(wf, "sess-fanin", portTypes, blocks, &noopRecorder{})
	require.NoError(t, err)

	total := chunksPerInput * 2
	deadline := time.After(500 * time.Millisecond)
	for {
		if an1.count() >= total {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("fan-in: an1 received %d (want %d)", an1.count(), total)
		case <-time.After(10 * time.Millisecond):
		}
	}

	assert.Equal(t, total, an1.count(), "fan-in: should receive chunks from both inputs")
	require.NoError(t, engine.Stop())
}

func TestEngineContextCancel(t *testing.T) {
	engine := NewEngine(nil)

	inBlock := &stubInputBlock{id: "in1", chunks: nil}
	anBlock := &stubAnalysisBlock{id: "an1"}

	wf := makeTestWorkflow(
		[]workflow.BlockDef{
			{ID: "in1", Type: "stub-input", Category: "input"},
			{ID: "an1", Type: "stub-analysis", Category: "analysis"},
		},
		[]workflow.ConnectionDef{
			{ID: "c1", FromBlockID: "in1", FromPortID: "out", ToBlockID: "an1", ToPortID: "in"},
		},
	)
	portTypes := map[string]map[string]DataType{
		"in1": {"out": DataTypeNum},
		"an1": {"in": DataTypeNum},
	}
	blocks := map[string]Block{"in1": inBlock, "an1": anBlock}

	err := engine.Start(wf, "sess-1", portTypes, blocks, &noopRecorder{})
	require.NoError(t, err)

	err = engine.Stop()
	require.NoError(t, err)
	assert.Equal(t, FlowStateIdle, engine.GetState())
}

// slowAnalysisBlock blocks on reads for blockDuration to simulate backpressure.
type slowAnalysisBlock struct {
	id           string
	blockDuration time.Duration
}

func (s *slowAnalysisBlock) ID() string              { return s.id }
func (s *slowAnalysisBlock) Type() string            { return "slow-analysis" }
func (s *slowAnalysisBlock) Category() BlockCategory { return CategoryAnalysis }
func (s *slowAnalysisBlock) Configure(map[string]any) error { return nil }
func (s *slowAnalysisBlock) OutputPorts() []Port             { return nil }
func (s *slowAnalysisBlock) InputPorts() []Port {
	return []Port{{ID: "in", Direction: PortDirInput, DataType: DataTypeNum}}
}
func (s *slowAnalysisBlock) Run(ctx context.Context, inputs map[string]<-chan DataChunk, _ map[string]chan<- DataChunk, _ chan<- BlockError) error {
	in := inputs["in"]
	for {
		select {
		case _, ok := <-in:
			if !ok {
				return nil
			}
			// Simulate slow processing — creates backpressure
			select {
			case <-time.After(s.blockDuration):
			case <-ctx.Done():
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// fastInputBlock floods chunks as fast as possible (no channel blocking on its end).
type fastInputBlock struct {
	id     string
	count  int
	sentCh chan struct{}
}

func (f *fastInputBlock) ID() string              { return f.id }
func (f *fastInputBlock) Type() string            { return "fast-input" }
func (f *fastInputBlock) Category() BlockCategory { return CategoryInput }
func (f *fastInputBlock) Configure(map[string]any) error { return nil }
func (f *fastInputBlock) InputPorts() []Port              { return nil }
func (f *fastInputBlock) OutputPorts() []Port {
	return []Port{{ID: "out", Direction: PortDirOutput, DataType: DataTypeNum}}
}
func (f *fastInputBlock) Run(ctx context.Context, _ map[string]<-chan DataChunk, outputs map[string]chan<- DataChunk, _ chan<- BlockError) error {
	out := outputs["out"]
	for i := 0; i < f.count; i++ {
		chunk := DataChunk{Timestamp: int64(i), SourceID: f.id, Values: []float64{float64(i)}}
		select {
		case out <- chunk:
		case <-ctx.Done():
			return nil
		}
	}
	if f.sentCh != nil {
		close(f.sentCh)
	}
	<-ctx.Done()
	return nil
}

// TestEngineBackpressureFrameDrop verifies that when a downstream block is slow,
// the engine drops frames after 100 consecutive drops and emits a block-status event
// with "frames dropped: 100" (T104a — test written before T104 implementation).
func TestEngineBackpressureFrameDrop(t *testing.T) {
	engine := NewEngine(nil)
	emitter := &mockEmitter{}
	engine.SetEmitter(emitter)

	// Fast input: sends 500 chunks; slow analysis: processes one per 50ms
	sentCh := make(chan struct{})
	inBlock := &fastInputBlock{id: "in1", count: 500, sentCh: sentCh}
	anBlock := &slowAnalysisBlock{id: "an1", blockDuration: 50 * time.Millisecond}

	wf := makeTestWorkflow(
		[]workflow.BlockDef{
			{ID: "in1", Type: "fast-input", Category: "input"},
			{ID: "an1", Type: "slow-analysis", Category: "analysis"},
		},
		[]workflow.ConnectionDef{
			{ID: "c1", FromBlockID: "in1", FromPortID: "out", ToBlockID: "an1", ToPortID: "in"},
		},
	)
	portTypes := map[string]map[string]DataType{
		"in1": {"out": DataTypeNum},
		"an1": {"in": DataTypeNum},
	}
	blocks := map[string]Block{"in1": inBlock, "an1": anBlock}

	err := engine.Start(wf, "sess-bp", portTypes, blocks, &noopRecorder{})
	require.NoError(t, err)

	// Wait for input to finish sending
	select {
	case <-sentCh:
	case <-time.After(2 * time.Second):
		t.Fatal("fast input did not finish sending")
	}

	// Give engine time to detect drops and emit event
	time.Sleep(200 * time.Millisecond)

	// T104 implementation will emit "frames dropped: 100" when 100 consecutive drops happen.
	// With current implementation (no backpressure), this assertion will FAIL (TDD red phase).
	found := emitter.findEvent("pipeline:block-status", "frames dropped: 100")
	assert.True(t, found, "expected pipeline:block-status event with 'frames dropped: 100' after backpressure; got events: %v", emitter.events)

	require.NoError(t, engine.Stop())
}

// stubRawInputBlock emits DataChunks with Raw bytes (no Values).
type stubRawInputBlock struct {
	id     string
	chunks []DataChunk
}

func (s *stubRawInputBlock) ID() string                                { return s.id }
func (s *stubRawInputBlock) Type() string                              { return "stub-raw-input" }
func (s *stubRawInputBlock) Category() BlockCategory                   { return CategoryInput }
func (s *stubRawInputBlock) Configure(map[string]any) error            { return nil }
func (s *stubRawInputBlock) InputPorts() []Port                        { return nil }
func (s *stubRawInputBlock) OutputPorts() []Port {
	return []Port{{ID: "out", Direction: PortDirOutput, DataType: DataTypeRaw}}
}
func (s *stubRawInputBlock) Run(ctx context.Context, _ map[string]<-chan DataChunk, outputs map[string]chan<- DataChunk, _ chan<- BlockError) error {
	out := outputs["out"]
	for _, chunk := range s.chunks {
		select {
		case out <- chunk:
		case <-ctx.Done():
			return nil
		}
	}
	<-ctx.Done()
	return nil
}

// stubRawAnalysisBlock receives raw chunks.
type stubRawAnalysisBlock struct {
	id       string
	mu       sync.Mutex
	received []DataChunk
}

func (s *stubRawAnalysisBlock) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.received)
}

func (s *stubRawAnalysisBlock) ID() string                                { return s.id }
func (s *stubRawAnalysisBlock) Type() string                              { return "stub-raw-analysis" }
func (s *stubRawAnalysisBlock) Category() BlockCategory                   { return CategoryAnalysis }
func (s *stubRawAnalysisBlock) Configure(map[string]any) error            { return nil }
func (s *stubRawAnalysisBlock) OutputPorts() []Port                       { return nil }
func (s *stubRawAnalysisBlock) InputPorts() []Port {
	return []Port{{ID: "in", Direction: PortDirInput, DataType: DataTypeRaw}}
}
func (s *stubRawAnalysisBlock) Run(ctx context.Context, inputs map[string]<-chan DataChunk, _ map[string]chan<- DataChunk, _ chan<- BlockError) error {
	in := inputs["in"]
	for {
		select {
		case chunk, ok := <-in:
			if !ok {
				return nil
			}
			s.mu.Lock()
			s.received = append(s.received, chunk)
			s.mu.Unlock()
		case <-ctx.Done():
			return nil
		}
	}
}

// TestEngineAnalysisInterceptRawData verifies that the engine intercept emits
// pipeline:data events with raw and mode fields for raw byte data.
func TestEngineAnalysisInterceptRawData(t *testing.T) {
	engine := NewEngine(nil)
	emitter := &mockEmitter{}
	engine.SetEmitter(emitter)

	chunks := []DataChunk{
		{Timestamp: 1, SourceID: "in1", Raw: []byte{0x0A, 0xFF, 0x10}},
		{Timestamp: 2, SourceID: "in1", Raw: []byte{0xDE, 0xAD}},
	}
	inBlock := &stubRawInputBlock{id: "in1", chunks: chunks}
	anBlock := &stubRawAnalysisBlock{id: "an1"}

	wf := makeTestWorkflow(
		[]workflow.BlockDef{
			{ID: "in1", Type: "stub-raw-input", Category: "input"},
			{ID: "an1", Type: "stub-raw-analysis", Category: "analysis"},
		},
		[]workflow.ConnectionDef{
			{ID: "c1", FromBlockID: "in1", FromPortID: "out", ToBlockID: "an1", ToPortID: "in"},
		},
	)
	portTypes := map[string]map[string]DataType{
		"in1": {"out": DataTypeRaw},
		"an1": {"in": DataTypeRaw},
	}
	blocks := map[string]Block{"in1": inBlock, "an1": anBlock}

	err := engine.Start(wf, "sess-raw", portTypes, blocks, &noopRecorder{})
	require.NoError(t, err)

	// Wait for chunks to arrive at the analysis block
	deadline := time.After(500 * time.Millisecond)
	for {
		if anBlock.count() >= 2 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("only received %d chunks before timeout", anBlock.count())
		case <-time.After(10 * time.Millisecond):
		}
	}

	require.NoError(t, engine.Stop())

	// Verify emitted pipeline:data events contain raw and mode fields
	emitter.mu.Lock()
	defer emitter.mu.Unlock()

	var rawEvents []map[string]any
	for _, ev := range emitter.events {
		if ev.name != "pipeline:data" {
			continue
		}
		m, ok := ev.data.(map[string]any)
		if !ok {
			continue
		}
		if m["blockId"] != "an1" {
			continue
		}
		points, ok := m["points"].([]map[string]any)
		if !ok || len(points) == 0 {
			continue
		}
		pt := points[0]
		if pt["mode"] == "raw" {
			rawEvents = append(rawEvents, pt)
		}
	}

	require.Len(t, rawEvents, 2, "expected 2 raw pipeline:data events")

	// First chunk: [0x0A, 0xFF, 0x10] → []int{10, 255, 16}
	raw0, ok := rawEvents[0]["raw"].([]int)
	require.True(t, ok, "raw field should be []int")
	assert.Equal(t, []int{10, 255, 16}, raw0)

	// Second chunk: [0xDE, 0xAD] → []int{222, 173}
	raw1, ok := rawEvents[1]["raw"].([]int)
	require.True(t, ok, "raw field should be []int")
	assert.Equal(t, []int{222, 173}, raw1)
}

// TestEngineMultiplePauseResumeCycles verifies that multiple Pause→Resume cycles do not panic
// with "close of closed channel" (regression test for the resumeCh double-close bug).
func TestEngineMultiplePauseResumeCycles(t *testing.T) {
	engine := NewEngine(nil)

	inBlock := &stubInputBlock{id: "in1", chunks: nil}
	anBlock := &stubAnalysisBlock{id: "an1"}

	wf := makeTestWorkflow(
		[]workflow.BlockDef{
			{ID: "in1", Type: "stub-input", Category: "input"},
			{ID: "an1", Type: "stub-analysis", Category: "analysis"},
		},
		[]workflow.ConnectionDef{
			{ID: "c1", FromBlockID: "in1", FromPortID: "out", ToBlockID: "an1", ToPortID: "in"},
		},
	)
	portTypes := map[string]map[string]DataType{
		"in1": {"out": DataTypeNum},
		"an1": {"in": DataTypeNum},
	}
	blocks := map[string]Block{"in1": inBlock, "an1": anBlock}

	require.NoError(t, engine.Start(wf, "sess-pr", portTypes, blocks, &noopRecorder{}))
	assert.Equal(t, FlowStateRunning, engine.GetState())

	// Three full Pause→Resume cycles — each Resume used to panic on the second iteration.
	for i := 0; i < 3; i++ {
		require.NoError(t, engine.Pause(), "Pause cycle %d", i)
		assert.Equal(t, FlowStatePaused, engine.GetState())

		require.NoError(t, engine.Resume(), "Resume cycle %d", i)
		assert.Equal(t, FlowStateRunning, engine.GetState())
	}

	require.NoError(t, engine.Stop())
	assert.Equal(t, FlowStateIdle, engine.GetState())
}
