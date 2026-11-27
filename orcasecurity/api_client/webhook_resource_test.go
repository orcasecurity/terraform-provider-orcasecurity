package api_client

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestWebhookCustomHeaderValues_Decode(t *testing.T) {
	want := WebhookCustomHeaderValues{{Custom: "v"}}
	cases := map[string]string{
		"legacy plain string": `{"X": "v"}`,
		"list of custom":      `{"X": [{"custom":"v"}]}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			var got map[string]WebhookCustomHeaderValues
			if err := json.Unmarshal([]byte(body), &got); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got["X"], want) {
				t.Errorf("got %+v, want %+v", got["X"], want)
			}
		})
	}
}

func TestWebhookCustomHeaderValues_NullIsEmpty(t *testing.T) {
	var got map[string]WebhookCustomHeaderValues
	if err := json.Unmarshal([]byte(`{"X": null}`), &got); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got["X"]) != 0 {
		t.Errorf("got %+v, want no values", got["X"])
	}
}

func TestWebhookCustomHeaderValues_RejectsOtherShapes(t *testing.T) {
	var got map[string]WebhookCustomHeaderValues
	if err := json.Unmarshal([]byte(`{"X": 5}`), &got); err == nil {
		t.Fatal("expected an error for a number value")
	}
}

func TestWebhookCustomHeaderValues_MarshalsAsList(t *testing.T) {
	out, err := json.Marshal(WebhookCustomHeaderValues{{Custom: "v"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(out) != `[{"custom":"v"}]` {
		t.Errorf("got %s", out)
	}
}

func TestWebhookDataSourceConfig_DecodesListHeaders(t *testing.T) {
	body := `{"custom_headers":{"Authorization":[{"custom":"__orca_secret_set__"}],"X-Plain":"p"},"type":"generic","webhook_url":"https://h"}`
	var cfg Config
	if err := json.Unmarshal([]byte(body), &cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := cfg.CustomHeaders["Authorization"]; len(got) != 1 || got[0].Custom != WebhookSecretSetMarker {
		t.Errorf("Authorization: got %+v", got)
	}
	if got := cfg.CustomHeaders["X-Plain"]; len(got) != 1 || got[0].Custom != "p" {
		t.Errorf("X-Plain: got %+v", got)
	}
}
