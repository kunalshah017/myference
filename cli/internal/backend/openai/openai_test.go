package openai

import "testing"

func TestEndpointAcceptsBaseURLsWithOrWithoutV1(t *testing.T) {
	for baseURL, want := range map[string]string{
		"https://provider.example":           "https://provider.example/v1/models",
		"https://provider.example/v1":        "https://provider.example/v1/models",
		"https://provider.example/openai/v1": "https://provider.example/openai/v1/models",
	} {
		client, err := New(baseURL, "secret", nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := client.endpoint("/v1/models"); got != want {
			t.Fatalf("base=%q endpoint=%q want=%q", baseURL, got, want)
		}
	}
}
