# OpenAPI to Jira

The tool converts API spec in OpenAPI fka Swagger 2.0 format to text in [JIRA-compatible](https://jira.atlassian.com/secure/WikiRendererHelpAction.jspa?section=all) format, 
which can be then used to create User Stories or Tasks or whatever ticket types you use in JIRA for API development.

## Usage

### Command Line
```
go run openapi2jira.go -in=myapi.yml -out=userstory.txt
```

### In Your Code
```go
import (
	"github.com/vshisterov/openapi2jira/jira"
	"github.com/vshisterov/openapi2jira/openapi"
)

	api, err := openapi.ParseFile("myapi.yml")
  
  // or if you have the spec already loaded as string:
  api, err := openapi.Parse(myapi)

	text := jira.ToJira(api)
  
```

## Acknowledgments

* Thanks [Gustavo Niemeyer](https://github.com/niemeyer) for the [yaml package](https://github.com/go-yaml/yaml)
