from pydantic import BaseModel



class GrafanaRequest(BaseModel):
    ai: bool = False
