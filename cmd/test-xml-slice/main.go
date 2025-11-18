package main

import (
	"encoding/json"
	"fmt"

	"github.com/bondowe/webfram/internal/bind"
	"github.com/bondowe/webfram/openapi"
)

type User struct {
	ID   int    `xml:"id,attr"`
	Name string `xml:"name"`
}

func main() {
	components := &openapi.Components{}

	// Test top-level slice with custom XMLRootName
	var users []User

	fmt.Println("=== Testing with custom XMLRootName ===")
	sliceSchema := bind.GenerateXMLSchema(users, "users", components)

	fmt.Printf("Schema type: %v\n", sliceSchema)
	if sliceSchema.Schema != nil {
		fmt.Printf("Type: %s\n", sliceSchema.Schema.Type)
		if sliceSchema.Schema.XML != nil {
			fmt.Printf("XML NodeType: %s\n", sliceSchema.Schema.XML.NodeType)
		}
		if sliceSchema.Schema.Items != nil && sliceSchema.Schema.Items.Schema != nil {
			fmt.Printf("Items Type: %s\n", sliceSchema.Schema.Items.Schema.Type)
		}
	}

	schemaBytes, _ := json.MarshalIndent(sliceSchema, "", "  ")
	fmt.Printf("\nFull schema:\n%s\n", string(schemaBytes))

	// Test with empty XMLRootName (should fall back to lowercase type name)
	components2 := &openapi.Components{}
	fmt.Println("\n=== Testing without XMLRootName (fallback to type name) ===")
	sliceSchema2 := bind.GenerateXMLSchema(users, "", components2)
	schemaBytes2, _ := json.MarshalIndent(sliceSchema2, "", "  ")
	fmt.Printf("Full schema:\n%s\n", string(schemaBytes2))
}
