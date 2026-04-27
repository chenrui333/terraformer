// SPDX-License-Identifier: Apache-2.0

//nolint:gosec // lint triage: legacy provider/API/security baseline is tracked in #175.
package tencentcloud

func Bool(i bool) *bool { return &i }

func String(i string) *string { return &i }

func Int(i int) *int { return &i }

func Uint(i uint) *uint { return &i }

func Int64(i int64) *int64 { return &i }

func Float64(i float64) *float64 { return &i }

func Uint64(i uint64) *uint64 { return &i }

func IntInt64(i int) *int64 {
	i64 := int64(i)
	return &i64
}

func IntUint64(i int) *uint64 {
	u := uint64(i)
	return &u
}

func Int64Uint64(i int64) *uint64 {
	u := uint64(i)
	return &u
}

func UInt64Int64(i uint64) *int64 {
	u := int64(i)
	return &u
}
