#!/bin/bash
# Create a test collection in the init_test database
mongosh --quiet <<'JSEOF'
db = db.getSiblingDB('init_test');
db.createCollection('test_collection');
JSEOF
