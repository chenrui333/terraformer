// SPDX-License-Identifier: Apache-2.0

package gcp

import (
	"context"

	"github.com/chenrui333/terraformer/terraformutils"

	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

var cloudSQLAllowEmptyValues = []string{}

var cloudSQLAdditionalFields = map[string]interface{}{}

type CloudSQLGenerator struct {
	GCPService
}

func (g *CloudSQLGenerator) loadDBInstances(svc *sqladmin.Service, project string) error {
	dbInstances, err := svc.Instances.List(project).Do()
	if err != nil {
		return err
	}
	for _, dbInstance := range dbInstances.Items {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			dbInstance.Name,
			dbInstance.Name,
			"google_sql_database_instance",
			g.ProviderName,
			map[string]string{
				"project": project,
				"name":    dbInstance.Name,
			},
			cloudSQLAllowEmptyValues,
			cloudSQLAdditionalFields,
		))
		err := g.loadDBs(svc, dbInstance.Name, project)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *CloudSQLGenerator) loadDBs(svc *sqladmin.Service, instanceName, project string) error {
	DBs, err := svc.Databases.List(project, instanceName).Do()
	if err != nil {
		return err
	}
	for _, db := range DBs.Items {
		g.Resources = append(g.Resources, terraformutils.NewResource(
			instanceName+":"+db.Name,
			instanceName+"-"+db.Name,
			"google_sql_database",
			g.ProviderName,
			map[string]string{
				"instance": instanceName,
				"project":  project,
				"name":     db.Name,
			},

			cloudSQLAllowEmptyValues,
			cloudSQLAdditionalFields,
		))
	}
	return nil
}

// Generate TerraformResources from GCP API,
// from each databases create many TerraformResource(dbinstance + databases)
// Need dbinstance name as ID for terraform resource
func (g *CloudSQLGenerator) InitResources() error {
	project := g.GetArgs()["project"].(string)
	ctx := context.Background()
	svc, err := sqladmin.NewService(ctx)
	if err != nil {
		return err
	}
	if err := g.loadDBInstances(svc, project); err != nil {
		return err
	}

	return nil
}
