package verify

// VarField validates a single field based on the tag string.
// It returns an error if the validation fails.
func Field(field any, tag string) error {
	return Validate().Var(field, tag)
}

// VarWithValue validates a single field based on the tag string, with a comparison value.
// It returns an error if the validation fails.
func WithValue(field1, field2 any, tag string) error {
	return Validate().VarWithValue(field1, field2, tag)
}

// VarStruct validates a struct based on the validation rules defined in the struct's tags.
func Struct(s any) error {
	return Validate().Struct(s)
}

// VarMap validates a map based on the validation rules defined in the map's tags, 返回一个 err 的 map.
func Map(m map[string]any, rules map[string]any) map[string]any {
	return Validate().ValidateMap(m, rules)
}
