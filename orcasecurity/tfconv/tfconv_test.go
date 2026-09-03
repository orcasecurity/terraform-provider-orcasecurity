package tfconv

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestKnown(t *testing.T) {
	cases := []struct {
		name string
		val  attr.Value
		want bool
	}{
		{"null string", types.StringNull(), false},
		{"unknown string", types.StringUnknown(), false},
		{"known string", types.StringValue("x"), true},
		{"known empty string", types.StringValue(""), true},
		{"null bool", types.BoolNull(), false},
		{"unknown bool", types.BoolUnknown(), false},
		{"known bool", types.BoolValue(false), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Known(tc.val); got != tc.want {
				t.Errorf("Known(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

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

func TestListToStringSlice(t *testing.T) {
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
			if got := ListToStringSlice(tt.list); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestListToStringSliceNonNull(t *testing.T) {
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
			got := ListToStringSliceNonNull(tt.list)
			if got == nil {
				t.Fatal("must never return nil")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestStringSliceToListPreserveNull(t *testing.T) {
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
			got := StringSliceToListPreserveNull(tt.prior, tt.values)
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetToStringSlice(t *testing.T) {
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
			if got := SetToStringSlice(tt.set); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSetToStringSliceNonNull(t *testing.T) {
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
			got := SetToStringSliceNonNull(tt.set)
			if got == nil {
				t.Fatal("must never return nil")
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestStringSliceToSet(t *testing.T) {
	if got := StringSliceToSet(nil); !got.IsNull() {
		t.Errorf("empty slice must map to null set, got %v", got)
	}
	if got := StringSliceToSet([]string{"a"}); !got.Equal(stringSet(t, []string{"a"})) {
		t.Errorf("expected {a}, got %v", got)
	}
}

func TestStringSliceToSetPreserveNull(t *testing.T) {
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
			got := StringSliceToSetPreserveNull(tt.prior, tt.values)
			if !got.Equal(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
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

func TestStringPtrOrNull(t *testing.T) {
	if got := StringPtrOrNull(nil); !got.IsNull() {
		t.Errorf("nil must map to null, got %v", got)
	}
	empty := ""
	if got := StringPtrOrNull(&empty); !got.IsNull() {
		t.Errorf("empty must map to null, got %v", got)
	}
	s := "x"
	if got := StringPtrOrNull(&s); got.ValueString() != "x" {
		t.Errorf("got %v", got)
	}
}

func TestBoolPtrOrNull(t *testing.T) {
	if got := BoolPtrOrNull(nil); !got.IsNull() {
		t.Errorf("nil must map to null, got %v", got)
	}
	v := true
	if got := BoolPtrOrNull(&v); !got.ValueBool() {
		t.Errorf("got %v", got)
	}
}

func TestStringListFromAPI(t *testing.T) {
	ctx := context.Background()
	got, d := StringListFromAPI(ctx, nil)
	if d.HasError() || !got.IsNull() {
		t.Errorf("nil must be null, got %v %v", got, d)
	}
	got, d = StringListFromAPI(ctx, []string{})
	if d.HasError() || got.IsNull() || len(got.Elements()) != 0 {
		t.Errorf("empty must be empty list, got %v %v", got, d)
	}
	got, d = StringListFromAPI(ctx, []string{"a"})
	if d.HasError() || len(got.Elements()) != 1 {
		t.Errorf("got %v %v", got, d)
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
