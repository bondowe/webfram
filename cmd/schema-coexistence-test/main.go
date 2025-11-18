package main

import (
	"encoding/json"
	"fmt"

	"github.com/bondowe/webfram/internal/bind"
	"github.com/bondowe/webfram/openapi"
)

type User struct {
	ID    int    `json:"id" xml:"id,attr"`
	Name  string `json:"name" xml:"name"`
	Email string `json:"email" xml:"email"`
}

func main() {
	components := &openapi.Components{}

	// Generate both JSON and XML schemas for the same struct
	var user User

	fmt.Println("Generating JSON schema...")
	jsonSchema := bind.GenerateJSONSchema(user, components)
	fmt.Printf("JSON schema ref: %s\n\n", jsonSchema.Ref)

	fmt.Println("Generating XML schema...")
	xmlSchema := bind.GenerateXMLSchema(user, components)
	fmt.Printf("XML schema ref: %s\n\n", xmlSchema.Ref)

	// Verify both exist in components
	fmt.Printf("Total schemas in components: %d\n", len(components.Schemas))
	fmt.Println("\nComponent names:")
	for name := range components.Schemas {
		fmt.Printf("  - %s\n", name)
	}

	// Pretty print both schemas
	fmt.Println("\n=== JSON Schema ===")
	jsonBytes, _ := json.MarshalIndent(components.Schemas["main.User"], "", "  ")
	fmt.Println(string(jsonBytes))

	fmt.Println("\n=== XML Schema ===")
	xmlBytes, _ := json.MarshalIndent(components.Schemas["main.User.XML"], "", "  ")
	fmt.Println(string(xmlBytes))
}
