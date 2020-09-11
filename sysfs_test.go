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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestReadAttrBytes(t *testing.T) {
	t.Run("Seek0", func(t *testing.T) {
		f := tempFile(t)
		const want = "Hello, World!"
		if _, err := f.WriteString(want); err != nil {
			t.Fatal(err)
		}
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}

		var buf [32]byte
		n, err := readAttrBytes(f, buf[:])
		if n != len(want) || err != nil {
			t.Errorf("readAttrBytes(f, buf) = %d, %v; want %d, <nil>", n, err, len(want))
		}
		if got := string(buf[:n]); got != want {
			t.Errorf("buf[:n] = %q; want %q", got, want)
		}
	})
	t.Run("AtEnd", func(t *testing.T) {
		f := tempFile(t)
		const want = "Hello, World!"
		if _, err := f.WriteString(want); err != nil {
			t.Fatal(err)
		}

		var buf [32]byte
		n, err := readAttrBytes(f, buf[:])
		if n != len(want) || err != nil {
			t.Errorf("readAttrBytes(f, buf) = %d, %v; want %d, <nil>", n, err, len(want))
		}
		if got := string(buf[:n]); got != want {
			t.Errorf("buf[:n] = %q; want %q", got, want)
		}
	})
}

func TestWriteAttr(t *testing.T) {
	f := tempFile(t)
	const want = "Hello, World!"

	if err := writeAttr(f, []byte(want)); err != nil {
		t.Errorf("writeAttr(f, []byte(%q)): %v", want, err)
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	got, err := ioutil.ReadAll(f)
	if err != nil {
		t.Error(err)
	}
	if string(got) != want {
		t.Errorf("file contents = %q; want %q", got, want)
	}
}

func TestFindByAddress(t *testing.T) {
	type call struct {
		files  map[string]string
		remove []string

		addr string

		want    string
		wantErr bool
	}
	tests := []struct {
		name   string
		prefix string
		calls  []call
	}{
		{
			name:   "Simple",
			prefix: "sensor",
			calls: []call{{
				files: map[string]string{
					"sensor0/address": "iface:S3\n",
					"sensor1/address": "iface:S1\n",
					"sensor2/address": "iface:S2\n",
				},
				addr: "iface:S1",
				want: "sensor1",
			}},
		},
		{
			name:   "NotFound",
			prefix: "sensor",
			calls: []call{{
				files: map[string]string{
					"sensor0/address": "iface:S3\n",
					"sensor1/address": "iface:S1\n",
					"sensor2/address": "iface:S2\n",
				},
				addr:    "iface:S4",
				wantErr: true,
			}},
		},
		{
			name:   "SkipsNotMatching",
			prefix: "sensor",
			calls: []call{{
				files: map[string]string{
					"bacon/address":   "iface:S3\n",
					"sensorX/address": "iface:S3\n",
				},
				addr:    "iface:S3",
				wantErr: true,
			}},
		},
		{
			name:   "DoesNotProduceSameDeviceTwice",
			prefix: "sensor",
			calls: []call{
				{
					files: map[string]string{
						"sensor0/address": "iface:S3\n",
						"sensor1/address": "iface:S1\n",
						"sensor2/address": "iface:S2\n",
					},
					addr: "iface:S1",
					want: "sensor1",
				},
				{
					addr:    "iface:S1",
					wantErr: true,
				},
			},
		},
		{
			name:   "ProducesReconfiguredPort",
			prefix: "sensor",
			calls: []call{
				{
					files: map[string]string{
						"sensor0/address": "iface:S3\n",
						"sensor1/address": "iface:S1\n",
						"sensor2/address": "iface:S2\n",
					},
					addr: "iface:S1",
					want: "sensor1",
				},
				{
					remove: []string{"sensor1"},
					files: map[string]string{
						"sensor3/address": "iface:S1\n",
					},
					addr: "iface:S1",
					want: "sensor3",
				},
			},
		},
		{
			name:   "ProducesSkippedDevices",
			prefix: "sensor",
			calls: []call{
				{
					files: map[string]string{
						"sensor0/address": "iface:S3\n",
						"sensor1/address": "iface:S1\n",
						"sensor2/address": "iface:S2\n",
					},
					addr: "iface:S1",
					want: "sensor1",
				},
				{
					addr: "iface:S3",
					want: "sensor0",
				},
			},
		},
		{
			name:   "DoesNotProduceDeletedSkippedDevice",
			prefix: "sensor",
			calls: []call{
				{
					files: map[string]string{
						"sensor0/address": "iface:S3\n",
						"sensor1/address": "iface:S1\n",
						"sensor2/address": "iface:S2\n",
					},
					addr: "iface:S1",
					want: "sensor1",
				},
				{
					remove:  []string{"sensor0"},
					addr:    "iface:S3",
					wantErr: true,
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			dev := newDeviceDir(dir, test.prefix)
			for i, call := range test.calls {
				for _, fname := range call.remove {
					dest := filepath.Join(dir, filepath.FromSlash(fname))
					if err := os.RemoveAll(dest); err != nil {
						t.Fatal(err)
					}
				}
				for fname, content := range call.files {
					dest := filepath.Join(dir, filepath.FromSlash(fname))
					if err := os.MkdirAll(filepath.Dir(dest), 0777); err != nil {
						t.Fatal(err)
					}
					if err := ioutil.WriteFile(dest, []byte(content), 0666); err != nil {
						t.Fatal(err)
					}
				}
				a, err := newAddress(call.addr)
				if err != nil {
					t.Fatal(err)
				}
				got, err := dev.findByAddress(a)
				want := filepath.Join(dir, filepath.FromSlash(call.want))
				if call.want == "" {
					want = ""
				}
				if got != want || (err != nil) != call.wantErr {
					errStr := "<nil>"
					if call.wantErr {
						errStr = "<error>"
					}
					t.Errorf("findByAddress(%q) #%d = %q, %v; want %q, %s", call.addr, i+1, got, err, want, errStr)
				}
			}
		})
	}
}

func tempFile(tb testing.TB) *os.File {
	f, err := ioutil.TempFile("", "ev3dev_sysfs")
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		name := f.Name()
		f.Close()
		os.Remove(name)
	})
	return f
}
