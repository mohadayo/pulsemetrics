import logging
import os
import time
import uuid
from flask import Flask, request, jsonify
from pydantic import BaseModel, ValidationError, field_validator
from typing import Optional

logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO"),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("ingestor")

app = Flask(__name__)


class MetricPayload(BaseModel):
    name: str
    value: float
    tags: Optional[dict] = None
    timestamp: Optional[float] = None

    @field_validator("name")
    @classmethod
    def name_not_empty(cls, v: str) -> str:
        if not v.strip():
            raise ValueError("name must not be empty")
        return v.strip()


metrics_store: list[dict] = []


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "ok", "service": "ingestor"}), 200


@app.route("/metrics", methods=["POST"])
def ingest_metric():
    body = request.get_json(silent=True)
    if body is None:
        logger.warning("Received request with invalid JSON body")
        return jsonify({"error": "Invalid JSON body"}), 400

    try:
        payload = MetricPayload(**body)
    except ValidationError as e:
        logger.warning("Validation failed: %s", e.error_count())
        errors = [
            {"field": err["loc"], "message": err["msg"]} for err in e.errors()
        ]
        return jsonify({"error": "Validation failed", "details": errors}), 422

    metric = {
        "id": str(uuid.uuid4()),
        "name": payload.name,
        "value": payload.value,
        "tags": payload.tags or {},
        "timestamp": payload.timestamp or time.time(),
    }
    metrics_store.append(metric)
    logger.info("Ingested metric: %s = %s", metric["name"], metric["value"])
    return jsonify({"id": metric["id"], "status": "accepted"}), 201


@app.route("/metrics", methods=["GET"])
def list_metrics():
    name_filter = request.args.get("name")
    if name_filter:
        filtered = [m for m in metrics_store if m["name"] == name_filter]
        return jsonify(filtered), 200
    return jsonify(metrics_store), 200


def create_app():
    return app


if __name__ == "__main__":
    port = int(os.getenv("INGESTOR_PORT", "8001"))
    logger.info("Starting ingestor on port %d", port)
    app.run(host="0.0.0.0", port=port)
