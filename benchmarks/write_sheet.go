package main

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// sheetRows renders a whole regular table as rows of text.
//
// The layout is the four-region one Archmage expects: a superheader in column
// A, field headers to its right, config IDs down the first column, and the
// config data filling the rest. A multi-column field repeats its Desc, Name
// and Type across every column it spans; Archmage groups columns by Name.
func (c *Corpus) sheetRows(t *Table, labels [4]string) [][]string {
	width := 1
	hasOpts := false
	for _, f := range t.Fields {
		width += f.Cols
		if f.ColOpts != nil {
			hasOpts = true
		}
	}

	descRow := make([]string, 0, width)
	nameRow := make([]string, 0, width)
	typeRow := make([]string, 0, width)
	optsRow := make([]string, 0, width)
	descRow = append(descRow, labels[0])
	nameRow = append(nameRow, labels[1])
	typeRow = append(typeRow, labels[2])
	optsRow = append(optsRow, labels[3])
	for _, f := range t.Fields {
		for i := range f.Cols {
			// Continuation columns of a multi-column field carry the lookback
			// operator "<1" rather than a repeated value. It is the documented
			// alternative to merged header cells and reads the same way in
			// xlsx and csv.
			if i == 0 {
				descRow = append(descRow, f.Desc)
				nameRow = append(nameRow, f.Name)
				typeRow = append(typeRow, f.Type)
			} else {
				descRow = append(descRow, "<1")
				nameRow = append(nameRow, "<1")
				typeRow = append(typeRow, "<1")
			}
			if f.ColOpts != nil {
				optsRow = append(optsRow, f.ColOpts[i])
			} else {
				optsRow = append(optsRow, "")
			}
		}
	}

	rows := make([][]string, 0, t.rows+4)
	rows = append(rows, descRow, nameRow, typeRow)
	if hasOpts {
		rows = append(rows, optsRow)
	}
	for row := range t.rows {
		line := make([]string, 0, width)
		line = append(line, strconv.FormatInt(t.ID(row), 10))
		for _, f := range t.Fields {
			line = append(line, f.Cells(row)...)
		}
		rows = append(rows, line)
	}
	return rows
}

// writeXLSX writes one worksheet per file, which is the plain case: without an
// index table Archmage exports the first worksheet and derives the export name
// from the file name.
func (c *Corpus) writeXLSX(dir string, t *Table) (int, error) {
	rows := c.sheetRows(t, [4]string{"Desc", "Name", "Type", "Options"})

	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()
	idx, err := f.NewSheet(t.Name)
	if err != nil {
		return 0, err
	}
	f.SetActiveSheet(idx)
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return 0, err
	}
	sw, err := f.NewStreamWriter(t.Name)
	if err != nil {
		return 0, err
	}
	nonEmpty := 0
	buf := make([]any, 0, len(rows[0]))
	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return 0, err
		}
		buf = buf[:0]
		for _, v := range row {
			buf = append(buf, v)
		}
		if err := sw.SetRow(cell, buf); err != nil {
			return 0, err
		}
		if i >= len(rows)-t.rows {
			nonEmpty += countFilled(row[1:])
		}
	}
	if err := sw.Flush(); err != nil {
		return 0, err
	}
	return nonEmpty, f.SaveAs(filepath.Join(dir, t.File))
}

// writeCSV writes a regular table in CSV form. The layout matches the xlsx
// one; only the superheader labels are spelled the way CSV authors write them.
func (c *Corpus) writeCSV(dir string, t *Table) (int, error) {
	rows := c.sheetRows(t, [4]string{"description", "name", "type", "options"})

	fh, err := os.Create(filepath.Join(dir, t.File))
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = fh.Close()
	}()

	w := csv.NewWriter(fh)
	nonEmpty := 0
	for i, row := range rows {
		if err := w.Write(row); err != nil {
			return 0, err
		}
		if i >= len(rows)-t.rows {
			nonEmpty += countFilled(row[1:])
		}
	}
	w.Flush()
	return nonEmpty, w.Error()
}

func countFilled(cells []string) int {
	n := 0
	for _, v := range cells {
		if v != "" {
			n++
		}
	}
	return n
}
