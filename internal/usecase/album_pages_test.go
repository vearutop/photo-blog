package usecase

import (
	"encoding/json"
	"html/template"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vearutop/photo-blog/internal/domain/photo"
	"github.com/vearutop/photo-blog/pkg/txt"
)

// TestBuildAlbumTimeline_dropsEmptyText covers the defensive guard: there's no use case for a
// text-only timeline entry with blank text (a plain, non-marker chrono text that's just empty
// would otherwise slip through filterPageTexts), so buildAlbumTimeline must never emit one —
// regardless of whether it's blank to begin with, or a leftover with nothing after the images.
func TestBuildAlbumTimeline_dropsEmptyText(t *testing.T) {
	images := []Image{{Hash: "i1", UTime: 100}}
	texts := []txt.Chronological{
		{Time: mustTime(t, "1970-01-01T00:01:30Z"), Text: "   "}, // blank, before the image
		{Time: mustTime(t, "1970-01-01T00:02:00Z"), Text: ""},    // blank, after the image (leftover)
	}

	timeline := buildAlbumTimeline(images, texts, false)

	assert.Len(t, timeline, 1, "only the image should remain, both blank texts dropped")
	assert.NotNil(t, timeline[0].Image)
}

// TestAlbumTimelineItem_JSON locks down two things at once, because a one-way marshal check alone
// already shipped a real bug once: a lightweight "image" key (just the hash) for a client-side
// renderer to walk without ever needing the full Image object, AND — the part a marshal-only test
// missed — that Image itself survives a full marshal/unmarshal ROUND TRIP. The album-page
// persistent cache stores albumPageData as JSON and, on a cache hit, build() never re-runs: the
// server template's {{if $item.Image}} check depends entirely on Image surviving that exact
// round trip, not just being present on a freshly-computed value.
func TestAlbumTimelineItem_JSON(t *testing.T) {
	items := []albumTimelineItem{
		{Image: &Image{Hash: "abc123", Width: 100}, ImageHash: "abc123", Ts: 111},
		{Text: "<p>hi</p>", Ts: 222},
	}

	b, err := json.Marshal(items)
	assert.NoError(t, err)
	assert.JSONEq(t, `[{"image":"abc123","image_full":{"hash":"abc123","width":100,"height":0,"name":"","utime":0}},{"text":"<p>hi</p>"}]`, string(b))

	var roundTripped []albumTimelineItem
	assert.NoError(t, json.Unmarshal(b, &roundTripped))

	assert.NotNil(t, roundTripped[0].Image, "the template's {{if $item.Image}} must still see a non-nil pointer after a cache-hit deserialize")
	assert.Equal(t, "abc123", roundTripped[0].Image.Hash)
	assert.Equal(t, int64(100), roundTripped[0].Image.Width)
	assert.Nil(t, roundTripped[1].Image)
	assert.Equal(t, template.HTML("<p>hi</p>"), roundTripped[1].Text)
}

// TestDisplayBefore locks down the one place NewestFirst is actually decided — pageForTime,
// buildAlbumTimeline, and defaultPage all route through this (and displayReached, its mirror).
func TestDisplayBefore(t *testing.T) {
	early := mustTime(t, "2024-01-01T00:00:00Z")
	late := mustTime(t, "2024-06-01T00:00:00Z")

	assert.True(t, displayBefore(early, late, false), "oldest-first: earlier time displays first")
	assert.False(t, displayBefore(late, early, false))

	assert.True(t, displayBefore(late, early, true), "newest-first: later time displays first")
	assert.False(t, displayBefore(early, late, true))

	assert.True(t, displayReached(late, early, false), "oldest-first: an earlier marker has been reached by a later moment")
	assert.False(t, displayReached(early, late, false))

	assert.True(t, displayReached(early, late, true), "newest-first: a later marker has been reached by an earlier moment")
	assert.False(t, displayReached(late, early, true))
}

func TestSplitPageMarker(t *testing.T) {
	name, rest := splitPageMarker("[page:aftermath]\nSome heading text")
	assert.Equal(t, "aftermath", name)
	assert.Equal(t, "Some heading text", rest)

	name, rest = splitPageMarker("[page:aftermath]")
	assert.Equal(t, "aftermath", name)
	assert.Equal(t, "", rest)

	name, rest = splitPageMarker("plain chrono text, no marker")
	assert.Equal(t, "", name)
	assert.Equal(t, "plain chrono text, no marker", rest)
}

