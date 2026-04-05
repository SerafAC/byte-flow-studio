// Package services provides Wails v3 services exposed to the frontend.
package services

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"byteflow-studio/internal/logging"
	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"
	"byteflow-studio/internal/workflow"

	"github.com/google/uuid"
	"go.bug.st/serial/enumerator"
)

// BlockTypeDescriptor describes a registered block type for the frontend catalogue.
type BlockTypeDescriptor struct {
	Type          string                `json:"type"`
	Category      pipeline.BlockCategory `json:"category"`
	Label         string                `json:"label"`
	Description   string                `json:"description"`
	DefaultParams map[string]any        `json:"defaultParams"`
}

// SerialPortInfo describes a serial port available on the machine.
type SerialPortInfo struct {
	Name        string `json:"name"`
	VendorID    string `json:"vendorId"`
	ProductID   string `json:"productId"`
	Description string `json:"description"`
}

// WorkflowService manages the workflow definition exposed to the frontend.
type WorkflowService struct {
	store   *workflow.Store
	current *workflow.Workflow
	engine  PipelineStateGetter
	log     *logging.Logger
}

// PipelineStateGetter is a minimal interface to check pipeline state.
type PipelineStateGetter interface {
	GetState() pipeline.FlowState
}

// NewWorkflowService creates a WorkflowService.
func NewWorkflowService(store *workflow.Store, log *logging.Logger) *WorkflowService {
	wf := &workflow.Workflow{
		ID:            uuid.New().String(),
		Name:          "Untitled Workflow",
		CreatedAt:     time.Now().UnixMilli(),
		UpdatedAt:     time.Now().UnixMilli(),
		Blocks:        []workflow.BlockDef{},
		Connections:   []workflow.ConnectionDef{},
		SessionConfig: workflow.DefaultSessionConfig(),
	}
	return &WorkflowService{store: store, current: wf, log: log}
}

// SetEngine sets the pipeline state getter (to enforce editing locks).
func (s *WorkflowService) SetEngine(e PipelineStateGetter) {
	s.engine = e
}

// GetWorkflow returns the currently loaded workflow.
func (s *WorkflowService) GetWorkflow() (workflow.Workflow, error) {
	if s.current == nil {
		return workflow.Workflow{}, fmt.Errorf("no workflow loaded")
	}
	return *s.current, nil
}

// SaveWorkflow persists the current workflow to a .byteflow file.
func (s *WorkflowService) SaveWorkflow(path string) error {
	if !strings.HasSuffix(path, ".byteflow") {
		return fmt.Errorf("path must end in .byteflow")
	}
	s.log.Debug("saving workflow", "path", path, "workflow_id", s.current.ID)
	fileStore, err := workflow.OpenStore(path)
	if err != nil {
		s.log.Error("failed to open workflow file for save", logging.KeyError, err, "path", path)
		return fmt.Errorf("failed to write workflow file: %w", err)
	}
	defer fileStore.Close()
	if err := fileStore.SaveWorkflow(s.current); err != nil {
		s.log.Error("failed to save workflow", logging.KeyError, err, "path", path)
		return fmt.Errorf("failed to write workflow file: %w", err)
	}
	s.log.Info("workflow saved", "path", path, "workflow_id", s.current.ID)
	return nil
}

// LoadWorkflow loads a workflow from a .byteflow file.
func (s *WorkflowService) LoadWorkflow(path string) (workflow.Workflow, error) {
	if !strings.HasSuffix(path, ".byteflow") {
		return workflow.Workflow{}, fmt.Errorf("not a valid byteflow file: path must end in .byteflow")
	}
	if !fileExists(path) {
		return workflow.Workflow{}, fmt.Errorf("file not found: %s", path)
	}
	s.log.Debug("loading workflow", "path", path)
	fileStore, err := workflow.OpenStore(path)
	if err != nil {
		s.log.Error("failed to open workflow file for load", logging.KeyError, err, "path", path)
		return workflow.Workflow{}, fmt.Errorf("not a valid byteflow file: %w", err)
	}
	defer fileStore.Close()

	wf, err := fileStore.LoadAnyWorkflow()
	if err != nil {
		s.log.Error("failed to load workflow from file", logging.KeyError, err, "path", path)
		return workflow.Workflow{}, fmt.Errorf("not a valid byteflow file: %w", err)
	}
	if wf == nil {
		s.log.Error("no workflow found in file", "path", path)
		return workflow.Workflow{}, fmt.Errorf("not a valid byteflow file: no workflow found")
	}

	// Check hardware availability for Input blocks
	ports, _ := listSerialPortNames()
	portSet := map[string]bool{}
	for _, p := range ports {
		portSet[p] = true
	}
	for i, b := range wf.Blocks {
		if b.Category == "input" && b.Type == "uart" {
			port, _ := b.Params["port"].(string)
			if port != "" && !portSet[port] {
				s.log.Warn("hardware not available for UART block", logging.KeyBlockID, b.ID, "port", port)
				wf.Blocks[i].Status = "error"
				wf.Blocks[i].ErrorMessage = "hardware not available: " + port
			}
		}
	}

	s.current = wf
	s.log.Info("workflow loaded", "path", path, "workflow_id", wf.ID, "blocks", len(wf.Blocks))
	return *wf, nil
}

