from common.telemetry.src.tracing.wrappers import traced



class Processor:
    @staticmethod
    @traced()
    def process(response, span=None):
        result = []

        answer = response.content.rstrip('"').lstrip('"')
        if answer.startswith(' '):
            answer = answer[1:]

        result += [{
            'answer': answer,
            'duration_seconds': round((int(response.response_metadata['total_duration']) / 1_000_000_000), 1),
            'provider': response.response_metadata['model_provider'],
            'model': response.response_metadata['model_name']
        }]

        return {
            'total_items': len(result),
            'items': result
        }
