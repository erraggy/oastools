// Collision Resolution example demonstrating joiner collision strategies.
//
// This example shows how to:
//   - Handle schema collisions when merging APIs
//   - Use fail-on-collision to detect conflicts
//   - Use accept-left/accept-right to resolve conflicts
//   - Understand what is lost with each strategy
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/parser"
)

func main() {
	paymentsPath := findSpecPath("specs/payments-api.yaml")
	ordersPath := findSpecPath("specs/orders-api.yaml")

	fmt.Println("Collision Resolution Strategies")
	fmt.Println("================================")
	fmt.Println()
	fmt.Println("Scenario: Both APIs define a 'Transaction' schema with different structures")
	fmt.Printf("  - %s: Transaction for payments (amount, currency, paymentMethod)\n", filepath.Base(paymentsPath))
	fmt.Printf("  - %s: Transaction for orders (orderId, items, total)\n", filepath.Base(ordersPath))
	fmt.Println()

	// Demo 1: fail-on-collision (default)
	fmt.Println("[1/3] Strategy: fail-on-collision (default)")
	fmt.Println("---------------------------------------------")
	demonstrateFailOnCollision(paymentsPath, ordersPath)

	// Demo 2: accept-left
	fmt.Println()
	fmt.Println("[2/3] Strategy: accept-left")
	fmt.Println("---------------------------------------------")
	demonstrateAcceptLeft(paymentsPath, ordersPath)

	// Demo 3: accept-right
	fmt.Println()
	fmt.Println("[3/3] Strategy: accept-right")
	fmt.Println("---------------------------------------------")
	demonstrateAcceptRight(paymentsPath, ordersPath)

	fmt.Println()
	fmt.Println("===============================================")
	fmt.Println("Key Takeaway: accept-left/right silently drops one schema.")
	fmt.Println("If you need BOTH schemas, use rename-left/right instead.")
	fmt.Println("See: examples/workflows/schema-renaming/")
}

func demonstrateFailOnCollision(paymentsPath, ordersPath string) {
	config := joiner.DefaultConfig()
	// DefaultConfig already uses StrategyFailOnCollision for DefaultStrategy
	// but SchemaStrategy defaults to StrategyAcceptLeft, so we need to set it explicitly
	config.SchemaStrategy = joiner.StrategyFailOnCollision

	j := joiner.New(config)
	_, err := j.Join([]string{paymentsPath, ordersPath})

	if err != nil {
		fmt.Printf("  Result: Error (as expected)\n")
		fmt.Printf("  Message: %v\n", err)
		fmt.Println()
		fmt.Println("  This is the safest default - it forces you to explicitly")
		fmt.Println("  choose how to handle the conflict.")
	} else {
		fmt.Println("  Result: Unexpected success")
	}
}

func demonstrateAcceptLeft(paymentsPath, ordersPath string) {
	config := joiner.DefaultConfig()
	config.SchemaStrategy = joiner.StrategyAcceptLeft
	config.PathStrategy = joiner.StrategyAcceptLeft

	j := joiner.New(config)
	result, err := j.Join([]string{paymentsPath, ordersPath})
	if err != nil {
		log.Printf("  Error: %v", err)
		return
	}

	fmt.Println("  Result: Success")
	fmt.Printf("  Collisions resolved: %d\n", result.CollisionCount)

	// Show which Transaction schema won
	doc := result.Document.(*parser.OAS3Document)
	if schema, ok := doc.Components.Schemas["Transaction"]; ok {
		props := getPropertyNames(schema)
		fmt.Printf("  Transaction schema kept: payments-api (left)\n")
		fmt.Printf("  Properties: %v\n", props)
	}

	if len(result.Warnings) > 0 {
		fmt.Println("  Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("    - %s\n", w)
		}
	}

	fmt.Println()
	fmt.Println("  The orders-api Transaction schema was DROPPED.")
	fmt.Println("  Any code expecting orderId/items/total will break!")
}

func demonstrateAcceptRight(paymentsPath, ordersPath string) {
	config := joiner.DefaultConfig()
	config.SchemaStrategy = joiner.StrategyAcceptRight
	config.PathStrategy = joiner.StrategyAcceptRight

	j := joiner.New(config)
	result, err := j.Join([]string{paymentsPath, ordersPath})
	if err != nil {
		log.Printf("  Error: %v", err)
		return
	}

	fmt.Println("  Result: Success")
	fmt.Printf("  Collisions resolved: %d\n", result.CollisionCount)

	// Show which Transaction schema won
	doc := result.Document.(*parser.OAS3Document)
	if schema, ok := doc.Components.Schemas["Transaction"]; ok {
		props := getPropertyNames(schema)
		fmt.Printf("  Transaction schema kept: orders-api (right)\n")
		fmt.Printf("  Properties: %v\n", props)
	}

	if len(result.Warnings) > 0 {
		fmt.Println("  Warnings:")
		for _, w := range result.Warnings {
			fmt.Printf("    - %s\n", w)
		}
	}

	fmt.Println()
	fmt.Println("  The payments-api Transaction schema was DROPPED.")
	fmt.Println("  Any code expecting amount/currency/paymentMethod will break!")
}

func getPropertyNames(schema *parser.Schema) []string {
	if schema == nil || schema.Properties == nil {
		return nil
	}
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	// Sort for consistent output
	slices.Sort(names)
	return names
}

func findSpecPath(relativePath string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}
	return filepath.Join(filepath.Dir(filename), relativePath)
}
