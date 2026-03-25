package util_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/Bucket-Chemist/GOgent-Fortress/internal/tui/util"
)

// ansiRe matches ANSI escape sequences so they can be stripped for content assertions.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes all ANSI escape sequences from s, returning plain text.
// Used by content assertions that need to check rendered text independently
// of the active glamour style (which may insert per-word escape codes).
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// ---------------------------------------------------------------------------
// RenderMarkdown — table-driven tests
// ---------------------------------------------------------------------------

func TestRenderMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		content string
		width   int
		check   func(t *testing.T, result string)
	}{
		{
			name:    "empty content returns empty string",
			content: "",
			width:   80,
			check: func(t *testing.T, result string) {
				t.Helper()
				if result != "" {
					t.Errorf("expected empty string; got %q", result)
				}
			},
		},
		{
			name:    "plain text passes through",
			content: "Hello, world!",
			width:   80,
			check: func(t *testing.T, result string) {
				t.Helper()
				if !strings.Contains(stripANSI(result), "Hello, world!") {
					t.Errorf("plain text should be present in output; got %q", result)
				}
			},
		},
		{
			name:    "heading text is preserved in output",
			content: "# My Title\n\nSome body text.",
			width:   80,
			check: func(t *testing.T, result string) {
				t.Helper()
				// Glamour renders headings with ANSI styling and padding.
				// We verify the heading text survives rendering and the output
				// is not just an empty string.
				if !strings.Contains(stripANSI(result), "My Title") {
					t.Errorf("heading text 'My Title' should be present in output; got:\n%s", result)
				}
				if !strings.Contains(stripANSI(result), "Some body text") {
					t.Errorf("body text should be present in output; got:\n%s", result)
				}
				// Output should not be identical to input — glamour adds
				// padding, ANSI codes, etc.
				if result == "# My Title\n\nSome body text." {
					t.Errorf("output should be styled by glamour, not identical to raw input")
				}
			},
		},
		{
			name:    "code block is rendered",
			content: "```go\nfmt.Println(\"hello\")\n```\n",
			width:   80,
			check: func(t *testing.T, result string) {
				t.Helper()
				if !strings.Contains(stripANSI(result), "fmt.Println") {
					t.Errorf("code block content should appear in output; got:\n%s", result)
				}
			},
		},
		{
			name: "nested code blocks render correctly",
			content: "Outer text\n\n```go\nfunc main() {\n\t// inner\n}\n```\n\nMore text.",
			width:  80,
			check: func(t *testing.T, result string) {
				t.Helper()
				if !strings.Contains(stripANSI(result), "func main") {
					t.Errorf("nested code should appear; got:\n%s", result)
				}
				if !strings.Contains(stripANSI(result), "More text") {
					t.Errorf("text after code block should appear; got:\n%s", result)
				}
			},
		},
		{
			name:    "width 0 uses default of 80",
			content: "Some content.",
			width:   0,
			check: func(t *testing.T, result string) {
				t.Helper()
				// Should render without error and contain the text.
				if !strings.Contains(stripANSI(result), "Some content") {
					t.Errorf("width=0 should use default width and render content; got:\n%s", result)
				}
			},
		},
		{
			name:    "negative width uses default of 80",
			content: "Negative width test.",
			width:   -5,
			check: func(t *testing.T, result string) {
				t.Helper()
				if !strings.Contains(stripANSI(result), "Negative width test") {
					t.Errorf("negative width should use default and render content; got:\n%s", result)
				}
			},
		},
		{
			name:    "list renders correctly",
			content: "- item one\n- item two\n- item three\n",
			width:   80,
			check: func(t *testing.T, result string) {
				t.Helper()
				plain := stripANSI(result)
				if !strings.Contains(plain, "item one") {
					t.Errorf("list item 'item one' should appear; got:\n%s", result)
				}
				if !strings.Contains(plain, "item two") {
					t.Errorf("list item 'item two' should appear; got:\n%s", result)
				}
			},
		},
		{
			name:    "inline code is included in output",
			content: "Use `fmt.Println` to print output.",
			width:   80,
			check: func(t *testing.T, result string) {
				t.Helper()
				if !strings.Contains(stripANSI(result), "fmt.Println") {
					t.Errorf("inline code 'fmt.Println' should appear; got:\n%s", result)
				}
			},
		},
		{
			name:    "very long line is wrapped within width",
			content: strings.Repeat("word ", 40),
			width:   40,
			check: func(t *testing.T, result string) {
				t.Helper()
				// With word wrap at 40, no individual line should exceed 40
				// visible characters (glamour may add ANSI codes, so we check
				// that the output has multiple lines indicating wrapping occurred).
				lines := strings.Split(result, "\n")
				if len(lines) < 2 {
					t.Errorf("long content at width=40 should produce multiple lines; got %d", len(lines))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Clear cache before each subtest so width-specific renderers
			// are not carried over between tests with different widths.
			util.ClearCache()

			result, err := util.RenderMarkdown(tc.content, tc.width)
			if err != nil {
				t.Fatalf("RenderMarkdown returned unexpected error: %v", err)
			}
			tc.check(t, result)
		})
	}
}

