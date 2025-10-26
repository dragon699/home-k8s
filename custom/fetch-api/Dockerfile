FROM python:3.13-slim

WORKDIR /app

COPY ./src ./requirements.txt /app/

RUN pip install --no-cache-dir -r requirements.txt

CMD ["python", "fetch-api.py"]
