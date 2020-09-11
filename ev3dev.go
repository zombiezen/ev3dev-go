// Copyright 2020 Ross Light
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0

// Package ev3dev provides a high-level interface to the EV3Dev drivers.
package ev3dev

import (
	"fmt"
	"path/filepath"
	"sync"
	"time"
)

// Brick is the root handle to the EV3Dev drivers.
type Brick struct {
	ports   deviceDir
	devices *devices
}

type devices struct {
	mu          sync.Mutex
	tachoMotors deviceDir
	sensors     deviceDir
}

func newBrick(root string) *Brick {
	portsDir := newDeviceDir(filepath.Join(root, "sys", "class", "lego-port"), "port")
	tachoMotorsDir := newDeviceDir(filepath.Join(root, "sys", "class", "tacho-motor"), "motor")
	sensorsDir := newDeviceDir(filepath.Join(root, "sys", "class", "lego-sensor"), "sensor")
	return &Brick{
		ports: *portsDir,
		devices: &devices{
			tachoMotors: *tachoMotorsDir,
			sensors:     *sensorsDir,
		},
	}
}

// PortByAddress searches for the port with the given address. Subsequent calls
// for the same address will return an error.
func (brick *Brick) PortByAddress(addr string) (*Port, error) {
	a, err := newAddress(addr)
	if err != nil {
		return nil, fmt.Errorf("find port %q: %w", addr, err)
	}
	path, err := brick.ports.findByAddress(a)
	if err != nil {
		return nil, fmt.Errorf("find port %q: %w", addr, err)
	}
	return &Port{
		path:    path,
		addr:    a,
		devices: brick.devices,
	}, nil
}

var System = newBrick("/")

// Port represents a configurable I/O port.
type Port struct {
	path    string
	addr    address
	devices *devices
}

// Addr returns the port address, like "spi0.1:S3".
func (p *Port) Addr() string {
	return p.addr.String()
}

// OpenSensor opens the port as a sensor.
func (p *Port) OpenSensor(typ SensorType) (*Sensor, error) {
	mode := typ.portMode()
	if len(mode) == 0 {
		return nil, fmt.Errorf("open sensor for port %q: invalid type %v", p.addr, typ)
	}

	modeFile, err := openAttrWrite(filepath.Join(p.path, "mode"))
	if err != nil {
		return nil, fmt.Errorf("open sensor for port %q: %w", p.addr, err)
	}
	err = writeAttr(modeFile, mode)
	closeErr := modeFile.Close()
	if err != nil {
		return nil, fmt.Errorf("open sensor for port %q: %w", p.addr, err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("open sensor for port %q: %w", p.addr, closeErr)
	}

	driverFile, err := openAttrWrite(filepath.Join(p.path, "set_device"))
	if err != nil {
		return nil, fmt.Errorf("open sensor for port %q: %w", p.addr, err)
	}
	err = writeAttr(driverFile, typ.driver())
	closeErr = driverFile.Close()
	if err != nil {
		return nil, fmt.Errorf("open sensor for port %q: %w", p.addr, err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("open sensor for port %q: %w", p.addr, closeErr)
	}
	// TODO(someday): This is a gross hack.
	time.Sleep(250 * time.Millisecond)

	p.devices.mu.Lock()
	defer p.devices.mu.Unlock()
	path, err := p.devices.sensors.findByAddress(p.addr)
	if err != nil {
		return nil, fmt.Errorf("open sensor for port %q: %w", p.addr, err)
	}
	s, err := newSensor(path)
	if err != nil {
		return nil, fmt.Errorf("open sensor for port %q: %w", p.addr, err)
	}
	return s, nil
}

// OpenTachoMotor opens the port as a tacho motor.
func (p *Port) OpenTachoMotor() (*TachoMotor, error) {
	modeFile, err := openAttrWrite(filepath.Join(p.path, "mode"))
	if err != nil {
		return nil, fmt.Errorf("open tacho motor for port %q: %w", p.addr, err)
	}
	err = writeAttr(modeFile, []byte("tacho-motor"))
	closeErr := modeFile.Close()
	if err != nil {
		return nil, fmt.Errorf("open tacho motor for port %q: %w", p.addr, err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("open tacho motor for port %q: %w", p.addr, closeErr)
	}
	// TODO(someday): This is a gross hack.
	time.Sleep(250 * time.Millisecond)

	p.devices.mu.Lock()
	defer p.devices.mu.Unlock()
	path, err := p.devices.tachoMotors.findByAddress(p.addr)
	if err != nil {
		return nil, fmt.Errorf("open tacho motor for port %q: %w", p.addr, err)
	}
	m, err := newTachoMotor(path)
	if err != nil {
		return nil, fmt.Errorf("open tacho motor for port %q: %w", p.addr, err)
	}
	return m, nil
}
