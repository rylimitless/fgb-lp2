from fastembed import TextEmbedding
from flask import Flask, jsonify, request

app = Flask(__name__)

# ONNX-backed, ~150-250MB RSS. Loaded once at import and reused across requests.
model = TextEmbedding("sentence-transformers/all-MiniLM-L6-v2")  # 384-dim

# Warm up so the model is resident before we accept traffic. Because this runs
# before app.run(), the healthcheck only turns green once the model is ready.
list(model.embed(["warmup"]))


@app.post("/embeddings")
def embeddings():
    data = request.get_json(silent=True) or {}
    inputs = data.get("input")

    if isinstance(inputs, str):
        inputs = [inputs]
    if not inputs or not isinstance(inputs, list):
        return jsonify(
            {"error": "input must be a non-empty string or list of strings"}
        ), 400

    vectors = [v.tolist() for v in model.embed(inputs)]

    return jsonify(
        {
            "data": [{"embedding": v, "index": i} for i, v in enumerate(vectors)],
            "dim": len(vectors[0]) if vectors else 0,
        }
    )


@app.get("/health")
def health():
    return jsonify({"status": "ok", "model": "all-MiniLM-L6-v2", "dim": 384})


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=7100)
