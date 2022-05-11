// Copyright 2014 Oleku Konko All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// This module is a Table Writer  API for the Go Programming Language.
// The protocols were written in pure Go and works on windows and unix systems

// Create & Generate text based table
package tablewriter

import (
    "bytes"
    "errors"
    "fmt"
    "io"
    "reflect"
    "regexp"
    "strings"
)

const (
    MAX_ROW_WIDTH = 30
)

const (
    CENTER_ALL = "┼"
    CENTER_NES = "\x1b[2m├"
    CENTER_NSW = "┤"
    CENTER_NEW = "┴"
    CENTER_ESW = "┬"
    CENTER_NE  = "\x1b[2m└"
    CENTER_WN  = "┘"
    CENTER_SW  = "┐"
    CENTER_ES  = "\x1b[2m┌"
    ROW        = "─"
    COLUMN     = "\x1b[2m│\x1b[0m"
    SPACE      = " "
    NEWLINE    = "\n"
)

const (
    ALIGN_DEFAULT = iota
    ALIGN_CENTER
    ALIGN_RIGHT
    ALIGN_LEFT
)

var (
    decimal = regexp.MustCompile(`^-?(?:\d{1,3}(?:,\d{3})*|\d+)(?:\.\d+)?$`)
    percent = regexp.MustCompile(`^-?\d+\.?\d*%$`)
)

type Border struct {
    Left   bool
    Right  bool
    Top    bool
    Bottom bool
}

type Table struct {
    out                     io.Writer
    rows                    [][]string
    lines                   [][][]string
    cs                      map[int]int
    rs                      map[int]int
    headers                 [][]string
    autoFmt                 bool
    autoWrap                bool
    reflowText              bool
    mW                      int
    tColumn                 int
    tRow                    int
    hAlign                  int
    fAlign                  int
    align                   int
    newLine                 string
    rowLine                 bool
    autoMergeCells          bool
    columnsToAutoMergeCells map[int]bool
    noWhiteSpace            bool
    tablePadding            string
    hdrLine                 bool
    colSize                 int
    headerParams            []string
    columnsParams           []string
    columnsAlign            []int
}

// Start New Table
// Take io.Writer Directly
func NewWriter(writer io.Writer) *Table {
    t := &Table{
        out:           writer,
        rows:          [][]string{},
        lines:         [][][]string{},
        cs:            make(map[int]int),
        rs:            make(map[int]int),
        headers:       [][]string{},
        autoFmt:       true,
        autoWrap:      true,
        reflowText:    true,
        mW:            MAX_ROW_WIDTH,
        tColumn:       -1,
        tRow:          -1,
        hAlign:        ALIGN_DEFAULT,
        fAlign:        ALIGN_DEFAULT,
        align:         ALIGN_DEFAULT,
        newLine:       NEWLINE,
        rowLine:       false,
        hdrLine:       true,
        colSize:       -1,
        headerParams:  []string{},
        columnsParams: []string{},
        columnsAlign:  []int{}}
    return t
}

// Render table output
func (t *Table) Render() {
    t.printLine(true, true, false)
    t.printHeading()
    if t.autoMergeCells {
        t.printRowsMergeCells()
    } else {
        t.printRows()
    }
    if !t.rowLine {
        t.printLine(true, false, true)
    }
}

const (
    headerRowIdx = -1
    footerRowIdx = -2
)

// Set table header
func (t *Table) SetHeader(keys []string) {
    t.colSize = len(keys)
    for i, v := range keys {
        lines := t.parseDimension(v, i, headerRowIdx)
        t.headers = append(t.headers, lines)
    }
}

// Set the Default column width
func (t *Table) SetColWidth(width int) {
    t.mW = width
}

// Set the minimal width for a column
func (t *Table) SetColMinWidth(column int, width int) {
    t.cs[column] = width
}

// Set Header Alignment
func (t *Table) SetHeaderAlignment(hAlign int) {
    t.hAlign = hAlign
}

// Set Table Alignment
func (t *Table) SetAlignment(align int) {
    t.align = align
}