// TestPageForUTime_newestFirst mirrors the "selected" album: newest-first, markers dated at each
// year's END, closing out the year that just finished.
func TestPageForUTime_newestFirst(t *testing.T) {
	texts := []txt.Chronological{
		{Time: mustTime(t, "2021-12-31T23:59:59Z"), Text: "[page:2021]"},
		{Time: mustTime(t, "2022-12-31T23:59:59Z"), Text: "[page:2022]\n### Happy new year"},
	}

	bounds := pageBoundaries(texts)

	assert.Equal(t, "2021", pageForUTime(bounds, mustTime(t, "2020-06-01T00:00:00Z").Unix(), true), "predates every marker, absorbed by the first cut")
	assert.Equal(t, "2021", pageForUTime(bounds, mustTime(t, "2021-12-31T23:59:59Z").Unix(), true), "exactly at the cut belongs to the page it closes")
	assert.Equal(t, "2022", pageForUTime(bounds, mustTime(t, "2022-06-01T00:00:00Z").Unix(), true), "between the two cuts, closed out by the second")
	assert.Equal(t, "", pageForUTime(bounds, mustTime(t, "2023-01-01T00:00:00Z").Unix(), true), "after the last cut: the current, not-yet-closed-out page")
}

// TestPageForUTime_oldestFirst mirrors the "selected-inv" album: oldest-first, markers dated at
// each year's START, opening the year that's about to begin.
func TestPageForUTime_oldestFirst(t *testing.T) {
	texts := []txt.Chronological{
		{Time: mustTime(t, "2023-01-01T00:00:00Z"), Text: "[page:2023]"},
		{Time: mustTime(t, "2024-01-01T00:00:00Z"), Text: "[page:2024]\n### Happy new year"},
	}

	bounds := pageBoundaries(texts)

	assert.Equal(t, "", pageForUTime(bounds, mustTime(t, "2022-06-01T00:00:00Z").Unix(), false), "predates every marker: not opened yet")
	assert.Equal(t, "2023", pageForUTime(bounds, mustTime(t, "2023-01-01T00:00:00Z").Unix(), false), "exactly at the cut belongs to the page it opens")
	assert.Equal(t, "2023", pageForUTime(bounds, mustTime(t, "2023-06-01T00:00:00Z").Unix(), false), "between the two cuts, opened by the first")
	assert.Equal(t, "2024", pageForUTime(bounds, mustTime(t, "2025-01-01T00:00:00Z").Unix(), false), "after the last cut: still belongs to the last opened page")
}

func TestDefaultPage(t *testing.T) {
	images := []photo.Image{
		{UTime: mustTime(t, "2022-06-01T00:00:00Z").Unix(), BlurHash: "x"},
		{UTime: mustTime(t, "2023-06-01T00:00:00Z").Unix(), BlurHash: "x"},
		{UTime: mustTime(t, "2020-01-01T00:00:00Z").Unix(), BlurHash: ""}, // unprocessed, ignored
	}

	// newest-first: marker closes out 2022 at its end; the latest image (2023) is after the last
	// cut, so it's the current, not-yet-closed-out page.
	closingBounds := pageBoundaries([]txt.Chronological{
		{Time: mustTime(t, "2022-12-31T23:59:59Z"), Text: "[page:2022]"},
	})
	assert.Equal(t, "", defaultPage(images, true, closingBounds), "newest-first: latest image is after the last cut, the current open page")

	// oldest-first: marker opens 2023 at its start; the earliest image (2022) predates it, so
	// it's the not-yet-opened page.
	openingBounds := pageBoundaries([]txt.Chronological{
		{Time: mustTime(t, "2023-01-01T00:00:00Z"), Text: "[page:2023]"},
	})
	assert.Equal(t, "", defaultPage(images, false, openingBounds), "oldest-first: earliest image predates the first cut, not opened yet")

	// oldest-first, marker predates every image: the earliest image (2022) is already opened.
	earlyOpeningBounds := pageBoundaries([]txt.Chronological{
		{Time: mustTime(t, "2021-01-01T00:00:00Z"), Text: "[page:2022]"},
	})
	assert.Equal(t, "2022", defaultPage(images, false, earlyOpeningBounds), "oldest-first: earliest image is after the opening cut")
}

func TestFilterImagesByPage(t *testing.T) {
	bounds := pageBoundaries([]txt.Chronological{
		{Time: mustTime(t, "2022-12-31T23:59:59Z"), Text: "[page:2022]"},
	})

	images := []photo.Image{
		{UTime: mustTime(t, "2022-06-01T00:00:00Z").Unix()},
		{UTime: mustTime(t, "2023-06-01T00:00:00Z").Unix()},
	}

	page2022 := filterImagesByPage(images, bounds, "2022", true)
	assert.Len(t, page2022, 1)
	assert.Equal(t, mustTime(t, "2022-06-01T00:00:00Z").Unix(), page2022[0].UTime)

	current := filterImagesByPage(images, bounds, "", true)
	assert.Len(t, current, 1)
	assert.Equal(t, mustTime(t, "2023-06-01T00:00:00Z").Unix(), current[0].UTime)
}

