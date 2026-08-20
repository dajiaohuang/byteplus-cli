package paramdescriptions

import (
	"encoding/json"
	"testing"
)

func TestAssetParamsJSONLoads(t *testing.T) {
	data, err := Asset("params.json")
	if err != nil {
		t.Fatalf("Asset(params.json): %v", err)
	}
	var payload struct {
		Apis map[string]interface{} `json:"apis"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.Apis == nil {
		t.Fatal("apis map must be present")
	}
}
