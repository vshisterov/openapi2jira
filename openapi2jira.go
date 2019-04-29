package main

import (
	"fmt"
	"github.com/vshisterov/openapi2jira/jira"
	"github.com/vshisterov/openapi2jira/openapi"
	"io/ioutil"
)

func main() {

	in := "test.yml"
	out := "test.txt"

	g, err := openapi.Parse(in)
	if err != nil {
		fmt.Println("Error reading spec", in, ":", err)
	}

	s := jira.ToJira(g)

	fmt.Println("Writing results")

	err = ioutil.WriteFile(out, []byte(s), 0644)
	if err != nil {
		fmt.Println("Error writing results", in, ":", err)
	}

	fmt.Println(s)

}