// Set No White Space
func (t *Table) SetNoWhiteSpace(allow bool) {
    t.noWhiteSpace = allow
}

// Set Table Padding
func (t *Table) SetTablePadding(padding string) {
    t.tablePadding = padding
}

func (t *Table) SetColumnAlignment(keys []int) {
    for _, v := range keys {
        switch v {
        case ALIGN_CENTER:
            break
        case ALIGN_LEFT:
            break
        case ALIGN_RIGHT:
            break
        default:
            v = ALIGN_DEFAULT
        }
        t.columnsAlign = append(t.columnsAlign, v)
    }
}

// Set New Line
func (t *Table) SetNewLine(nl string) {
    t.newLine = nl
}

// Set Header Line
// This would enable / disable a line after the header
func (t *Table) SetHeaderLine(line bool) {
    t.hdrLine = line
}

// Set Row Line
// This would enable / disable a line on each row of the table
func (t *Table) SetRowLine(line bool) {
    t.rowLine = line
}

// Set Auto Merge Cells
// This would enable / disable the merge of cells with identical values
func (t *Table) SetAutoMergeCells(auto bool) {
    t.autoMergeCells = auto
}

// Set Auto Merge Cells By Column Index
// This would enable / disable the merge of cells with identical values for specific columns
// If cols is empty, it is the same as `SetAutoMergeCells(true)`.
func (t *Table) SetAutoMergeCellsByColumnIndex(cols []int) {
    t.autoMergeCells = true

    if len(cols) > 0 {
        m := make(map[int]bool)
        for _, col := range cols {
            m[col] = true
        }
        t.columnsToAutoMergeCells = m
    }
}

// SetStructs sets header and rows from slice of struct.
// If something that is not a slice is passed, error will be returned.
// The tag specified by "tablewriter" for the struct becomes the header.
// If not specified or empty, the field name will be used.
// The field of the first element of the slice is used as the header.
// If the element implements fmt.Stringer, the result will be used.
// And the slice contains nil, it will be skipped without rendering.
func (t *Table) SetStructs(v interface{}) error {
    if v == nil {
        return errors.New("nil value")
    }
    vt := reflect.TypeOf(v)
    vv := reflect.ValueOf(v)
    switch vt.Kind() {
    case reflect.Slice, reflect.Array:
        if vv.Len() < 1 {
            return errors.New("empty value")
        }

        // check first element to set header
        first := vv.Index(0)
        e := first.Type()
        switch e.Kind() {
        case reflect.Struct:
            // OK
        case reflect.Ptr:
            if first.IsNil() {
                return errors.New("the first element is nil")
            }
            e = first.Elem().Type()
            if e.Kind() != reflect.Struct {
                return fmt.Errorf("invalid kind %s", e.Kind())
            }
        default:
            return fmt.Errorf("invalid kind %s", e.Kind())
        }
        n := e.NumField()
        headers := make([]string, n)
        for i := 0; i < n; i++ {
            f := e.Field(i)
            header := f.Tag.Get("tablewriter")
            if header == "" {
                header = f.Name
            }
            headers[i] = header
        }
        t.SetHeader(headers)

        for i := 0; i < vv.Len(); i++ {
            item := reflect.Indirect(vv.Index(i))
            itemType := reflect.TypeOf(item)
            switch itemType.Kind() {
            case reflect.Struct:
                // OK
            default:
                return fmt.Errorf("invalid item type %v", itemType.Kind())
            }
            if !item.IsValid() {
                // skip rendering
                continue
            }
            nf := item.NumField()
            if n != nf {
                return errors.New("invalid num of field")
            }
            rows := make([]string, nf)
            for j := 0; j < nf; j++ {
                f := reflect.Indirect(item.Field(j))
                if f.Kind() == reflect.Ptr {
                    f = f.Elem()
                }
                if f.IsValid() {
                    if s, ok := f.Interface().(fmt.Stringer); ok {
                        rows[j] = s.String()
                        continue
                    }
                    rows[j] = fmt.Sprint(f)
                } else {
                    rows[j] = "nil"
                }
            }
            t.Append(rows)
        }
    default:
        return fmt.Errorf("invalid type %T", v)
    }
    return nil
}

