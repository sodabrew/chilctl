// Package cx34 provides a client for working with the Chiltrix CX34 heat pump.
package cx34

import (
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/goburrow/modbus"
	"github.com/goburrow/serial"
	"github.com/golang/glog"
	"github.com/howeyc/crc16"
	"go.uber.org/multierr"

	"github.com/sodabrew/chilctl/units"
)

// Parameters from https://www.chiltrix.com/control-options/Remote-Gateway-BACnet-Guide-rev2.pdf
const (
	baudRate = 9600
	parity   = "N"
	stopBits = 1
	dataBits = 8
)

const (
	// Valid Holding register range
	firstHoldingRegister Register = 1
	lastHoldingRegister  Register = 350
	registersPerRead              = 120
)

// Mode indicates the protocol that should be used to communicate with the CX34.
type Mode string

// Valid modes.
const (
	// Modbus mode uses modbus to communicate with the CX34. This is a
	// standardized protocol, but so far the library doesn't support it.
	Modbus Mode = "modbus"

	// CX34Text uses a proprietary protocol from Omron to communicate with the CX34.
	CX34Text Mode = "cx34text"
)

// Params configures the connection to the Chiltrix.
type Params struct {
	// The /dev/ttyX device shown by dmesg for the RS-485 connection to the heat pump.
	TTYDevice string
	LogWriter io.Writer
	Mode      Mode
	UnitId    int
}

// Client is used to communicate with the Chiltrix CX34 heat pump.
type Client struct {
	c modbus.Client
}

// Connect connects a new client to the heat pump or returns an error.
func Connect(p *Params) (*Client, error) {
	if p.Mode != Modbus && p.Mode != CX34Text {
		return nil, fmt.Errorf("Invalid mode %q", p.Mode)
	}
	// Modbus RTU/ASCII
	handler := modbus.NewRTUClientHandler(p.TTYDevice)
	handler.BaudRate = baudRate
	handler.DataBits = dataBits
	handler.Parity = parity
	handler.StopBits = stopBits
	handler.SlaveId = uint8(p.UnitId)
	handler.Timeout = 10 * time.Second
	handler.RS485 = serial.RS485Config{
		Enabled:            false,
		DelayRtsAfterSend:  0,
		DelayRtsBeforeSend: 0,
		RtsHighAfterSend:   false,
		RtsHighDuringSend:  false,
		RxDuringTx:         true,
	}

	if p.Mode == CX34Text {
		port, err := serial.Open(&handler.Config)
		if err != nil {
			return nil, fmt.Errorf("serial.Open failure: %w", err)
		}
		_, err = io.Copy(p.LogWriter, port)
		if err != nil {
			return nil, fmt.Errorf("io.Copy failure: %w", err)
		}
		return nil, nil
	}

	err := handler.Connect()
	defer handler.Close()
	if err != nil {
		return nil, fmt.Errorf("Connect failed: %w", err)
	}

	client := modbus.NewClient(handler)
	c := &Client{c: client}

	if err := c.CheckConnection(); err != nil {
		return nil, err
	}
	return c, nil
}

// ReadState returns a snapshot of the state of the heat pump.
func (c *Client) ReadState() (*State, error) {
	// ReadCoils, ReadInputRegisters, and ReadDiscreteInputs are not supported.
	// However, ReadHoldingRegisters is.
	m := make(map[Register]uint16)
	for i := firstHoldingRegister; i <= lastHoldingRegister; i += registersPerRead {
		count := registersPerRead
		if i+Register(count) > lastHoldingRegister {
			count = int(lastHoldingRegister) - int(i+1)
		}
		results, err := c.c.ReadHoldingRegisters(uint16(i), uint16(count))
		if err != nil {
			return nil, fmt.Errorf("ReadHoldingRegisters() failed: %w", err)
		}
		if len(results)%2 != 0 {
			return nil, fmt.Errorf("got register data of length %d, want modulus of 2", len(results))
		}
		if len(results)/2 > count {
			return nil, fmt.Errorf("returned register data of length %d exceeds expected length %d", len(results)/2, count)
		}
		for j := 0; j < len(results)/2; j++ {
			value := uint16(results[j*2])<<8 + uint16(results[j*2+1])
			m[Register(j)+i] = value
		}
	}
	return &State{time.Now(), m}, nil
}

func (c *Client) SetACMode(m AirConditioningMode) error {
	if m < 0 || m > 4 {
		return fmt.Errorf("mode is out of range 0-4: %v", m)
	}
	registerValue := uint16(m)
	res, err := c.c.WriteSingleRegister(ACMode.uint16(), registerValue)
	if err != nil {
		return fmt.Errorf("WriteSingleRegister error: %w (returned bytes %v)", err, res)
	}
	return nil
}

// SetHeatingTemp sets the target heating temperature for the CX34.
func (c *Client) SetHeatingTemp(t units.Temperature) error {
	deg := t.Celsius()
	if deg < 5 || deg > 70 {
		return fmt.Errorf("temperature is out of range: %v", t)
	}
	registerValue := uint16(math.Round(t.Celsius()))

	res, err := c.c.WriteSingleRegister(TargetACHeatingModeTemp.uint16(), registerValue)
	if err != nil {
		return fmt.Errorf("WriteSingleRegister error: %w (returned bytes %v)", err, res)
	}
	glog.Infof("set target heating temperature to %.2f°C/%.2f°F", t.Celsius(), t.Fahrenheit())
	return nil
}

