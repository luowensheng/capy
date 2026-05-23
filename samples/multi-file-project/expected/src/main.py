"""Generated FastAPI app for todo-api."""
from fastapi import FastAPI
from . import handlers

app = FastAPI(title="todo-api")

@app.get("/health")
async def health_check_endpoint(*args, **kwargs):
    return await handlers.health_check(*args, **kwargs)

@app.get("/todos")
async def list_todos_endpoint(*args, **kwargs):
    return await handlers.list_todos(*args, **kwargs)

@app.post("/todos")
async def create_todo_endpoint(*args, **kwargs):
    return await handlers.create_todo(*args, **kwargs)

@app.get("/todos/{id}")
async def get_todo_endpoint(*args, **kwargs):
    return await handlers.get_todo(*args, **kwargs)

@app.delete("/todos/{id}")
async def delete_todo_endpoint(*args, **kwargs):
    return await handlers.delete_todo(*args, **kwargs)