// RecoverWorkflow attempts to open a corrupted .byteflow file by re-initialising
// the schema (discarding sessions) and returning an empty workflow for the file (T106).
func (s *WorkflowService) RecoverWorkflow(path string) (workflow.Workflow, error) {
	if !strings.HasSuffix(path, ".byteflow") {
		return workflow.Workflow{}, fmt.Errorf("not a valid byteflow file: path must end in .byteflow")
	}
	s.log.Info("attempting workflow recovery", "path", path)
	// Attempt to open; SQLite will re-initialise schema on a corrupted file
	fileStore, err := workflow.OpenStore(path)
	if err != nil {
		s.log.Error("workflow recovery failed", logging.KeyError, err, "path", path)
		return workflow.Workflow{}, fmt.Errorf("recovery failed: %w", err)
	}
	defer fileStore.Close()

	// Try to load existing workflow; if none found, return empty
	wf, _ := fileStore.LoadAnyWorkflow()
	if wf == nil {
		s.log.Warn("no workflow found during recovery, creating empty workflow", "path", path)
		empty := workflow.Workflow{
			ID:            fmt.Sprintf("recovered-%d", time.Now().UnixMilli()),
			Name:          "Recovered Workflow",
			CreatedAt:     time.Now().UnixMilli(),
			UpdatedAt:     time.Now().UnixMilli(),
			Blocks:        []workflow.BlockDef{},
			Connections:   []workflow.ConnectionDef{},
			SessionConfig: workflow.DefaultSessionConfig(),
		}
		s.current = &empty
		return empty, nil
	}
	s.current = wf
	s.log.Info("workflow recovered", "path", path, "workflow_id", wf.ID)
	return *wf, nil
}

// AddBlock creates and registers a new block at the given canvas position.
func (s *WorkflowService) AddBlock(blockType string, x, y float64) (workflow.BlockDef, error) {
	if s.engine != nil && s.engine.GetState() == pipeline.FlowStateRunning {
		return workflow.BlockDef{}, fmt.Errorf("cannot add blocks while flow is Running")
	}
	factory, ok := processing.Registry[blockType]
	if !ok {
		s.log.Error("unknown block type", "type", blockType)
		return workflow.BlockDef{}, fmt.Errorf("unknown block type: %s", blockType)
	}
	block := factory(uuid.New().String())
	def := workflow.BlockDef{
		ID:        block.ID(),
		Type:      blockType,
		Category:  string(block.Category()),
		Label:     blockType,
		Params:    defaultParamsForType(blockType),
		PositionX: x,
		PositionY: y,
	}
	s.current.Blocks = append(s.current.Blocks, def)
	s.current.UpdatedAt = time.Now().UnixMilli()
	s.log.Debug("block added", logging.KeyBlockID, def.ID, "type", blockType)
	return def, nil
}

// RemoveBlock removes a block and its attached connections.
func (s *WorkflowService) RemoveBlock(blockID string) error {
	if s.engine != nil && s.engine.GetState() == pipeline.FlowStateRunning {
		return fmt.Errorf("cannot remove blocks while flow is Running")
	}
	idx := -1
	for i, b := range s.current.Blocks {
		if b.ID == blockID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return fmt.Errorf("block not found: %s", blockID)
	}
	s.current.Blocks = append(s.current.Blocks[:idx], s.current.Blocks[idx+1:]...)

	// Remove attached connections
	filtered := s.current.Connections[:0]
	for _, c := range s.current.Connections {
		if c.FromBlockID != blockID && c.ToBlockID != blockID {
			filtered = append(filtered, c)
		}
	}
	s.current.Connections = filtered
	s.current.UpdatedAt = time.Now().UnixMilli()
	s.log.Debug("block removed", logging.KeyBlockID, blockID)
	return nil
}

