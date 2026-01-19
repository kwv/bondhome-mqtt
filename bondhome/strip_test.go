package bondhome

import (
	"encoding/json"
	"os"
	"testing"
)

func TestStripLocalHash_SimpleObject(t *testing.T) {
	// Test that "__" is stripped but "_" is kept
	input := json.RawMessage(`{"_":"subtree_hash","__":"local_hash","power":1,"speed":2}`)
	expected := `{"_":"subtree_hash","power":1,"speed":2}`

	result := stripLocalHash(input)

	var resultObj, expectedObj map[string]interface{}
	if err := json.Unmarshal(result, &resultObj); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	if err := json.Unmarshal([]byte(expected), &expectedObj); err != nil {
		t.Fatalf("Failed to unmarshal expected: %v", err)
	}

	if !mapsEqual(resultObj, expectedObj) {
		t.Errorf("Expected %s but got %s", expected, string(result))
	}

	// Verify "__" is not present
	if _, ok := resultObj["__"]; ok {
		t.Error("Expected '__' field to be stripped")
	}

	// Verify "_" is present
	if _, ok := resultObj["_"]; !ok {
		t.Error("Expected '_' field to be kept")
	}
}

func TestStripLocalHash_NestedObject(t *testing.T) {
	// Test that "__" is stripped from nested objects
	input := json.RawMessage(`{"_":"root_hash","__":"root_local","device":{"_":"device_hash","__":"device_local","name":"Fan","state":{"_":"state_hash","__":"state_local","power":1}}}`)

	result := stripLocalHash(input)

	var resultObj map[string]interface{}
	if err := json.Unmarshal(result, &resultObj); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Check root level
	if _, ok := resultObj["__"]; ok {
		t.Error("Expected root '__' field to be stripped")
	}
	if _, ok := resultObj["_"]; !ok {
		t.Error("Expected root '_' field to be kept")
	}

	// Check nested device level
	device, ok := resultObj["device"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'device' to be an object")
	}
	if _, ok := device["__"]; ok {
		t.Error("Expected device '__' field to be stripped")
	}
	if _, ok := device["_"]; !ok {
		t.Error("Expected device '_' field to be kept")
	}

	// Check nested state level
	state, ok := device["state"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected 'state' to be an object")
	}
	if _, ok := state["__"]; ok {
		t.Error("Expected state '__' field to be stripped")
	}
	if _, ok := state["_"]; !ok {
		t.Error("Expected state '_' field to be kept")
	}
}

func TestStripLocalHash_Array(t *testing.T) {
	// Test that "__" is stripped from objects within arrays
	input := json.RawMessage(`{"devices":[{"_":"hash1","__":"local1","id":"a"},{"_":"hash2","__":"local2","id":"b"}]}`)

	result := stripLocalHash(input)

	var resultObj map[string]interface{}
	if err := json.Unmarshal(result, &resultObj); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	devices, ok := resultObj["devices"].([]interface{})
	if !ok {
		t.Fatal("Expected 'devices' to be an array")
	}

	for i, device := range devices {
		deviceObj, ok := device.(map[string]interface{})
		if !ok {
			t.Fatalf("Expected device[%d] to be an object", i)
		}
		if _, ok := deviceObj["__"]; ok {
			t.Errorf("Expected device[%d] '__' field to be stripped", i)
		}
		if _, ok := deviceObj["_"]; !ok {
			t.Errorf("Expected device[%d] '_' field to be kept", i)
		}
	}
}

func TestStripLocalHash_OnlyLocalHash(t *testing.T) {
	// Test object with only "__" field (should become empty object)
	input := json.RawMessage(`{"__":"local_hash"}`)
	expected := `{}`

	result := stripLocalHash(input)

	var resultObj, expectedObj map[string]interface{}
	if err := json.Unmarshal(result, &resultObj); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	if err := json.Unmarshal([]byte(expected), &expectedObj); err != nil {
		t.Fatalf("Failed to unmarshal expected: %v", err)
	}

	if len(resultObj) != 0 {
		t.Errorf("Expected empty object but got %s", string(result))
	}
}

func TestStripLocalHash_NoHashFields(t *testing.T) {
	// Test object without any hash fields
	input := json.RawMessage(`{"power":1,"speed":2,"name":"Fan"}`)
	expected := `{"power":1,"speed":2,"name":"Fan"}`

	result := stripLocalHash(input)

	var resultObj, expectedObj map[string]interface{}
	if err := json.Unmarshal(result, &resultObj); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	if err := json.Unmarshal([]byte(expected), &expectedObj); err != nil {
		t.Fatalf("Failed to unmarshal expected: %v", err)
	}

	if !mapsEqual(resultObj, expectedObj) {
		t.Errorf("Expected %s but got %s", expected, string(result))
	}
}

