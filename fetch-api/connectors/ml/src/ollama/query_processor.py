from common.telemetry.src.tracing.wrappers import traced



class Processor:
    @staticmethod
    @traced()
    def process(query_response: str, span=None):
        result = []

        query_response = query_response.rstrip('"').lstrip('"')
        if query_response.startswith(' '):
            query_response = query_response[1:]

        result += [{'answer': query_response}]

        return {
            'total_items': len(result),
            'items': result
        }
