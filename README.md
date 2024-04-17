# Staking API Service

The `staking-api-service` is a critical component of Babylon Mainnet's Phase 1, focused on managing data for the Frontend Staking Dashboard and handling the unbonding path.

## Architectural Design

![Staking Service Architecture](images/architectural-design.jpg)

This architecture is centered around a message-driven approach, utilizing RabbitMQ queues for inter-service communication. Such a design facilitates high concurrency and enhances fault tolerance by allowing for horizontal scaling and leveraging RabbitMQ's message retry features. The primary infrastructures involved include:

1. MongoDB
2. RabbitMQ
3. Redis cache (Work In Progress)

### Key Features

- **Asynchronous Communication**: Enables decoupled, non-blocking inter-service interactions, aside from the unbonding pipeline which follows a different interaction pattern.
- **Fault Tolerance**: Utilizes RabbitMQ's message retry mechanism for resilience against transient failures.
- **Horizontal Scalability**: Supports increasing system capacity by adding more processing nodes as demand grows.

For more detailed rules applied to message processing, refer to the [queue handler documentation](internal/queue/handlers/REAME.md).

### Workflow

#### Standard Staking Path

The standard staking path encompasses user-initiated staking through CLI/UI, followed by waiting for the staking period (timelock) to expire.

1. **Transaction Submission**: User-submitted staking transactions are confirmed and picked up by the [indexer](https://github.com/babylonchain/staking-indexer) after approximately 100 blocks.
2. **Event Queuing**: The indexer sends `ActiveStakingEvent` [messages](https://github.com/babylonchain/staking-queue-client/blob/main/client/schema.go#L24) to the Active Event Queue for the staking API service to process.
3. **Processing and State Management**: The staking API service executes statistical calculations, data transformation, and staking state management, inserting records into the `timelock_queue` collection.
4. **Timelock Expiry Monitoring**: A dedicated service monitors the `timelock_queue` for records with expired timelocks, signaling the staking API service to update the staking delegation status to 'unbonded', allowing users to withdraw their staked BTC.

#### Early Unbonding Path

The early unbonding path allows users to withdraw their staked BTC before the timelock expiry.

1. **Signature Submission**: Via the UI/CLI, users can initiate an early unbonding action, which involves signing and submitting a signature to the staking API service.
2. **Signature Verification and Storage**: The staking API service validates the signature and stores it for further processing by the unbonding pipeline.
3. **Committee Co-Signing**: The unbonding pipeline collects additional signatures from the covenant committee and submits the unbonding transaction to the Bitcoin network.
4. **Transaction Detection**: The unbonding transaction is eventually detected by the staking-indexer, which places a corresponding [unbonding event](https://github.com/babylonchain/staking-queue-client/blob/main/client/schema.go#L70) into the RabbitMQ queue.
5. **Processing and State Management**: Similar to the standard path, the staking API service handles statistical updates, adjusts the staking state, and inserts a record into the `timelock_queue`.
6. **Finalization**: The expire-service processes items from the `timelock_queue` in MongoDB and emits an [expired event](https://github.com/babylonchain/staking-queue-client/blob/main/client/schema.go#L130) to RabbitMQ for the staking-api-service to process. This marks completed staking transactions as 'unbonded', allowing users to withdraw their funds.

## Getting Started

### Prerequisites

- Docker
- Go

### Installation

1. Clone the repository:

```bash
git clone git@github.com:babylonchain/staking-api-service.git
```

2. Run the service:

```
make run-local
```

OR, you can run as a docker container

```
make start-staking-api-service
```

3. Open your browser and navigate to `http://localhost` to see the api server running.


### Tests

The service only contains integration tests so far, run below:

```
make tests
```

### Update Mocks
1. Make sure the interfaces such as the `DBClient`is up to date
2. Install `mockery`: https://vektra.github.io/mockery/latest/
3. Run `make generate-mock-interface`

## Contribution

Feel free to submit a pull request or open an issue.