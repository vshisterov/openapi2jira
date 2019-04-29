package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

type apiGroup struct {
	Name 	string
	Methods	[]apiMethod
}

type apiMethod struct {
	Summary				string
	Method				string
	Description			string
	QueryParameters 	[]apiParameter
	RequestSchema		apiParameterSchema
	ResponseSchema		apiParameterSchema
	Examples			[]apiExample
	CustomTags			map[string]string
}

type apiParameter struct {
	Name 		string
	Type 		string
	Description	string
	Schema		apiParameterSchema
}

type apiParameterSchema struct {
	Name		string
	Attributes	[]apiParameter
}

type apiExample struct {
	Title		string
	Request		string
	Response	string
}

type MapSlice yaml.MapSlice


func main() {

	generate()

}

func generate() error {

	specFile := "petstore.yml"

	spec, err := loadSpec(specFile)

	if err != nil {
		fmt.Println("Error reading spec", specFile, ":", err)
		return err
	}

	apiGroups := parseSpec(spec)

	description := toJira(apiGroups)

	fmt.Println(description)

	return nil

}

func toJira(groups map[string]apiGroup) string {

	builder := strings.Builder {}

	for _, group := range groups{
		fmt.Fprintf(&builder,"h3. %s", group.Name)
		fmt.Fprintln(&builder)

		for _, method := range group.Methods {
			fmt.Fprintf(&builder, "h4. %s\n", method.Summary)


			if len(method.Description)>0 {fmt.Fprintln(&builder, method.Description) }

			fmt.Fprintf(&builder, "*Method*: {noformat}%s{noformat}\n", method.Method)

			for customTag, customTagValue := range method.CustomTags {
				fmt.Fprintf(&builder,"*%s*: %s\n", customTag, customTagValue)
			}

			printParameters(&builder, "Query Parameters", method.QueryParameters)
			printParameters(&builder, "Request Parameters", method.RequestSchema.Attributes)
			printParameters(&builder, "Response Attributes", method.ResponseSchema.Attributes)

			fmt.Fprintln(&builder)

		}

		fmt.Fprintln(&builder)

	}

	return builder.String()

}

func printParameters(builder *strings.Builder, header string, parameters []apiParameter ) {
	if len(parameters) > 0 {

		fmt.Fprintf(builder, "*%s*:\n", header)
		fmt.Fprintln(builder, "|| Name || Type || Description ||")

		for _, parameter := range parameters {
			printParameter(builder, parameter, "")
		}
	}
}

func printParameter(builder *strings.Builder, parameter apiParameter, parameterPrefix string) {

	fmt.Fprintf(builder, "| {{%s}} | %s | %s |\n", parameterPrefix + parameter.Name, parameter.Type, parameter.Description)

	parameterPrefix += parameter.Name + "."

	for _, nestedParameter := range parameter.Schema.Attributes {
		printParameter(builder, nestedParameter, parameterPrefix)
	}
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

func parseSpec(spec MapSlice) map[string]apiGroup {

	result := map[string]apiGroup{}

	definitions := map[string]apiParameterSchema{}

	for i := range spec {

		tag := spec[i].Key.(string)

		switch tag {

		case "paths":

			result = parsePaths(spec[i].Value.(MapSlice))

		case "definitions":

			definitionsNode := spec[i].Value.(MapSlice)

			for _, definitionNode := range definitionsNode {

				definition := apiParameterSchema{}
				definition.Name = definitionNode.Key.(string)

				var attributes []apiParameter

				for _, definitionTagsNode := range definitionNode.Value.(MapSlice){

					if definitionTagsNode.Key.(string) == "properties" {

						for _, propertyNode := range definitionTagsNode.Value.(MapSlice) {

								attribute := apiParameter{}

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
									}


								}
								attributes = append(attributes, attribute)
						}

					}

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

	for _, group := range result{


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

func parsePaths(pathNodes MapSlice) map[string]apiGroup {

	result :=map[string]apiGroup{}

	for _, pathNode := range pathNodes {

		parsePath(pathNode, result)

	}

	return result

}

func parsePath(pathNode yaml.MapItem, groups map[string]apiGroup) {
	path := pathNode.Key.(string)
	fmt.Println("Parsing path", path)
	for _, methodNode := range pathNode.Value.(MapSlice) {

		parseMethod(methodNode, groups, path)

	}
}

func parseMethod(methodNode yaml.MapItem, groups map[string]apiGroup, path string) {

	httpMethod := strings.ToUpper(methodNode.Key.(string))

	group, ok := groups["Unknown"]
	if !ok {
		group = apiGroup{"Unknown", []apiMethod{}}
	}

	method := apiMethod{}
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
			for _, responseNode := range  methodPropertyNode.Value.(MapSlice) {
				if responseNode.Key.(string) == "200" || responseNode.Key.(string) == "201" || responseNode.Key.(string) == "default" {
					for _, responsePropertyNode := range responseNode.Value.(MapSlice) {
						if responsePropertyNode.Key.(string) == "schema" {
							for _, schemaPropertyNode := range responsePropertyNode.Value.(MapSlice) {
								if schemaPropertyNode.Key.(string) == "$ref" {
									method.ResponseSchema.Name = strings.TrimPrefix(schemaPropertyNode.Value.(string), "#/definitions/")
								}
								if schemaPropertyNode.Key.(string) == "items" {
									for _, itemsPropertyNode := range schemaPropertyNode.Value.(MapSlice){
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

func addCustomTag(method apiMethod, name string,  value string) {
	customTag := strings.Title(strings.ReplaceAll(strings.TrimPrefix(name, "x-"), "-", " "))
	method.CustomTags[customTag] = value
}

func parseParameter(parameterNode interface{}, method *apiMethod) {

	parameter := apiParameter{}

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
					for _, itemsPropertyNode := range schemaPropertyNode.Value.(MapSlice){
						if itemsPropertyNode.Key.(string) == "$ref" {
							parameter.Schema.Name = strings.TrimPrefix(itemsPropertyNode.Value.(string), "#/definitions/")
						}
					}
				}
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

	if isFormData{

		method.RequestSchema.Attributes = append(method.RequestSchema.Attributes, parameter)
	}

}

func getGroupByName(groups map[string]apiGroup, group apiGroup, name string) apiGroup {
	existingGroup, ok := groups[name]
	if ok {
		group = existingGroup
	} else {
		group.Name = name
	}
	return group
}