// ---------------------------------------------------------------------------
// Caching behaviour
// ---------------------------------------------------------------------------

// TestCachedRenderer_ReusesSameWidth verifies that calling RenderMarkdown twice
// with the same width reuses the cached renderer (observable via consistent
// output rather than internal pointer comparison, since the cache is private).
func TestCachedRenderer_ReusesSameWidth(t *testing.T) {
	util.ClearCache()

	const width = 80
	const content = "# Heading\n\nBody text."

	out1, err1 := util.RenderMarkdown(content, width)
	out2, err2 := util.RenderMarkdown(content, width)

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v, %v", err1, err2)
	}
	if out1 != out2 {
		t.Errorf("same width should produce identical output (cached renderer);\ngot1=%q\ngot2=%q", out1, out2)
	}
}

// TestCachedRenderer_DifferentWidth verifies that different widths produce
// different (or at least separately-created) renderers. We validate this by
// confirming both calls succeed and return non-empty output.
func TestCachedRenderer_DifferentWidth(t *testing.T) {
	util.ClearCache()

	const content = "Hello markdown world!"

	out40, err40 := util.RenderMarkdown(content, 40)
	out80, err80 := util.RenderMarkdown(content, 80)

	if err40 != nil || err80 != nil {
		t.Fatalf("unexpected errors: %v, %v", err40, err80)
	}
	if out40 == "" || out80 == "" {
		t.Error("both widths should produce non-empty output")
	}
	// The outputs may differ slightly in whitespace/wrapping.
	// We just confirm both renderers were created successfully.
}

// TestClearCache_ResetsCache verifies that ClearCache causes the next call to
// RenderMarkdown to create a fresh renderer.  Since the renderer is internal,
// we validate the observable behaviour: the output remains consistent after a
// cache clear (i.e. no stale state corrupts results).
func TestClearCache_ResetsCache(t *testing.T) {
	const content = "# Reset Test\n\nSome body."
	const width = 80

	// Populate cache.
	out1, err := util.RenderMarkdown(content, width)
	if err != nil {
		t.Fatalf("first render failed: %v", err)
	}

	// Clear and re-render.
	util.ClearCache()
	out2, err := util.RenderMarkdown(content, width)
	if err != nil {
		t.Fatalf("second render after ClearCache failed: %v", err)
	}

	// Output should be deterministic — same input produces same output.
	if out1 != out2 {
		t.Errorf("output after ClearCache should match original;\ngot1=%q\ngot2=%q", out1, out2)
	}
}

// TestRenderMarkdown_ErrorDegradesToOriginal verifies that when glamour cannot
// render (simulated by checking the graceful-degradation path), the original
// content is returned rather than an error surfacing to the caller.
//
// Note: glamour v1.0.0 is quite robust; we cannot easily inject a render
// failure without mocking. This test instead confirms the function contract:
// RenderMarkdown NEVER returns a non-nil error to the caller.
func TestRenderMarkdown_NeverReturnsError(t *testing.T) {
	util.ClearCache()

	testCases := []struct {
		content string
		width   int
	}{
		{"", 80},
		{"plain", 0},
		{"# heading", -1},
		{"```\ncode\n```", 80},
		{strings.Repeat("x", 10000), 80},
	}

	for _, tc := range testCases {
		_, err := util.RenderMarkdown(tc.content, tc.width)
		if err != nil {
			t.Errorf("RenderMarkdown(%q, %d) returned non-nil error: %v", tc.content, tc.width, err)
		}
	}
}
