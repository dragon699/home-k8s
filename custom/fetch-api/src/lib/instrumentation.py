import atexit
from os import getenv

from opentelemetry import trace
from opentelemetry.trace.status import Status, StatusCode
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter as OTLPGrpcSpanExporter

from opentelemetry.instrumentation.flask import FlaskInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor


otel_meta = {
    'service_name': getenv('OTEL_SERVICE_NAME', 'fetch-api'),
    'service_namespace': getenv('OTEL_SERVICE_NAMESPACE', 'fetch-api'),
    'service_version': getenv('OTEL_SERVICE_VERSION', '0.2.5'),
    'otlp_endpoint': getenv('OTLP_ENDPOINT', 'grafana-alloy.monitoring.svc:4317')
}


class FetchAPIInstrumentation:
    def __init__(self, **otel_meta):
        self.resource = Resource.create({
            'service.name': otel_meta['service_name'],
            'service.namespace': otel_meta['service_namespace'],
            'service.version': otel_meta['service_version']
        })

        self.span_exporter = OTLPGrpcSpanExporter(
            endpoint = otel_meta['otlp_endpoint'],
            insecure = True
        )

        self.provider = TracerProvider(resource=self.resource)
        self.provider.add_span_processor(
            BatchSpanProcessor(self.span_exporter)
        )

        trace.set_tracer_provider(self.provider)
        self.tracer = trace.get_tracer(otel_meta['service_name'])

        atexit.register(lambda: self.provider.shutdown())


    def instrument_requests(self, app):
        FlaskInstrumentor().instrument_app(app)
        RequestsInstrumentor().instrument()


    def get_tracer(self):
        return self.tracer
