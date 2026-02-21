package diaglog

// sensitiveKeys are the field names whose values are replaced with
// "[REDACTED]" before any log entry is written (FR-013).
var sensitiveKeys = map[string]bool{
	"authentication": true,
	"password":       true,
	"secret":         true,
	"challenge":      true,
	"salt":           true,
	"auth":           true,
}

// Redact recursively traverses v and replaces the values of any key found in
// sensitiveKeys with the literal string "[REDACTED]". v is not mutated; a new
// map is returned. Non-map types are returned unchanged.
func Redact(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, child := range val {
			if sensitiveKeys[k] {
				out[k] = "[REDACTED]"
			} else {
				out[k] = Redact(child)
			}
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, elem := range val {
			out[i] = Redact(elem)
		}
		return out
	default:
		return v
	}
}
