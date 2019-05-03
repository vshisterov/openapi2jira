package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/hoisie/web"
	"github.com/vshisterov/openapi2jira/jira"
	"github.com/vshisterov/openapi2jira/openapi"
	"io/ioutil"
)

func main() {

	var in, out string

	flag.StringVar(&in, "in", "test.yml", "source file name")
	flag.StringVar(&out, "out", "test.txt", "target file name")

	flag.Parse()

	if len(flag.Args()) > 0 && flag.Arg(0) == "serve" {

		web.Post("/convert", convert)
		web.Run("0.0.0.0:9999")

	} else {
		fmt.Println("Converting file:", in)
		Convert(in, out)
		fmt.Println("Completed:", out)
	}

}

func convert(ctx *web.Context) string {

	buf := new(bytes.Buffer)
	buf.ReadFrom(ctx.Request.Body)
	source := buf.String()

	g, _ := openapi.Parse(source)
	s := jira.ToJira(g)

	return s
}

func Convert(in string, out string) {
	
	g, err := openapi.ParseFile(in)
	if err != nil {
		fmt.Println("Error reading spec", in, ":", err)
		return
	}

	s := jira.ToJira(g)

	err = ioutil.WriteFile(out, []byte(s), 0644)
	if err != nil {
		fmt.Println("Error writing results", in, ":", err)
	}
}