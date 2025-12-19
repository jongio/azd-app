package azure

import (
	"testing"
)

func TestTableCategories_Structure(t *testing.T) {
	if len(TableCategories) == 0 {
		t.Fatal("TableCategories should not be empty")
	}

	expectedCategories := []string{"containerapp", "appservice", "function", "aks", "aci"}
	for _, cat := range expectedCategories {
		if _, ok := TableCategories[cat]; !ok {
			t.Errorf("TableCategories missing expected category: %s", cat)
		}
	}
}

func TestTableCategories_ContainerApp(t *testing.T) {
	cat := TableCategories["containerapp"]

	if cat.Name != "containerapp" {
		t.Errorf("Name = %q, want %q", cat.Name, "containerapp")
	}
	if cat.DisplayName != "Container Apps" {
		t.Errorf("DisplayName = %q, want %q", cat.DisplayName, "Container Apps")
	}
	if len(cat.Tables) < 2 {
		t.Error("ContainerApp category should have at least 2 tables")
	}

	// Verify expected tables
	expectedTables := []string{"ContainerAppConsoleLogs_CL", "ContainerAppSystemLogs_CL"}
	for _, expected := range expectedTables {
		found := false
		for _, table := range cat.Tables {
			if table == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ContainerApp category missing table: %s", expected)
		}
	}
}

func TestTableCategories_AppService(t *testing.T) {
	cat := TableCategories["appservice"]

	if len(cat.Tables) < 4 {
		t.Error("AppService category should have at least 4 tables")
	}

	expectedTables := []string{
		"AppServiceConsoleLogs",
		"AppServiceHTTPLogs",
		"AppServicePlatformLogs",
		"AppServiceAppLogs",
	}

	for _, expected := range expectedTables {
		found := false
		for _, table := range cat.Tables {
			if table == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("AppService category missing table: %s", expected)
		}
	}
}

func TestTableDescriptions_Coverage(t *testing.T) {
	if len(TableDescriptions) == 0 {
		t.Fatal("TableDescriptions should not be empty")
	}

	// Verify key tables have descriptions
	expectedTables := []string{
		"ContainerAppConsoleLogs_CL",
		"AppServiceConsoleLogs",
		"FunctionAppLogs",
		"ContainerLogV2",
	}

	for _, table := range expectedTables {
		desc, ok := TableDescriptions[table]
		if !ok {
			t.Errorf("TableDescriptions missing entry for %s", table)
			continue
		}
		if desc == "" {
			t.Errorf("TableDescriptions[%s] should not be empty", table)
		}
	}
}

func TestTableColumns_Coverage(t *testing.T) {
	if len(TableColumns) == 0 {
		t.Fatal("TableColumns should not be empty")
	}

	// Verify key tables have column definitions
	expectedTables := []string{
		"ContainerAppConsoleLogs_CL",
		"AppServiceConsoleLogs",
		"FunctionAppLogs",
	}

	for _, table := range expectedTables {
		columns, ok := TableColumns[table]
		if !ok {
			t.Errorf("TableColumns missing entry for %s", table)
			continue
		}
		if len(columns) == 0 {
			t.Errorf("TableColumns[%s] should not be empty", table)
		}
		// Verify TimeGenerated is included
		foundTime := false
		for _, col := range columns {
			if col == "TimeGenerated" {
				foundTime = true
				break
			}
		}
		if !foundTime {
			t.Errorf("TableColumns[%s] should include TimeGenerated", table)
		}
	}
}

func TestDefaultTablesByResourceType(t *testing.T) {
	tests := []struct {
		resourceType ResourceType
		wantCount    int
	}{
		{ResourceTypeContainerApp, 1},
		{ResourceTypeAppService, 1},
		{ResourceTypeFunction, 1},
		{ResourceTypeAKS, 1},
		{ResourceTypeContainerInstance, 1},
	}

	for _, tt := range tests {
		t.Run(string(tt.resourceType), func(t *testing.T) {
			tables, ok := DefaultTablesByResourceType[tt.resourceType]
			if !ok {
				t.Fatalf("DefaultTablesByResourceType missing entry for %s", tt.resourceType)
			}
			if len(tables) < tt.wantCount {
				t.Errorf("DefaultTablesByResourceType[%s] has %d tables, want at least %d",
					tt.resourceType, len(tables), tt.wantCount)
			}
		})
	}
}