func TestStripLocalHash_Disabled(t *testing.T) {
	// Test that stripping is disabled when BOND_STRIP_LOCAL_HASH=false
	oldEnv := os.Getenv("BOND_STRIP_LOCAL_HASH")
	os.Setenv("BOND_STRIP_LOCAL_HASH", "false")
	defer os.Setenv("BOND_STRIP_LOCAL_HASH", oldEnv)

	input := json.RawMessage(`{"_":"subtree_hash","__":"local_hash","power":1}`)

	result := stripLocalHash(input)

	var resultObj map[string]interface{}
	if err := json.Unmarshal(result, &resultObj); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Verify "__" is still present
	if _, ok := resultObj["__"]; !ok {
		t.Error("Expected '__' field to be kept when stripping is disabled")
	}
}

func TestStripLocalHash_DisabledWithZero(t *testing.T) {
	// Test that stripping is disabled when BOND_STRIP_LOCAL_HASH=0
	oldEnv := os.Getenv("BOND_STRIP_LOCAL_HASH")
	os.Setenv("BOND_STRIP_LOCAL_HASH", "0")
	defer os.Setenv("BOND_STRIP_LOCAL_HASH", oldEnv)

	input := json.RawMessage(`{"__":"local_hash","power":1}`)

	result := stripLocalHash(input)

	var resultObj map[string]interface{}
	if err := json.Unmarshal(result, &resultObj); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Verify "__" is still present
	if _, ok := resultObj["__"]; !ok {
		t.Error("Expected '__' field to be kept when BOND_STRIP_LOCAL_HASH=0")
	}
}

func TestStripLocalHash_EnabledExplicitly(t *testing.T) {
	// Test that stripping works when BOND_STRIP_LOCAL_HASH=true
	oldEnv := os.Getenv("BOND_STRIP_LOCAL_HASH")
	os.Setenv("BOND_STRIP_LOCAL_HASH", "true")
	defer os.Setenv("BOND_STRIP_LOCAL_HASH", oldEnv)

	input := json.RawMessage(`{"__":"local_hash","power":1}`)

	result := stripLocalHash(input)

	var resultObj map[string]interface{}
	if err := json.Unmarshal(result, &resultObj); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	// Verify "__" is stripped
	if _, ok := resultObj["__"]; ok {
		t.Error("Expected '__' field to be stripped when BOND_STRIP_LOCAL_HASH=true")
	}
}

func TestStripLocalHash_InvalidJSON(t *testing.T) {
	// Test that invalid JSON is returned as-is
	input := json.RawMessage(`{invalid json}`)

	result := stripLocalHash(input)

	// Should return the original input
	if string(result) != string(input) {
		t.Error("Expected invalid JSON to be returned as-is")
	}
}

func TestStripLocalHash_EmptyObject(t *testing.T) {
	// Test empty object
	input := json.RawMessage(`{}`)
	expected := `{}`

	result := stripLocalHash(input)

	var resultObj, expectedObj map[string]interface{}
	if err := json.Unmarshal(result, &resultObj); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	if err := json.Unmarshal([]byte(expected), &expectedObj); err != nil {
		t.Fatalf("Failed to unmarshal expected: %v", err)
	}

	if len(resultObj) != 0 {
		t.Errorf("Expected empty object but got %s", string(result))
	}
}

func TestStripLocalHash_PrimitiveTypes(t *testing.T) {
	// Test that primitive types are preserved
	input := json.RawMessage(`{"string":"value","number":42,"bool":true,"null":null,"__":"local"}`)

	result := stripLocalHash(input)

	var resultObj map[string]interface{}
	if err := json.Unmarshal(result, &resultObj); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if resultObj["string"] != "value" {
		t.Error("Expected string value to be preserved")
	}
	if resultObj["number"] != float64(42) {
		t.Error("Expected number value to be preserved")
	}
	if resultObj["bool"] != true {
		t.Error("Expected bool value to be preserved")
	}
	if resultObj["null"] != nil {
		t.Error("Expected null value to be preserved")
	}
	if _, ok := resultObj["__"]; ok {
		t.Error("Expected '__' field to be stripped")
	}
}

// Helper function to compare maps
func mapsEqual(a, b map[string]interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || !valuesEqual(v, bv) {
			return false
		}
	}
	return true
}

// Helper function to compare values (handles nested structures)
func valuesEqual(a, b interface{}) bool {
	switch av := a.(type) {
	case map[string]interface{}:
		bv, ok := b.(map[string]interface{})
		if !ok {
			return false
		}
		return mapsEqual(av, bv)
	case []interface{}:
		bv, ok := b.([]interface{})
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !valuesEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
