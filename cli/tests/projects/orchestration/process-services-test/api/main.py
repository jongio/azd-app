from fastapi import FastAPI

app = FastAPI()

@app.get("/health")
def health():
    return {"status": "healthy", "service": "api"}

@app.get("/")
def root():
    return {"message": "Python API Service"}
