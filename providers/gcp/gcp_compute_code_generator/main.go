// SPDX-License-Identifier: Apache-2.0

//nolint:gosec,staticcheck // lint triage: legacy provider/API/security baseline is tracked in #175.
package main

import (
	"bytes"
	"encoding/json"
	"go/format"
	"log"
	"os"
	"strings"
	"text/template"
)

const pathForGenerateFiles = "/providers/gcp/"
const serviceTemplate = `
// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"context"
	"fmt"
	{{ if .byZone  }}"strings"{{end}}

	"github.com/chenrui333/terraformer/terraformutils"

	"google.golang.org/api/compute/v1"
)

var {{.resource}}AllowEmptyValues = []string{"{{join .allowEmptyValues "\",\"" }}"}

var {{.resource}}AdditionalFields = map[string]interface{}{
	{{ range $key,$value := .additionalFields}}
	"{{$key}}":			"{{$value}}",{{end}}
}

type {{.titleResourceName}}Generator struct {
	GCPService
}

// Run on {{.resource}}List and create for each TerraformResource
func (g {{.titleResourceName}}Generator) createResources(ctx context.Context, {{.resource}}List *compute.{{.titleResourceName}}ListCall{{ if .byZone  }}, zone string{{end}}) ([]terraformutils.Resource, error) {
	resources := []terraformutils.Resource{}
	if err := {{.resource}}List.Pages(ctx, func(page *compute.{{.responseName}}) error {
		for _, obj := range page.Items {
			resources = append(resources, terraformutils.NewResource(
				{{ if .idWithZone  }}zone+"/"+obj.Name,{{else}}obj.Name,{{end}}
				{{ if .idWithZone  }}zone+"/"+obj.Name,{{else}}obj.Name,{{end}}
				"{{.terraformName}}",
				g.ProviderName,
				map[string]string{
					"name":    obj.Name,
					"project": g.GetArgs()["project"].(string),
					{{ if .needRegion}}"region":  g.GetArgs()["region"].(compute.Region).Name,{{end}}
					{{ if .byZone  }}"zone":    zone,{{end}}
					{{ range $key, $value := .additionalFieldsForRefresh}}
					"{{$key}}":			"{{$value}}",{{end}}
				},
				{{.resource}}AllowEmptyValues,
				{{.resource}}AdditionalFields,
			))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("list {{.resource}}: %w", err)
	}
	return resources, nil
}

// Generate TerraformResources from GCP API,
// from each {{.resource}} create 1 TerraformResource
// Need {{.resource}} name as ID for terraform resource
func (g *{{.titleResourceName}}Generator) InitResources() error {
	ctx := context.Background()
	computeService, err := compute.NewService(ctx)
	if err != nil {
		return err
	}
	{{ if .byZone  }}
	for _, zoneLink := range g.GetArgs()["region"].(compute.Region).Zones {
		t := strings.Split(zoneLink, "/")
		zone := t[len(t)-1]
		{{.resource}}List := computeService.{{.titleResourceName}}.List(g.GetArgs()["project"].(string), zone)
		resources, err := g.createResources(ctx, {{.resource}}List, zone)
		if err != nil {
			return err
		}
		g.Resources = append(g.Resources, resources...)
	}
	{{else}}
		{{.resource}}List := computeService.{{.titleResourceName}}.List({{.parameterOrder}})
		resources, err := g.createResources(ctx, {{.resource}}List)
		if err != nil {
			return err
		}
		g.Resources = resources
	{{end}}

	return nil

}

`
const computeTemplate = `
// SPDX-License-Identifier: Apache-2.0

// AUTO-GENERATED CODE. DO NOT EDIT.
package gcp

import (
	"github.com/chenrui333/terraformer/terraformutils"
)

// Map of supported GCP compute service with code generate
var ComputeServices = map[string]terraformutils.ServiceGenerator{
{{ range $key, $value := .services }}
	"{{$key}}":                   &GCPFacade{service: &{{title $key}}Generator{}},{{ end }}

}

`

func main() {
	computeAPIData, err := os.ReadFile(os.Getenv("GOPATH") + "/src/google.golang.org/api/compute/v1/compute-api.json") // TODO delete this hack
	if err != nil {
		log.Fatal(err)
	}
	computeAPI := map[string]interface{}{}
	err = json.Unmarshal(computeAPIData, &computeAPI)
	if err != nil {
		log.Fatal(err)
	}
	funcMap := template.FuncMap{
		"title":   strings.Title,
		"toLower": strings.ToLower,
		"join":    strings.Join,
	}
	for resource, v := range computeAPI["resources"].(map[string]interface{}) {
		if _, exist := terraformResources[resource]; !exist {
			continue
		}
		if value, exist := v.(map[string]interface{})["methods"].(map[string]interface{})["list"]; exist {
			parameters := []string{}
			for _, param := range value.(map[string]interface{})["parameterOrder"].([]interface{}) {
				switch param.(string) {
				case "region":
					parameters = append(parameters, `g.GetArgs()["region"].(compute.Region).Name`)
				case "project":
					parameters = append(parameters, `g.GetArgs()["project"].(string)`)
				case "zone":
					parameters = append(parameters, `g.GetArgs()["zone"].(string)`)
				}
			}
			parameterOrder := strings.Join(parameters, ", ")
			var tpl bytes.Buffer
			t := template.Must(template.New("resource.go").Funcs(funcMap).Parse(serviceTemplate))
			err := t.Execute(&tpl, map[string]interface{}{
				"titleResourceName":          strings.Title(resource),
				"resource":                   resource,
				"responseName":               value.(map[string]interface{})["response"].(map[string]interface{})["$ref"].(string),
				"terraformName":              terraformResources[resource].getTerraformName(),
				"additionalFields":           terraformResources[resource].getAdditionalFields(),
				"additionalFieldsForRefresh": terraformResources[resource].getAdditionalFieldsForRefresh(),
				"allowEmptyValues":           terraformResources[resource].getAllowEmptyValues(),
				"needRegion":                 terraformResources[resource].ifNeedRegion(),
				"resourcePackageName":        resource,
				"parameterOrder":             parameterOrder,
				"byZone":                     terraformResources[resource].ifNeedZone(strings.Contains(parameterOrder, "zone")),
				"idWithZone":                 terraformResources[resource].ifIDWithZone(strings.Contains(parameterOrder, "zone")),
			})
			if err != nil {
				log.Print(resource, err)
				continue
			}
			rootPath, _ := os.Getwd()
			currentPath := rootPath + pathForGenerateFiles
			err = os.MkdirAll(currentPath, os.ModePerm)
			if err != nil {
				log.Print(resource, err)
				continue
			}
			err = os.WriteFile(currentPath+"/"+resource+"_gen.go", codeFormat(tpl.Bytes()), os.ModePerm)
			if err != nil {
				log.Print(resource, err)
				continue
			}
		} else {
			log.Println(resource)
		}
	}
	var tpl bytes.Buffer
	t := template.Must(template.New("compute.go").Funcs(funcMap).Parse(computeTemplate))
	err = t.Execute(&tpl, map[string]interface{}{
		"services": terraformResources,
	})
	if err != nil {
		log.Print(err)
	}
	rootPath, _ := os.Getwd()
	err = os.WriteFile(rootPath+pathForGenerateFiles+"compute.go", codeFormat(tpl.Bytes()), os.ModePerm)
	if err != nil {
		log.Println(err)
	}
}

func codeFormat(src []byte) []byte {
	code, err := format.Source(src)
	if err != nil {
		log.Println(err)
	}
	return code
}
