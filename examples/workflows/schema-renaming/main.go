// Schema Renaming example demonstrating joiner rename strategies.
//
// This example shows how to:
//   - Preserve both conflicting schemas using rename strategies
//   - Use rename-left vs rename-right
//   - Customize renamed schema names with templates
//   - Apply namespace prefixes for consistent naming
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/erraggy/oastools/joiner"
	"github.com/erraggy/oastools/parser"
)

func main() {
	billingPath := findSpecPath("specs/billing-api.yaml")
	crmPath := findSpecPath("specs/crm-api.yaml")

	fmt.Println("Schema Renaming Strategies")
	fmt.Println("==========================")
	fmt.Println()
	fmt.Println("Scenario: Both APIs legitimately need different 'Account' schemas")
	fmt.Println("  - billing-api.yaml: Account {accountId, balance, creditLimit, paymentTerms}")
	fmt.Println("  - crm-api.yaml: Account {accountId, companyName, industry, employeeCount}")
	fmt.Println()
	fmt.Println("Unlike accept-left/right, we need BOTH schemas preserved!")
	fmt.Println()

	// Demo 1: rename-right
	fmt.Println("[1/4] Strategy: rename-right")
	fmt.Println("---------------------------------------------")
	demonstrateRenameRight(billingPath, crmPath)

	// Demo 2: rename-left
	fmt.Println()
	fmt.Println("[2/4] Strategy: rename-left")
	fmt.Println("---------------------------------------------")
	demonstrateRenameLeft(billingPath, crmPath)

	// Demo 3: custom template
	fmt.Println()
	fmt.Println("[3/4] Custom rename template")
	fmt.Println("---------------------------------------------")
	demonstrateCustomTemplate(billingPath, crmPath)

	// Demo 4: namespace prefixes
	fmt.Println()
	fmt.Println("[4/4] Namespace prefixes")
	fmt.Println("---------------------------------------------")
	demonstrateNamespacePrefixes(billingPath, crmPath)

	fmt.Println()
	fmt.Println("===============================================")
	fmt.Println("Key Takeaway: Rename strategies preserve BOTH schemas.")
	fmt.Println("The joiner automatically rewrites all $ref pointers!")
}

func demonstrateRenameRight(billingPath, crmPath string) {
	config := joiner.DefaultConfig()
	config.SchemaStrategy = joiner.StrategyRenameRight
	config.PathStrategy = joiner.StrategyAcceptLeft // Handle /accounts path collision
	// Default template: "{{.Name}}_{{.Source}}"

	j := joiner.New(config)
	result, err := j.Join([]string{billingPath, crmPath})
	if err != nil {
		log.Printf("  Error: %v", err)
		return
	}

	accessor := result.ToParseResult().AsAccessor()
	if accessor == nil {
		log.Printf("  Could not access document")
		return
	}
	schemas := getSortedSchemaNames(accessor)

	fmt.Println("  Result: Success")
	fmt.Printf("  Schemas: %v\n", schemas)
	fmt.Println()
	fmt.Println("  How it works:")
	fmt.Println("    - billing-api's Account -> Account (kept original name)")
	fmt.Println("    - crm-api's Account -> Account_crm_api (renamed)")
	fmt.Println()
	fmt.Println("  All $refs in crm-api paths now point to Account_crm_api")

	showAccountSchemas(accessor)
}

func demonstrateRenameLeft(billingPath, crmPath string) {
	config := joiner.DefaultConfig()
	config.SchemaStrategy = joiner.StrategyRenameLeft
	config.PathStrategy = joiner.StrategyAcceptLeft // Handle /accounts path collision

	j := joiner.New(config)
	result, err := j.Join([]string{billingPath, crmPath})
	if err != nil {
		log.Printf("  Error: %v", err)
		return
	}

	accessor := result.ToParseResult().AsAccessor()
	if accessor == nil {
		log.Printf("  Could not access document")
		return
	}
	schemas := getSortedSchemaNames(accessor)

	fmt.Println("  Result: Success")
	fmt.Printf("  Schemas: %v\n", schemas)
	fmt.Println()
	fmt.Println("  How it works:")
	fmt.Println("    - billing-api's Account -> Account_billing_api (renamed)")
	fmt.Println("    - crm-api's Account -> Account (kept original name)")
	fmt.Println()
	fmt.Println("  All $refs in billing-api paths now point to Account_billing_api")

	showAccountSchemas(accessor)
}

