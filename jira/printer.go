package jira

import (
	"fmt"
	"github.com/vshisterov/openapi2jira/openapi"
	"io"
	"strings"
)

const HeaderTop = 3
const HeaderRegular = 4
const HeaderLite = -1

func ToJira(groups map[string]openapi.APIGroup) string {

	b := strings.Builder{}

	for _, g := range groups {
		printAPIGroup(&b, g)
	}

	return b.String()

}

func printAPIGroup(w io.Writer, g openapi.APIGroup) {
	printHeader(w, g.Name, HeaderTop)
	printAPIMethods(w, g.Methods)
	printNewLine(w)
}

func printHeader(w io.Writer, s string, level int)  {
	if level > HeaderLite {
		fmt.Fprintf(w, "h%d. %s\n", level, s)
	} else {
		fmt.Fprintf(w, "*%s*:\n", s)
	}
}

func printAPIMethods(w io.Writer, methods []openapi.APIMethod) {
	for _, m := range methods {
		printAPIMethod(w, m)
	}
}

func printAPIMethod(w io.Writer, m openapi.APIMethod) {
	printHeader(w, m.Summary, HeaderRegular)
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


func printMethod(w io.Writer, method string) {
	printPair(w, "Method", fmt.Sprintf("{noformat}%s{noformat}", method))
}

func printPair(w io.Writer, key string, value string) {
	fmt.Fprintf(w, "*%s*: %s\n", key, value)
}

func printExtensions(w io.Writer, tags map[string]string ) {
	for t, v := range tags {
		printPair(w, t, v)
	}
}

func printParams(w  io.Writer, header string, params []openapi.APIParam, mandatory bool) {
	if len(params) > 0 {

		printHeader(w, header, HeaderLite)

		printColumns(w, mandatory)

		for _, parameter := range params {
			printParam(w, parameter, "", mandatory)
		}
	}
}

func printColumns(w io.Writer, mandatory bool) {

	columns := []string { "Name", "Type"}
	if mandatory {
		columns = append(columns, "Mandatory")
	}
	columns = append(columns, "Description")

	fmt.Fprint(w, "||")

	for _, c := range columns {
		fmt.Fprint(w, " ", c, " ||")
	}

	fmt.Fprintln(w)
}

func printParam(w io.Writer, p openapi.APIParam, prefix string, mandatory bool) {

	printCellDelimiter(w)
	
	n := getMonospaced(prefix + p.Name)
	printCell(w, n)
	
	t := getParamType(p)
	printCell(w, t)

	if mandatory {
		m := getCheck(p.Mandatory)
		printCell(w, m)
	}

	printCell(w, p.Description)

	printNewLine(w)

	prefix += p.Name + "."

	for _, nestedParameter := range p.Schema.Attributes {
		printParam(w, nestedParameter, prefix, mandatory)
	}
}

func printCellDelimiter(w io.Writer) {
	fmt.Fprint(w, "|")
}

func printCell(w io.Writer, s string){
	fmt.Fprintf(w, " %s |", s)
}

func getMonospaced(s string) string {
	return fmt.Sprintf("{{%s}}", s)
}

func getParamType(p openapi.APIParam) string {
	s := p.Type
	if len(p.Enum) > 0 {
		enumDelimiter := ""
		s = ""
		for _, enumValue := range p.Enum {
			s += enumDelimiter + "{{" + enumValue + "}}"
			enumDelimiter = " \\| "
		}
	}
	return s
}

func getCheck(b bool) string {
	check := " "
	if b {
		check = "(/)"
	}
	return check
}

func printNewLine(w io.Writer) {
	fmt.Fprintln(w)
}