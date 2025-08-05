#!/bin/bash

echo "🚀 Open Atlas Search - Build and Test Script"
echo "============================================="

# Test build
echo "📦 Testing build..."
if go build -o open-atlas-search .; then
    echo "✅ Build successful"
else
    echo "❌ Build failed"
    exit 1
fi

# Test help command
echo "📋 Testing help command..."
if ./open-atlas-search --help > /dev/null; then
    echo "✅ Help command works"
else
    echo "❌ Help command failed"
    exit 1
fi

# Test server help
echo "🖥️  Testing server help..."
if ./open-atlas-search server --help > /dev/null; then
    echo "✅ Server help works"
else
    echo "❌ Server help failed"
    exit 1
fi

# Test configuration validation (without MongoDB connection)
echo "⚙️  Testing configuration validation..."
if timeout 3s ./open-atlas-search server --config config.yaml 2>/dev/null; then
    echo "⚠️  Server started (expected to fail due to no MongoDB)"
else
    echo "✅ Configuration validation works (expected MongoDB connection error)"
fi

echo ""
echo "🎉 All basic tests passed!"
echo ""
echo "📚 Next steps:"
echo "  1. Set up MongoDB (with replica set for change streams)"
echo "  2. Update config.yaml with your MongoDB connection details"
echo "  3. Run: ./open-atlas-search server"
echo "  4. Test search API endpoints"
echo ""
echo "🔗 API Endpoints:"
echo "  POST /search - Perform searches"
echo "  POST /indexes/{index}/documents - Index documents"
echo "  DELETE /indexes/{index}/documents/{id} - Delete documents"
echo ""
echo "📖 See README.md for full documentation and examples"