func demonstrateCustomTemplate(billingPath, crmPath string) {
	config := joiner.DefaultConfig()
	config.SchemaStrategy = joiner.StrategyRenameRight
	config.PathStrategy = joiner.StrategyAcceptLeft
	config.RenameTemplate = "{{.Source | pascalCase}}{{.Name}}"

	j := joiner.New(config)
	result, err := j.Join([]string{billingPath, crmPath})
	if err != nil {
		log.Printf("  Error: %v", err)
		return
	}

	accessor := result.ToParseResult().AsAccessor()
	if accessor == nil {
		log.Printf("  Could not access document")
		return
	}
	schemas := getSortedSchemaNames(accessor)

	fmt.Println("  Result: Success")
	fmt.Printf("  Schemas: %v\n", schemas)
	fmt.Println()
	fmt.Println("  Template: {{.Source | pascalCase}}{{.Name}}")
	fmt.Println()
	fmt.Println("  Available template variables:")
	fmt.Println("    - {{.Name}}   - Original schema name")
	fmt.Println("    - {{.Source}} - Source file name (without extension)")
	fmt.Println("    - {{.Index}}  - Document index (0-based)")
	fmt.Println()
	fmt.Println("  Available functions:")
	fmt.Println("    - pascalCase  - UserName")
	fmt.Println("    - camelCase   - userName")
	fmt.Println("    - snakeCase   - user_name")
	fmt.Println("    - kebabCase   - user-name")
}

func demonstrateNamespacePrefixes(billingPath, crmPath string) {
	config := joiner.DefaultConfig()
	config.SchemaStrategy = joiner.StrategyRenameRight
	config.PathStrategy = joiner.StrategyAcceptLeft
	// NamespacePrefix uses full file paths as keys
	config.NamespacePrefix = map[string]string{
		billingPath: "Billing",
		crmPath:     "CRM",
	}
	config.AlwaysApplyPrefix = false // Only on collision (default)

	j := joiner.New(config)
	result, err := j.Join([]string{billingPath, crmPath})
	if err != nil {
		log.Printf("  Error: %v", err)
		return
	}

	accessor := result.ToParseResult().AsAccessor()
	if accessor == nil {
		log.Printf("  Could not access document")
		return
	}
	schemas := getSortedSchemaNames(accessor)

	fmt.Println("  Result: Success")
	fmt.Printf("  Schemas: %v\n", schemas)
	fmt.Println()
	fmt.Println("  Configuration:")
	fmt.Printf("    NamespacePrefix: %s -> Billing\n", filepath.Base(billingPath))
	fmt.Printf("                     %s -> CRM\n", filepath.Base(crmPath))
	fmt.Println("    AlwaysApplyPrefix: false (only on collision)")
	fmt.Println()
	fmt.Println("  Result:")
	fmt.Println("    - Account (billing) -> Account (kept, first document)")
	fmt.Println("    - Account (CRM) -> CRM_Account (prefixed due to collision)")
	fmt.Println("    - Invoice, Contact -> unchanged (no collision)")
	fmt.Println()
	fmt.Println("  Tip: Set AlwaysApplyPrefix=true to prefix ALL schemas,")
	fmt.Println("       useful for consistent naming across large merges.")
}

func showAccountSchemas(accessor parser.DocumentAccessor) {
	fmt.Println()
	fmt.Println("  Schema properties comparison:")
	schemas := accessor.GetSchemas()
	if schemas == nil {
		return
	}
	for name, schema := range schemas {
		if len(name) >= 7 && name[:7] == "Account" || name == "Account" {
			props := getPropertyNames(schema)
			fmt.Printf("    %s: %v\n", name, props)
		}
	}
}

func getPropertyNames(schema *parser.Schema) []string {
	if schema == nil || schema.Properties == nil {
		return nil
	}
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func getSortedSchemaNames(accessor parser.DocumentAccessor) []string {
	schemas := accessor.GetSchemas()
	if schemas == nil {
		return nil
	}
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func findSpecPath(relativePath string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("Unable to get current file path")
	}
	return filepath.Join(filepath.Dir(filename), relativePath)
}
