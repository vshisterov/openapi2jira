package openapi

import (
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

type Group struct {
	Name       string
	Operations []Operation
}

type Operation struct {
	Summary        string
	Method         string
	Description    string
	QueryParams    []Param
	RequestSchema  Schema
	ResponseSchema Schema
	Examples       []Example
	CustomTags     map[string]string
}

type Param struct {
	Name        string
	Type        string
	Description string
	Mandatory   bool
	Schema      Schema
	Enum        []string
}

type Schema struct {
	Name               string
	Attributes         []Param
	HasMandatoryParams bool
}

type Example struct {
	Title    string
	Request  string
	Response string
}

const DefaultGroup = "API Specifics"

func Parse(s string) (map[string]Group, error) {

	return ParseBytes([]byte(s))

}

func ParseFile(name string) (map[string]Group, error) {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	g, err := ParseBytes(data)
	if err != nil {
		return nil, err
	}

	return g, nil
}

func ParseBytes(data []byte) (map[string]Group, error) {
	m := yaml.MapSlice(nil)
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	g := parseSpec(m)

	return g, nil
}

func parseSpec(s yaml.MapSlice) map[string]Group {

	g := map[string]Group{}
	d := map[string]Schema{}

	for _, item := range s {
		switch item.Key.(string) {
		case "paths":
			g = parsePaths(item.Value.(yaml.MapSlice))
		case "definitions":

			d = parseDefinitions(item.Value.(yaml.MapSlice))
		}
	}

	linkDefinitions(d)
	linkSchemas(g, d)

	return g
}

func parsePaths(s yaml.MapSlice) map[string]Group {

	g := map[string]Group{}

	for i := range s {
		path := s[i].Key.(string)

		fmt.Println("Parsing path", path)

		for _, o := range s[i].Value.(yaml.MapSlice) {

			parseOperation(o, g, path)

		}
	}

	return g

}

func parseOperation(s yaml.MapItem, groups map[string]Group, path string) {

	o := Operation{}

	group, ok := groups[DefaultGroup]
	if !ok {
		group = Group{DefaultGroup, []Operation{}}
	}

	m := strings.ToUpper(s.Key.(string))
	o.Method = fmt.Sprintf("%s %s", m, path)

	o.CustomTags = map[string]string{}

	for _, item := range s.Value.(yaml.MapSlice) {

		key := item.Key.(string)

		switch key {

		case "tags":
			tag := item.Value.([]interface{})[0].(string)
			group = findOrName(groups, group, tag)

		case "summary":
			o.Summary = item.Value.(string)

		case "description":
			o.Description = item.Value.(string)

		case "parameters":
			for _, p := range item.Value.([]interface{}) {
				parseParameter(p, &o)
			}

		case "responses":
			for _, r := range item.Value.(yaml.MapSlice) {
				parseResponse(r, &o)
			}

		}

		if strings.HasPrefix(key, "x-") {
			addCustomTag(o, key, item.Value.(string))
		}

	}
	group.Operations = append(group.Operations, o)
	groups[group.Name] = group
}

func parseResponse(responseNode yaml.MapItem, o *Operation) {
	if responseNode.Key.(string) == "200" || responseNode.Key.(string) == "201" || responseNode.Key.(string) == "default" {
		for _, responsePropertyNode := range responseNode.Value.(yaml.MapSlice) {
			if responsePropertyNode.Key.(string) == "schema" {
				for _, schemaPropertyNode := range responsePropertyNode.Value.(yaml.MapSlice) {
					if schemaPropertyNode.Key.(string) == "$ref" {
						o.ResponseSchema.Name = strings.TrimPrefix(schemaPropertyNode.Value.(string), "#/definitions/")
					}
					if schemaPropertyNode.Key.(string) == "items" {
						for _, itemsPropertyNode := range schemaPropertyNode.Value.(yaml.MapSlice) {
							if itemsPropertyNode.Key.(string) == "$ref" {
								o.ResponseSchema.Name = strings.TrimPrefix(itemsPropertyNode.Value.(string), "#/definitions/")
							}
						}
					}
				}
			}
		}
	}
}

func addCustomTag(method Operation, name string, value string) {
	customTag := strings.Title(strings.ReplaceAll(strings.TrimPrefix(name, "x-"), "-", " "))
	method.CustomTags[customTag] = value
}

func parseParameter(parameterNode interface{}, method *Operation) {

	parameter := Param{}

	isQuery := false
	isBody := false
	isFormData := false

	for _, parameterPropertyNode := range parameterNode.(yaml.MapSlice) {

		switch parameterPropertyNode.Key.(string) {

		case "name":
			parameter.Name = parameterPropertyNode.Value.(string)

		case "type":
			parameter.Type = parameterPropertyNode.Value.(string)

		case "description":
			parameter.Description = parameterPropertyNode.Value.(string)

		case "schema":
			for _, schemaPropertyNode := range parameterPropertyNode.Value.(yaml.MapSlice) {
				if schemaPropertyNode.Key.(string) == "$ref" {
					parameter.Schema.Name = strings.TrimPrefix(schemaPropertyNode.Value.(string), "#/definitions/")
				}
				if schemaPropertyNode.Key.(string) == "items" {
					for _, itemsPropertyNode := range schemaPropertyNode.Value.(yaml.MapSlice) {
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
		method.QueryParams = append(method.QueryParams, parameter)
	}

	if isBody {
		method.RequestSchema.Name = parameter.Schema.Name
	}

	if isFormData {

		method.RequestSchema.Attributes = append(method.RequestSchema.Attributes, parameter)
	}

}

func findOrName(groups map[string]Group, group Group, name string) Group {
	existingGroup, ok := groups[name]
	if ok {
		group = existingGroup
	} else {
		group.Name = name
	}
	return group
}

func parseDefinitions(source yaml.MapSlice) map[string]Schema {

	r := map[string]Schema{}

	for _, item := range source {

		definition := Schema{}
		definition.Name = item.Key.(string)

		var attributes []Param
		var requiredParameters []string

		for _, definitionTagsNode := range item.Value.(yaml.MapSlice) {

			switch definitionTagsNode.Key.(string) {
			case "properties":

				for _, propertyNode := range definitionTagsNode.Value.(yaml.MapSlice) {

					attribute := Param{}

					attribute.Name = propertyNode.Key.(string)

					for _, propertyAttributeNode := range propertyNode.Value.(yaml.MapSlice) {
						switch propertyAttributeNode.Key.(string) {
						case "type":

							attribute.Type = propertyAttributeNode.Value.(string)
						case "description":
							attribute.Description = propertyAttributeNode.Value.(string)
						case "$ref":
							attribute.Schema.Name = strings.TrimPrefix(propertyAttributeNode.Value.(string),
								"#/definitions/")
						case "items":
							for _, itemsNode := range propertyAttributeNode.Value.(yaml.MapSlice) {

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

		r[definition.Name] = definition

	}

	return r
}

func linkDefinitions(d map[string]Schema) {
	for k := range d {

		for i, a := range d[k].Attributes {

			if len(a.Schema.Name) > 0 {

				d[k].Attributes[i].Schema = d[a.Schema.Name]
				if a.Type != "array" {
					d[k].Attributes[i].Type = "struct"
				}

			}
		}

	}
}

func linkSchemas(g map[string]Group, s map[string]Schema) {
	for k := range g {

		for i, o := range g[k].Operations {

			if schema, ok := s[o.RequestSchema.Name]; ok {
				g[k].Operations[i].RequestSchema = schema
			}

			if schema, ok := s[o.ResponseSchema.Name]; ok {
				g[k].Operations[i].ResponseSchema = schema
			}
		}

	}
}
