#!/bin/bash

# Start MongoDB service in the background
mongod --replSet "RS" --bind_ip_all &

# Wait for MongoDB to start
sleep 10

# Initiate the replica set
mongosh --eval "rs.initiate({_id: 'RS', members: [{ _id: 0, host: 'mongodb:27017' }]})"

# Wait for replica set to initiate
sleep 5

# Create the necessary indexes
mongosh --eval "
db = db.getSiblingDB('staking-api-service');
db.unbonding_queue.createIndex({'staker_pk_hex': 1}, {unique: true});
db.delegations.createIndex({'staker_pk_hex': 1, 'staking_start_height': -1}, {unique: false});
"

# Keep the container running
tail -f /dev/null
