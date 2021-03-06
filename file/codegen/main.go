package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"reflect"
	"text/template"

	"github.com/alecthomas/jsonschema"
	"github.com/kong/deck/file"
	"github.com/kong/go-kong/kong"
)

const templateContent = `// Code generated by go generate; DO NOT EDIT.
package file

const contentSchema = ` + "`{{.Schema}}`\n"

type templateData struct {
	Schema string
}

var (
	// routes and services
	anyOfNameOrID = []*jsonschema.Type{
		{
			Required: []string{"name"},
		},
		{
			Required: []string{"id"},
		},
	}

	anyOfUsernameOrID = []*jsonschema.Type{
		{
			Required: []string{"username"},
		},
		{
			Required: []string{"id"},
		},
	}
)

func main() {
	var reflector jsonschema.Reflector
	reflector.ExpandedStruct = true
	reflector.TypeMapper = func(typ reflect.Type) *jsonschema.Type {
		// plugin configuration
		if typ == reflect.TypeOf(kong.Configuration{}) {
			return &jsonschema.Type{
				Type:                 "object",
				Properties:           map[string]*jsonschema.Type{},
				AdditionalProperties: []byte("true"),
			}
		}
		return nil
	}
	schema := reflector.Reflect(file.Content{})
	schema.Definitions["Service"].AnyOf = anyOfNameOrID
	schema.Definitions["FService"].AnyOf = anyOfNameOrID

	schema.Definitions["Route"].AnyOf = anyOfNameOrID
	schema.Definitions["FRoute"].AnyOf = anyOfNameOrID

	schema.Definitions["Consumer"].AnyOf = anyOfUsernameOrID
	schema.Definitions["FConsumer"].AnyOf = anyOfUsernameOrID

	schema.Definitions["Upstream"].Required = []string{"name"}
	schema.Definitions["FUpstream"].Required = []string{"name"}

	schema.Definitions["FTarget"].Required = []string{"target"}
	schema.Definitions["FCACertificate"].Required = []string{"cert"}
	schema.Definitions["FPlugin"].Required = []string{"name"}

	schema.Definitions["FCertificate"].Required = []string{"id", "cert", "key"}
	schema.Definitions["FCertificate"].Properties["snis"] = &jsonschema.Type{
		Type: "array",
		Items: &jsonschema.Type{
			Type: "object",
			Properties: map[string]*jsonschema.Type{
				"name": {
					Type: "string",
				},
			},
		},
	}

	// creds
	schema.Definitions["ACLGroup"].Required = []string{"group"}
	schema.Definitions["BasicAuth"].Required = []string{"username", "password"}
	schema.Definitions["HMACAuth"].Required = []string{"username", "secret"}
	schema.Definitions["JWTAuth"].Required = []string{"algorithm", "key",
		"secret"}
	schema.Definitions["KeyAuth"].Required = []string{"key"}
	schema.Definitions["Oauth2Credential"].Required = []string{"name",
		"client_id", "redirect_uris", "client_secret"}
	schema.Definitions["MTLSAuth"].Required = []string{"id", "subject_name"}

	// RBAC resources
	schema.Definitions["FRBACRole"].Required = []string{"name"}
	schema.Definitions["FRBACEndpointPermission"].Required = []string{"workspace", "endpoint"}

	// Foreign references
	stringType := &jsonschema.Type{Type: "string"}
	schema.Definitions["FPlugin"].Properties["consumer"] = stringType
	schema.Definitions["FPlugin"].Properties["service"] = stringType
	schema.Definitions["FPlugin"].Properties["route"] = stringType

	schema.Definitions["FService"].Properties["client_certificate"] = stringType

	// konnect resources
	schema.Definitions["FServicePackage"].Required = []string{"name"}
	schema.Definitions["FServiceVersion"].Required = []string{"version"}
	schema.Definitions["Implementation"].Required = []string{"type", "kong"}

	jsonSchema, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		log.Fatalln(err)
	}

	tmpl := template.New("codegen")
	tmpl, err = tmpl.Parse(templateContent)
	if err != nil {
		log.Fatalln(err)
	}

	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, templateData{string(jsonSchema)})
	if err != nil {
		log.Fatalln(err)
	}
	err = ioutil.WriteFile("schema.go", buffer.Bytes(), 0644)
	if err != nil {
		log.Fatalln(err)
	}
}
