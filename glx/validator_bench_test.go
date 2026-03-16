// Copyright 2025 Oracynth, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"runtime"
	"testing"
)

// generateTestGLXDoc creates a minimal valid GLX document for validation testing
func generateTestGLXDoc() map[string]any {
	return map[string]any{
		"persons": map[string]any{
			"person-1": map[string]any{
				"properties": map[string]any{
					"name": map[string]any{
						"value": "Test Person",
						"fields": map[string]any{
							"given":   "Test",
							"surname": "Person",
						},
					},
				},
			},
		},
	}
}

// measureValidationMemory runs fn once and reports total bytes allocated using
// TotalAlloc, which counts all allocations regardless of GC. This gives an
// accurate picture of memory pressure during validation.
func measureValidationMemory(fn func()) uint64 {
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	fn()

	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	return after.TotalAlloc - before.TotalAlloc
}

// BenchmarkValidateGLXFileStructure_SingleFile benchmarks validation of a single file.
// This establishes baseline performance with schema caching.
func BenchmarkValidateGLXFileStructure_SingleFile(b *testing.B) {
	doc := generateTestGLXDoc()
	b.ReportAllocs()
	for b.Loop() {
		issues := ValidateGLXFileStructure(doc)
		if len(issues) > 0 {
			b.Fatalf("validation failed: %v", issues)
		}
	}
}

// BenchmarkValidateGLXFileStructure_MultiFile simulates validating multiple files
// to verify schema caching works correctly. With caching, memory should remain
// constant regardless of file count.
func BenchmarkValidateGLXFileStructure_MultiFile(b *testing.B) {
	fileCounts := []int{10, 100, 1000}

	for _, count := range fileCounts {
		b.Run(fmt.Sprintf("%dfiles", count), func(b *testing.B) {
			for b.Loop() {
				peak := measureValidationMemory(func() {
					doc := generateTestGLXDoc()
					for i := 0; i < count; i++ {
						issues := ValidateGLXFileStructure(doc)
						if len(issues) > 0 {
							b.Fatalf("validation failed: %v", issues)
						}
					}
				})
				b.ReportMetric(float64(peak)/(1024*1024), "alloc-MB")
				b.ReportMetric(float64(peak)/float64(count), "alloc/file")
			}
		})
	}
}
