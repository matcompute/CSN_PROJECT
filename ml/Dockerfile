FROM python:3.10-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends libgomp1 curl && rm -rf /var/lib/apt/lists/*
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt uvicorn fastapi prometheus-client pydantic
COPY ml/ ml/
COPY models/ models/
ENV CSN_MODELS_DIR=/app/models
EXPOSE 8000
CMD ["uvicorn","ml.serve_predictor:app","--host","0.0.0.0","--port","8000"]
