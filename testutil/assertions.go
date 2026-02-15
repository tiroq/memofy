package testutil

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

// AssertEqual checks if two values are equal
func AssertEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("%s: expected %v, got %v", msg, expected, actual)
	}
}

// AssertNotEqual checks if two values are not equal
func AssertNotEqual(t *testing.T, expected, actual interface{}, msg string) {
	t.Helper()
	if expected == actual {
		t.Fatalf("%s: expected values to differ, both are %v", msg, expected)
	}
}

// AssertTrue checks if a condition is true
func AssertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Fatalf("%s: expected true, got false", msg)
	}
}

// AssertFalse checks if a condition is false
func AssertFalse(t *testing.T, condition bool, msg string) {
	t.Helper()
	if condition {
		t.Fatalf("%s: expected false, got true", msg)
	}
}

// AssertNil checks if a value is nil
func AssertNil(t *testing.T, value interface{}, msg string) {
	t.Helper()
	if value != nil {
		t.Fatalf("%s: expected nil, got %v", msg, value)
	}
}

// AssertNotNil checks if a value is not nil
func AssertNotNil(t *testing.T, value interface{}, msg string) {
	t.Helper()
	if value == nil {
		t.Fatalf("%s: expected non-nil value", msg)
	}
}

// AssertNoError checks if an error is nil
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: unexpected error: %v", msg, err)
	}
}

// AssertError checks if an error is not nil
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected an error but got nil", msg)
	}
}

// AssertErrorContains checks if an error contains a specific substring
func AssertErrorContains(t *testing.T, err error, substr string, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected an error but got nil", msg)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("%s: error %q does not contain %q", msg, err.Error(), substr)
	}
}

// AssertStringContains checks if a string contains a substring
func AssertStringContains(t *testing.T, str, substr string, msg string) {
	t.Helper()
	if !strings.Contains(str, substr) {
		t.Fatalf("%s: string %q does not contain %q", msg, str, substr)
	}
}

// AssertStringNotContains checks if a string does not contain a substring
func AssertStringNotContains(t *testing.T, str, substr string, msg string) {
	t.Helper()
	if strings.Contains(str, substr) {
		t.Fatalf("%s: string %q should not contain %q", msg, str, substr)
	}
}

// VersionEquals checks if a version string matches expected format
func VersionEquals(t *testing.T, version, expected string) {
	t.Helper()
	if version != expected {
		t.Fatalf("Version mismatch: expected %s, got %s", expected, version)
	}
}

// VersionValid checks if a version string has valid format (X.Y.Z)
func VersionValid(t *testing.T, version string) {
	t.Helper()
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		t.Fatalf("Invalid version format: %s (expected X.Y.Z or X.Y)", version)
	}
}

// WithinDuration checks if a duration is within expected range
func WithinDuration(t *testing.T, actual, expected, tolerance time.Duration, msg string) {
	t.Helper()
	diff := actual - expected
	if diff < 0 {
		diff = -diff
	}

	if diff > tolerance {
		t.Fatalf("%s: duration %v not within %v of expected %v (diff: %v)",
			msg, actual, tolerance, expected, diff)
	}
}

// AssertJSONValid checks if a string is valid JSON
func AssertJSONValid(t *testing.T, jsonStr string, msg string) {
	t.Helper()
	var result interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("%s: invalid JSON: %v", msg, err)
	}
}

// AssertJSONContainsKey checks if JSON contains a specific key
func AssertJSONContainsKey(t *testing.T, jsonStr, key string, msg string) {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		t.Fatalf("%s: invalid JSON: %v", msg, err)
	}

	if _, exists := result[key]; !exists {
		t.Fatalf("%s: JSON does not contain key %q", msg, key)
	}
}

// WaitForCondition waits for a condition to become true within timeout
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("%s: condition not met within %v", msg, timeout)
}

// AssertEventually checks if a condition becomes true within timeout
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, interval time.Duration, msg string) {
	t.Helper()
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}

	t.Fatalf("%s: condition did not become true within %v", msg, timeout)
}

// AssertInRange checks if a value is within a range
func AssertInRange(t *testing.T, value, min, max float64, msg string) {
	t.Helper()
	if value < min || value > max {
		t.Fatalf("%s: value %v not in range [%v, %v]", msg, value, min, max)
	}
}

// SourceExists checks if a source with given name exists in OBS response
type OBSSource struct {
	Name    string
	Enabled bool
	Kind    string
}

// ParseOBSResponse parses a common OBS WebSocket response
func ParseOBSResponse(t *testing.T, response map[string]interface{}) map[string]interface{} {
	t.Helper()

	d, ok := response["d"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'd' field")
	}

	return d
}

// GetResponseData extracts responseData from OBS response
func GetResponseData(t *testing.T, response map[string]interface{}) map[string]interface{} {
	t.Helper()

	d := ParseOBSResponse(t, response)
	data, ok := d["responseData"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'responseData' field")
	}

	return data
}

// GetRequestStatus extracts requestStatus from OBS response
func GetRequestStatus(t *testing.T, response map[string]interface{}) map[string]interface{} {
	t.Helper()

	d := ParseOBSResponse(t, response)
	status, ok := d["requestStatus"].(map[string]interface{})
	if !ok {
		t.Fatal("Response missing 'requestStatus' field")
	}

	return status
}

// AssertStatusCode checks OBS response status code
func AssertStatusCode(t *testing.T, response map[string]interface{}, expectedCode int, msg string) {
	t.Helper()

	status := GetRequestStatus(t, response)
	code, ok := status["code"].(float64)
	if !ok {
		t.Fatalf("%s: status code is not a number", msg)
	}

	if int(code) != expectedCode {
		comment := status["comment"]
		t.Fatalf("%s: expected status code %d, got %d (comment: %v)", msg, expectedCode, int(code), comment)
	}
}

// AssertSuccessResponse checks if OBS response indicates success
func AssertSuccessResponse(t *testing.T, response map[string]interface{}, msg string) {
	t.Helper()

	status := GetRequestStatus(t, response)
	result, ok := status["result"].(bool)
	if !ok {
		t.Fatalf("%s: result is not a boolean", msg)
	}

	if !result {
		code := status["code"]
		comment := status["comment"]
		t.Fatalf("%s: request failed with code %v: %v", msg, code, comment)
	}
}

// MustMarshalJSON marshals data to JSON or fails the test
func MustMarshalJSON(t *testing.T, data interface{}) string {
	t.Helper()
	bytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	return string(bytes)
}

// MustUnmarshalJSON unmarshals JSON or fails the test
func MustUnmarshalJSON(t *testing.T, jsonStr string, target interface{}) {
	t.Helper()
	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
}

// Retry retries a function until it succeeds or max attempts reached
func Retry(t *testing.T, maxAttempts int, delay time.Duration, fn func() error) error {
	t.Helper()

	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err != nil {
			lastErr = err
			if i < maxAttempts-1 {
				time.Sleep(delay)
			}
			continue
		}
		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", maxAttempts, lastErr)
}
