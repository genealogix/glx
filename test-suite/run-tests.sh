#!/bin/bash

set -e

VALIDATOR=${1:-glx}
PASSED=0
FAILED=0

echo "🧪 Running GENEALOGIX Conformance Tests"
echo "Using validator: $VALIDATOR"
echo ""

# Test valid files
echo "Testing valid files..."
for file in valid/*.glx; do
    if [ ! -e "$file" ]; then continue; fi
    if $VALIDATOR validate "$file" > /dev/null 2>&1; then
        echo "✓ $file"
        ((PASSED++))
    else
        echo "✗ $file (should be valid)"
        ((FAILED++))
    fi
done

# Test invalid files
echo ""
echo "Testing invalid files..."
for file in invalid/*.glx; do
    if [ ! -e "$file" ]; then continue; fi
    if $VALIDATOR validate "$file" > /dev/null 2>&1; then
        echo "✗ $file (should be invalid)"
        ((FAILED++))
    else
        echo "✓ $file (correctly rejected)"
        ((PASSED++))
    fi
done

echo ""
echo "Results: $PASSED passed, $FAILED failed"

if [ $FAILED -eq 0 ]; then
    echo "✅ All tests passed!"
    exit 0
else
    echo "❌ Some tests failed"
    exit 1
fi


