import common.utils.requests as req
from langchain_ollama import ChatOllama
from connectors.ml.settings import settings
from common.telemetry.src.tracing.wrappers import traced
from common.telemetry.src.tracing.helpers import reword



class OllamaClient:
    def __init__(self):
        self.url = settings.url


    @staticmethod
    @traced()
    def ping(span=None):
        span.set_attributes({
            'ollama.operation': 'ping',
            'ollama.url': settings.url,
            'ollama.endpoint': settings.health_endpoint
        })

        response = req.get(
            settings.health_endpoint,
            timeout=5
        )

        return response


    @traced()
    def ask(self, prompt: str, model: str, instructions: str='', span=None):
        span.set_attributes(
            reword({
                'ollama.operation': 'ask',
                'ollama.url': self.url,
                'ollama.model': model,
                'ollama.prompt': prompt,
                'ollama.instructions': instructions,
                'ollama.keep_alive': f'{settings.default_keep_alive_minutes}m',
                'ollama.temperature': settings.default_temperature
            })
        )

        model = ChatOllama(
            base_url=self.url,
            model=model,
            keep_alive=f'{settings.default_keep_alive_minutes}m',
            temperature=settings.default_temperature,
            num_ctx=3072,
            num_thread=8
        )

        response = model.invoke(
            [
                {
                    'role': 'system',
                    'content': instructions
                },
                {
                    'role': 'user',
                    'content': prompt
                }
            ]
        )

        return response
