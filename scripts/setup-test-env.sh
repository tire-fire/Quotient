#!/bin/bash
#
# setup-test-env.sh - Sets up a complete test environment
#
# This script configures PostgreSQL and Redis for integration testing
#

set -e

echo "🔧 Setting up test environment..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. Configure PostgreSQL
echo -e "${YELLOW}Configuring PostgreSQL...${NC}"

# Disable SSL for testing
if ! grep -q "ssl = off" /etc/postgresql/16/main/postgresql.conf; then
    echo "ssl = off" >> /etc/postgresql/16/main/postgresql.conf
fi

# Allow trust authentication for postgres user
sed -i 's/local   all             postgres                                peer/local   all             postgres                                trust/' /etc/postgresql/16/main/pg_hba.conf

# Start PostgreSQL
service postgresql start || service postgresql restart
sleep 2

# Create test database and user
su - postgres -c "psql -c \"SELECT 1 FROM pg_database WHERE datname = 'quotient_test'\" | grep -q 1 || psql -c 'CREATE DATABASE quotient_test;'" 2>/dev/null

su - postgres -c "psql -c \"SELECT 1 FROM pg_roles WHERE rolname = 'quotient_test'\" | grep -q 1 || psql -c \\\"CREATE USER quotient_test WITH PASSWORD 'test123';\\\"" 2>/dev/null

su - postgres -c "psql -c 'GRANT ALL PRIVILEGES ON DATABASE quotient_test TO quotient_test;'" 2>/dev/null

# Grant schema permissions
su - postgres -c "psql -d quotient_test -c 'GRANT ALL ON SCHEMA public TO quotient_test;'" 2>/dev/null

# Add authentication for test user
if ! grep -q "quotient_test" /etc/postgresql/16/main/pg_hba.conf; then
    echo "local   quotient_test   quotient_test                           md5" >> /etc/postgresql/16/main/pg_hba.conf
    service postgresql reload
fi

echo -e "${GREEN}✓ PostgreSQL configured${NC}"

# 2. Start Redis
echo -e "${YELLOW}Starting Redis...${NC}"
redis-server --daemonize yes --port 6379 --loglevel warning 2>/dev/null || true
sleep 1

if redis-cli ping > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Redis started${NC}"
else
    echo -e "${YELLOW}⚠ Redis not responding${NC}"
fi

# 3. Test connections
echo -e "${YELLOW}Testing connections...${NC}"

# Test PostgreSQL
export PGPASSWORD=test123
if psql -U quotient_test -d quotient_test -h localhost -c "SELECT 1;" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PostgreSQL connection works${NC}"
else
    echo -e "${YELLOW}⚠ PostgreSQL connection failed${NC}"
fi

# Test Redis
if redis-cli ping > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Redis connection works${NC}"
else
    echo -e "${YELLOW}⚠ Redis connection failed${NC}"
fi

echo ""
echo -e "${GREEN}🎉 Test environment ready!${NC}"
echo ""
echo "Connection details:"
echo "  PostgreSQL: host=localhost user=quotient_test password=test123 dbname=quotient_test port=5432"
echo "  Redis: localhost:6379"
echo ""
echo "Run tests with:"
echo "  go test ./tests/integration/..."
echo "  go test ./tests/chaos/..."
echo "  go test ./... (all tests)"
