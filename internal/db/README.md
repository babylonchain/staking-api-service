# Logical Sharding in MongoDB

## Overview

**WARNING** If you plan to change the `logical-shard-count` to a higher number, consult the team first!

Logical sharding distributes data across multiple logical partitions or shards to enhance performance and scalability by reducing write contention. This method is especially beneficial in environments with high write throughput, where many operations update the same data points concurrently.

## Implementation

Sharding is achieved by appending a shard number to each document's ID, such as `finalityProviderPkHex:shardNumber`. This approach effectively spreads writes across multiple documents, reducing bottlenecks.

### Example
```go
func (db *Database) generateFinalityProviderStatsId(finalityProviderPkHex string) string {
    randomShardNum := uint64(rand.Intn(int(db.cfg.LogicalShardCount)))
    return fmt.Sprintf("%s:%d", finalityProviderPkHex, randomShardNum)
}
```

## Considerations

### Query Complexity

Logical sharding increases query complexity. Operations like FindFinalityProviderStatsByPkHex must now access multiple shards, leading to costlier queries as shard count increases.

### Configuration Sensitivity

Increasing the LogicalShardCount can further complicate queries. Crucially, once increased, the shard count should never be decreased to avoid disrupting the sharding logic and causing data inconsistencies.
