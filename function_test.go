package funcstore

import "testing"

func TestNewFunctionDefaults(t *testing.T) {
	fn := NewFunction()

	if fn == nil {
		t.Fatalf("NewFunction() returned nil")
	}
	if fn.GetID() == "" {
		t.Errorf("NewFunction() must set a non-empty ID")
	}
	if fn.GetAlias() != "" {
		t.Errorf("NewFunction() Alias = %q, want empty", fn.GetAlias())
	}
	if fn.GetName() != "" {
		t.Errorf("NewFunction() Name = %q, want empty", fn.GetName())
	}
	if fn.GetRuntime() != "" {
		t.Errorf("NewFunction() Runtime = %q, want empty", fn.GetRuntime())
	}
	if fn.GetSource() != "" {
		t.Errorf("NewFunction() Source = %q, want empty", fn.GetSource())
	}
	if fn.GetTimeoutMs() != 0 {
		t.Errorf("NewFunction() TimeoutMs = %d, want 0", fn.GetTimeoutMs())
	}
	if fn.GetMemLimit() != 0 {
		t.Errorf("NewFunction() MemLimit = %d, want 0", fn.GetMemLimit())
	}
	if fn.GetMetadata() != "" {
		t.Errorf("NewFunction() Metadata = %q, want empty", fn.GetMetadata())
	}
	if fn.GetCreatedAt() == "" {
		t.Errorf("NewFunction() CreatedAt must not be empty")
	}
	if fn.GetUpdatedAt() == "" {
		t.Errorf("NewFunction() UpdatedAt must not be empty")
	}
}

func TestFunctionFluentSetters(t *testing.T) {
	fn := NewFunction()

	fn.SetID("abc123").
		SetAlias("double").
		SetName("Doubler").
		SetRuntime("js").
		SetSource("return input * 2").
		SetTimeoutMs(1000).
		SetMemLimit(1024).
		SetMetadata(`{"author":"dev"}`).
		SetCreatedAt("1700000000").
		SetUpdatedAt("1700000001")

	if fn.GetID() != "abc123" {
		t.Errorf("GetID() = %q, want %q", fn.GetID(), "abc123")
	}
	if fn.GetAlias() != "double" {
		t.Errorf("GetAlias() = %q, want %q", fn.GetAlias(), "double")
	}
	if fn.GetName() != "Doubler" {
		t.Errorf("GetName() = %q, want %q", fn.GetName(), "Doubler")
	}
	if fn.GetRuntime() != "js" {
		t.Errorf("GetRuntime() = %q, want %q", fn.GetRuntime(), "js")
	}
	if fn.GetSource() != "return input * 2" {
		t.Errorf("GetSource() = %q, want %q", fn.GetSource(), "return input * 2")
	}
	if fn.GetTimeoutMs() != 1000 {
		t.Errorf("GetTimeoutMs() = %d, want %d", fn.GetTimeoutMs(), 1000)
	}
	if fn.GetMemLimit() != 1024 {
		t.Errorf("GetMemLimit() = %d, want %d", fn.GetMemLimit(), 1024)
	}
	if fn.GetMetadata() != `{"author":"dev"}` {
		t.Errorf("GetMetadata() = %q, want %q", fn.GetMetadata(), `{"author":"dev"}`)
	}
	if fn.GetCreatedAt() != "1700000000" {
		t.Errorf("GetCreatedAt() = %q, want %q", fn.GetCreatedAt(), "1700000000")
	}
	if fn.GetUpdatedAt() != "1700000001" {
		t.Errorf("GetUpdatedAt() = %q, want %q", fn.GetUpdatedAt(), "1700000001")
	}
}

func TestFunctionHydrate(t *testing.T) {
	fn := NewFunction()

	fn.Hydrate(map[string]string{
		COLUMN_ID:         "hydrated-id",
		COLUMN_ALIAS:      "h-alias",
		COLUMN_NAME:       "Hydrated",
		COLUMN_RUNTIME:    "js",
		COLUMN_SOURCE:     "return 1",
		COLUMN_TIMEOUT_MS: "500",
		COLUMN_MEM_LIMIT:  "2048",
		COLUMN_METADATA:   `{"k":"v"}`,
		COLUMN_CREATED_AT: "1600000000",
		COLUMN_UPDATED_AT: "1600000001",
	})

	if fn.GetID() != "hydrated-id" {
		t.Errorf("after Hydrate() GetID() = %q, want %q", fn.GetID(), "hydrated-id")
	}
	if fn.GetAlias() != "h-alias" {
		t.Errorf("after Hydrate() GetAlias() = %q, want %q", fn.GetAlias(), "h-alias")
	}
	if fn.GetTimeoutMs() != 500 {
		t.Errorf("after Hydrate() GetTimeoutMs() = %d, want %d", fn.GetTimeoutMs(), 500)
	}
	if fn.GetMemLimit() != 2048 {
		t.Errorf("after Hydrate() GetMemLimit() = %d, want %d", fn.GetMemLimit(), 2048)
	}
}

func TestFunctionGetSet(t *testing.T) {
	fn := NewFunction()

	fn.Set(COLUMN_ID, "xyz")
	if got := fn.Get(COLUMN_ID); got != "xyz" {
		t.Errorf("Get(COLUMN_ID) = %q, want %q", got, "xyz")
	}

	fn.Set("custom_key", "custom_value")
	if got := fn.Get("custom_key"); got != "custom_value" {
		t.Errorf("Get(\"custom_key\") = %q, want %q", got, "custom_value")
	}
}

func TestFunctionTimeoutMsZero(t *testing.T) {
	fn := NewFunction()
	if fn.GetTimeoutMs() != 0 {
		t.Errorf("GetTimeoutMs() = %d, want 0", fn.GetTimeoutMs())
	}
}

func TestFunctionTimeoutMsInvalidString(t *testing.T) {
	fn := NewFunction()
	fn.Set(COLUMN_TIMEOUT_MS, "not-a-number")
	if fn.GetTimeoutMs() != 0 {
		t.Errorf("GetTimeoutMs() with invalid string = %d, want 0", fn.GetTimeoutMs())
	}
}