func TestFilterPageTexts(t *testing.T) {
	newTexts := func() []txt.Chronological {
		return []txt.Chronological{
			{Time: mustTime(t, "2021-06-01T00:00:00Z"), Text: "regular 2021 note"},
			{Time: mustTime(t, "2021-12-31T23:59:59Z"), Text: "[page:2021]\n## 2021 recap"},
			{Time: mustTime(t, "2022-06-01T00:00:00Z"), Text: "regular 2022 note"},
		}
	}

	// filterPageTexts mutates its input in place, so each call needs its own fresh slice.
	page2021 := filterPageTexts(newTexts(), "2021", true)
	assert.Len(t, page2021, 2, "everything up to and including the cut belongs to the page it closes")
	assert.Equal(t, "regular 2021 note", page2021[0].Text)
	assert.Equal(t, "## 2021 recap", page2021[1].Text, "the marker's own trailing text is a normal chrono entry, positioned by its own time")

	current := filterPageTexts(newTexts(), "", true)
	assert.Len(t, current, 1, "the empty [page:2021] marker itself carries no visible text")
	assert.Equal(t, "regular 2022 note", current[0].Text)
}

func TestFilterPageTexts_wildcard(t *testing.T) {
	newTexts := func() []txt.Chronological {
		return []txt.Chronological{
			{Time: mustTime(t, "2021-06-01T00:00:00Z"), Text: "[page:*]\n[next](wee/)"},
			{Time: mustTime(t, "2022-01-01T00:00:00Z"), Text: "[page:2022]"},
		}
	}

	// "*" is not a real page, so it never defines a boundary of its own...
	bounds := pageBoundaries(newTexts())
	assert.Len(t, bounds, 1)
	assert.Equal(t, "2022", bounds[0].name)

	// ...and its text survives filtering for every page, unlike a regular chrono text at the same time.
	// filterPageTexts mutates its input in place, so each call needs its own fresh slice.
	for _, page := range []string{"", "2022", "anything"} {
		filtered := filterPageTexts(newTexts(), page, true)
		assert.Len(t, filtered, 1, "page %q", page)
		assert.Equal(t, "[next](wee/)", filtered[0].Text)
	}
}

// TestPageSplitEndToEnd runs the real pipeline (page filtering + timeline building) against a
// concrete worked example: items in order i1, i2, t3, t4 (splits "p1"), i5, t6, i7, t7,
// t8 (splits "p2"), i9 should produce exactly 3 pages:
//
//	"":   i1, i2, t3
//	"p1": t4, i5, t6, i7, t7
//	"p2": t8, i9
//
// Timestamps are arbitrary, evenly-spaced offsets — only their relative order (matching the
// example's own ordering) matters, not the specific values.
func TestPageSplitEndToEnd(t *testing.T) {
	base := mustTime(t, "2024-03-01T00:00:00Z")
	at := func(n int) time.Time { return base.Add(time.Duration(n) * 11 * time.Minute) }

	label := map[int64]string{
		at(10).Unix(): "i1",
		at(20).Unix(): "i2",
		at(30).Unix(): "t3",
		at(40).Unix(): "t4",
		at(50).Unix(): "i5",
		at(60).Unix(): "t6",
		at(70).Unix(): "i7",
		at(80).Unix(): "t7",
		at(85).Unix(): "t8",
		at(90).Unix(): "i9",
	}

	rawImages := []photo.Image{
		{UTime: at(10).Unix(), BlurHash: "x"},
		{UTime: at(20).Unix(), BlurHash: "x"},
		{UTime: at(50).Unix(), BlurHash: "x"},
		{UTime: at(70).Unix(), BlurHash: "x"},
		{UTime: at(90).Unix(), BlurHash: "x"},
	}

	texts := []txt.Chronological{
		{Time: at(30), Text: "t3"},
		{Time: at(40), Text: "[page:p1]\nt4"},
		{Time: at(60), Text: "t6"},
		{Time: at(80), Text: "t7"},
		{Time: at(85), Text: "[page:p2]\nt8"},
	}

	const newestFirst = false

	bounds := pageBoundaries(texts)
	assert.Equal(t, "", defaultPage(rawImages, newestFirst, bounds), "bare URL: earliest image predates every marker")

	cases := []struct {
		page string
		want []string
	}{
		{"", []string{"i1", "i2", "t3"}},
		{"p1", []string{"t4", "i5", "t6", "i7", "t7"}},
		{"p2", []string{"t8", "i9"}},
	}

	for _, c := range cases {
		pageImages := filterImagesByPage(append([]photo.Image(nil), rawImages...), bounds, c.page, newestFirst)
		pageTexts := filterPageTexts(append([]txt.Chronological(nil), texts...), c.page, newestFirst)

		images := make([]Image, len(pageImages))
		for i, img := range pageImages {
			images[i] = Image{UTime: img.UTime}
		}

		timeline := buildAlbumTimeline(images, pageTexts, newestFirst)

		got := make([]string, len(timeline))
		for i, item := range timeline {
			got[i] = label[item.Ts]
		}

		assert.Equal(t, c.want, got, "page %q timeline", c.page)
	}
}

func mustTime(t *testing.T, s string) time.Time {
	t.Helper()

	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}

	return tm
}