// AddConnection adds a connection between two block ports.
func (s *WorkflowService) AddConnection(fromBlockID, fromPortID, toBlockID, toPortID string) (workflow.ConnectionDef, error) {
	if s.engine != nil && s.engine.GetState() == pipeline.FlowStateRunning {
		return workflow.ConnectionDef{}, fmt.Errorf("cannot add connections while flow is Running")
	}

	fromBlock := s.findBlock(fromBlockID)
	toBlock := s.findBlock(toBlockID)
	if fromBlock == nil {
		return workflow.ConnectionDef{}, fmt.Errorf("source block not found: %s", fromBlockID)
	}
	if toBlock == nil {
		return workflow.ConnectionDef{}, fmt.Errorf("target block not found: %s", toBlockID)
	}

	// Check input port is not already connected
	for _, c := range s.current.Connections {
		if c.ToBlockID == toBlockID && c.ToPortID == toPortID {
			return workflow.ConnectionDef{}, fmt.Errorf("input port already has a connection")
		}
	}

	// Build a temporary port type map for cycle + type checking
	portTypes := s.buildPortTypeMap()

	// Type compatibility check
	srcType, hasSrc := portTypeOf(portTypes, fromBlockID, fromPortID)
	dstType, hasDst := portTypeOf(portTypes, toBlockID, toPortID)
	if hasSrc && hasDst && srcType != dstType {
		return workflow.ConnectionDef{}, fmt.Errorf("incompatible port types: %s → %s", srcType, dstType)
	}

	newConn := workflow.ConnectionDef{
		ID:          uuid.New().String(),
		FromBlockID: fromBlockID,
		FromPortID:  fromPortID,
		ToBlockID:   toBlockID,
		ToPortID:    toPortID,
	}

	// Check for cycle by running validation with the new connection
	testConns := append(s.current.Connections, newConn)
	if _, err := pipeline.ValidateGraph(s.current.Blocks, testConns, portTypes); err != nil {
		if strings.Contains(err.Error(), "cycle") {
			return workflow.ConnectionDef{}, fmt.Errorf("connection would create a cycle")
		}
		// Other validation errors (e.g. no path) are acceptable at edit time
	}

	s.current.Connections = append(s.current.Connections, newConn)
	s.current.UpdatedAt = time.Now().UnixMilli()
	s.log.Debug("connection added", "connection_id", newConn.ID, "from", fromBlockID+":"+fromPortID, "to", toBlockID+":"+toPortID)
	return newConn, nil
}

// RemoveConnection removes a connection by ID.
func (s *WorkflowService) RemoveConnection(connectionID string) error {
	for i, c := range s.current.Connections {
		if c.ID == connectionID {
			s.current.Connections = append(s.current.Connections[:i], s.current.Connections[i+1:]...)
			s.current.UpdatedAt = time.Now().UnixMilli()
			s.log.Debug("connection removed", "connection_id", connectionID)
			return nil
		}
	}
	return fmt.Errorf("connection not found: %s", connectionID)
}

// UpdateBlockParams updates block parameters.
func (s *WorkflowService) UpdateBlockParams(blockID string, params map[string]any) error {
	b := s.findBlockRef(blockID)
	if b == nil {
		return fmt.Errorf("block not found: %s", blockID)
	}
	if b.Params == nil {
		b.Params = map[string]any{}
	}
	for k, v := range params {
		b.Params[k] = v
	}
	s.current.UpdatedAt = time.Now().UnixMilli()
	return nil
}

// UpdateBlockPosition updates the canvas position of a block.
func (s *WorkflowService) UpdateBlockPosition(blockID string, x, y float64) error {
	b := s.findBlockRef(blockID)
	if b == nil {
		return fmt.Errorf("block not found: %s", blockID)
	}
	b.PositionX = x
	b.PositionY = y
	s.current.UpdatedAt = time.Now().UnixMilli()
	return nil
}

// UpdateBlockSize persists a new canvas width and height for a block.
func (s *WorkflowService) UpdateBlockSize(blockID string, width, height float64) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("invalid size: width=%f height=%f", width, height)
	}
	b := s.findBlockRef(blockID)
	if b == nil {
		return fmt.Errorf("block not found: %s", blockID)
	}
	b.Width = width
	b.Height = height
	s.current.UpdatedAt = time.Now().UnixMilli()
	return nil
}

