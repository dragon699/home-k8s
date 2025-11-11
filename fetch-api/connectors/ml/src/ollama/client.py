from langchain_ollama import ChatOllama
from connectors.ml.settings import settings
from common.utils.web import create_session
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

        session = create_session(timeout=5)
        response = session.get(settings.health_endpoint)

        return response


    @traced()
    def ask(self, prompt: str, model: str, span=None):
        span.set_attributes(
            reword({
                'ollama.operation': 'ask',
                'ollama.url': self.url,
                'ollama.model': model,
                'ollama.prompt': prompt,
                'ollama.keep_alive': f'{settings.default_keep_alive_minutes}m'
            })
        )

        model = ChatOllama(
            base_url=self.url,
            model=model,
            keep_alive=f'{settings.default_keep_alive_minutes}m'
        )

        response = model.invoke(prompt)

        return response
