"""Smoke tests — just confirms every route is mounted."""
from fastapi.testclient import TestClient
from src.main import app

client = TestClient(app)

def test_health_check_is_mounted():
    # Method allowed; bodies raise NotImplementedError until you fill them in.
    response = client.get("/health")
    assert response.status_code != 404

def test_list_todos_is_mounted():
    # Method allowed; bodies raise NotImplementedError until you fill them in.
    response = client.get("/todos")
    assert response.status_code != 404

def test_create_todo_is_mounted():
    # Method allowed; bodies raise NotImplementedError until you fill them in.
    response = client.post("/todos")
    assert response.status_code != 404

def test_get_todo_is_mounted():
    # Method allowed; bodies raise NotImplementedError until you fill them in.
    response = client.get("/todos/{id}")
    assert response.status_code != 404

def test_delete_todo_is_mounted():
    # Method allowed; bodies raise NotImplementedError until you fill them in.
    response = client.delete("/todos/{id}")
    assert response.status_code != 404


