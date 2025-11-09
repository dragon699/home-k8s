from opentelemetry.sdk.trace import SpanProcessor



class HttpSpanProcessor(SpanProcessor):
    def on_start(self, span, parent_context=None):
        if span.name in ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS']:
            url = span.attributes.get('http.url') or span.attributes.get('url.full')

            if url:
                span.update_name(f"{span.name} {url}")
    
    def on_end(self, span):
        pass
    
    def shutdown(self):
        pass
    
    def force_flush(self, timeout_millis=None):
        pass
