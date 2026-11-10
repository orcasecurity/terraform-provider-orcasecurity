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

func configOp[P any](c *api_client.APIClient, service string, create func() (*P, error), id func(*P) string) runner {
	return runOp(api_client.ExternalServiceConfigCreateOp(c, service, stubTag, create), id)
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
			return configOp(c, api_client.OpsgenieServiceName, func() (*api_client.OpsgenieExternalServiceConfig, error) {
				return c.CreateOpsgenieConfig(api_client.OpsgenieExternalServiceConfig{TemplateName: stubTag})
			}, func(o *api_client.OpsgenieExternalServiceConfig) string { return o.ID })
		}),
		configCase("s3_bucket", api_client.S3BucketServiceName, func(c *api_client.APIClient) runner {
			return configOp(c, api_client.S3BucketServiceName, func() (*api_client.S3BucketExternalServiceConfig, error) {
				return c.CreateS3BucketConfig(api_client.S3BucketExternalServiceConfig{TemplateName: stubTag})
			}, func(o *api_client.S3BucketExternalServiceConfig) string { return o.ID })
		}),
		configCase("jira_cloud_template", api_client.JiraCloudServiceName, func(c *api_client.APIClient) runner {
			return configOp(c, api_client.JiraCloudServiceName, func() (*api_client.JiraCloudTemplate, error) {
				return c.CreateJiraCloudTemplate(api_client.JiraCloudTemplate{TemplateName: stubTag})
			}, func(o *api_client.JiraCloudTemplate) string { return o.ID })
		}),
		resourceCase("monday_resource", api_client.MondayResourceServiceName, func(c *api_client.APIClient) runner {
			return runOp(c.MondayResourceCreateOp(api_client.MondayResource{Name: stubTag}),
				func(m *api_client.MondayResource) string { return m.ID })
		}),
		resourceCase("servicenow_itsm", api_client.ServiceNowITSMServiceName, func(c *api_client.APIClient) runner {
			return runOp(c.ServiceNowITSMResourceCreateOp(api_client.ServiceNowITSMResource{Name: stubTag}),
				func(s *api_client.ServiceNowITSMResource) string { return s.ID })
		}),
	)
}
