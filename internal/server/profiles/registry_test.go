package profiles

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

func TestRegistryMatchesProfileArtifacts(t *testing.T) {
	if err := ValidateRegistryArtifacts(testRepoRoot(t)); err != nil {
		t.Fatal(err)
	}
}

func TestProfileArtifactsSatisfySchema(t *testing.T) {
	repoRoot := testRepoRoot(t)
	compiler := jsonschema.NewCompiler()
	schemaPath := filepath.Join(repoRoot, "ops", "schemas", "profile.schema.json")
	schema, err := compiler.Compile("file://" + filepath.ToSlash(schemaPath))
	if err != nil {
		t.Fatalf("compile schema with santhosh-tekuri/jsonschema v6: %v", err)
	}

	for _, profile := range All() {
		artifact, err := LoadArtifact(filepath.Join(repoRoot, profile.ArtifactPath))
		if err != nil {
			t.Fatalf("load artifact %q: %v", profile.ArtifactPath, err)
		}
		payload, err := json.Marshal(artifact)
		if err != nil {
			t.Fatalf("marshal artifact %q: %v", profile.Key, err)
		}
		var instance any
		if err := json.Unmarshal(payload, &instance); err != nil {
			t.Fatalf("normalize artifact %q: %v", profile.Key, err)
		}
		if err := schema.Validate(instance); err != nil {
			t.Fatalf("artifact %q fails profile schema: %v", profile.Key, err)
		}
	}
}

func testRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "../../.."))
}
