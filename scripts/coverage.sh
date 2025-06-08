#!/bin/bash
echo "Generating test coverage report..."
go test -coverprofile=coverage.out ./...
echo ""
echo "=== TEST COVERAGE SUMMARY ==="
echo ""
echo "Package Coverage:"
go tool cover -func=coverage.out | grep -E "^github.com" | awk '{print $1 " " $3}' | sed 's/github.com\/oetiker\/go-acme-dns-manager\///g' | column -t
echo ""
echo "Overall Coverage:"
go tool cover -func=coverage.out | tail -1
echo ""
echo "HTML report generated: coverage.html"
go tool cover -html=coverage.out -o coverage.html
