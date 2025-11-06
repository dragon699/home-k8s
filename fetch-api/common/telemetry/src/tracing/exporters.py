from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.trace import Status, StatusCode


class StatusSettingSpanExporter(OTLPSpanExporter):
    def export(self, spans):
        for span in spans:
            if span.status.status_code == StatusCode.UNSET:
                status_code = span.attributes.get('http.status_code')
                
                if status_code:
                    if 200 <= status_code < 400:
                        span._status = Status(StatusCode.OK)

                    elif status_code >= 400:
                        span._status = Status(StatusCode.ERROR)
        
        return super().export(spans)
