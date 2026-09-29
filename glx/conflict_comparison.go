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

	"github.com/spf13/cobra"

	glxlib "github.com/genealogix/glx/go-glx"
)

func comparisonOptions(options []glxlib.ComparisonOptions) glxlib.ComparisonOptions {
	if len(options) > 0 {
		return options[0]
	}

	return glxlib.ComparisonOptions{}
}

const approximationFlag = "approximation-years"

func init() {
	for _, cmd := range []*cobra.Command{analyzeCmd, proofCmd, evidenceCmd, mergePersonsCmd} {
		cmd.Flags().Int(approximationFlag, glxlib.DefaultApproximationYears, "Tolerance in years for ABT, EST and CAL date values (0–10000)")
	}
}

func commandComparisonOptions(cmd *cobra.Command) (glxlib.ComparisonOptions, error) {
	width, err := cmd.Flags().GetInt(approximationFlag)
	if err != nil {
		return glxlib.ComparisonOptions{}, err
	}
	opts := glxlib.ComparisonOptions{ApproximationYears: &width}
	if err := opts.Validate(); err != nil {
		return glxlib.ComparisonOptions{}, fmt.Errorf("--%s: %w", approximationFlag, err)
	}

	return glxlib.ComparisonOptions{ApproximationYears: &width}, nil
}
