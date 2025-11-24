from opentelemetry.sdk.trace import SpanProcessor
from urllib.parse import urlparse



class HttpSpanProcessor(SpanProcessor):
    def on_start(self, span, parent_context=None):
        if span.name in ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS']:
            url = span.attributes.get('http.url') or span.attributes.get('url.full')
            target = span.attributes.get('http.target') or span.attributes.get('url.path')

            if target:
                span.update_name(f"{span.name} {target}")

            elif url:
                span.update_name(f"{span.name} {urlparse(url).path}")
    
    def on_end(self, span):
        pass
    
    def shutdown(self):
        pass
    
    def force_flush(self, timeout_millis=None):
        pass
