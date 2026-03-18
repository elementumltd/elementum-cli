#!/bin/bash
# Copyright (c) ElementumAI, Inc.
# SPDX-License-Identifier: MPL-2.0

# Test script for Elementum CLI against a real app

set -e

echo "🧪 Testing Elementum CLI"
echo "======================="
echo ""

# Test app from https://appdemo.elementum.io/admin/apps/dealclosingprocess/flow
APP_NAMESPACE="dealclosingprocess"

echo "1️⃣  Running unit tests..."
go test ./... -v | grep -E "(PASS|FAIL|ok)" | tail -5
echo ""

echo "2️⃣  Building CLI..."
go build -o ~/go/bin/ei .
echo "✅ CLI built"
echo ""

echo "3️⃣  Testing commands (requires authentication)..."
echo ""

echo "📋 Test: List objects"
echo "Command: ei list objects"
echo "Expected: Shows all apps and elements"
echo ""

echo "🔍 Test: Show app details"
echo "Command: ei show app $APP_NAMESPACE"
echo "Expected: Tree view with:"
echo "  - Fields"
echo "  - Layouts"
echo "  - Flows"
echo "  - Agents"
echo "  - Widgets"
echo "  - AI File Readers"
echo "  - Approval Processes"
echo ""

echo "📦 Test: Export app (interactive)"
echo "Command: ei export app $APP_NAMESPACE"
echo "Expected: Interactive selector with all resource types"
echo ""

echo "📦 Test: Export app (all resources)"
echo "Command: ei export app $APP_NAMESPACE --all"
echo "Expected: Exports all resources to Terraform"
echo ""

echo "🔧 Manual Testing Steps:"
echo "========================"
echo ""
echo "1. Authenticate:"
echo "   ei auth login"
echo ""
echo "2. Show app:"
echo "   ei show app $APP_NAMESPACE"
echo ""
echo "3. Export app (all):"
echo "   ei export app $APP_NAMESPACE --all -o test-export.tf"
echo ""
echo "4. Verify Terraform:"
echo "   cd /tmp/elementum-export-*"
echo "   cat generated.tf"
echo "   terraform validate"
echo "   terraform plan"
echo ""
echo "5. Expected resources in export:"
echo "   ✓ elementum_app"
echo "   ✓ elementum_*_field (multiple types)"
echo "   ✓ elementum_layout"
echo "   ✓ elementum_flow"
echo "   ✓ elementum_agent"
echo "   ✓ elementum_agent_tool"
echo "   ✓ elementum_widget"
echo "   ✓ elementum_ai_file_reader"
echo "   ✓ elementum_approval_process"
echo ""
echo "✅ All tests configured!"
