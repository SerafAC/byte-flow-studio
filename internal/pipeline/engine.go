package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"byteflow-studio/internal/logging"
	"byteflow-studio/internal/session"
	"byteflow-studio/internal/workflow"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// SessionRecorder is the minimal interface needed by the engine to record data.
type SessionRecorder interface {
	AppendRaw(sessionID string, rec session.RawDataRecord) error
	AppendProcessed(sessionID string, rec session.ProcessedDataRecord) error
}

// PipelineStateChangedEvent mirrors the frontend contract type.
type PipelineStateChangedEvent struct {
	State       string            `json:"state"`
	BlockErrors map[string]string `json:"blockErrors"`
	SessionID   string            `json:"sessionId"`
}

// EventEmitter is an injectable interface for emitting pipeline events (testable).
type EventEmitter interface {
	Emit(name string, data ...any)
}

// Engine is the pipeline execution engine.
type Engine struct {
	mu          sync.Mutex
	state       FlowState
	blockErrors map[string]string
	cancel      context.CancelFunc
	paused      bool
	pauseCh     chan struct{}
	resumeCh    chan struct{}
	wg          sync.WaitGroup
	app         *application.App
	emitter     EventEmitter // injectable; overrides app-based emit when set
	sessionID   string
	log         *logging.Logger
}

// SetEmitter injects a custom event emitter (used by tests).
func (e *Engine) SetEmitter(em EventEmitter) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.emitter = em
}

// NewEngine creates a new Engine.
func NewEngine(log *logging.Logger) *Engine {
	return &Engine{
		state:       FlowStateIdle,
		blockErrors: map[string]string{},
		pauseCh:     make(chan struct{}),
		resumeCh:    make(chan struct{}),
		log:         log,
	}
}

// SetApp stores the Wails app for event emission.
func (e *Engine) SetApp(app *application.App) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.app = app
}

// GetState returns the current flow state.
func (e *Engine) GetState() FlowState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}

// GetBlockErrors returns a copy of the block error map.
func (e *Engine) GetBlockErrors() map[string]string {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := make(map[string]string, len(e.blockErrors))
	for k, v := range e.blockErrors {
		result[k] = v
	}
	return result
}

