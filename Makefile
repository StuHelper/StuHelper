.PHONY: help bootstrap-dev-ubuntu2404 dev dev-init dev-up dev-docker-up dev-down dev-reset dev-smoke dev-status dev-logs e2e e2e-web e2e-admin e2e-uni e2e-koishi obs-up obs-down obs-smoke prod-init prod-render prod-deploy prod-rollback deploy prod-down prod-reset prod-smoke prod-identity-smoke prod-open-platform-evidence prod-backup-evidence prod-parity-ingress prod-parity-up prod-parity-down prod-parity-reset prod-parity-smoke prod-parity-datastore-smoke prod-parity-browser-smoke deploy-bundle ansible-bootstrap ansible-deploy-staging ansible-deploy-prod ansible-rollback-staging ansible-rollback-prod check-docs check-infra-contracts check-semgrep-custom

.DEFAULT_GOAL := help

PROD_ENV_FILE := $(CURDIR)/.env.prod.shared
PROD_SECRETS_ENV_FILE := $(CURDIR)/.env.prod.secrets.local
PROD_GENERATED_ENV_FILE := $(CURDIR)/.env.prod.generated
PROD_GENERATED_SECRET_ENV_FILE := $(CURDIR)/.env.prod.generated.secrets
PROD_RUNTIME_ENV := ENV_FILE="$(PROD_ENV_FILE)" SECRETS_ENV_FILE="$(PROD_SECRETS_ENV_FILE)" GENERATED_ENV_FILE="$(PROD_GENERATED_ENV_FILE)" GENERATED_SECRET_ENV_FILE="$(PROD_GENERATED_SECRET_ENV_FILE)"
PLAYWRIGHT_REUSE_SERVER ?= $(if $(CI),0,1)
export PLAYWRIGHT_REUSE_SERVER

help:
	@echo "StuHelper automation entrypoints"
	@echo ""
	@echo "Development:"
	@echo "  make bootstrap-dev-ubuntu2404 - install Ubuntu 24.04 local dev prerequisites"
	@echo "  make dev        - alias for dev-up"
	@echo "  make dev-init   - generate runnable local .env and derived files"
	@echo "  make dev-up     - one-click start local hot-reload dev (Vite + air) with Docker infra"
	@echo "  make dev-docker-up - start full dockerized dev stack (legacy/CI style)"
	@echo "  make dev-down   - stop dev stack"
	@echo "  make dev-reset  - stop dev stack and remove volumes"
	@echo "  make dev-smoke  - run API/Web/Admin smoke checks"
	@echo "  make dev-status - show local dev processes + docker service status"
	@echo "  make dev-logs   - tail local dev process logs"
	@echo "  make e2e-web    - run Web Playwright E2E locally"
	@echo "  make e2e-admin  - run Admin Playwright E2E locally"
	@echo "  make e2e-uni    - run UniAppX H5 Playwright E2E locally"
	@echo "  make e2e-koishi - run Koishi Console Playwright E2E locally"
	@echo "  make e2e        - run Web/Admin/UniAppX H5 Playwright E2E locally"
	@echo ""
	@echo "Observability:"
	@echo "  make obs-up     - render configs and start observability stack"
	@echo "  make obs-down   - stop observability stack"
	@echo "  make obs-smoke  - run observability smoke checks"
	@echo ""
	@echo "Production:"
	@echo "  make prod-init   - generate production shared/secrets env skeletons with required placeholders"
	@echo "  make prod-render - render generated observability configs"
	@echo "  make prod-deploy - pull pinned production images, bootstrap, deploy, and smoke-check prod stack"
	@echo "  make prod-rollback - rollback to the previous successful production tag"
	@echo "  make deploy      - alias for prod-deploy"
	@echo "  make deploy-bundle - create CI/remote deploy bundle tarball"
	@echo "  make prod-down   - stop prod stack"
	@echo "  make prod-reset  - stop prod stack and remove volumes"
	@echo "  make prod-smoke  - run app + observability smoke checks"
	@echo "  make prod-identity-smoke - verify public stuhelper/id/sso identity ingress"
	@echo "  make prod-open-platform-evidence - run token/OpenFGA production evidence smokes"
	@echo "  make prod-backup-evidence - verify local/fetched PostgreSQL backup evidence"
	@echo "  make prod-parity-ingress - install local stuhelper/id/sso host ingress"
	@echo "  make prod-parity-up - build and run local production-equivalent stack"
	@echo "  make prod-parity-down - stop local production-equivalent stack"
	@echo "  make prod-parity-reset - stop local production-equivalent stack and remove volumes"
	@echo "  make prod-parity-smoke - smoke-check local production-equivalent stack"
	@echo "  make prod-parity-datastore-smoke - verify local parity PostgreSQL/Redis isolation"
	@echo "  make prod-parity-browser-smoke - run browser smoke against local production-equivalent Web/Admin"
	@echo ""
	@echo "Ansible:"
	@echo "  make ansible-bootstrap      - bootstrap a remote Ubuntu host from infra/ansible"
	@echo "  make ansible-deploy-staging - deploy the current workspace to staging via Ansible"
	@echo "  make ansible-deploy-prod    - deploy the current workspace to production via Ansible"
	@echo "  make ansible-rollback-staging - rollback staging via Ansible"
	@echo "  make ansible-rollback-prod    - rollback production via Ansible"
	@echo ""
	@echo "Security:"
	@echo "  make check-infra-contracts - run ops shell and Node contract tests"
	@echo "  make check-semgrep-custom - run StuHelper custom Semgrep rules and fixtures"

