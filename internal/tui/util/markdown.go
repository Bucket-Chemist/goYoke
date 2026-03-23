// Package util provides shared utility functions for the GOgent-Fortress TUI.
package util

import (
	"sync"

	"github.com/charmbracelet/glamour"
)

// defaultWidth is used when the caller supplies a width of zero or negative.
const defaultWidth = 80

// rendererCache maps terminal width → cached TermRenderer. Creating a
// TermRenderer is expensive (goldmark initialisation + style compilation), so
// we reuse renderers for the same width across calls.
var (
	rendererMu    sync.Mutex
	rendererCache = map[int]*glamour.TermRenderer{}
)

// RenderMarkdown renders markdown content to styled terminal text using
// Glamour. It caches the renderer per width to avoid expensive re-creation.
//
// Edge cases:
//   - Empty content returns "" immediately (no renderer created).
//   - Width ≤ 0 uses a default of 80.
//   - If Glamour returns an error the original content is returned as-is
//     (graceful degradation — the panel stays usable even in degraded envs).
func RenderMarkdown(content string, width int) (string, error) {
	if content == "" {
		return "", nil
	}

	if width <= 0 {
		width = defaultWidth
	}

	r, err := cachedRenderer(width)
	if err != nil {
		// Graceful degradation: renderer could not be created.
		return content, nil
	}

	rendered, err := r.Render(content)
	if err != nil {
		// Graceful degradation: render failed (e.g. malformed content).
		return content, nil
	}

	return rendered, nil
}

// ClearCache clears the renderer cache, forcing new renderers to be created on
// the next RenderMarkdown call. This is useful when the terminal theme changes
// at runtime (e.g. the user switches between dark and light mode).
func ClearCache() {
	rendererMu.Lock()
	rendererCache = map[int]*glamour.TermRenderer{}
	rendererMu.Unlock()
}

// cachedRenderer returns a *glamour.TermRenderer for the given width, creating
// and caching it on first use.
func cachedRenderer(width int) (*glamour.TermRenderer, error) {
	rendererMu.Lock()
	defer rendererMu.Unlock()

	if r, ok := rendererCache[width]; ok {
		return r, nil
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil, err
	}

	rendererCache[width] = r
	return r, nil
}
