package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

type APIGroup struct {
	Name    string
	Methods []APIMethod
}

type APIMethod struct {
	Summary         string
	Method          string
	Description     string
	QueryParameters []APIParam
	RequestSchema   APIParamaSchema
	ResponseSchema  APIParamaSchema
	Examples        []APIExample
	CustomTags      map[string]string
}

type APIParam struct {
	Name        string
	Type        string
	Description string
	Mandatory   bool
	Schema      APIParamaSchema
	Enum        []string
}

type APIParamaSchema struct {
	Name               string
	Attributes         []APIParam
	HasMandatoryParams bool
}

type APIExample struct {
	Title    string
	Request  string
	Response string
}

type MapSlice yaml.MapSlice

const HeaderTop = 3
const HeaderRegular = 4
const HeaderLite = -1

func main() {

	generate()

}

func generate() error {

	in := "test.yml"
	out := "text.txt"

	spec, err := loadSpec(in)
	if err != nil {
		fmt.Println("Error reading spec", in, ":", err)
		return err
	}

	g := parse(spec)
	s := toJira(g)

	fmt.Println("Writing results")

	err = ioutil.WriteFile(out, []byte(s), 0644)
	if err != nil {
		fmt.Println("Error writing results", in, ":", err)
		return err
	}

	fmt.Println(s)

	return nil

}

func toJira(groups map[string]APIGroup) string {

	b := strings.Builder{}

	for _, g := range groups {
		printAPIGroup(&b, g)
	}

	return b.String()

}

