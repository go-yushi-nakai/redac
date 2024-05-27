package redac

import (
	"encoding/csv"
	"encoding/json"
	"io"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

type Renderer interface {
	SetShowHeader(bool)
	Render(io.Writer, [][]string) error
}

type rendererBase struct {
	ShowHeader bool
}

func (r *rendererBase) SetShowHeader(showHeader bool) {
	r.ShowHeader = showHeader
}

type tableType int

const (
	TableType1 tableType = iota
	TableType2
)

type TableRenderer struct {
	rendererBase
	TableType tableType
}

func (r *TableRenderer) Render(w io.Writer, data [][]string) error {
	table := tablewriter.NewWriter(w)
	switch r.TableType {
	case TableType1:
		table.SetBorder(false)
	case TableType2:
		table.SetAutoWrapText(false)
		table.SetAutoFormatHeaders(true)
		table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetCenterSeparator("")
		table.SetColumnSeparator("")
		table.SetRowSeparator("")
		table.SetHeaderLine(false)
		table.SetBorder(false)
		table.SetTablePadding("\t")
		table.SetNoWhiteSpace(true)
	}
	if r.ShowHeader {
		table.SetHeader(data[0])
	}

	for _, v := range data[1:] {
		table.Append(v)
	}
	table.Render()
	return nil
}

type CSVRenderer struct {
	rendererBase
}

func (r *CSVRenderer) Render(w io.Writer, data [][]string) error {
	if !r.ShowHeader {
		data = data[1:]
	}
	return csv.NewWriter(w).WriteAll(data)
}

type JSONRenderer struct {
	rendererBase
}

func (r *JSONRenderer) Render(w io.Writer, data [][]string) error {
	if !r.ShowHeader {
		data = data[1:]
	}
	return json.NewEncoder(w).Encode(data)
}

type YAMLRenderer struct {
	rendererBase
}

func (r *YAMLRenderer) Render(w io.Writer, data [][]string) error {
	if !r.ShowHeader {
		data = data[1:]
	}
	return yaml.NewEncoder(w).Encode(data)
}
