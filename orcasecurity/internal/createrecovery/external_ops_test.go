package createrecovery_test

import (
	"fmt"

	"terraform-provider-orcasecurity/orcasecurity/api_client"
)

func externalRow(nameField, service string) func(id string) string {
	return func(id string) string {
		return fmt.Sprintf(`{"id":%q,"service_name":%q,%q:%q,"config":{}}`, id, service, nameField, stubTag)
	}
}

func configCase(name, service string, op func(c *api_client.APIClient) runner) opCase {
	return opCase{
		name: name,
		post: "/api/external_service/config",
		get: map[string]func(bool) string{
			"/api/external_service/config": func(created bool) string {
				return rows(created, externalRow("template_name", service), dataList)
			},
		},
		op: op,
	}
}

func resourceCase(name, service string, op func(c *api_client.APIClient) runner) opCase {
	return opCase{
		name: name,
		post: "/api/external_service/resources",
		get: map[string]func(bool) string{
			"/api/external_service/resources": func(created bool) string {
				return rows(created, externalRow("name", service), dataList)
			},
			"/api/external_service/resources/" + newID: func(bool) string {
				return `{"data":` + externalRow("name", service)(newID) + `}`
			},
		},
		op: op,
	}
}

func init() {
	opCases = append(opCases,
		configCase("external_service_config", api_client.OpsgenieServiceName, func(c *api_client.APIClient) runner {
			return runOp(api_client.ExternalServiceConfigCreateOp(c, api_client.OpsgenieServiceName, api_client.OpsgenieExternalServiceConfig{TemplateName: stubTag}))
		}),
		configCase("s3_bucket", api_client.S3BucketServiceName, func(c *api_client.APIClient) runner {
			return runOp(api_client.S3BucketConfigCreateOp(c, api_client.S3BucketServiceName, api_client.S3BucketExternalServiceConfig{TemplateName: stubTag}))
		}),
		configCase("jira_cloud_template", api_client.JiraCloudServiceName, func(c *api_client.APIClient) runner {
			return runOp(api_client.JiraCloudTemplateCreateOp(c, api_client.JiraCloudServiceName, api_client.JiraCloudTemplate{TemplateName: stubTag}))
		}),
		resourceCase("monday_resource", api_client.MondayResourceServiceName, func(c *api_client.APIClient) runner {
			return runOp(c.MondayResourceCreateOp(api_client.MondayResource{Name: stubTag}))
		}),
		resourceCase("servicenow_itsm", api_client.ServiceNowITSMServiceName, func(c *api_client.APIClient) runner {
			return runOp(c.ServiceNowITSMResourceCreateOp(api_client.ServiceNowITSMResource{Name: stubTag}))
		}),
	)
}
