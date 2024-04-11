package verify

// VarField validates a single field based on the tag string.
// It returns an error if the validation fails.
func VarField(field any, tag string) error {
	return Validate().Var(field, tag)
}

// VarStruct validates a struct based on the validation rules defined in the struct's tags.
func VarStruct(s any) error {
	return Validate().Struct(s)
}

// VarMap validates a map based on the validation rules defined in the map's tags.
func VarMap(m map[string]any, rules map[string]any) map[string]any {
	return Validate().ValidateMap(m, rules)
}
