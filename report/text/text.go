package text

import (
	"fmt"
	"io"

	"github.com/tufin/oasdiff/diff"
)

// Report is a simplified OpenAPI diff report in text format
type Report struct {
	Writer io.Writer
	level  int
}

func (report *Report) indent() *Report {
	return &Report{
		Writer: report.Writer,
		level:  report.level + 1,
	}
}

func (report *Report) print(output ...interface{}) (n int, err error) {
	return fmt.Fprintln(report.Writer, addPrefix(report.level, output)...)
}

func addPrefix(level int, output []interface{}) []interface{} {
	return append(getPrefix(level), output...)
}

func getPrefix(level int) []interface{} {
	switch level {
	case 1:
		return []interface{}{"*"}
	case 2:
		return []interface{}{"  -"}
	}

	return []interface{}{}
}

// Output outputs a textual diff report
func (report *Report) Output(d *diff.Diff) {

	if d.Empty() {
		report.print("No changes")
		return
	}

	if d.EndpointsDiff.Empty() {
		report.print("No endpoint changes")
		return
	}

	report.print("### New Endpoints")
	report.print("-----------------")
	for _, added := range d.EndpointsDiff.Added {
		report.print(added.Method, added.Path)
	}
	report.print("")

	report.print("### Deleted Endpoints")
	report.print("---------------------")
	for _, deleted := range d.EndpointsDiff.Deleted {
		report.print(deleted.Method, deleted.Path)
	}
	report.print("")

	report.print("### Modified Endpoints")
	report.print("----------------------")
	for endpoint, methodDiff := range d.EndpointsDiff.Modified {
		report.print(endpoint.Method, endpoint.Path)
		report.indent().printMethod(methodDiff)
		report.print("")
	}
}

func (report *Report) printMethod(d *diff.MethodDiff) {
	if d.Empty() {
		return
	}

	if !d.DescriptionDiff.Empty() {
		report.print("Description changed from: ", d.DescriptionDiff.From, "To:", d.DescriptionDiff.To)
	}

	report.printParams(d.ParametersDiff)

	if !d.RequestBodyDiff.Empty() {
		report.print("Request body changed")
	}

	if !d.ResponsesDiff.Empty() {
		report.print("Response changed")
		report.indent().printResponses(d.ResponsesDiff)
	}

	if !d.CallbacksDiff.Empty() {
		report.print("Callbacks changed")
	}

	if !d.SecurityDiff.Empty() {
		report.print("Security changed")
	}
}

func (report *Report) printParams(d *diff.ParametersDiff) {
	if d.Empty() {
		return
	}

	for location, params := range d.Added {
		for _, param := range params {
			report.print("New", location, "param:", param)
		}
	}

	for location, params := range d.Deleted {
		for _, param := range params {
			report.print("Deleted", location, "param:", param)
		}
	}

	for location, paramDiffs := range d.Modified {
		for param, paramDiff := range paramDiffs {
			report.print("Modified", location, "param:", param)
			report.indent().printParam(paramDiff)
		}
	}
}

func (report *Report) printParam(d *diff.ParameterDiff) {
	if !d.SchemaDiff.Empty() {
		report.print("Schema changed")
		report.printSchema(d.SchemaDiff)
	}

	if !d.ContentDiff.Empty() {
		report.print("Content changed")
	}
}

func (report *Report) printSchema(d *diff.SchemaDiff) {
	if d.Empty() {
		return
	}
}

func (report *Report) printResponses(d *diff.ResponsesDiff) {
	if d.Empty() {
		return
	}

	for _, added := range d.Added {
		report.print("New response:", added)
	}

	for _, deleted := range d.Deleted {
		report.print("Deleted response:", deleted)
	}

	for response := range d.Modified {
		report.print("Modified response:", response)
	}
}
