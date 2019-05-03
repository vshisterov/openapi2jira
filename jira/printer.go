package jira

import (
	"fmt"
	"github.com/vshisterov/openapi2jira/openapi"
	"io"
	"strings"
)

const HeaderBig = 3
const HeaderNormal = 4
const HeaderLite = 99

func ToJira(groups map[string]openapi.Group) string {

	b := strings.Builder{}

	for _, g := range groups {
		printAPIGroup(&b, g)
	}

	return b.String()

}

func printAPIGroup(w io.Writer, g openapi.Group) {
	printHeader(w, g.Name, HeaderBig)
	printAPIMethods(w, g.Operations)
	printNewLine(w)
}

func printHeader(w io.Writer, s string, level int) {
	if level < 7 {
		fmt.Fprintf(w, "h%d. %s\n", level, s)
	} else {
		fmt.Fprintf(w, "%s:\n", getBold(s))
	}
}

func printAPIMethods(w io.Writer, methods []openapi.Operation) {
	for _, m := range methods {
		printAPIMethod(w, m)
	}
}

func printNewLine(w io.Writer) {
	fmt.Fprintln(w)
}

func printAPIMethod(w io.Writer, m openapi.Operation) {
	printHeader(w, m.Summary, HeaderNormal)
	printNotEmpty(w, m.Description)
	printMethod(w, m.Method)
	printExtensions(w, m.CustomTags)
	printParams(w, "Query Parameters", m.QueryParams, false)
	printParams(w, "Request Parameters", m.RequestSchema.Attributes, m.RequestSchema.HasMandatoryParams)
	printParams(w, "Response Attributes", m.ResponseSchema.Attributes, false)
	printNewLine(w)
}

func printNotEmpty(w io.Writer, s string) {
	if len(s) > 0 {
		fmt.Fprintln(w, s)
	}
}

func printMethod(w io.Writer, m string) {
	printPair(w, "Method", getNonformatted(m))
}

func printExtensions(w io.Writer, tags map[string]string) {
	for t, v := range tags {
		printPair(w, t, v)
	}
}

func printParams(w io.Writer, header string, params []openapi.Param, mandatory bool) {
	if len(params) > 0 {

		printHeader(w, header, HeaderLite)

		printColumns(w, mandatory)

		for _, parameter := range params {
			printParam(w, parameter, "", mandatory)
		}
	}
}

func printPair(w io.Writer, key string, value string) {
	fmt.Fprintf(w, "%s: %s\n", getBold(key), value)
}

func printColumns(w io.Writer, mandatory bool) {

	columns := []string{"Name", "Type"}
	if mandatory {
		columns = append(columns, "Mandatory")
	}
	columns = append(columns, "Description")

	fmt.Fprint(w, getHeaderCellDelimiter())

	for _, c := range columns {
		fmt.Fprint(w, c, getHeaderCellDelimiter())
	}

	fmt.Fprintln(w)
}

func printParam(w io.Writer, p openapi.Param, prefix string, mandatory bool) {

	fmt.Fprint(w, getCellDelimiter())
	fmt.Fprint(w, getMonospaced(prefix+p.Name))
	fmt.Fprint(w, getCellDelimiter())

	fmt.Fprint(w, getParamType(p))
	fmt.Fprint(w, getCellDelimiter())

	if mandatory {
		fmt.Fprint(w, getCheck(p.Mandatory))
		fmt.Fprint(w, getCellDelimiter())
	}

	fmt.Fprint(w, p.Description)
	fmt.Fprintln(w, getCellDelimiter())

	prefix += p.Name + "."

	for _, nestedParam := range p.Schema.Attributes {
		printParam(w, nestedParam, prefix, mandatory)
	}
}

func getParamType(p openapi.Param) string {
	s := p.Type
	if len(p.Enum) > 0 {
		delimiter := ""
		s = ""
		for _, v := range p.Enum {
			s += delimiter + getMonospaced(v)
			delimiter = getPipe()
		}
	}
	return s
}

func getMonospaced(s string) string {
	return fmt.Sprintf("{{%s}}", s)
}

func getNonformatted(s string) string {
	return fmt.Sprintf("{noformat}%s{noformat}", s)
}

func getBold(s string) string {
	return fmt.Sprintf("*%s*", s)
}

func getHeaderCellDelimiter() string {
	return "||"
}

func getCellDelimiter() string {
	return "|"
}

func getCheck(b bool) string {
	check := " "
	if b {
		check = "(/)"
	}
	return check
}

func getPipe() string {
	return " \\| "
}
