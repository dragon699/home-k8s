import logging
from pythonjsonlogger.jsonlogger import JsonFormatter


class Formatters:
    class Logfmt(logging.Formatter):
        def format(self, record):
            attr = [
                f'time="{self.formatTime(record, self.datefmt)}"',
                f'level={record.level}',
                f'component={record.component}',
                f'msg="{record.getMessage()}"'
            ]

            if hasattr(record, '__dict__'):
                for key, value in record.__dict__.items():
                    if key in ['component', 'msg', 'args', 'created', 'filename', 'funcName', 
                              'level', 'levelno', 'lineno', 'module', 'msecs', 
                              'message', 'pathname', 'process', 'processName', 'relativeCreated',
                              'thread', 'threadName', 'exc_info', 'exc_text', 'stack_info',
                              'asctime', 'taskName']:
                        continue

                    if isinstance(value, str):
                        attr += [(f'{key}="{value}"')]

                    elif value is None:
                        attr += [f'{key}=null']

                    elif isinstance(value, bool):
                        attr += [f'{key}={str(value).lower()}']

                    else:
                        attr += [f'{key}={value}']

            return ' '.join(attr)


    class Json(JsonFormatter):
        def add_fields(self, log_record, record, message_dict):
            super().add_fields(log_record, record, message_dict)

            if 'msg' in log_record:
                log_record['msg'] = record.getMessage()
