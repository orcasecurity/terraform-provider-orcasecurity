package crown_jewel

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestSchemaGroupUniqueIDRequiredRequiresReplace(t *testing.T) {
	r := &crownJewelResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["group_unique_id"].(schema.StringAttribute)
	if !ok {
		t.Fatal("group_unique_id is not a StringAttribute")
	}
	if !attr.Required {
		t.Error("group_unique_id must be Required")
	}
	if len(attr.PlanModifiers) != 1 {
		t.Errorf("group_unique_id must have RequiresReplace, got %d modifiers", len(attr.PlanModifiers))
	}
}

func TestSchemaDescriptionOptional(t *testing.T) {
	r := &crownJewelResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["description"].(schema.StringAttribute)
	if !ok {
		t.Fatal("description is not a StringAttribute")
	}
	if !attr.Optional {
		t.Error("description must be Optional")
	}
	if attr.Required {
		t.Error("description must not be Required")
	}
}

func TestSchemaIDComputed(t *testing.T) {
	r := &crownJewelResource{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	attr, ok := resp.Schema.Attributes["id"].(schema.StringAttribute)
	if !ok {
		t.Fatal("id is not a StringAttribute")
	}
	if !attr.Computed {
		t.Error("id must be Computed")
	}
}

func TestMetadataTypeName(t *testing.T) {
	r := &crownJewelResource{}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "orcasecurity"}, resp)
	if resp.TypeName != "orcasecurity_crown_jewel" {
		t.Errorf("unexpected type name: %s", resp.TypeName)
	}
}
