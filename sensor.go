package ev3dev

import (
	"fmt"
	"os"
	"path/filepath"

	"zombiezen.com/go/ev3dev/fixedpoint"
)

// SensorType enumerates known sensors.
type SensorType int

// Known sensor types
const (
	// NXT Touch sensor.
	LegoNXTTouch SensorType = 1 + iota
)

func (typ SensorType) portMode() []byte {
	switch typ {
	case LegoNXTTouch:
		return []byte("nxt-analog")
	default:
		return nil
	}
}

func (typ SensorType) driver() []byte {
	switch typ {
	case LegoNXTTouch:
		return []byte("lego-nxt-touch")
	default:
		return nil
	}
}

// A Sensor represents an input device.
type Sensor struct {
	decimals int16
	values   [8]*os.File
}

func newSensor(path string) (_ *Sensor, err error) {
	s := new(Sensor)

	decimalsFile, err := os.Open(filepath.Join(path, "decimals"))
	if err != nil {
		return nil, err
	}
	decimals, err := readAttrInt(decimalsFile, 16)
	decimalsFile.Close()
	if err != nil {
		return nil, err
	}
	s.decimals = int16(decimals)

	numValuesFile, err := os.Open(filepath.Join(path, "num_values"))
	if err != nil {
		return nil, err
	}
	nvalues, err := readAttrInt(numValuesFile, 8)
	numValuesFile.Close()
	if err != nil {
		return nil, err
	}
	if nvalues < 1 || nvalues > int64(len(s.values)) {
		return nil, fmt.Errorf("sensor has %d values", nvalues)
	}
	for i := 0; i < int(nvalues); i++ {
		var err error
		s.values[i], err = os.Open(filepath.Join(path, fmt.Sprintf("value%d", i)))
		if err != nil {
			s.Close()
			return nil, err
		}
	}

	return s, nil
}

// Close cleans up any resources for this sensor.
func (s *Sensor) Close() error {
	var firstErr error
	for _, f := range s.values {
		if f == nil {
			break
		}
		if err := f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return fmt.Errorf("close sensor: %w", firstErr)
	}
	return nil
}

// NValues returns the number of values the sensor provides.
func (s *Sensor) NValues() int {
	for i, f := range s.values {
		if f == nil {
			return i
		}
	}
	return len(s.values)
}

// Value reads the i'th value from the sensor. It returns an error if i is not
// in the range [0, s.NValues()).
func (s *Sensor) Value(i int) (fixedpoint.Value, error) {
	if i < 0 || i >= len(s.values) || s.values[i] == nil {
		return fixedpoint.Value{}, fmt.Errorf("read sensor value %d: no such value", i)
	}
	v, err := readAttrInt(s.values[i], 32)
	if err != nil {
		return fixedpoint.Value{}, fmt.Errorf("read sensor value %d: %w", i, err)
	}
	return fixedpoint.FromInt(int32(v)).Shift10(-int16(s.decimals)), nil
}
