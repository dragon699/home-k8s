from opentelemetry import trace
from functools import wraps
from os import getenv


tracer = trace.get_tracer(getenv("OTEL_SERVICE_NAME", __name__))


def traced(span_name=None, attributes=None, inject_span=True):
    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            if span_name:
                name = span_name

            else:
                if args and hasattr(args[0].__class__, func.__name__):
                    name = f"{args[0].__class__.__name__}.{func.__name__}"

                else:
                    name = f"{func.__module__}.{func.__name__}"

            with tracer.start_as_current_span(name) as span:
                span.set_attribute("func.name", func.__name__)
                span.set_attribute("func.module", func.__module__)

                if attributes:
                    for key, value in attributes.items():
                        span.set_attribute(key, value)

                try:
                    if inject_span:
                        kwargs['span'] = span

                    result = func(*args, **kwargs)

                    return result

                except Exception as e:
                    span.set_status(trace.Status(trace.StatusCode.ERROR))
                    span.record_exception(e)

                    raise

        return wrapper
    return decorator