// GetAvailableBlockTypes returns the full catalogue of registered block types.
func (s *WorkflowService) GetAvailableBlockTypes() []BlockTypeDescriptor {
	descriptors := blockTypeDescriptors()
	result := make([]BlockTypeDescriptor, 0, len(descriptors))
	for _, d := range descriptors {
		if _, ok := processing.Registry[d.Type]; ok {
			result = append(result, d)
		}
	}
	return result
}

// ListSerialPorts enumerates available serial ports.
func (s *WorkflowService) ListSerialPorts() ([]SerialPortInfo, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		s.log.Error("failed to enumerate serial ports", logging.KeyError, err)
		return nil, err
	}
	s.log.Debug("serial ports enumerated", "count", len(ports))
	result := make([]SerialPortInfo, 0, len(ports))
	for _, p := range ports {
		result = append(result, SerialPortInfo{
			Name:        p.Name,
			VendorID:    p.VID,
			ProductID:   p.PID,
			Description: p.Product,
		})
	}
	return result, nil
}

// RestoreBlocks replaces the in-memory blocks and connections (used for undo/redo sync).
func (s *WorkflowService) RestoreBlocks(blocks []workflow.BlockDef, connections []workflow.ConnectionDef) error {
	if s.engine != nil && s.engine.GetState() == pipeline.FlowStateRunning {
		return fmt.Errorf("cannot modify workflow while flow is Running")
	}
	if blocks == nil {
		blocks = []workflow.BlockDef{}
	}
	if connections == nil {
		connections = []workflow.ConnectionDef{}
	}
	s.current.Blocks = blocks
	s.current.Connections = connections
	s.current.UpdatedAt = time.Now().UnixMilli()
	s.log.Debug("blocks restored", "blocks", len(blocks), "connections", len(connections))
	return nil
}

// GetCurrentWorkflow returns a pointer to the mutable current workflow (internal use).
func (s *WorkflowService) GetCurrentWorkflow() *workflow.Workflow {
	return s.current
}

// SetCurrentWorkflow replaces the in-memory workflow.
func (s *WorkflowService) SetCurrentWorkflow(wf *workflow.Workflow) {
	s.current = wf
}

// --- helpers ---

func (s *WorkflowService) findBlock(id string) *workflow.BlockDef {
	for i := range s.current.Blocks {
		if s.current.Blocks[i].ID == id {
			return &s.current.Blocks[i]
		}
	}
	return nil
}

func (s *WorkflowService) findBlockRef(id string) *workflow.BlockDef {
	return s.findBlock(id)
}

func (s *WorkflowService) buildPortTypeMap() map[string]map[string]pipeline.DataType {
	portTypes := map[string]map[string]pipeline.DataType{}
	for _, b := range s.current.Blocks {
		factory, ok := processing.Registry[b.Type]
		if !ok {
			continue
		}
		block := factory(b.ID)
		portTypes[b.ID] = map[string]pipeline.DataType{}
		for _, p := range block.InputPorts() {
			portTypes[b.ID][p.ID] = p.DataType
		}
		for _, p := range block.OutputPorts() {
			portTypes[b.ID][p.ID] = p.DataType
		}
	}
	return portTypes
}

func portTypeOf(portTypes map[string]map[string]pipeline.DataType, blockID, portID string) (pipeline.DataType, bool) {
	if m, ok := portTypes[blockID]; ok {
		if dt, ok2 := m[portID]; ok2 {
			return dt, true
		}
	}
	return "", false
}

func fileExists(path string) bool {
	_, err := filepath.Abs(path)
	return err == nil
}

func listSerialPortNames() ([]string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(ports))
	for _, p := range ports {
		names = append(names, p.Name)
	}
	return names, nil
}

func defaultParamsForType(blockType string) map[string]any {
	for _, d := range blockTypeDescriptors() {
		if d.Type == blockType {
			return d.DefaultParams
		}
	}
	return map[string]any{}
}

