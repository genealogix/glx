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
	"path/filepath"
	"strings"
	"testing"
)

// yearRows builds timeline rows for the given years, labeled by year.
func yearRows(years ...int) []timelineRow {
	rows := make([]timelineRow, 0, len(years))
	for _, year := range years {
		rows = append(rows, timelineRow{
			Date:  displayYear(year),
			Label: "Event",
			Year:  year,
		})
	}

	return rows
}

func TestBuildTimelineStrip_NilWithoutASpanToPlot(t *testing.T) {
	cases := map[string][]timelineRow{
		"no rows at all":     nil,
		"no dated rows":      {{Label: "Birth", Undated: true}},
		"a single year":      yearRows(1850),
		"one year, repeated": yearRows(1850, 1850, 1850),
	}
	for name, rows := range cases {
		if strip := buildTimelineStrip(rows); strip != nil {
			t.Errorf("%s: expected no strip, got %+v", name, strip)
		}
	}
}

func TestBuildTimelineStrip_PlotsEveryDatedRow(t *testing.T) {
	rows := yearRows(1850, 1875, 1880, 1920)
	rows = append(rows, timelineRow{Label: "Burial", Undated: true})

	strip := buildTimelineStrip(rows)
	if strip == nil {
		t.Fatal("expected a strip for rows spanning several years")
	}
	if got, want := len(strip.Dots), 4; got != want {
		t.Fatalf("dot count = %d, want %d (the undated row is not plottable)", got, want)
	}
	if got, want := strip.Dots[0].X, stripInsetLeft; got != want {
		t.Errorf("earliest event at x=%d, want the bar's left edge %d", got, want)
	}
	if got, want := strip.Dots[3].X, stripWidth-stripInsetRight; got != want {
		t.Errorf("latest event at x=%d, want the bar's right edge %d", got, want)
	}

	// 1875 is a quarter of the way through 1850–1920.
	wantX := stripInsetLeft + (stripWidth-stripInsetLeft-stripInsetRight)*25/70
	if got := strip.Dots[1].X; got < wantX-1 || got > wantX+1 {
		t.Errorf("1875 plotted at x=%d, want ~%d (proportional to its position in the span)", got, wantX)
	}
	if !strings.Contains(strip.Label, "1850") || !strings.Contains(strip.Label, "1920") {
		t.Errorf("accessible label %q should name the span's endpoints", strip.Label)
	}
}

func TestStripTicks_LabelsEndpointsInRoundYearsWithoutCollisions(t *testing.T) {
	ticks := stripTicks(1531, 1646)
	if len(ticks) < 2 {
		t.Fatalf("expected at least the two endpoints, got %d ticks", len(ticks))
	}
	if first, last := ticks[0].Label, ticks[len(ticks)-1].Label; first != "1531" || last != "1646" {
		t.Errorf("endpoint labels = %q…%q, want 1531…1646", first, last)
	}
	for i := 1; i < len(ticks); i++ {
		if gap := ticks[i].X - ticks[i-1].X; gap < stripMinTickGap {
			t.Errorf("labels %q and %q are %d apart, closer than the %d minimum",
				ticks[i-1].Label, ticks[i].Label, gap, stripMinTickGap)
		}
	}
	for _, tick := range ticks[1 : len(ticks)-1] {
		if year := tick.Label; !strings.HasSuffix(year, "0") {
			t.Errorf("interior label %q is not a round year", year)
		}
	}
}

func TestStripTicks_WideSpanStaysWithinTheLabelBudget(t *testing.T) {
	// A span of two millennia must not label every decade.
	ticks := stripTicks(-44, 1980)
	if len(ticks) > stripMaxTicks+2 {
		t.Errorf("got %d labels for a 2000-year span, want at most %d plus the two endpoints",
			len(ticks), stripMaxTicks)
	}
	if ticks[0].Label != "44 BCE" {
		t.Errorf("first label = %q, want %q", ticks[0].Label, "44 BCE")
	}
}

func TestStripDotTitle_NamesTheEventInFull(t *testing.T) {
	title := stripDotTitle(timelineRow{
		Date:   "January 15, 1850",
		Label:  "Birth",
		Detail: "Boston, Massachusetts",
		Year:   1850,
	})
	for _, want := range []string{"January 15, 1850", "Birth", "Boston, Massachusetts"} {
		if !strings.Contains(title, want) {
			t.Errorf("tooltip %q missing %q", title, want)
		}
	}
}

func TestBuildTimelineRows_RecordsTheYearOfEachEvent(t *testing.T) {
	model := buildSiteModel(buildModelTestArchive(), siteModelOptions{})

	var john *personPage
	for _, page := range model.Persons {
		if page.ID == "person-john-smith" {
			john = page
		}
	}
	if john == nil {
		t.Fatal("expected John Smith in the model")
	}
	if len(john.Timeline) == 0 {
		t.Fatal("expected John to have timeline rows")
	}
	if john.Timeline[0].Year != 1850 {
		t.Errorf("first row's year = %d, want 1850", john.Timeline[0].Year)
	}
	if john.TimelineStrip == nil {
		t.Fatal("John's events span 1850–1920 and should produce a strip")
	}
}

func TestRenderSite_DrawsTheTimelineStrip(t *testing.T) {
	model := buildSiteModel(buildModelTestArchive(), siteModelOptions{Title: "Smith Family"})
	out := t.TempDir()
	if err := renderSite(model, out); err != nil {
		t.Fatalf("renderSite: %v", err)
	}

	john := readFile(t, filepath.Join(out, "persons", "person-john-smith.html"))
	for _, want := range []string{
		`<figure class="strip">`,
		`class="strip-bar"`,
		`<circle cx=`,
		`<title>January 15, 1850 · Birth · Boston</title>`,
	} {
		if !strings.Contains(john, want) {
			t.Errorf("John's profile missing %q", want)
		}
	}

	// The living person has no dated events at all, so no strip is drawn.
	living := readFile(t, filepath.Join(out, "persons", "person-living-soul.html"))
	if strings.Contains(living, `<figure class="strip">`) {
		t.Error("a person with no dated events should not get a proportional strip")
	}
}
