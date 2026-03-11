// Insert a document into the test_collection
db = db.getSiblingDB('init_test');
db.test_collection.insertOne({ name: 'init_test_doc', source: 'js_init_script', timestamp: new Date() });
