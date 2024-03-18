DOCKER = $(shell which docker)

babylon-api:
	$(DOCKER) build --secret id=sshKey,src=${BBN_PRIV_DEPLOY_KEY} --tag babylonchain/babylon-api -f api-server/Dockerfile \
				$(shell git rev-parse --show-toplevel)

babylon-api-rmi:
	$(DOCKER) rmi babylonchain/babylon-api 2>/dev/null; true
