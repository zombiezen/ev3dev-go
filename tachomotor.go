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

package ev3dev

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A TachoMotor is a motor with a quadrature encoder.
type TachoMotor struct {
	command          *os.File
	countPerRot      TachoDelta
	maxSpeed         TachoSpeed
	position         *os.File
	positionSetPoint *os.File
	speed            *os.File
	speedSetPoint    *os.File
	stopAction       *os.File
	stopActions      [3]bool
	timeSetPoint     *os.File
}

func newTachoMotor(path string) (_ *TachoMotor, err error) {
	m := new(TachoMotor)
	defer func() {
		if err == nil {
			return
		}
		for _, f := range m.files() {
			if f != nil {
				f.Close()
			}
		}
	}()
	if m.command, err = openAttrWrite(filepath.Join(path, "command")); err != nil {
		return nil, err
	}
	countPerRotFile, err := os.Open(filepath.Join(path, "count_per_rot"))
	if err != nil {
		return nil, err
	}
	countPerRot, err := readAttrInt(countPerRotFile, 32)
	countPerRotFile.Close()
	if err != nil {
		return nil, err
	}
	m.countPerRot = TachoDelta(countPerRot)
	maxSpeedFile, err := os.Open(filepath.Join(path, "max_speed"))
	if err != nil {
		return nil, err
	}
	maxSpeed, err := readAttrInt(maxSpeedFile, 32)
	maxSpeedFile.Close()
	if err != nil {
		return nil, err
	}
	m.maxSpeed = TachoSpeed(maxSpeed)
	if m.position, err = os.Open(filepath.Join(path, "position")); err != nil {
		return nil, err
	}
	if m.positionSetPoint, err = openAttrWrite(filepath.Join(path, "position_sp")); err != nil {
		return nil, err
	}
	if m.speed, err = os.Open(filepath.Join(path, "speed")); err != nil {
		return nil, err
	}
	if m.speedSetPoint, err = openAttrWrite(filepath.Join(path, "speed_sp")); err != nil {
		return nil, err
	}
	if m.stopAction, err = openAttrWrite(filepath.Join(path, "stop_action")); err != nil {
		return nil, err
	}
	stopActionsFile, err := os.Open(filepath.Join(path, "stop_actions"))
	if err != nil {
		return nil, err
	}
	var stopActionsBuf [64]byte
	n, err := readAttrBytes(stopActionsFile, stopActionsBuf[:])
	stopActionsFile.Close()
	if err != nil {
		return nil, err
	}
	for _, action := range strings.Fields(string(stopActionsBuf[:n])) {
		for aa := range m.stopActions {
			if StopAction(aa).String() == action {
				m.stopActions[aa] = true
				break
			}
		}
	}
	stopActionsFile.Close()
	if m.timeSetPoint, err = openAttrWrite(filepath.Join(path, "time_sp")); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *TachoMotor) files() []*os.File {
	return []*os.File{
		m.command,
		m.position,
		m.positionSetPoint,
		m.speed,
		m.speedSetPoint,
		m.stopAction,
		m.timeSetPoint,
	}
}

// Reset stops the motor and resets all options.
func (m *TachoMotor) Reset() error {
	if err := writeAttr(m.command, []byte("reset")); err != nil {
		return fmt.Errorf("reset motor: %w", err)
	}
	return nil
}

// Close stops the motor and cleans up its resources.
func (m *TachoMotor) Close() error {
	firstErr := m.Reset()
	for _, f := range m.files() {
		if err := f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return fmt.Errorf("close motor: %w", firstErr)
	}
	return nil
}

// Run instructs the motor to run at the given speed until another command
// is given.
func (m *TachoMotor) Run(speed TachoSpeed) error {
	if err := m.setSpeed(speed); err != nil {
		return fmt.Errorf("run motor: %w", err)
	}
	if err := writeAttr(m.command, []byte("run-forever")); err != nil {
		return fmt.Errorf("run motor: %w", err)
	}
	return nil
}

// RunToPosition instructs the motor to run until it reaches an absolute
// position then stop.
func (m *TachoMotor) RunToPosition(pos TachoPosition, params *TachoMotorParams) error {
	if err := m.setPosition(int32(pos)); err != nil {
		return fmt.Errorf("run motor to position: %w", err)
	}
	if err := m.setParams(params); err != nil {
		return fmt.Errorf("run motor to position: %w", err)
	}
	if err := writeAttr(m.command, []byte("run-to-abs-pos")); err != nil {
		return fmt.Errorf("run motor to position: %w", err)
	}
	return nil
}

// RunToDelta instructs the motor to run until it reaches a position
// relative to the current position then stop.
func (m *TachoMotor) RunToDelta(delta TachoDelta, params *TachoMotorParams) error {
	if err := m.setPosition(int32(delta)); err != nil {
		return fmt.Errorf("run motor to position: %w", err)
	}
	if err := m.setParams(params); err != nil {
		return fmt.Errorf("run motor to position: %w", err)
	}
	if err := writeAttr(m.command, []byte("run-to-rel-pos")); err != nil {
		return fmt.Errorf("run motor to position: %w", err)
	}
	return nil
}