// blockTypeDescriptors returns the full catalogue of block type metadata.
func blockTypeDescriptors() []BlockTypeDescriptor {
	return []BlockTypeDescriptor{
		{
			Type:        "uart",
			Category:    pipeline.CategoryInput,
			Label:       "UART/Serial",
			Description: "Read data from a UART/serial port",
			DefaultParams: map[string]any{
				"port": "", "baudRate": 115200, "dataBits": 8, "stopBits": 1.0, "parity": "none",
			},
		},
		{
			Type:        "websocket",
			Category:    pipeline.CategoryInput,
			Label:       "WebSocket",
			Description: "Receive data from a WebSocket server",
			DefaultParams: map[string]any{
				"url": "ws://localhost:8080/data", "subprotocol": "", "reconnectIntervalMs": 2000,
			},
		},
		{
			Type:        "simulator",
			Category:    pipeline.CategoryInput,
			Label:       "Signal Simulator",
			Description: "Generate synthetic waveform data for testing",
			DefaultParams: map[string]any{
				"waveform": "sine", "frequencyHz": 1.0, "amplitude": 1.0, "offset": 0.0, "sampleRateHz": 100.0,
			},
		},
		{
			Type:        "passthrough",
			Category:    pipeline.CategoryProcessing,
			Label:       "Passthrough",
			Description: "Pass data unchanged (useful for forking streams)",
			DefaultParams: map[string]any{},
		},
		{
			Type:        "moving-average",
			Category:    pipeline.CategoryProcessing,
			Label:       "Moving Average",
			Description: "Compute rolling average over a sliding window",
			DefaultParams: map[string]any{"windowSize": 10},
		},
		{
			Type:        "summation",
			Category:    pipeline.CategoryProcessing,
			Label:       "Summation",
			Description: "Accumulate a running sum of incoming values",
			DefaultParams: map[string]any{},
		},
		{
			Type:        "fft",
			Category:    pipeline.CategoryProcessing,
			Label:       "FFT",
			Description: "Compute frequency spectrum via Fast Fourier Transform",
			DefaultParams: map[string]any{"windowSize": 512, "windowFunction": "hann"},
		},
		{
			Type:        "scaling",
			Category:    pipeline.CategoryProcessing,
			Label:       "Value Scaling",
			Description: "Apply a scale factor and DC offset to each value",
			DefaultParams: map[string]any{"scale": 1.0, "offset": 0.0},
		},
		{
			Type:        "byte-parser",
			Category:    pipeline.CategoryProcessing,
			Label:       "Byte Parser",
			Description: "Parse raw bytes into numeric values using a frame format",
			DefaultParams: map[string]any{"format": "float32-le", "channels": 1, "frameSize": 4},
		},
		{
			Type:        "sampler",
			Category:    pipeline.CategoryProcessing,
			Label:       "Sampler",
			Description: "Reduce stream rate: pass every N-th sample, first/last in a time window, or the first value only",
			DefaultParams: map[string]any{"mode": "every-n-samples", "n": 10, "intervalMs": 100.0},
		},
		{
			Type:        "value-display",
			Category:    pipeline.CategoryAnalysis,
			Label:       "Value Display",
			Description: "Display the latest numeric or raw byte value",
			DefaultParams: map[string]any{
				"label": "Value", "unit": "", "decimals": 2, "showHistory": false,
				"bufferMode": "samples", "bufferSamples": 100, "bufferDurationSec": 10.0,
			},
		},
		{
			Type:        "line-chart",
			Category:    pipeline.CategoryAnalysis,
			Label:       "Line Chart",
			Description: "Plot a streaming time-series line chart",
			DefaultParams: map[string]any{
				"title": "", "xLabel": "Time", "yLabel": "Value", "unit": "", "decimals": 2,
				"bufferMode": "samples", "bufferSamples": 500, "bufferDurationSec": 30.0,
			},
		},
		{
			Type:        "data-table",
			Category:    pipeline.CategoryAnalysis,
			Label:       "Data Table",
			Description: "Display incoming data as a scrolling table",
			DefaultParams: map[string]any{
				"maxRows": 100,
				"bufferMode": "samples", "bufferSamples": 100, "bufferDurationSec": 10.0,
			},
		},
		{
			Type:        "bar-chart",
			Category:    pipeline.CategoryAnalysis,
			Label:       "Bar Chart",
			Description: "Display the latest value per channel as vertical bars",
			DefaultParams: map[string]any{
				"title": "", "yLabel": "Value", "unit": "", "decimals": 2,
				"bufferMode": "samples", "bufferSamples": 50, "bufferDurationSec": 5.0,
			},
		},
		{
			Type:        "fft-spectrum",
			Category:    pipeline.CategoryAnalysis,
			Label:       "FFT Spectrum Viewer",
			Description: "Display frequency-domain magnitude spectrum",
			DefaultParams: map[string]any{
				"xLabel": "Frequency (Hz)", "yLabel": "Magnitude", "decimals": 2, "logScaleY": false,
				"bufferMode": "samples", "bufferSamples": 10, "bufferDurationSec": 5.0,
			},
		},
	}
}
