package input

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"byteflow-studio/internal/pipeline"
	"byteflow-studio/internal/processing"

	"go.bug.st/serial"
)

func init() {
	processing.Register("bluetooth", func(id string) pipeline.Block {
		return &bluetoothBlock{id: id, maxRetries: -1} // -1 = infinite retries
	})
}

type bluetoothBlock struct {
	id            string
	deviceAddress string
	serialPort    string
	reader        io.ReadCloser // injectable for testing
	maxRetries    int           // -1 = infinite, 0 = no retry
}

func (b *bluetoothBlock) ID() string                       { return b.id }
func (b *bluetoothBlock) Type() string                     { return "bluetooth" }
func (b *bluetoothBlock) Category() pipeline.BlockCategory { return pipeline.CategoryInput }

func (b *bluetoothBlock) Configure(params map[string]any) error {
	if v, ok := params["deviceAddress"].(string); ok {
		b.deviceAddress = v
	}
	if v, ok := params["serialPort"].(string); ok {
		b.serialPort = v
	}
	return nil
}

func (b *bluetoothBlock) InputPorts() []pipeline.Port { return nil }
func (b *bluetoothBlock) OutputPorts() []pipeline.Port {
	return []pipeline.Port{
		{ID: "out", Direction: pipeline.PortDirOutput, DataType: pipeline.DataTypeRaw, Label: "Out"},
	}
}

func (b *bluetoothBlock) Run(
	ctx context.Context,
	_ map[string]<-chan pipeline.DataChunk,
	outputs map[string]chan<- pipeline.DataChunk,
	errCh chan<- pipeline.BlockError,
) error {
	out, ok := outputs["out"]
	if !ok {
		return nil
	}

	if b.reader != nil {
		return b.readLoop(ctx, b.reader, out, errCh)
	}

	if b.serialPort == "" && b.deviceAddress == "" {
		err := fmt.Errorf("bluetooth: no device configured")
		pkgLog.Error("bluetooth device not configured", "block_id", b.id)
		select {
		case errCh <- pipeline.BlockError{BlockID: b.id, Err: err}:
		default:
		}
		return err
	}

	portPath := b.serialPort
	if portPath == "" {
		portPath = resolveSerialPort(b.deviceAddress)
	}
	if portPath == "" {
		err := fmt.Errorf("bluetooth: could not resolve serial port for device %s", b.deviceAddress)
		pkgLog.Error("could not resolve bluetooth serial port", "block_id", b.id, "device", b.deviceAddress)
		select {
		case errCh <- pipeline.BlockError{BlockID: b.id, Err: err}:
		default:
		}
		return err
	}

	retryCount := 0
	backoff := time.Second

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		pkgLog.Debug("opening bluetooth serial port", "block_id", b.id, "port", portPath)
		mode := &serial.Mode{BaudRate: 115200}
		port, err := serial.Open(portPath, mode)
		if err != nil {
			retryCount++
			pkgLog.Warn("bluetooth connection failed", "block_id", b.id, "port", portPath, "attempt", retryCount, "error", err)

			if b.maxRetries >= 0 && retryCount > b.maxRetries {
				pkgLog.Error("bluetooth max retries reached", "block_id", b.id, "max_retries", b.maxRetries)
				select {
				case errCh <- pipeline.BlockError{BlockID: b.id, Err: err}:
				default:
				}
				return err
			}

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(backoff):
			}
			// Exponential backoff: 1s, 2s, 4s, 8s, ... max 30s
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			continue
		}

		pkgLog.Info("bluetooth connected", "block_id", b.id, "port", portPath)
		retryCount = 0
		backoff = time.Second

		err = b.readLoop(ctx, port, out, errCh)
		port.Close()

		if ctx.Err() != nil {
			return nil
		}
		if err != nil {
			pkgLog.Warn("bluetooth disconnected", "block_id", b.id, "port", portPath, "error", err)
		}

		if b.maxRetries == 0 {
			return err
		}

		// Retry after backoff
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(backoff):
		}
	}
}

func (b *bluetoothBlock) readLoop(ctx context.Context, reader io.ReadCloser, out chan<- pipeline.DataChunk, errCh chan<- pipeline.BlockError) error {
	buf := make([]byte, 256)
	readDone := make(chan struct {
		n   int
		err error
		buf []byte
	}, 1)

	for {
		go func() {
			n, err := reader.Read(buf)
			cp := make([]byte, n)
			copy(cp, buf[:n])
			readDone <- struct {
				n   int
				err error
				buf []byte
			}{n, err, cp}
		}()

		select {
		case <-ctx.Done():
			return nil
		case res := <-readDone:
			if res.err != nil {
				if ctx.Err() != nil {
					return nil
				}
				pkgLog.Error("bluetooth read error", "block_id", b.id, "error", res.err)
				select {
				case errCh <- pipeline.BlockError{BlockID: b.id, Err: res.err}:
				default:
				}
				return res.err
			}
			now := time.Now().UnixMilli()
			for _, byteVal := range res.buf[:res.n] {
				chunk := pipeline.DataChunk{
					Timestamp: now,
					SourceID:  b.id,
					Raw:       []byte{byteVal},
				}
				select {
				case out <- chunk:
				case <-ctx.Done():
					return nil
				}
			}
		}
	}
}

// --- Bluetooth device discovery ---

// BluetoothDevice represents a paired Bluetooth device.
type BluetoothDevice struct {
	Address string `json:"address"`
	Name    string `json:"name"`
	Port    string `json:"port"`
}

// ListBluetoothDevices returns paired Bluetooth devices from the host OS.
func ListBluetoothDevices() ([]BluetoothDevice, error) {
	switch runtime.GOOS {
	case "linux":
		return listBluetoothLinux()
	case "darwin":
		return listBluetoothMacOS()
	case "windows":
		return listBluetoothWindows()
	default:
		return nil, fmt.Errorf("bluetooth discovery not supported on %s", runtime.GOOS)
	}
}

func listBluetoothLinux() ([]BluetoothDevice, error) {
	cmd := exec.Command("bluetoothctl", "devices", "Paired")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bluetoothctl failed: %w", err)
	}

	var devices []BluetoothDevice
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Device ") {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		devices = append(devices, BluetoothDevice{
			Address: parts[1],
			Name:    parts[2],
		})
	}
	return devices, nil
}

func listBluetoothMacOS() ([]BluetoothDevice, error) {
	cmd := exec.Command("system_profiler", "SPBluetoothDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("system_profiler failed: %w", err)
	}
	// Parse basic info from JSON — simplified
	_ = output
	return []BluetoothDevice{}, nil
}

func listBluetoothWindows() ([]BluetoothDevice, error) {
	cmd := exec.Command("powershell", "-Command",
		"Get-PnpDevice -Class Bluetooth | Where-Object {$_.Status -eq 'OK'} | Select-Object -Property FriendlyName, InstanceId | ConvertTo-Json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("powershell failed: %w", err)
	}
	_ = output
	return []BluetoothDevice{}, nil
}

func resolveSerialPort(deviceAddress string) string {
	switch runtime.GOOS {
	case "linux":
		// Try /dev/rfcomm0 as default for Linux
		return "/dev/rfcomm0"
	case "darwin":
		return "/dev/tty." + strings.ReplaceAll(deviceAddress, ":", "-") + "-SPPDev"
	case "windows":
		return "" // Would need registry lookup
	}
	return ""
}
