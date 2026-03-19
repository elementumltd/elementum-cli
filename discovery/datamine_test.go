// Copyright (c) ElementumAI, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"testing"
)

func TestDatamine_GetUUIDMappings(t *testing.T) {
	dm := &Datamine{
		ID:          "dm-uuid-123",
		Name:        "Test Datamine",
		TableID:     "table-uuid-456",
		Description: "A test datamine",
	}

	resourceName := "test_datamine"
	mappings := dm.GetUUIDMappings(resourceName)

	expectedID := "elementum_datamine." + resourceName + ".id"
	if mappings[dm.ID] != expectedID {
		t.Errorf("expected mapping for datamine ID to be '%s', got '%s'", expectedID, mappings[dm.ID])
	}
}

func TestDatamine_ParseSchedule(t *testing.T) {
	tests := []struct {
		name             string
		schedule         map[string]interface{}
		expectedType     string
		expectedTimeUnit string
		expectedValue    int64
		expectedCron     string
	}{
		{
			name: "fixed schedule",
			schedule: map[string]interface{}{
				"__typename": "DatamineScheduleFixed",
				"timeUnit":   "HOUR",
				"value":      float64(4),
			},
			expectedType:     "fixed",
			expectedTimeUnit: "HOUR",
			expectedValue:    4,
			expectedCron:     "",
		},
		{
			name: "cron schedule",
			schedule: map[string]interface{}{
				"__typename":     "DatamineScheduleCron",
				"cronExpression": "0 0 * * *",
			},
			expectedType:     "cron",
			expectedTimeUnit: "",
			expectedValue:    0,
			expectedCron:     "0 0 * * *",
		},
		{
			name: "unknown schedule type",
			schedule: map[string]interface{}{
				"__typename": "UnknownType",
			},
			expectedType:     "",
			expectedTimeUnit: "",
			expectedValue:    0,
			expectedCron:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dm := &Datamine{}
			parseSchedule(tt.schedule, dm)

			if dm.ScheduleType != tt.expectedType {
				t.Errorf("expected schedule type '%s', got '%s'", tt.expectedType, dm.ScheduleType)
			}
			if dm.TimeUnit != tt.expectedTimeUnit {
				t.Errorf("expected time unit '%s', got '%s'", tt.expectedTimeUnit, dm.TimeUnit)
			}
			if dm.Value != tt.expectedValue {
				t.Errorf("expected value %d, got %d", tt.expectedValue, dm.Value)
			}
			if dm.CronExpression != tt.expectedCron {
				t.Errorf("expected cron '%s', got '%s'", tt.expectedCron, dm.CronExpression)
			}
		})
	}
}

func TestDatamine_FixedScheduleValues(t *testing.T) {
	validTimeUnits := []string{"DAY", "HOUR", "MINUTE", "MONTH", "WEEK"}

	for _, unit := range validTimeUnits {
		schedule := map[string]interface{}{
			"__typename": "DatamineScheduleFixed",
			"timeUnit":   unit,
			"value":      float64(1),
		}

		dm := &Datamine{}
		parseSchedule(schedule, dm)

		if dm.ScheduleType != "fixed" {
			t.Errorf("expected schedule type 'fixed' for time unit '%s'", unit)
		}
		if dm.TimeUnit != unit {
			t.Errorf("expected time unit '%s', got '%s'", unit, dm.TimeUnit)
		}
	}
}

func TestDatamine_CronScheduleExpression(t *testing.T) {
	cronExpressions := []string{
		"0 0 * * *",       // Daily at midnight
		"0 */4 * * *",     // Every 4 hours
		"0 9 * * MON-FRI", // Weekdays at 9 AM
		"0 0 1 * *",       // First day of month
		"*/15 * * * *",    // Every 15 minutes
	}

	for _, expr := range cronExpressions {
		schedule := map[string]interface{}{
			"__typename":     "DatamineScheduleCron",
			"cronExpression": expr,
		}

		dm := &Datamine{}
		parseSchedule(schedule, dm)

		if dm.ScheduleType != "cron" {
			t.Errorf("expected schedule type 'cron' for expression '%s'", expr)
		}
		if dm.CronExpression != expr {
			t.Errorf("expected cron expression '%s', got '%s'", expr, dm.CronExpression)
		}
	}
}

func TestDatamine_PrimaryColumns(t *testing.T) {
	dm := &Datamine{
		ID:      "dm-123",
		Name:    "Test",
		TableID: "table-456",
		PrimaryColumnIDs: []string{
			"col-1",
			"col-2",
			"col-3",
		},
	}

	if len(dm.PrimaryColumnIDs) != 3 {
		t.Errorf("expected 3 primary columns, got %d", len(dm.PrimaryColumnIDs))
	}

	expectedColumns := []string{"col-1", "col-2", "col-3"}
	for i, expected := range expectedColumns {
		if dm.PrimaryColumnIDs[i] != expected {
			t.Errorf("expected column %d to be '%s', got '%s'", i, expected, dm.PrimaryColumnIDs[i])
		}
	}
}

func TestDatamine_EmptySchedule(t *testing.T) {
	dm := &Datamine{}

	// With nil schedule, nothing should be set
	parseSchedule(nil, dm)

	if dm.ScheduleType != "" {
		t.Errorf("expected empty schedule type, got '%s'", dm.ScheduleType)
	}
}

func TestDatamine_FullModel(t *testing.T) {
	dm := &Datamine{
		ID:               "dm-uuid-123",
		Name:             "Stale Records Monitor",
		Description:      "Monitors for stale records",
		Active:           true,
		TableID:          "table-uuid-456",
		TableName:        "Sales Data",
		PrimaryColumnIDs: []string{"col-1", "col-2"},
		ScheduleType:     "fixed",
		TimeUnit:         "HOUR",
		Value:            4,
	}

	if dm.ID != "dm-uuid-123" {
		t.Errorf("ID mismatch")
	}
	if dm.Name != "Stale Records Monitor" {
		t.Errorf("Name mismatch")
	}
	if dm.Description != "Monitors for stale records" {
		t.Errorf("Description mismatch")
	}
	if !dm.Active {
		t.Errorf("Expected Active to be true")
	}
	if dm.TableID != "table-uuid-456" {
		t.Errorf("TableID mismatch")
	}
	if dm.TableName != "Sales Data" {
		t.Errorf("TableName mismatch")
	}
	if len(dm.PrimaryColumnIDs) != 2 {
		t.Errorf("Expected 2 primary columns")
	}
	if dm.ScheduleType != "fixed" {
		t.Errorf("ScheduleType mismatch")
	}
	if dm.TimeUnit != "HOUR" {
		t.Errorf("TimeUnit mismatch")
	}
	if dm.Value != 4 {
		t.Errorf("Value mismatch")
	}
}