// Append row to table
func (t *Table) Append(row []string) {
    rowSize := len(t.headers)
    if rowSize > t.colSize {
        t.colSize = rowSize
    }

    n := len(t.lines)
    line := [][]string{}
    for i, v := range row {

        // Detect string  width
        // Detect String height
        // Break strings into words
        out := t.parseDimension(v, i, n)

        // Append broken words
        line = append(line, out)
    }
    t.lines = append(t.lines, line)
}

// Append row to table with color attributes
func (t *Table) Rich(row []string, colors []Colors) {
    rowSize := len(t.headers)
    if rowSize > t.colSize {
        t.colSize = rowSize
    }

    n := len(t.lines)
    line := [][]string{}
    for i, v := range row {

        // Detect string  width
        // Detect String height
        // Break strings into words
        out := t.parseDimension(v, i, n)

        if len(colors) > i {
            color := colors[i]
            out[0] = format(out[0], color)
        }

        // Append broken words
        line = append(line, out)
    }
    t.lines = append(t.lines, line)
}

// Allow Support for Bulk Append
// Eliminates repeated for loops
func (t *Table) AppendBulk(rows [][]string) {
    for _, row := range rows {
        t.Append(row)
    }
}

// NumLines to get the number of lines
func (t *Table) NumLines() int {
    return len(t.lines)
}

// Clear rows
func (t *Table) ClearRows() {
    t.lines = [][][]string{}
}

// Print line based on row width
func (t *Table) printLine(nl bool, firstRow bool, lastRow bool) {

    switch {
    case firstRow:
        fmt.Fprint(t.out, CENTER_ES)
    case lastRow:
        fmt.Fprint(t.out, CENTER_NE)
    default:
        fmt.Fprint(t.out, CENTER_NES)
    }
    for i := 0; i < len(t.cs); i++ {

        lastCol := i == len(t.cs)-1

        v := t.cs[i]
        fmt.Fprintf(t.out, "%s%s%s",
            ROW,
            strings.Repeat(string(ROW), v),
            ROW)

        switch {
        case lastCol && firstRow:
            fmt.Fprint(t.out, CENTER_SW)
        case lastCol && lastRow:
            fmt.Fprint(t.out, CENTER_WN)
        case lastCol:
            fmt.Fprint(t.out, CENTER_NSW)
        case firstRow:
            fmt.Fprint(t.out, CENTER_ESW)
        case lastRow:
            fmt.Fprint(t.out, CENTER_NEW)
        default:
            fmt.Fprint(t.out, CENTER_ALL)
        }
    }
    if nl {
        fmt.Fprint(t.out, t.newLine)
    }
}

// Print line based on row width with our without cell separator
func (t *Table) printLineOptionalCellSeparators(nl bool, displayCellSeparator []bool) {

    lastHasBorder := false
    nextHasBorder := false

    for i := 0; i < len(t.cs); i++ {

        nextHasBorder = i > len(displayCellSeparator) || displayCellSeparator[i]

        switch {
        case nextHasBorder && lastHasBorder:
            fmt.Fprint(t.out, CENTER_ALL)
        case nextHasBorder:
            fmt.Fprint(t.out, CENTER_NES)
        case lastHasBorder:
            fmt.Fprint(t.out, CENTER_NSW)
        default:
            fmt.Fprint(t.out, COLUMN)
        }

        v := t.cs[i]
        if nextHasBorder {
            // Display the cell separator
            fmt.Fprintf(t.out, "%s%s%s",
                ROW,
                strings.Repeat(string(ROW), v),
                ROW)
        } else {
            // Don't display the cell separator for this cell
            fmt.Fprintf(t.out, "%s",
                strings.Repeat(" ", v+2))
        }

        lastHasBorder = nextHasBorder
    }
    switch {
    case lastHasBorder:
        fmt.Fprint(t.out, CENTER_NSW)
    default:
        fmt.Fprint(t.out, COLUMN)
    }
    if nl {
        fmt.Fprint(t.out, t.newLine)
    }
}

