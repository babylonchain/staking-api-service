# Staking API Service

The Staking API Service is a crucial component designed to support the Babylon Staking UI and CLI,


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