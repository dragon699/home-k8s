FROM python:3.13-slim AS builder

WORKDIR /app

COPY common/requirements.txt .

RUN pip install --user --no-cache-dir -r requirements.txt



FROM python:3.13-slim

WORKDIR /app

COPY --from=builder /root/.local /root/.local

COPY connectors/grafana/src /app/connectors/grafana/src
COPY connectors/grafana/templates /app/connectors/grafana/templates
COPY connectors/grafana/*.py /app/connectors/grafana/
COPY common /app/common

ENV PATH=/root/.local/bin:$PATH

CMD ["python", "-m", "fetch_api.app"]
