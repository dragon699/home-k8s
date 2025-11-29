FROM python:3.13-slim

ARG VAULT_VERSION=1.17.1

RUN apt-get update \
  && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    zip \
    unzip \
    postgresql-client \
  && curl -fsSLo /tmp/vault.zip https://releases.hashicorp.com/vault/${VAULT_VERSION}/vault_${VAULT_VERSION}_linux_amd64.zip \
  && unzip /tmp/vault.zip -d /usr/local/bin \
  && rm -rf /tmp/vault.zip /var/lib/apt/lists/*

RUN pip install --no-cache-dir google-cloud-storage
