package main

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
)

const previewBytesLimit int64 = 10_000

type Renderer struct {
	EdgePadding int
	offsetMem   int
}

func (r *Renderer) Render(tree *Tree, winHeight, winWidth int) string {
	// rendering tree
	renderedTree, selectedRow := r.renderTree(tree)
	croppedTree := r.cropTree(renderedTree, selectedRow, winHeight)

	sectionWidth := int(math.Floor(0.5 * float64(winWidth)))
	treeStyle := lipgloss.
		NewStyle().
		MaxWidth(sectionWidth)
	renderedStyledTree := treeStyle.Render(strings.Join(croppedTree, "\n"))

	// rendering file content
	content := make([]byte, previewBytesLimit)
	n, err := tree.ReadSelectedChildContent(content, previewBytesLimit)
	if err != nil {
		return renderedStyledTree
	}
	content = content[:n]

	leftMargin := sectionWidth - lipgloss.Width(renderedStyledTree)
	contentStyle := lipgloss.
		NewStyle().
		Italic(true).
		MarginLeft(leftMargin).
		MaxWidth(sectionWidth + leftMargin - 1).
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true)

	var contentLines []string
	if !utf8.Valid(content) {
		contentLines = []string{"<binary content>"}
	} else {
		contentLines = strings.Split(string(content), "\n")
		contentLines = contentLines[:max(min(winHeight, len(contentLines)), 0)]
	}
	renderedStyledTree = lipgloss.JoinHorizontal(
		0,
		renderedStyledTree,
		contentStyle.Render(strings.Join(contentLines, "\n")),
	)
	return renderedStyledTree
}

func (r *Renderer) cropTree(lines []string, currentLine int, windowHeight int) []string {
	linesLen := len(lines)

	// determining offset and limit based on selected row
	offset := r.offsetMem
	limit := linesLen
	if windowHeight > 0 {
		// cursor is out for 'top' boundary
		if currentLine+1 > windowHeight+offset-r.EdgePadding {
			offset = min(currentLine+1-windowHeight+r.EdgePadding, linesLen-windowHeight)
		}
		// cursor is out for 'bottom' boundary
		if currentLine < r.EdgePadding+offset {
			offset = max(currentLine-r.EdgePadding, 0)
		}
		r.offsetMem = offset
		limit = min(windowHeight+offset, linesLen)
	}
	return lines[offset:limit]
}

// Returns lines as slice and index of selected line
func (r *Renderer) renderTree(tree *Tree) ([]string, int) {
	cnt := -1
	selectedRow := 0

	type stackEl struct {
		*Node
		int
	}
	lines := []string{}
	s := newStack(stackEl{tree.Root, 0})

	for s.Len() > 0 {
		el := s.Pop()
		cnt += 1

		node := el.Node
		depth := el.int

		if node == nil {
			continue
		}
		name := node.Info.Name()
		if node.Info.IsDir() {
			name = color.BlueString(node.Info.Name())
		}
		repr := strings.Repeat("  ", depth) + name
		if tree.GetSelectedChild() == node {
			repr += color.YellowString(" <-")
			selectedRow = cnt
		}
		lines = append(lines, repr)

		if node.Children != nil {
			// current directory is empty
			if len(node.Children) == 0 && tree.CurrentDir == node {
				lines = append(lines, strings.Repeat("  ", depth+1)+"..."+color.YellowString(" <-"))
				selectedRow = cnt + 1
			}
			for i := len(node.Children) - 1; i >= 0; i-- {
				ch := node.Children[i]
				s.Push(stackEl{ch, depth + 1})
			}
		}
	}
	return lines, selectedRow
}

type stack[T any] struct {
	items []T
}

func (s *stack[T]) Push(el ...T) {
	s.items = append(s.items, el...)
}
func (s *stack[_]) Len() int {
	return len(s.items)
}
func (s *stack[T]) Pop() T {
	el := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return el
}

func newStack[T any](els ...T) stack[T] {
	s := stack[T]{}
	s.Push(els...)
	return s
}
