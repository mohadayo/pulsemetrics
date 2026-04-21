import pytest
from app import app, metrics_store


@pytest.fixture
def client():
    app.config["TESTING"] = True
    metrics_store.clear()
    with app.test_client() as c:
        yield c


def test_health(client):
    resp = client.get("/health")
    assert resp.status_code == 200
    data = resp.get_json()
    assert data["status"] == "ok"
    assert data["service"] == "ingestor"


def test_ingest_valid_metric(client):
    resp = client.post("/metrics", json={"name": "cpu_usage", "value": 72.5})
    assert resp.status_code == 201
    data = resp.get_json()
    assert data["status"] == "accepted"
    assert "id" in data


def test_ingest_with_tags_and_timestamp(client):
    payload = {
        "name": "memory",
        "value": 1024.0,
        "tags": {"host": "srv-1"},
        "timestamp": 1700000000.0,
    }
    resp = client.post("/metrics", json=payload)
    assert resp.status_code == 201


def test_ingest_invalid_json(client):
    resp = client.post("/metrics", data="not-json", content_type="application/json")
    assert resp.status_code == 400


def test_ingest_empty_name(client):
    resp = client.post("/metrics", json={"name": "  ", "value": 1.0})
    assert resp.status_code == 422


def test_ingest_missing_value(client):
    resp = client.post("/metrics", json={"name": "cpu"})
    assert resp.status_code == 422


def test_list_metrics(client):
    client.post("/metrics", json={"name": "a", "value": 1.0})
    client.post("/metrics", json={"name": "b", "value": 2.0})
    resp = client.get("/metrics")
    assert resp.status_code == 200
    assert len(resp.get_json()) == 2


def test_list_metrics_filter(client):
    client.post("/metrics", json={"name": "a", "value": 1.0})
    client.post("/metrics", json={"name": "b", "value": 2.0})
    resp = client.get("/metrics?name=a")
    assert resp.status_code == 200
    data = resp.get_json()
    assert len(data) == 1
    assert data[0]["name"] == "a"