// RunTimed instructs the motor to run for a set duration then stop.
func (m *TachoMotor) RunTimed(t time.Duration, params *TachoMotorParams) error {
	if err := m.setTime(t); err != nil {
		return fmt.Errorf("run motor for time: %w", err)
	}
	if err := m.setParams(params); err != nil {
		return fmt.Errorf("run motor for time: %w", err)
	}
	if err := writeAttr(m.command, []byte("run-timed")); err != nil {
		return fmt.Errorf("run motor for time: %w", err)
	}
	return nil
}

// Stop instructs the motor to stop.
func (m *TachoMotor) Stop(action StopAction) error {
	if err := m.setStopAction(action); err != nil {
		return fmt.Errorf("stop motor: %w", err)
	}
	if err := writeAttr(m.command, []byte("stop")); err != nil {
		return fmt.Errorf("stop motor: %w", err)
	}
	return nil
}

func (m *TachoMotor) setSpeed(speed TachoSpeed) error {
	// TODO(maybe): add safety cap?
	if speed > m.maxSpeed || speed < -m.maxSpeed {
		return fmt.Errorf("set motor speed: speed %d beyond max speed %d", speed, m.maxSpeed)
	}
	if err := writeAttrInt(m.speedSetPoint, int64(speed)); err != nil {
		return fmt.Errorf("set motor speed: %w", err)
	}
	return nil
}

func (m *TachoMotor) setPosition(pos int32) error {
	if err := writeAttrInt(m.positionSetPoint, int64(pos)); err != nil {
		return fmt.Errorf("set motor position: %w", err)
	}
	return nil
}

func (m *TachoMotor) setTime(t time.Duration) error {
	if err := writeAttrInt(m.timeSetPoint, t.Milliseconds()); err != nil {
		return fmt.Errorf("set motor time: %w", err)
	}
	return nil
}

func (m *TachoMotor) setStopAction(action StopAction) error {
	if !action.isValid() {
		return fmt.Errorf("set motor stop action: invalid action %v", action)
	}
	if err := writeAttr(m.stopAction, []byte(action.String())); err != nil {
		return fmt.Errorf("set motor stop action: %w", err)
	}
	return nil
}

func (m *TachoMotor) setParams(params *TachoMotorParams) error {
	if params != nil && params.Speed != 0 {
		if err := m.setSpeed(params.Speed); err != nil {
			return err
		}
	}
	action := Coast
	if params != nil {
		action = params.StopAction
	}
	if err := m.setStopAction(action); err != nil {
		return err
	}
	return nil
}

// MaxSpeed returns the maximum value accepted by the speed commands.
func (m *TachoMotor) MaxSpeed() TachoSpeed {
	return m.maxSpeed
}

// CountPerRotation returns the number of tacho counts in one rotation.
func (m *TachoMotor) CountPerRotation() TachoDelta {
	return m.countPerRot
}

// Position reads the current value of the encoder.
func (m *TachoMotor) Position() (TachoPosition, error) {
	i, err := readAttrInt(m.position, 32)
	if err != nil {
		return 0, fmt.Errorf("read motor position: %w", err)
	}
	return TachoPosition(i), nil
}

// Speed reads the current measured speed of the motor.
func (m *TachoMotor) Speed() (TachoSpeed, error) {
	i, err := readAttrInt(m.speed, 32)
	if err != nil {
		return 0, fmt.Errorf("read motor speed: %w", err)
	}
	return TachoSpeed(i), nil
}

// TachoMotorParams is the set of optional parameters for motor commands.
type TachoMotorParams struct {
	// Speed sets the speed of the motor. If zero, uses the speed from the last
	// issued command.
	Speed TachoSpeed

	// StopAction is the action to take when the motor stops.
	// Default is Coast.
	StopAction StopAction
}

// TachoPosition is a position value from a tacho motor.
type TachoPosition int32

// Add adds a distance to a position.
func (pos TachoPosition) Add(delta TachoDelta) TachoPosition {
	return pos + TachoPosition(delta)
}

// Sub finds the distance between two positions.
func (pos TachoPosition) Sub(pos2 TachoPosition) TachoDelta {
	return TachoDelta(pos - pos2)
}

// TachoDelta is a difference in position value from a tacho motor.
type TachoDelta int32

// TachoSpeed is the speed of a tacho motor in tacho counts per second.
type TachoSpeed int32

// Mul multiplies a speed by a duration to produce a distance.
func (speed TachoSpeed) Mul(dt time.Duration) TachoDelta {
	return TachoDelta(float64(speed) * dt.Seconds())
}

// StopAction is a motor stop mode.
type StopAction int

// Motor stop modes.
const (
	// Remove power from the motor.
	Coast StopAction = iota
	// Remove power from the motor and create a passive load.
	Brake
	// Cause the motor to actively try to hold position.
	Hold
)

// String returns the lowercase name of the action.
func (action StopAction) String() string {
	switch action {
	case Coast:
		return "coast"
	case Brake:
		return "brake"
	case Hold:
		return "hold"
	default:
		return fmt.Sprintf("StopAction(%d)", int(action))
	}
}

func (action StopAction) isValid() bool {
	return action == Coast || action == Brake || action == Hold
}
