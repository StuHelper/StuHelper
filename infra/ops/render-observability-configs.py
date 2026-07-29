#!/usr/bin/env python3
from __future__ import annotations

import argparse
import json
import os
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[2]
PROM_TEMPLATE = REPO_ROOT / "infra/observability/prometheus/prometheus.yml.tmpl"
GENERATED_OBS_DIR = Path(
    os.environ.get(
        "GENERATED_OBS_DIR",
        REPO_ROOT / "infra/generated/observability",
    )
)
PROM_OUTPUT = GENERATED_OBS_DIR / "prometheus/prometheus.yml"
ALERT_OUTPUT = GENERATED_OBS_DIR / "alertmanager/alertmanager.yml"
PRODUCTION_EXTERNAL_HTTP_TARGETS = (
    "          - https://sso.stuhelper.com/.well-known/openid-configuration"
)


def env(name: str, default: str = "") -> str:
    return os.environ.get(name, default)


def yaml_quote(value: str) -> str:
    return json.dumps(value)


def write_prometheus(mode: str) -> None:
    metrics_user = env("METRICS_USER", "prometheus")
    metrics_password = env("METRICS_PASSWORD")
    if not metrics_password:
        raise SystemExit("METRICS_PASSWORD is required to render Prometheus config")

    rendered = PROM_TEMPLATE.read_text()
    rendered = rendered.replace("__METRICS_USER__", yaml_quote(metrics_user))
    rendered = rendered.replace("__METRICS_PASSWORD__", yaml_quote(metrics_password))
    external_targets = PRODUCTION_EXTERNAL_HTTP_TARGETS if mode == "prod" else ""
    rendered = rendered.replace("__BLACKBOX_PRODUCTION_HTTP_TARGETS__", external_targets)
    PROM_OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    PROM_OUTPUT.write_text(rendered)


def write_alertmanager(mode: str) -> None:
    webhook = env("ALERTMANAGER_WEBHOOK_URL")
    if mode == "prod" and not webhook:
        raise SystemExit("ALERTMANAGER_WEBHOOK_URL is required in production deploy mode")

    if webhook:
        rendered = f"""global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'job', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: webhook
  routes:
    - matchers:
        - severity = critical
      receiver: webhook
    - matchers:
        - severity = warning
      receiver: webhook

receivers:
  - name: webhook
    webhook_configs:
              - url: {yaml_quote(webhook)}
                send_resolved: true
"""
    else:
        rendered = """global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'job', 'severity']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: default
  routes:
    - matchers:
        - severity = critical
      receiver: default
    - matchers:
        - severity = warning
      receiver: default

receivers:
  - name: default
"""

    ALERT_OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    ALERT_OUTPUT.write_text(rendered)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", choices=["dev", "observability", "prod"], required=True)
    args = parser.parse_args()

    write_prometheus(args.mode)
    write_alertmanager(args.mode)
    print(str(PROM_OUTPUT))
    print(str(ALERT_OUTPUT))


if __name__ == "__main__":
    main()
