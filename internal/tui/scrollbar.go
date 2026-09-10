package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
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