func printAPIGroup(w io.Writer, g APIGroup) {
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

func printAPIMethods(w io.Writer, methods []APIMethod) {
	for _, m := range methods {
		printAPIMethod(w, m)
	}
}

func printAPIMethod(w io.Writer, m APIMethod) {
	printHeader(w, m.Summary, HeaderRegular)
	printNotEmpty(w, m.Description)
	printMethod(w, m.Method)
	printExtensions(w, m.CustomTags)
	printParams(w, "Query Parameters", m.QueryParameters, false)
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

func printParams(w  io.Writer, header string, params []APIParam, mandatory bool) {
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

func printParam(w io.Writer, p APIParam, prefix string, mandatory bool) {

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

func getParamType(p APIParam) string {
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

func loadSpec(f string) (MapSlice, error) {
	m, err := readYaml(f)
	return m, err
}

func readYaml(fileName string) (MapSlice, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	m := MapSlice(nil)
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return m, nil
}

func parse(spec MapSlice) map[string]APIGroup {

	result := map[string]APIGroup{}

	definitions := map[string]APIParamaSchema{}

	for i := range spec {

		tag := spec[i].Key.(string)

		switch tag {

		case "paths":

			result = parsePaths(spec[i].Value.(MapSlice))

		case "definitions":

			definitionsNode := spec[i].Value.(MapSlice)

			for _, definitionNode := range definitionsNode {

				definition := APIParamaSchema{}
				definition.Name = definitionNode.Key.(string)

				var attributes []APIParam
				var requiredParameters []string

				for _, definitionTagsNode := range definitionNode.Value.(MapSlice) {

					switch definitionTagsNode.Key.(string) {
					case "properties":

						for _, propertyNode := range definitionTagsNode.Value.(MapSlice) {

							attribute := APIParam{}

							attribute.Name = propertyNode.Key.(string)

							for _, propertyAttributeNode := range propertyNode.Value.(MapSlice) {
								switch propertyAttributeNode.Key.(string) {
								case "type":
									attribute.Type = propertyAttributeNode.Value.(string)
								case "description":
									attribute.Description = propertyAttributeNode.Value.(string)
								case "$ref":
									attribute.Schema.Name = strings.TrimPrefix(propertyAttributeNode.Value.(string),
										"#/definitions/")
								case "items":
									for _, itemsNode := range propertyAttributeNode.Value.(MapSlice) {

										switch itemsNode.Key.(string) {
										case "$ref":
											attribute.Schema.Name = strings.TrimPrefix(itemsNode.Value.(string), "#/definitions/")
										case "type":
											attribute.Type = "array of " + itemsNode.Value.(string)
										case "description":
											attribute.Description = itemsNode.Value.(string)
										}
									}
								case "enum":
									for _, enumValueNode := range propertyAttributeNode.Value.([]interface{}) {
										attribute.Enum = append(attribute.Enum, enumValueNode.(string))
									}
								}

							}
							attributes = append(attributes, attribute)
						}
					case "required":
						for _, requiredNode := range definitionTagsNode.Value.([]interface{}) {
							requiredParameters = append(requiredParameters, requiredNode.(string))
						}
					}

				}

				for requiredParameter := range requiredParameters {
					attributes[requiredParameter].Mandatory = true
					definition.HasMandatoryParams = true
				}

				definition.Attributes = attributes

				definitions[definition.Name] = definition

			}

		}

	}

	for _, definition := range definitions {

		for i, attribute := range definition.Attributes {

			if len(attribute.Schema.Name) > 0 {

				definitions[definition.Name].Attributes[i].Schema = definitions[attribute.Schema.Name]
				if attribute.Type != "array" {
					definitions[definition.Name].Attributes[i].Type = "struct"
				}

			}
		}

	}

	for _, group := range result {

		for i, method := range group.Methods {

			if definition, ok := definitions[method.RequestSchema.Name]; ok {
				result[group.Name].Methods[i].RequestSchema = definition
			}

			if definition, ok := definitions[method.ResponseSchema.Name]; ok {
				result[group.Name].Methods[i].ResponseSchema = definition
			}
		}

	}

	return result
}

func parsePaths(pathNodes MapSlice) map[string]APIGroup {

	result := map[string]APIGroup{}

	for _, pathNode := range pathNodes {

		parsePath(pathNode, result)

	}

	return result

}

func parsePath(pathNode yaml.MapItem, groups map[string]APIGroup) {
	path := pathNode.Key.(string)
	fmt.Println("Parsing path", path)
	for _, methodNode := range pathNode.Value.(MapSlice) {

		parseMethod(methodNode, groups, path)

	}
}

func parseMethod(methodNode yaml.MapItem, groups map[string]APIGroup, path string) {

	httpMethod := strings.ToUpper(methodNode.Key.(string))

	group, ok := groups["Unknown"]
	if !ok {
		group = APIGroup{"Unknown", []APIMethod{}}
	}

	method := APIMethod{}
	method.CustomTags = map[string]string{}
	method.Method = fmt.Sprintf("%s %s", httpMethod, path)

	for _, methodPropertyNode := range methodNode.Value.(MapSlice) {

		methodPropertyName := methodPropertyNode.Key.(string)

		switch methodPropertyNode.Key.(string) {

		case "tags":
			tag := methodPropertyNode.Value.([]interface{})[0].(string)
			group = getGroupByName(groups, group, tag)

		case "summary":
			method.Summary = methodPropertyNode.Value.(string)

		case "description":
			method.Description = methodPropertyNode.Value.(string)

		case "parameters":
			for _, parameterNode := range methodPropertyNode.Value.([]interface{}) {
				parseParameter(parameterNode, &method)
			}

		case "responses":
			for _, responseNode := range methodPropertyNode.Value.(MapSlice) {
				if responseNode.Key.(string) == "200" || responseNode.Key.(string) == "201" || responseNode.Key.(string) == "default" {
					for _, responsePropertyNode := range responseNode.Value.(MapSlice) {
						if responsePropertyNode.Key.(string) == "schema" {
							for _, schemaPropertyNode := range responsePropertyNode.Value.(MapSlice) {
								if schemaPropertyNode.Key.(string) == "$ref" {
									method.ResponseSchema.Name = strings.TrimPrefix(schemaPropertyNode.Value.(string), "#/definitions/")
								}
								if schemaPropertyNode.Key.(string) == "items" {
									for _, itemsPropertyNode := range schemaPropertyNode.Value.(MapSlice) {
										if itemsPropertyNode.Key.(string) == "$ref" {
											method.ResponseSchema.Name = strings.TrimPrefix(itemsPropertyNode.Value.(string), "#/definitions/")
										}
									}
								}
							}
						}
					}
				}
			}

		}

		if strings.HasPrefix(methodPropertyName, "x-") {
			addCustomTag(method, methodPropertyName, methodPropertyNode.Value.(string))
		}

	}
	group.Methods = append(group.Methods, method)
	groups[group.Name] = group
}

func addCustomTag(method APIMethod, name string, value string) {
	customTag := strings.Title(strings.ReplaceAll(strings.TrimPrefix(name, "x-"), "-", " "))
	method.CustomTags[customTag] = value
}

func parseParameter(parameterNode interface{}, method *APIMethod) {

	parameter := APIParam{}

	isQuery := false
	isBody := false
	isFormData := false

	for _, parameterPropertyNode := range parameterNode.(MapSlice) {

		switch parameterPropertyNode.Key.(string) {

		case "name":
			parameter.Name = parameterPropertyNode.Value.(string)

		case "type":
			parameter.Type = parameterPropertyNode.Value.(string)

		case "description":
			parameter.Description = parameterPropertyNode.Value.(string)

		case "schema":
			for _, schemaPropertyNode := range parameterPropertyNode.Value.(MapSlice) {
				if schemaPropertyNode.Key.(string) == "$ref" {
					parameter.Schema.Name = strings.TrimPrefix(schemaPropertyNode.Value.(string), "#/definitions/")
				}
				if schemaPropertyNode.Key.(string) == "items" {
					for _, itemsPropertyNode := range schemaPropertyNode.Value.(MapSlice) {
						if itemsPropertyNode.Key.(string) == "$ref" {
							parameter.Schema.Name = strings.TrimPrefix(itemsPropertyNode.Value.(string), "#/definitions/")
						}
					}
				}
			}

		case "enum":
			for _, enumValueNode := range parameterPropertyNode.Value.([]interface{}) {
				parameter.Enum = append(parameter.Enum, enumValueNode.(string))
			}

		case "in":
			switch parameterPropertyNode.Value.(string) {
			case "query":
				isQuery = true
			case "body":
				isBody = true
			case "formData":
				isFormData = true
			}
		}

	}

	if isQuery {
		method.QueryParameters = append(method.QueryParameters, parameter)
	}

	if isBody {
		method.RequestSchema.Name = parameter.Schema.Name
	}

	if isFormData {

		method.RequestSchema.Attributes = append(method.RequestSchema.Attributes, parameter)
	}

}

func getGroupByName(groups map[string]APIGroup, group APIGroup, name string) APIGroup {
	existingGroup, ok := groups[name]
	if ok {
		group = existingGroup
	} else {
		group.Name = name
	}
	return group
}
