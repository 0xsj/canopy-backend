package main

import (
	"fmt"

	"github.com/0xsj/canopy-backend/pkg/errors"
)

// --- Domain layer (repository) ---

func findLeafByID(id string) error {
	// Simulates a database lookup that finds nothing.
	return errors.New("leaf not found").
		WithKind(errors.KindNotFound).
		WithCode("exploration_leaf_not_found").
		WithSeverity(errors.SeverityLow).
		WithMetadata("leaf_id", id)
}

// --- Service layer ---

func getLeaf(id string, workspaceID string) error {
	err := findLeafByID(id)
	if err != nil {
		return errors.Wrap(err, "exploration: get leaf").
			WithMetadata("workspace_id", workspaceID)
	}
	return nil
}

// --- Handler layer ---

func handleGetLeaf() {
	leafID := "leaf-abc-123"
	workspaceID := "ws-xyz-789"

	err := getLeaf(leafID, workspaceID)
	if err == nil {
		fmt.Println("success (unexpected in this demo)")
		return
	}

	fmt.Println("=== Error as seen by the handler ===")
	fmt.Println()

	// 1. The full error message (includes operation chain).
	fmt.Printf("Error:    %s\n", err)
	fmt.Println()

	// 2. Extract classification for HTTP response.
	kind := errors.GetKind(err)
	code := errors.GetCode(err)
	severity := errors.GetSeverity(err)

	fmt.Printf("Kind:     %s\n", kind)
	fmt.Printf("Code:     %s\n", code)
	fmt.Printf("Severity: %s\n", severity)
	fmt.Println()

	// 3. Collect metadata from the full chain.
	meta := errors.CollectMetadata(err)
	fmt.Println("Metadata (collected from all layers):")
	for k, v := range meta {
		fmt.Printf("  %s = %v\n", k, v)
	}
	fmt.Println()

	// 4. Stack trace from the origin.
	var ce errors.Error
	if errors.As(err, &ce) {
		stack := ce.ErrorStack()
		if stack != nil {
			fmt.Println("Stack (origin):")
			fmt.Println(stack)
		}
	}

	fmt.Println()
	fmt.Println("=== HTTP response the handler would send ===")
	fmt.Println()
	fmt.Printf("Status: 404\n")
	fmt.Printf("Body:   {\"error\": {\"code\": \"%s\", \"message\": \"leaf not found\"}}\n", code)
}

func main() {
	handleGetLeaf()
}
