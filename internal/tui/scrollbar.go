package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

func clipLine(s string, cols int) string {
	if cols <= 0 {
		return ""
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if lipgloss.Width(s) <= cols {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(cols).Render(s)
}

func blockHeight(s string) int {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0
	}
	return lipgloss.Height(s)
}

// stackTopMidBottom pins top to the first rows and bottom to the last rows.
// Leftover height goes into mid so it never shows up as padding under the
// status bar or sidebar footer.
func stackTopMidBottom(top, mid, bottom string, w, h int) string {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	top = strings.TrimRight(top, "\n")
	mid = strings.TrimRight(mid, "\n")
	bottom = strings.TrimRight(bottom, "\n")
	topH := blockHeight(top)
	botH := blockHeight(bottom)
	if topH+botH >= h {
		if botH >= h {
			return padBlock(bottom, w, h)
		}
		topH = max(0, h-botH)
		if topH+botH == h {
			if topH == 0 {
				return padBlock(bottom, w, h)
			}
			return lipgloss.JoinVertical(lipgloss.Left, padBlock(top, w, topH), padBlock(bottom, w, botH))
		}
	}
	midH := max(1, h-topH-botH)
	parts := make([]string, 0, 3)
	if topH > 0 {
		parts = append(parts, padBlock(top, w, topH))
	}
	parts = append(parts, padBlock(mid, w, midH))
	if botH > 0 {
		parts = append(parts, padBlock(bottom, w, botH))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func padBlock(s string, width, height int) string {
	if height < 1 {
		height = 1
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		w := lipgloss.Width(lines[i])
		if w < width {
			lines[i] += strings.Repeat(" ", width-w)
		} else if width > 0 && w > width {
			lines[i] = clipLine(lines[i], width)
		}
	}
	pad := strings.Repeat(" ", max(0, width))
	for len(lines) < height {
		lines = append(lines, pad)
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}

// boxNoWrap sizes s to w×h inside st without word-wrapping. Lip Gloss Width()
// wraps by default, which warps borders; we clip first so Width only pads.
func boxNoWrap(st lipgloss.Style, s string, w, h int) string {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	innerW := max(1, w-st.GetHorizontalFrameSize())
	innerH := max(1, h-st.GetVerticalFrameSize())
	return st.Width(w).Height(h).MaxWidth(w).MaxHeight(h).Render(padBlock(s, innerW, innerH))
}

func cropLine(s string, x, w int) string {
	if w <= 0 {
		return ""
	}
	if x < 0 {
		x = 0
	}
	if x >= lipgloss.Width(s) {
		return strings.Repeat(" ", w)
	}
	return padBlock(ansi.Cut(s, x, x+w), w, 1)
}

// cropBlock returns the w×h window of s whose top-left is (x, y) in cells.
func cropBlock(s string, x, y, w, h int) string {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if s == "" {
		return padBlock("", w, h)
	}
	lines := strings.Split(s, "\n")
	if y < 0 {
		y = 0
	}
	if y > len(lines) {
		y = len(lines)
	}
	end := min(len(lines), y+h)
	var out []string
	if y < len(lines) {
		for _, line := range lines[y:end] {
			out = append(out, cropLine(line, x, w))
		}
	}
	return padBlock(strings.Join(out, "\n"), w, h)
}

func renderScrollbar(viewH, totalH, offset int, s Styles) string {
	if viewH < 1 {
		return ""
	}
	thumbH := max(1, viewH*viewH/max(1, totalH))
	if thumbH > viewH {
		thumbH = viewH
	}
	maxOff := totalH - viewH
	thumbY := 0
	if maxOff > 0 {
		thumbY = offset * (viewH - thumbH) / maxOff
	}
	var b strings.Builder
	for i := 0; i < viewH; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i >= thumbY && i < thumbY+thumbH {
			b.WriteString(s.Accent.Render("┃"))
		} else {
			b.WriteString(s.Dim.Render("│"))
		}
	}
	return b.String()
}

// scrollbarCols is the bar plus the gap between it and the content.
const scrollbarCols = 2

// joinScrollbar pins a vertical scrollbar to the right edge of a frameW-wide
// block so the thumb stays on the frame, not on the content's ragged edge.
func joinScrollbar(content string, frameW, viewH, totalH, offset int, s Styles) string {
	if totalH <= viewH || viewH < 1 {
		return content
	}
	if frameW < 1 {
		frameW = lipgloss.Width(content) + scrollbarCols
	}
	contentW := max(0, frameW-scrollbarCols)
	return lipgloss.JoinHorizontal(lipgloss.Top,
		padBlock(content, contentW, viewH),
		" ",
		renderScrollbar(viewH, totalH, offset, s),
	)
}
