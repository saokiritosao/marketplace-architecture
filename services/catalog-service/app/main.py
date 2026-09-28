from fastapi import FastAPI

app = FastAPI(title="Catalog Service", version="0.1.0")


@app.get("/health", status_code=200)
def health():
    return {"status": "ok"}