// SetCoolingTemp sets the target heating temperature for the CX34.
func (c *Client) SetCoolingTemp(t units.Temperature) error {
	deg := t.Celsius()
	if deg < 5 || deg > 70 {
		return fmt.Errorf("temperature is out of range: %v", t)
	}
	registerValue := uint16(math.Round(t.Celsius()))

	res, err := c.c.WriteSingleRegister(TargetACCoolingModeTemp.uint16(), registerValue)
	if err != nil {
		return fmt.Errorf("WriteSingleRegister error: %w (returned bytes %v)", err, res)
	}
	glog.Infof("set target cooling temperature to %.2f°C/%.2f°F", t.Celsius(), t.Fahrenheit())
	return nil
}

// SetHeatingTemp sets the target heating temperature for the CX34.
func (c *Client) SetDomesticHotWaterTemp(t units.Temperature) error {
	deg := t.Celsius()
	if deg < 5 || deg > 70 {
		return fmt.Errorf("temperature is out of range: %v", t)
	}
	registerValue := uint16(math.Round(t.Celsius()))

	res, err := c.c.WriteSingleRegister(TargetDomesticHotWaterTemp.uint16(), registerValue)
	if err != nil {
		return fmt.Errorf("WriteSingleRegister error: %w (returned bytes %v)", err, res)
	}
	glog.Infof("set target DHW temperature to %.2f°C/%.2f°F", t.Celsius(), t.Fahrenheit())
	return nil
}

func (c *Client) setRegisterValue(reg, value uint16) error {
	res, err := c.c.WriteSingleRegister(reg, value)
	if err != nil {
		return fmt.Errorf("WriteSingleRegister error: %w (returned bytes %v)", err, res)
	}
	glog.Infof("set register value %d to %d; got response %v", reg, value, res)
	return nil
}

// CheckConnection attempts to connect to the heat pump and returns an error if the connection fails.
func (c *Client) CheckConnection() error {
	_, err := c.ReadState()
	return err
	var finalErr error
	if _, err := c.c.ReadCoils(1, 1); err != nil {
		finalErr = multierr.Append(finalErr, fmt.Errorf("ReadCoils error: %w", err))
	}
	if _, err := c.c.ReadInputRegisters(1, 1); err != nil {
		finalErr = multierr.Append(finalErr, fmt.Errorf("ReadInputRegisters error: %w", err))
	}
	if _, err := c.c.ReadFIFOQueue(1); err != nil {
		finalErr = multierr.Append(finalErr, fmt.Errorf("ReadFIFOQueue error: %w", err))
	}
	if _, err := c.c.ReadDiscreteInputs(1, 1); err != nil {
		finalErr = multierr.Append(finalErr, fmt.Errorf("ReadDiscreteInputs error: %w", err))
	}
	return finalErr
}

// State is a snapshot of the heat pump's state.
type State struct {
	collectionTime time.Time
	registerValues map[Register]uint16
}

// CollectionTime returns the collection time of the heat pump state log entry.
func (s *State) CollectionTime() time.Time {
	return s.collectionTime
}

// RegisterValues returns a map of register values. The map should not be modified by the caller.
func (s *State) RegisterValues() map[Register]uint16 {
	return s.registerValues
}

// Register is a modsbus register
type Register uint16

func (r Register) uint16() uint16 {
	return uint16(r)
}

// String returns a human-readable name of the modbus register.
func (r Register) String() string {
	if name, ok := registerNames[r]; ok {
		return name
	}
	return fmt.Sprintf("%d", r)
}

type Logger struct {
	debug, raw io.Writer
}

func NewLogger(debug, raw io.Writer) *Logger {
	return &Logger{debug, raw}
}

func (l *Logger) Write(p []byte) (n int, err error) {
	t := time.Now()
	{
		pdu, err := decodeFrame(p)
		msg := fmt.Sprintf("error decoding frame: %v", err)
		if err == nil {
			msg = fmt.Sprintf("decoded frame with code %v", pdu.FunctionCode)
		}
		glog.Infof("got %q: %s", string(p), msg)
	}
	if _, err := fmt.Fprintf(l.debug, "%d %q // %d bytes; %s\n", t.UnixNano(), hex.EncodeToString(p), len(p), t); err != nil {
		return 0, err
	}
	if l, err := l.raw.Write(p); err != nil {
		return 0, fmt.Errorf("raw output error: %w", err)
	} else if l != len(p) {
		return 0, fmt.Errorf("raw output Write() returned %d, want %d", l, len(p))
	}
	return len(p), nil
}

func decodeFrame(adu []byte) (*modbus.ProtocolDataUnit, error) {
	if len(adu) < 4 {
		return nil, fmt.Errorf("modbus: argument cannot possibly be a legitimate Application Data Unit: size is too small (%d bytes)", len(adu))
	}
	length := len(adu)
	// Calculate checksum
	crcCalculated := ^crc16.ChecksumIBM(adu[0 : length-2]) //crc16.Update(65535, crc16.IBMTable, adu[0:length-2])
	crcPacket := uint16(adu[length-1])<<8 | uint16(adu[length-2])
	if crcPacket != crcCalculated {
		return nil, fmt.Errorf("modbus: response crc %d does not match expected %d in %d-byte package", crcCalculated, crcPacket, length)
	}
	// Function code & data
	return &modbus.ProtocolDataUnit{
		FunctionCode: adu[1],
		Data:         adu[2 : length-2],
	}, nil
}

// func uint16ToBytes(i uint16) []byte {
// 	return []byte{(i & 0xFF00) >> 8, i & 0x00FF)}
// }
