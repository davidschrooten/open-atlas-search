#!/nix/store/lb33m49aslmvkx5l4xrkiy7m6nbh2kqf-bash-interactive-5.3p0/bin/bash

# Comprehensive test script for all Open Atlas Search clients

set -e

echo "🧪 Testing Open Atlas Search Clients"
echo "===================================="
echo

# Test TypeScript Client
echo "1️⃣ Testing TypeScript Client (oas-ts-client)"
echo "--------------------------------------------"

cd ts

echo "📦 Installing dependencies..."
if [ ! -d "node_modules" ]; then
    npm install > /dev/null 2>&1
fi

echo "🔨 Building TypeScript client..."
npm run build > /dev/null 2>&1

echo "🧪 Running tests..."
npm test

echo "✅ TypeScript client tests completed successfully!"
echo

# Test NestJS Client
echo "2️⃣ Testing NestJS Client (oas-ts-nestjs-client)"
echo "-----------------------------------------------"

cd ../ts-nestjs

echo "📦 Installing dependencies..."
if [ ! -d "node_modules" ]; then
    npm install > /dev/null 2>&1
fi

echo "🔨 Building NestJS client..."
npm run build > /dev/null 2>&1

echo "🧪 Running tests..."
npm test

echo "📊 Running test coverage..."
npm run test:coverage > /dev/null 2>&1

echo "🔍 Running validation script..."
node validate-client.js

echo "✅ NestJS client tests completed successfully!"
echo

# Summary
echo "📋 SUMMARY"
echo "=========="
echo "✅ TypeScript Client (oas-ts-client)"
echo "   - ✓ Dependencies installed"
echo "   - ✓ Build successful"
echo "   - ✓ All tests passing"
echo "   - ✓ Ready for use"
echo
echo "✅ NestJS Client (oas-ts-nestjs-client)"
echo "   - ✓ Dependencies installed" 
echo "   - ✓ Build successful"
echo "   - ✓ All tests passing"
echo "   - ✓ Good test coverage"
echo "   - ✓ Module validation passed"
echo "   - ✓ Ready for use"
echo
echo "🎉 All clients are working correctly!"
echo
echo "📚 Usage Information:"
echo "--------------------"
echo "TypeScript Client: Located in clients/ts/"
echo "• Package name: oas-ts-client"
echo "• Import: import { OpenAtlasSearchClient } from 'oas-ts-client'"
echo
echo "NestJS Client: Located in clients/ts-nestjs/"
echo "• Package name: oas-ts-nestjs-client"
echo "• Import: import { OpenAtlasSearchModule } from 'oas-ts-nestjs-client'"
echo "• Includes full NestJS integration with DI support"
echo
echo "✅ Both clients are production-ready!"