// Return the PadRight function if align is left, PadLeft if align is right,
// and Pad by default
func pad(align int) func(string, string, int) string {
    padFunc := Pad
    switch align {
    case ALIGN_LEFT:
        padFunc = PadRight
    case ALIGN_RIGHT:
        padFunc = PadLeft
    }
    return padFunc
}

// Print heading information
func (t *Table) printHeading() {
    // Check if headers is available
    if len(t.headers) < 1 {
        return
    }

    // Identify last column
    end := len(t.cs) - 1

    // Get pad function
    padFunc := pad(t.hAlign)

    // Checking for ANSI escape sequences for header
    is_esc_seq := false
    if len(t.headerParams) > 0 {
        is_esc_seq = true
    }

    // Maximum height.
    max := t.rs[headerRowIdx]

    // Print Heading
    for x := 0; x < max; x++ {
        // Check if border is set
        // Replace with space if not set
        if !t.noWhiteSpace {
            fmt.Fprint(t.out, COLUMN)
        }

        for y := 0; y <= end; y++ {
            v := t.cs[y]
            h := ""

            if y < len(t.headers) && x < len(t.headers[y]) {
                h = t.headers[y][x]
            }
            if t.autoFmt {
                h = fmt.Sprintf("\x1b[1m%s\x1b[0m", Title(h))
            }
            pad := COLUMN
            if t.noWhiteSpace {
                pad = t.tablePadding
            }
            if is_esc_seq {
                if !t.noWhiteSpace {
                    fmt.Fprintf(t.out, " %s %s",
                        format(padFunc(h, SPACE, v),
                            t.headerParams[y]), pad)
                } else {
                    fmt.Fprintf(t.out, "%s %s",
                        format(padFunc(h, SPACE, v),
                            t.headerParams[y]), pad)
                }
            } else {
                if !t.noWhiteSpace {
                    fmt.Fprintf(t.out, " %s %s",
                        padFunc(h, SPACE, v),
                        pad)
                } else {
                    // the spaces between breaks the kube formatting
                    fmt.Fprintf(t.out, "%s%s",
                        padFunc(h, SPACE, v),
                        pad)
                }
            }
        }
        // Next line
        fmt.Fprint(t.out, t.newLine)
    }
    if t.hdrLine {
        t.printLine(true, false, false)
    }
}

// Calculate the total number of characters in a row
func (t Table) getTableWidth() int {
    var chars int
    for _, v := range t.cs {
        chars += v
    }

    // Add chars, spaces, seperators to calculate the total width of the table.
    // ncols := t.colSize
    // spaces := ncols * 2
    // seps := ncols + 1

    return (chars + (3 * t.colSize) + 2)
}

func (t Table) printRows() {
    for i, lines := range t.lines {
        t.printRow(lines, i, i == len(t.lines)-1)
    }
}

func (t *Table) fillAlignment(num int) {
    if len(t.columnsAlign) < num {
        t.columnsAlign = make([]int, num)
        for i := range t.columnsAlign {
            t.columnsAlign[i] = t.align
        }
    }
}

// Print Row Information
// Adjust column alignment based on type

