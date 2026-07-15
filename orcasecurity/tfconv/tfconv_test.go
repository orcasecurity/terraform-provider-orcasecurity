package tfconv

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func stringList(t *testing.T, values []string) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		t.Fatalf("building list: %v", diags)
	}
	return list
}

func stringSet(t *testing.T, values []string) types.Set {
	t.Helper()
	set, diags := types.SetValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		t.Fatalf("building set: %v", diags)
	}
	return set
}

func TestStringListToAPI(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		list types.List
		want []string
	}{
		{"null becomes nil", types.ListNull(types.StringType), nil},
		{"unknown becomes nil", types.ListUnknown(types.StringType), nil},
		{"empty stays empty non-nil", stringList(t, []string{}), []string{}},
		{"values pass through", stringList(t, []string{"a", "b"}), []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringListToAPI(ctx, tt.list); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestStringListToAPINonNull(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		list types.List
		want []string
	}{
		{"null becomes empty slice", types.ListNull(types.StringType), []string{}},
		{"unknown becomes empty slice", types.ListUnknown(types.StringType), []string{}},
		{"values pass through", stringList(t, []string{"a"}), []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringListToAPINonNull(ctx, tt.list)
			if got == nil {
				t.Fatal("must never return nil")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestStringListFromAPIPreserveNull(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		prior  types.List
		values []string
		want   types.List
	}{
		{"empty remote + null prior stays null", types.ListNull(types.StringType), nil, types.ListNull(types.StringType)},
		{"empty remote + configured prior becomes empty list", stringList(t, []string{}), nil, stringList(t, []string{})},
		{"values override null prior", types.ListNull(types.StringType), []string{"a"}, stringList(t, []string{"a"})},
		{"values override prior", stringList(t, []string{"old"}), []string{"new"}, stringList(t, []string{"new"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := StringListFromAPIPreserveNull(ctx, tt.prior, tt.values)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringSetToAPI(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		set  types.Set
		want []string
	}{
		{"null becomes nil", types.SetNull(types.StringType), nil},
		{"unknown becomes nil", types.SetUnknown(types.StringType), nil},
		{"empty stays empty non-nil", stringSet(t, []string{}), []string{}},
		{"values pass through", stringSet(t, []string{"a", "b"}), []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringSetToAPI(ctx, tt.set); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestStringSetToAPINonNull(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		set  types.Set
		want []string
	}{
		{"null becomes empty slice", types.SetNull(types.StringType), []string{}},
		{"unknown becomes empty slice", types.SetUnknown(types.StringType), []string{}},
		{"values pass through", stringSet(t, []string{"a"}), []string{"a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringSetToAPINonNull(ctx, tt.set)
			if got == nil {
				t.Fatal("must never return nil")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestStringSetFromAPIPreserveNull(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		prior  types.Set
		values []string
		want   types.Set
	}{
		{"empty remote + null prior stays null", types.SetNull(types.StringType), nil, types.SetNull(types.StringType)},
		{"empty remote + configured prior becomes empty set", stringSet(t, []string{}), nil, stringSet(t, []string{})},
		{"values override null prior", types.SetNull(types.StringType), []string{"a"}, stringSet(t, []string{"a"})},
		{"values override prior", stringSet(t, []string{"old"}), []string{"new"}, stringSet(t, []string{"new"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := StringSetFromAPIPreserveNull(ctx, tt.prior, tt.values)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringOrNull(t *testing.T) {
	if got := StringOrNull(""); !got.IsNull() {
		t.Errorf("empty string must map to null, got %v", got)
	}
	if got := StringOrNull("x"); got.ValueString() != "x" {
		t.Errorf("expected x, got %v", got)
	}
}

func TestInt64ToAPIPtr(t *testing.T) {
	if got := Int64ToAPIPtr(types.Int64Null()); got != nil {
		t.Errorf("null must map to nil, got %v", got)
	}
	if got := Int64ToAPIPtr(types.Int64Unknown()); got != nil {
		t.Errorf("unknown must map to nil, got %v", got)
	}
	if got := Int64ToAPIPtr(types.Int64Value(7)); got == nil || *got != 7 {
		t.Errorf("expected 7, got %v", got)
	}
}

func TestInt64FromAPIPtr(t *testing.T) {
	if got := Int64FromAPIPtr(nil); !got.IsNull() {
		t.Errorf("nil must map to null, got %v", got)
	}
	value := int64(7)
	if got := Int64FromAPIPtr(&value); got.ValueInt64() != 7 {
		t.Errorf("expected 7, got %v", got)
	}
}