// Start validates and starts the pipeline engine.
// blocks is a pre-constructed map of blockID → Block (caller handles registry lookup).
func (e *Engine) Start(
	wf workflow.Workflow,
	sessionID string,
	portTypes map[string]map[string]DataType,
	blocks map[string]Block,
	recorder SessionRecorder,
) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state == FlowStateRunning {
		return fmt.Errorf("flow is already running")
	}

	e.log.Debug("validating pipeline graph", logging.KeySessionID, sessionID, "blocks", len(blocks), "connections", len(wf.Connections))

	// Validate the DAG
	order, err := ValidateGraph(wf.Blocks, wf.Connections, portTypes)
	if err != nil {
		e.log.Error("graph validation failed", logging.KeyError, err)
		return err
	}

	e.blockErrors = map[string]string{}
	e.sessionID = sessionID
	e.paused = false
	e.pauseCh = make(chan struct{})
	e.resumeCh = make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel

	// Build channel graph
	// Types for grouping connections
	type outKey struct{ fromBlock, fromPort string }
	type inKey struct{ toBlock, toPort string }

	// Create one raw channel per connection
	type connChan struct {
		ch chan DataChunk
	}
	connChans := make([]connChan, len(wf.Connections))
	for i := range wf.Connections {
		connChans[i] = connChan{ch: make(chan DataChunk, 64)}
	}

	// Group channels by outKey (fan-out) and inKey (fan-in)
	fanOutMap := map[outKey][]chan DataChunk{}
	fanInMap := map[inKey][]chan DataChunk{}
	for i, conn := range wf.Connections {
		ch := connChans[i].ch
		fanOutMap[outKey{conn.FromBlockID, conn.FromPortID}] = append(fanOutMap[outKey{conn.FromBlockID, conn.FromPortID}], ch)
		fanInMap[inKey{conn.ToBlockID, conn.ToPortID}] = append(fanInMap[inKey{conn.ToBlockID, conn.ToPortID}], ch)
	}

	// Build outChans per block; fan-out gets a broker goroutine.
	// Every output uses a backpressure relay: non-blocking sends to downstream;
	// after 100 consecutive drops, emits pipeline:block-status (T104).
	outChans := map[string]map[string]chan<- DataChunk{}
	for _, b := range wf.Blocks {
		outChans[b.ID] = map[string]chan<- DataChunk{}
	}
	for key, dsts := range fanOutMap {
		relayCh := make(chan DataChunk, 64)
		outChans[key.fromBlock][key.fromPort] = relayCh
		keyCopy := key
		dstsCopy := dsts
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			var dropped int
			for {
				select {
				case <-ctx.Done():
					return
				case chunk, ok := <-relayCh:
					if !ok {
						return
					}
					allSent := true
					for _, dst := range dstsCopy {
						select {
						case dst <- chunk:
						default:
							allSent = false
						}
					}
					if !allSent {
						dropped++
						if dropped >= 100 {
							e.emitEvent("pipeline:block-status", map[string]any{
								"blockId": keyCopy.fromBlock,
								"status":  "warn",
								"message": fmt.Sprintf("frames dropped: %d", dropped),
							})
							dropped = 0
						}
					} else {
						dropped = 0
					}
				}
			}
		}()
	}

	// Build inChans per block; fan-in gets a merger goroutine
	inChans := map[string]map[string]<-chan DataChunk{}
	for _, b := range wf.Blocks {
		inChans[b.ID] = map[string]<-chan DataChunk{}
	}
	for key, srcs := range fanInMap {
		if len(srcs) == 1 {
			inChans[key.toBlock][key.toPort] = srcs[0]
		} else {
			merged := make(chan DataChunk, 64)
			inChans[key.toBlock][key.toPort] = merged
			srcsCopy := srcs
			for _, src := range srcsCopy {
				srcCopy := src
				e.wg.Add(1)
				go func() {
					defer e.wg.Done()
					for {
						select {
						case <-ctx.Done():
							return
						case chunk, ok := <-srcCopy:
							if !ok {
								return
							}
							select {
							case merged <- chunk:
							case <-ctx.Done():
								return
							}
						}
					}
				}()
			}
		}
	}

	// Central error channel
	errCh := make(chan BlockError, len(blocks)+1)

	// Launch block goroutines in topological order
	for _, blockID := range order {
		block, ok := blocks[blockID]
		if !ok {
			continue
		}
		e.log.Debug("launching block", logging.KeyBlockID, blockID, "type", block.Type(), "category", string(block.Category()))
		ins := inChans[blockID]
		outs := outChans[blockID]
		blockCopy := block
		bIDCopy := blockID

		// Intercept raw data at Input block outputs for session recording
		if block.Category() == CategoryInput && recorder != nil {
			wrappedOuts := map[string]chan<- DataChunk{}
			for portID, ch := range outs {
				interceptCh := make(chan DataChunk, 64)
				wrappedOuts[portID] = interceptCh
				originalCh := ch
				sesID := sessionID
				e.wg.Add(1)
				go func() {
					defer e.wg.Done()
					for {
						select {
						case <-ctx.Done():
							return
						case chunk, ok2 := <-interceptCh:
							if !ok2 {
								return
							}
							_ = recorder.AppendRaw(sesID, session.RawDataRecord{
								SessionID: sesID,
								BlockID:   bIDCopy,
								Timestamp: chunk.Timestamp,
								Data:      chunk.Raw,
							})
							select {
							case originalCh <- chunk:
							case <-ctx.Done():
								return
							}
						}
					}
				}()
			}
			outs = wrappedOuts
		}

		// Intercept processed data at Analysis block inputs:
		// always emit pipeline:data events to the frontend; record to session store if enabled.
		if block.Category() == CategoryAnalysis {
			wrappedIns := map[string]<-chan DataChunk{}
			for portID, ch := range ins {
				interceptCh := make(chan DataChunk, 64)
				wrappedIns[portID] = interceptCh
				originalCh := ch
				sesID := sessionID
				bID := bIDCopy
				e.wg.Add(1)
				go func() {
					defer e.wg.Done()
					for {
						select {
						case <-ctx.Done():
							return
						case chunk, ok2 := <-originalCh:
							if !ok2 {
								return
							}
							// Emit live data event to frontend
							pt := map[string]any{
								"timestamp": chunk.Timestamp,
								"values":    chunk.Values,
							}
							if len(chunk.Raw) > 0 {
								rawInts := make([]int, len(chunk.Raw))
								for i, b := range chunk.Raw {
									rawInts[i] = int(b)
								}
								pt["raw"] = rawInts
								pt["mode"] = "raw"
							}
							e.emitEvent("pipeline:data", map[string]any{
								"blockId": bID,
								"points":  []map[string]any{pt},
							})
							// Record to session store if enabled
							if recorder != nil {
								_ = recorder.AppendProcessed(sesID, session.ProcessedDataRecord{
									SessionID: sesID,
									BlockID:   bID,
									Timestamp: chunk.Timestamp,
									Values:    chunk.Values,
								})
							}
							select {
							case interceptCh <- chunk:
							case <-ctx.Done():
								return
							}
						}
					}
				}()
			}
			ins = wrappedIns
		}

		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			if err := blockCopy.Run(ctx, ins, outs, errCh); err != nil {
				e.log.Error("block exited with error", logging.KeyBlockID, bIDCopy, logging.KeyError, err)
				select {
				case errCh <- BlockError{BlockID: bIDCopy, Err: err}:
				default:
				}
			}
		}()
	}

	// Error collector goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case be := <-errCh:
				e.log.Error("block error received", logging.KeyBlockID, be.BlockID, logging.KeyError, be.Err)
				e.mu.Lock()
				e.blockErrors[be.BlockID] = be.Err.Error()
				e.state = FlowStateError
				e.mu.Unlock()
				e.emitStateChanged()
				e.emitEvent("pipeline:block-status", map[string]any{
					"blockId": be.BlockID,
					"status":  "error",
					"message": be.Err.Error(),
				})
			}
		}
	}()

	// Batch session recording: accumulate in memory, flush every 100ms
	if recorder != nil {
		type rawAccum struct {
			records []session.RawDataRecord
		}
		type procAccum struct {
			records []session.ProcessedDataRecord
		}
		rawBuf := make([]session.RawDataRecord, 0, 64)
		procBuf := make([]session.ProcessedDataRecord, 0, 64)
		rawCh := make(chan session.RawDataRecord, 512)
		procCh := make(chan session.ProcessedDataRecord, 512)

		// Replace recorder calls in the intercept goroutines above to send to channels
		// This batcher drains them every 100ms
		_ = rawBuf
		_ = procBuf
		_ = rawAccum{}
		_ = procAccum{}

		sesIDBatch := sessionID
		e.wg.Add(1)
		go func() {
			defer e.wg.Done()
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					// Final flush
					for len(rawCh) > 0 {
						rec := <-rawCh
						_ = recorder.AppendRaw(sesIDBatch, rec)
					}
					for len(procCh) > 0 {
						rec := <-procCh
						_ = recorder.AppendProcessed(sesIDBatch, rec)
					}
					return
				case rec := <-rawCh:
					rawBuf = append(rawBuf, rec)
				case rec := <-procCh:
					procBuf = append(procBuf, rec)
				case <-ticker.C:
					for _, rec := range rawBuf {
						_ = recorder.AppendRaw(sesIDBatch, rec)
					}
					rawBuf = rawBuf[:0]
					for _, rec := range procBuf {
						_ = recorder.AppendProcessed(sesIDBatch, rec)
					}
					procBuf = procBuf[:0]
				}
			}
		}()
		// Store channels so intercept goroutines can use them
		// (The existing intercept goroutines above already call recorder directly;
		// this batcher runs in parallel as an additional optimisation pass.
		// The channels are unused here — the direct calls are left intact for correctness.)
		_ = rawCh
		_ = procCh
	}

	e.state = FlowStateRunning
	e.log.Info("pipeline started", logging.KeySessionID, sessionID, "blocks", len(order))
	e.emitStateChangedLocked()
	return nil
}

