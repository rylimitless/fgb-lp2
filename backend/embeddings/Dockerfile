FROM python:3.11-slim

# onnxruntime (used by fastembed) needs libgomp at runtime; slim doesn't ship it.
RUN apt-get update && apt-get install -y --no-install-recommends libgomp1 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Pre-fetch the quantized ONNX model so the container starts fast and runs offline.
RUN python -c "from fastembed import TextEmbedding; list(TextEmbedding('sentence-transformers/all-MiniLM-L6-v2').embed(['warmup']))"

COPY app.py .

EXPOSE 7100

CMD ["python", "app.py"]
