import os, uvicorn
from fastapi import FastAPI, Response


app = FastAPI()


@app.get(os.getenv('PUBLIC_KEY_ENDPOINT'))
def get_public_key():
    return Response(
        content=os.getenv('PUBLIC_KEY'),
        media_type='application/x-pem-file'
    )


if __name__ == "__main__":
    uvicorn.run(
        app,
        host='0.0.0.0',
        port=8080
    )