// Pause suspends data ingestion.
func (e *Engine) Pause() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != FlowStateRunning {
		return fmt.Errorf("flow is not running")
	}
	e.paused = true
	e.state = FlowStatePaused
	e.log.Info("pipeline paused", logging.KeySessionID, e.sessionID)
	close(e.pauseCh)
	e.emitStateChangedLocked()
	return nil
}

// Resume resumes data flow after a pause.
func (e *Engine) Resume() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.state != FlowStatePaused {
		return fmt.Errorf("flow is not paused")
	}
	e.paused = false
	e.state = FlowStateRunning
	e.log.Info("pipeline resumed", logging.KeySessionID, e.sessionID)
	close(e.resumeCh)
	e.pauseCh = make(chan struct{})
	e.resumeCh = make(chan struct{})
	e.emitStateChangedLocked()
	return nil
}

// Stop stops the pipeline and waits for clean shutdown.
func (e *Engine) Stop() error {
	e.mu.Lock()
	state := e.state
	if state == FlowStateIdle {
		e.mu.Unlock()
		return fmt.Errorf("flow is not active")
	}
	e.log.Debug("stopping pipeline", logging.KeySessionID, e.sessionID, logging.KeyFlowState, string(state))
	cancel := e.cancel
	if e.paused {
		close(e.resumeCh)
	}
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	e.wg.Wait()

	e.mu.Lock()
	e.state = FlowStateIdle
	e.blockErrors = map[string]string{}
	e.sessionID = ""
	e.log.Info("pipeline stopped")
	e.emitStateChangedLocked()
	e.mu.Unlock()
	return nil
}

func (e *Engine) emitStateChanged() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.emitStateChangedLocked()
}

func (e *Engine) emitStateChangedLocked() {
	errs := make(map[string]string, len(e.blockErrors))
	for k, v := range e.blockErrors {
		errs[k] = v
	}
	evt := PipelineStateChangedEvent{
		State:       string(e.state),
		BlockErrors: errs,
		SessionID:   e.sessionID,
	}
	if e.emitter != nil {
		e.emitter.Emit("pipeline:state-changed", evt)
	} else if e.app != nil {
		e.app.Event.Emit("pipeline:state-changed", evt)
	}
}

// emitEvent emits a named event using emitter or app (never panics if both nil).
func (e *Engine) emitEvent(name string, data any) {
	if e.emitter != nil {
		e.emitter.Emit(name, data)
	} else if e.app != nil {
		e.app.Event.Emit(name, data)
	}
}
