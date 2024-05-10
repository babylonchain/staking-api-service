#!/bin/sh

# Maximum number of retries
max_retries=10
current_retry=0

# Execute mongosh command in a loop
while [ $current_retry -lt $max_retries ]; do
  # Execute mongosh command
  kubectl exec mongodb-staging-0 -n $MONGODB_STAGING_NAMESPACE \
  -- mongosh --eval "db.adminCommand('ping')"
  return_code=$?

  # Check the return code of the command
  if [ $return_code -eq 0 ]; then
    # Command executed successfully
    kubectl exec mongodb-staging-0 -n $MONGODB_STAGING_NAMESPACE \
    -- mongosh \
    --eval "rs.reconfig({'_id': 'rs0', 'members': [{'_id': 0, 'host': 'mongodb-staging-0.mongodb-staging-headless.mongodb-staking-api.svc.cluster.local:27017', 'priority': 10}]}, {force: true})"
    break
  else
    # Command execution failed
    echo "Command execution failed. Return code: $return_code. Retrying in 10 second..."
    sleep $MONGODB_HEALTH_CHECK_INTERVAL
    current_retry=$((current_retry + 1))
  fi
done

# Check if maximum retries reached
if [ $current_retry -eq $max_retries ]; then
  echo "Maximum retries reached. Command execution failed."
fi