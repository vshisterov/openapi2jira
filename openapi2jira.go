package main

import (
	"flag"
	"fmt"
	"github.com/vshisterov/openapi2jira/jira"
	"github.com/vshisterov/openapi2jira/openapi"
	"io/ioutil"
)

func main() {

	var in, out string

	flag.StringVar(&in, "in", "test.yml", "source file name")
	flag.StringVar(&out, "out", "test.txt", "target file name")

	flag.Parse()

	fmt.Println("Conveting file:", in)

	Convert(in, out)

	fmt.Println("Completed:", out)
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