func (t *Table) printRow(columns [][]string, rowIdx int, last bool) {
    // Get Maximum Height
    max := t.rs[rowIdx]
    total := len(columns)

    // TODO Fix uneven col size
    // if total < t.colSize {
    //	for n := t.colSize - total; n < t.colSize ; n++ {
    //		columns = append(columns, []string{SPACE})
    //		t.cs[n] = t.mW
    //	}
    //}

    // Pad Each Height
    pads := []int{}

    // Checking for ANSI escape sequences for columns
    is_esc_seq := false
    if len(t.columnsParams) > 0 {
        is_esc_seq = true
    }
    t.fillAlignment(total)

    for i, line := range columns {
        length := len(line)
        pad := max - length
        pads = append(pads, pad)
        for n := 0; n < pad; n++ {
            columns[i] = append(columns[i], "  ")
        }
    }
    //fmt.Println(max, "\n")
    for x := 0; x < max; x++ {
        for y := 0; y < total; y++ {

            // Check if border is set
            if !t.noWhiteSpace {
                fmt.Fprint(t.out, COLUMN)
                fmt.Fprintf(t.out, SPACE)
            }

            str := columns[y][x]

            // Embedding escape sequence with column value
            if is_esc_seq {
                str = format(str, t.columnsParams[y])
            }

            // This would print alignment
            // Default alignment  would use multiple configuration
            switch t.columnsAlign[y] {
            case ALIGN_CENTER: //
                fmt.Fprintf(t.out, "%s", Pad(str, SPACE, t.cs[y]))
            case ALIGN_RIGHT:
                fmt.Fprintf(t.out, "%s", PadLeft(str, SPACE, t.cs[y]))
            case ALIGN_LEFT:
                fmt.Fprintf(t.out, "%s", PadRight(str, SPACE, t.cs[y]))
            default:
                if decimal.MatchString(strings.TrimSpace(str)) || percent.MatchString(strings.TrimSpace(str)) {
                    fmt.Fprintf(t.out, "%s", PadLeft(str, SPACE, t.cs[y]))
                } else {
                    fmt.Fprintf(t.out, "%s", PadRight(str, SPACE, t.cs[y]))

                    // TODO Custom alignment per column
                    //if max == 1 || pads[y] > 0 {
                    //	fmt.Fprintf(t.out, "%s", Pad(str, SPACE, t.cs[y]))
                    //} else {
                    //	fmt.Fprintf(t.out, "%s", PadRight(str, SPACE, t.cs[y]))
                    //}

                }
            }
            if !t.noWhiteSpace {
                fmt.Fprintf(t.out, SPACE)
            } else {
                fmt.Fprintf(t.out, t.tablePadding)
            }
        }
        // Check if border is set
        // Replace with space if not set
        if !t.noWhiteSpace {
            fmt.Fprint(t.out, COLUMN)
        }
        fmt.Fprint(t.out, t.newLine)
    }

    if t.rowLine {
        t.printLine(true, false, last)
    }
}

// Print the rows of the table and merge the cells that are identical
func (t *Table) printRowsMergeCells() {
    var previousLine []string
    var displayCellBorder []bool
    var tmpWriter bytes.Buffer
    for i, lines := range t.lines {
        // We store the display of the current line in a tmp writer, as we need to know which border needs to be print above
        previousLine, displayCellBorder = t.printRowMergeCells(&tmpWriter, lines, i, previousLine)
        if i > 0 { //We don't need to print borders above first line
            if t.rowLine {
                t.printLineOptionalCellSeparators(true, displayCellBorder)
            }
        }
        tmpWriter.WriteTo(t.out)
    }
    //Print the end of the table
    if t.rowLine {
        t.printLine(true, false, true)
    }
}

// Print Row Information to a writer and merge identical cells.
// Adjust column alignment based on type