func TestGetTableInfo(t *testing.T) {
	tests := []struct {
		tableName string
		wantName  string
	}{
		{"ContainerAppConsoleLogs_CL", "ContainerAppConsoleLogs_CL"},
		{"AppServiceConsoleLogs", "AppServiceConsoleLogs"},
		{"FunctionAppLogs", "FunctionAppLogs"},
		{"UnknownTable", "UnknownTable"},
	}

	for _, tt := range tests {
		t.Run(tt.tableName, func(t *testing.T) {
			info := GetTableInfo(tt.tableName)

			if info.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", info.Name, tt.wantName)
			}

			// Known tables should have descriptions and columns
			if _, ok := TableDescriptions[tt.tableName]; ok {
				if info.Description == "" {
					t.Error("Known table should have description")
				}
			}

			// Category should be set
			if info.Category == "" {
				t.Error("Category should be set")
			}
		})
	}
}

func TestGetTableCategory(t *testing.T) {
	tests := []struct {
		tableName    string
		wantCategory string
	}{
		{"ContainerAppConsoleLogs_CL", "containerapp"},
		{"AppServiceConsoleLogs", "appservice"},
		{"FunctionAppLogs", "function"},
		{"ContainerLogV2", "aks"},
		{"ContainerInstanceLog_CL", "aci"},
		{"UnknownTable", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.tableName, func(t *testing.T) {
			got := GetTableCategory(tt.tableName)
			if got != tt.wantCategory {
				t.Errorf("GetTableCategory(%q) = %q, want %q", tt.tableName, got, tt.wantCategory)
			}
		})
	}
}

func TestGetRecommendedTables(t *testing.T) {
	tests := []struct {
		resourceType ResourceType
		wantContain  string
	}{
		{ResourceTypeContainerApp, "ContainerAppConsoleLogs_CL"},
		{ResourceTypeAppService, "AppServiceConsoleLogs"},
		{ResourceTypeFunction, "FunctionAppLogs"},
		{ResourceTypeAKS, "ContainerLogV2"},
		{ResourceTypeContainerInstance, "ContainerInstanceLog_CL"},
	}

	for _, tt := range tests {
		t.Run(string(tt.resourceType), func(t *testing.T) {
			tables := GetRecommendedTables(tt.resourceType)

			if len(tables) == 0 {
				t.Fatal("GetRecommendedTables should return at least one table")
			}

			// Verify expected table is included
			found := false
			for _, table := range tables {
				if table == tt.wantContain {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("GetRecommendedTables(%s) should contain %q", tt.resourceType, tt.wantContain)
			}
		})
	}
}

func TestGetRecommendedTables_UnknownType(t *testing.T) {
	// Unknown resource types should default to Container App tables
	tables := GetRecommendedTables("unknown-type")

	if len(tables) == 0 {
		t.Fatal("GetRecommendedTables should return default tables for unknown type")
	}

	// Should default to Container App
	if tables[0] != "ContainerAppConsoleLogs_CL" {
		t.Errorf("Default should be Container App tables, got %q", tables[0])
	}
}

func TestGetTablesForCategory(t *testing.T) {
	tests := []struct {
		category  string
		wantCount int
		wantTable string
	}{
		{"containerapp", 2, "ContainerAppConsoleLogs_CL"},
		{"appservice", 4, "AppServiceConsoleLogs"},
		{"function", 2, "FunctionAppLogs"},
		{"aks", 7, "ContainerLogV2"},
		{"aci", 2, "ContainerInstanceLog_CL"},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			tables := GetTablesForCategory(tt.category)

			if len(tables) < tt.wantCount {
				t.Errorf("GetTablesForCategory(%q) returned %d tables, want at least %d",
					tt.category, len(tables), tt.wantCount)
			}

			// Verify expected table is included
			found := false
			for _, table := range tables {
				if table == tt.wantTable {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("GetTablesForCategory(%q) should contain %q", tt.category, tt.wantTable)
			}
		})
	}
}