bootstrap-dev-ubuntu2404:
	sudo bash infra/ops/bootstrap-dev-ubuntu2404.sh

dev-init:
	./infra/ops/init-dev-env.sh

dev: dev-up

dev-up:
	./infra/ops/dev-up.sh

dev-docker-up:
	DEV_UP_MODE=dockerized ./infra/ops/dev-up.sh

dev-down:
	./infra/ops/dev-down.sh

dev-reset:
	REMOVE_VOLUMES=true ./infra/ops/dev-down.sh

dev-smoke:
	./infra/ops/dev-smoke.sh

dev-status:
	./infra/ops/dev-status.sh

dev-logs:
	./infra/ops/dev-logs.sh

e2e-web:
	cd clients && pnpm test:e2e:web

e2e-admin:
	cd clients && pnpm test:e2e:admin

e2e-uni:
	cd clients && pnpm test:e2e:uni

e2e-koishi:
	cd bots/koishi && corepack yarn test:ui

e2e:
	cd clients && pnpm test:e2e

obs-up:
	./infra/ops/observability-up.sh

obs-down:
	./infra/ops/observability-down.sh

obs-smoke:
	./infra/ops/observability-smoke-check.sh

prod-init:
	$(PROD_RUNTIME_ENV) ./infra/ops/init-prod-env.sh

prod-render:
	$(PROD_RUNTIME_ENV) ./infra/ops/render-observability.sh prod

deploy-bundle:
	./infra/ops/build-deploy-bundle.sh

deploy: prod-deploy

prod-deploy:
	$(PROD_RUNTIME_ENV) ./infra/ops/prod-deploy.sh

prod-rollback:
	$(PROD_RUNTIME_ENV) ./infra/ops/prod-rollback.sh

prod-down:
	$(PROD_RUNTIME_ENV) ./infra/ops/prod-down.sh

prod-reset:
	$(PROD_RUNTIME_ENV) REMOVE_VOLUMES=true ./infra/ops/prod-down.sh

prod-smoke:
	$(PROD_RUNTIME_ENV) ./infra/ops/smoke-check.sh && \
	$(PROD_RUNTIME_ENV) ./infra/ops/observability-smoke-check.sh

prod-identity-smoke:
	$(PROD_RUNTIME_ENV) ./infra/ops/identity-public-smoke.sh

prod-open-platform-evidence:
	$(PROD_RUNTIME_ENV) ./infra/ops/open-platform-production-evidence.sh

prod-backup-evidence:
	$(PROD_RUNTIME_ENV) ./infra/ops/postgres-backup-evidence.sh

prod-parity-ingress:
	./infra/ops/install-local-prod-parity-ingress.sh

prod-parity-up:
	./infra/ops/prod-parity-up.sh

prod-parity-down:
	./infra/ops/prod-parity-down.sh

prod-parity-reset:
	REMOVE_VOLUMES=true ./infra/ops/prod-parity-down.sh

prod-parity-smoke:
	./infra/ops/prod-parity-smoke.sh

prod-parity-datastore-smoke:
	./infra/ops/prod-parity-datastore-smoke.sh

prod-parity-browser-smoke:
	./infra/ops/prod-parity-browser-smoke.sh

ansible-bootstrap:
	cd infra/ansible && ansible-playbook -i inventory/production.ini playbooks/bootstrap.yml

ansible-deploy-staging:
	cd infra/ansible && ansible-playbook -i inventory/staging.ini playbooks/deploy.yml -e env_name=staging

ansible-deploy-prod:
	cd infra/ansible && ansible-playbook -i inventory/production.ini playbooks/deploy.yml -e env_name=production

ansible-rollback-staging:
	cd infra/ansible && ansible-playbook -i inventory/staging.ini playbooks/rollback.yml -e env_name=staging

ansible-rollback-prod:
	cd infra/ansible && ansible-playbook -i inventory/production.ini playbooks/rollback.yml -e env_name=production

check-docs:
	node --test scripts/check-docs-hygiene.test.mjs
	bash scripts/check-docs-hygiene.sh

check-infra-contracts:
	bash infra/ops/tests/run-infra-contracts.sh

check-semgrep-custom:
	bash scripts/check-semgrep-custom-rules.sh