func (t *Table) printRowMergeCells(writer io.Writer, columns [][]string, rowIdx int, previousLine []string) ([]string, []bool) {
    // Get Maximum Height
    max := t.rs[rowIdx]
    total := len(columns)

    // Pad Each Height
    pads := []int{}

    // Checking for ANSI escape sequences for columns
    is_esc_seq := false
    if len(t.columnsParams) > 0 {
        is_esc_seq = true
    }
    for i, line := range columns {
        length := len(line)
        pad := max - length
        pads = append(pads, pad)
        for n := 0; n < pad; n++ {
            columns[i] = append(columns[i], "  ")
        }
    }

    var displayCellBorder []bool
    t.fillAlignment(total)
    for x := 0; x < max; x++ {
        for y := 0; y < total; y++ {

            // Check if border is set
            fmt.Fprint(writer, COLUMN)

            fmt.Fprintf(writer, SPACE)

            str := columns[y][x]

            // Embedding escape sequence with column value
            if is_esc_seq {
                str = format(str, t.columnsParams[y])
            }

            if t.autoMergeCells {
                var mergeCell bool
                if t.columnsToAutoMergeCells != nil {
                    // Check to see if the column index is in columnsToAutoMergeCells.
                    if t.columnsToAutoMergeCells[y] {
                        mergeCell = true
                    }
                } else {
                    // columnsToAutoMergeCells was not set.
                    mergeCell = true
                }
                //Store the full line to merge mutli-lines cells
                fullLine := strings.TrimRight(strings.Join(columns[y], " "), " ")
                if len(previousLine) > y && fullLine == previousLine[y] && fullLine != "" && mergeCell {
                    // If this cell is identical to the one above but not empty, we don't display the border and keep the cell empty.
                    displayCellBorder = append(displayCellBorder, false)
                    str = ""
                } else {
                    // First line or different content, keep the content and print the cell border
                    displayCellBorder = append(displayCellBorder, true)
                }
            }

            // This would print alignment
            // Default alignment  would use multiple configuration
            switch t.columnsAlign[y] {
            case ALIGN_CENTER: //
                fmt.Fprintf(writer, "%s", Pad(str, SPACE, t.cs[y]))
            case ALIGN_RIGHT:
                fmt.Fprintf(writer, "%s", PadLeft(str, SPACE, t.cs[y]))
            case ALIGN_LEFT:
                fmt.Fprintf(writer, "%s", PadRight(str, SPACE, t.cs[y]))
            default:
                if decimal.MatchString(strings.TrimSpace(str)) || percent.MatchString(strings.TrimSpace(str)) {
                    fmt.Fprintf(writer, "%s", PadLeft(str, SPACE, t.cs[y]))
                } else {
                    fmt.Fprintf(writer, "%s", PadRight(str, SPACE, t.cs[y]))
                }
            }
            fmt.Fprintf(writer, SPACE)
        }
        // Check if border is set
        // Replace with space if not set
        fmt.Fprint(writer, COLUMN)
        fmt.Fprint(writer, t.newLine)
    }

    //The new previous line is the current one
    previousLine = make([]string, total)
    for y := 0; y < total; y++ {
        previousLine[y] = strings.TrimRight(strings.Join(columns[y], " "), " ") //Store the full line for multi-lines cells
    }
    //Returns the newly added line and wether or not a border should be displayed above.
    return previousLine, displayCellBorder
}

func (t *Table) parseDimension(str string, colKey, rowKey int) []string {
    var (
        raw      []string
        maxWidth int
    )

    raw = getLines(str)
    maxWidth = 0
    for _, line := range raw {
        if w := DisplayWidth(line); w > maxWidth {
            maxWidth = w
        }
    }

    // If wrapping, ensure that all paragraphs in the cell fit in the
    // specified width.
    if t.autoWrap {
        // If there's a maximum allowed width for wrapping, use that.
        if maxWidth > t.mW {
            maxWidth = t.mW
        }

        // In the process of doing so, we need to recompute maxWidth. This
        // is because perhaps a word in the cell is longer than the
        // allowed maximum width in t.mW.
        newMaxWidth := maxWidth
        newRaw := make([]string, 0, len(raw))

        if t.reflowText {
            // Make a single paragraph of everything.
            raw = []string{strings.Join(raw, " ")}
        }
        for i, para := range raw {
            paraLines, _ := WrapString(para, maxWidth)
            for _, line := range paraLines {
                if w := DisplayWidth(line); w > newMaxWidth {
                    newMaxWidth = w
                }
            }
            if i > 0 {
                newRaw = append(newRaw, " ")
            }
            newRaw = append(newRaw, paraLines...)
        }
        raw = newRaw
        maxWidth = newMaxWidth
    }

    // Store the new known maximum width.
    v, ok := t.cs[colKey]
    if !ok || v < maxWidth || v == 0 {
        t.cs[colKey] = maxWidth
    }

    // Remember the number of lines for the row printer.
    h := len(raw)
    v, ok = t.rs[rowKey]

    if !ok || v < h || v == 0 {
        t.rs[rowKey] = h
    }
    //fmt.Printf("Raw %+v %d\n", raw, len(raw))
    return raw
}
