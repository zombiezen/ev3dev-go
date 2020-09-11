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

package fixedpoint

import "testing"

func TestEqual(t *testing.T) {
	tests := []struct {
		fp1, fp2 Value
		want     bool
	}{
		{
			fp1:  Value{value: 0, decimals: 0},
			fp2:  Value{value: 0, decimals: 0},
			want: true,
		},
		{
			fp1:  Value{value: 42, decimals: 0},
			fp2:  Value{value: 42, decimals: 0},
			want: true,
		},
		{
			fp1:  Value{value: 42, decimals: 0},
			fp2:  Value{value: 4200, decimals: 2},
			want: true,
		},
		{
			fp1:  Value{value: 42000, decimals: 3},
			fp2:  Value{value: 4200000, decimals: 5},
			want: true,
		},
		{
			fp1:  Value{value: 50, decimals: 1},
			fp2:  Value{value: 50, decimals: 2},
			want: false,
		},
		{
			fp1:  Value{value: 50, decimals: 1},
			fp2:  Value{value: 42, decimals: 1},
			want: false,
		},
	}
	for _, test := range tests {
		if got := test.fp1.Equal(test.fp2); got != test.want {
			t.Errorf("(%v).Equal(%v) = %t; want %t", test.fp1, test.fp2, got, test.want)
		}
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		fp1, fp2, want Value
	}{
		{
			fp1:  Value{value: 0, decimals: 0},
			fp2:  Value{value: 0, decimals: 0},
			want: Value{value: 0, decimals: 0},
		},
		{
			fp1:  Value{value: 1, decimals: 0},
			fp2:  Value{value: 2, decimals: 0},
			want: Value{value: 3, decimals: 0},
		},
		{
			fp1:  Value{value: 100, decimals: 2},
			fp2:  Value{value: 5, decimals: 1},
			want: Value{value: 150, decimals: 2},
		},
	}
	for _, test := range tests {
		if got := test.fp1.Add(test.fp2); !got.Equal(test.want) {
			t.Errorf("(%v).Add(%v) = %+v; want %v", test.fp1, test.fp2, got, test.want)
		}
	}
}

func TestFloat64(t *testing.T) {
	tests := []struct {
		fp   Value
		want float64
	}{
		{
			fp:   Value{value: 0, decimals: 0},
			want: 0.0,
		},
		{
			fp:   Value{value: 0, decimals: 10},
			want: 0.0,
		},
		{
			fp:   Value{value: 42, decimals: 0},
			want: 42.0,
		},
		{
			fp:   Value{value: 42, decimals: 1},
			want: 4.2,
		},
		{
			fp:   Value{value: 42, decimals: 2},
			want: 0.42,
		},
	}
	for _, test := range tests {
		if got := test.fp.Float64(); got != test.want {
			t.Errorf("Value{value: %d, decimals: %d}.Float64() = %v; want %v", test.fp.value, test.fp.decimals, got, test.want)
		}
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		fp   Value
		want string
	}{
		{
			fp:   Value{value: 0, decimals: 0},
			want: "0",
		},
		{
			fp:   Value{value: 1, decimals: 0},
			want: "1",
		},
		{
			fp:   Value{value: 1, decimals: 1},
			want: "0.1",
		},
		{
			fp:   Value{value: 1, decimals: 2},
			want: "0.01",
		},
		{
			fp:   Value{value: 1234, decimals: 2},
			want: "12.34",
		},
	}
	for _, test := range tests {
		if got := test.fp.String(); got != test.want {
			t.Errorf("Value{value: %d, decimals: %d}.String() = %q; want %q", test.fp.value, test.fp.decimals, got, test.want)
		}
	}
}
