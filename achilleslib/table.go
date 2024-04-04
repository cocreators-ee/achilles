package achilleslib

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
)

type bareRow struct {
	lib  string
	uses int32
}

// By is the type of a "less" function that defines the ordering of its Planet arguments.
type By func(r1, r2 *bareRow) bool

var madeTable = false

// Sort is a method on the function type, By, that sorts the argument slice according to the function.
func (by By) Sort(rows []bareRow) {
	ps := &rowSorter{
		rows: rows,
		by:   by, // The Sort method's receiver is the function (closure) that defines the sort order.
	}
	sort.Sort(ps)
}

type rowSorter struct {
	rows []bareRow
	by   func(r1, r2 *bareRow) bool // Closure used in the Less method.
}

// Len is part of sort.Interface.
func (s *rowSorter) Len() int {
	return len(s.rows)
}

// Swap is part of sort.Interface.
func (s *rowSorter) Swap(i, j int) {
	s.rows[i], s.rows[j] = s.rows[j], s.rows[i]
}

// Less is part of sort.Interface. It is implemented by calling the "by" closure in the sorter.
func (s *rowSorter) Less(i, j int) bool {
	return s.by(&s.rows[i], &s.rows[j])
}

func getTable() table.Model {
	// Mke rows in initial State, to be sorted
	bareRows := []bareRow{}
	longest := 20
	mapMutex.Lock()
	total := 1
	if GlobalModel != nil {
		total = len(GlobalModel.Libs) + len(GlobalModel.Bins)
		for lib, uses := range GlobalModel.Libs {
			length := len(lib)
			if length > longest {
				longest = length
			}
			bareRows = append(bareRows, bareRow{
				lib:  lib,
				uses: uses.Load(),
			})
		}
	}
	mapMutex.Unlock()

	// Sort
	uses_name := func(r1, r2 *bareRow) bool {
		if r1.uses == r2.uses {
			return r1.lib < r2.lib
		}
		return r1.uses > r2.uses
	}
	By(uses_name).Sort(bareRows)

	// Now format them into table rows
	rows := []table.Row{}
	for idx, br := range bareRows {
		pct := fmt.Sprintf("%.1f%%", float64(br.uses)/float64(total)*100)
		rows = append(rows, table.Row{IntFormat(idx + 1), br.lib, Int32Format(br.uses), pct})
	}

	// Dynamic columns
	columns := []table.Column{
		{Title: "#", Width: len(IntFormat(len(rows) + 1))},
		{Title: "Library", Width: longest},
		{Title: "Uses", Width: 6},
		{Title: "%", Width: 5},
	}

	if madeTable {
		// Update existing table
		t := GlobalModel.Table
		t.SetColumns(columns)
		t.SetRows(rows)
		return t
	}

	// Create a new table
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(25),
	)

	s := table.DefaultStyles()

	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("220")).
		Foreground(lipgloss.Color("33")).
		BorderBottom(true).
		Bold(false)

	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	t.SetStyles(s)

	madeTable = true
	return t
}