func TestGetTablesForCategory_Unknown(t *testing.T) {
	tables := GetTablesForCategory("nonexistent-category")

	if tables != nil {
		t.Error("GetTablesForCategory for unknown category should return nil")
	}
}

func TestGetAllKnownTables(t *testing.T) {
	tables := GetAllKnownTables()

	if len(tables) == 0 {
		t.Fatal("GetAllKnownTables should return tables")
	}

	// Verify no duplicates
	seen := make(map[string]bool)
	for _, info := range tables {
		if seen[info.Name] {
			t.Errorf("Duplicate table in GetAllKnownTables: %s", info.Name)
		}
		seen[info.Name] = true
	}

	// Verify all tables have required fields
	for _, info := range tables {
		if info.Name == "" {
			t.Error("Table should have name")
		}
		if info.Category == "" {
			t.Errorf("Table %s should have category", info.Name)
		}
	}
}

func TestIsRecommendedTable(t *testing.T) {
	tests := []struct {
		tableName    string
		resourceType ResourceType
		want         bool
	}{
		{"ContainerAppConsoleLogs_CL", ResourceTypeContainerApp, true},
		{"ContainerAppSystemLogs_CL", ResourceTypeContainerApp, false},
		{"AppServiceConsoleLogs", ResourceTypeAppService, true},
		{"AppServiceHTTPLogs", ResourceTypeAppService, false},
		{"FunctionAppLogs", ResourceTypeFunction, true},
		{"ContainerLogV2", ResourceTypeAKS, true},
		{"ContainerLog", ResourceTypeAKS, false},
		{"ContainerInstanceLog_CL", ResourceTypeContainerInstance, true},
		{"UnknownTable", ResourceTypeContainerApp, false},
	}

	for _, tt := range tests {
		t.Run(tt.tableName, func(t *testing.T) {
			got := IsRecommendedTable(tt.tableName, tt.resourceType)
			if got != tt.want {
				t.Errorf("IsRecommendedTable(%q, %s) = %v, want %v",
					tt.tableName, tt.resourceType, got, tt.want)
			}
		})
	}
}

func TestTableInfo_StructFields(t *testing.T) {
	info := GetTableInfo("ContainerAppConsoleLogs_CL")

	// Verify all fields are populated
	if info.Name == "" {
		t.Error("TableInfo.Name should be set")
	}
	if info.Category == "" {
		t.Error("TableInfo.Category should be set")
	}
	if info.Description == "" {
		t.Error("TableInfo.Description should be set for known table")
	}
	if len(info.Columns) == 0 {
		t.Error("TableInfo.Columns should be set for known table")
	}
}

func TestTableCategory_StructFields(t *testing.T) {
	cat := TableCategories["containerapp"]

	if cat.Name == "" {
		t.Error("TableCategory.Name should be set")
	}
	if cat.DisplayName == "" {
		t.Error("TableCategory.DisplayName should be set")
	}
	if len(cat.Tables) == 0 {
		t.Error("TableCategory.Tables should not be empty")
	}
}

func TestTableColumns_ConsistencyWithDescriptions(t *testing.T) {
	// Every table with columns should have a description
	for tableName := range TableColumns {
		if _, ok := TableDescriptions[tableName]; !ok {
			t.Errorf("Table %s has columns but no description", tableName)
		}
	}
}

func TestTableCategories_NoDuplicateTables(t *testing.T) {
	// Track which tables appear in which categories
	tableToCategories := make(map[string][]string)

	for catName, cat := range TableCategories {
		for _, table := range cat.Tables {
			tableToCategories[table] = append(tableToCategories[table], catName)
		}
	}

	// Some tables may legitimately appear in multiple categories (e.g., AppServiceConsoleLogs in function category)
	// But we want to ensure this is intentional
	for table, categories := range tableToCategories {
		if len(categories) > 2 {
			t.Errorf("Table %s appears in too many categories: %v", table, categories)
		}
	}
}
