FROM python:3.13-slim AS builder

WORKDIR /app

COPY requirements.txt .

RUN pip install --prefix=/install --no-cache-dir -r requirements.txt



FROM python:3.13-slim

WORKDIR /app

COPY --from=builder /install /usr/local

COPY app.py .

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

USER 65532:65532

CMD ["python", "app.py"]
