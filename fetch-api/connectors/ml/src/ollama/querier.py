import os, copy
from common.utils.system import read_file, render_template
from common.utils.helpers import time_now
from common.telemetry.src.tracing.wrappers import traced
from common.telemetry.src.tracing.helpers import reword
from connectors.ml.settings import settings
from connectors.ml.src.telemetry.logging import log
from connectors.ml.src.api import ollama_client
from connectors.ml.src.ollama.query_processor import Processor



class Querier:
    def __init__(self, client):
        self.instructions_template_path = settings.instructions_template_path
        
        self.client = client
        self.instructions = {}

        self.load_instructions()


    def load_instructions(self):
        if not os.path.exists(self.instructions_template_path):
            log.critical(f'Instructions template file does not exist', extra={
                'instructions_file': self.instructions_template_path
            })
            raise SystemExit(1)

        try:
            instructions = read_file(self.instructions_template_path, type='yaml')

        except Exception as e:
            log.critical(f'{self.instructions_template_path} is not valid YAML file', extra={
                'instructions_file': self.instructions_template_path,
                'error': str(e)
            })
            raise SystemExit(1)

        self.instructions = instructions

    
    @traced()
    def commit(self, prompt: str, model: str=None, instructions: str='', instructions_template: str=None, span=None):
        try:
            full_instructions = self.fetch(instructions, instructions_template)
            payload = self.render(prompt, model, full_instructions)
            response = self.send(*payload.values())
            result = self.process(response)

            span.set_attributes(
                reword({
                    'querier.query.status': 'successful',
                    'querier.query.prompt': payload['prompt'],
                    'querier.query.model': payload['model'],
                    'querier.query.instructions': payload['instructions'],
                    'querier.query.response': response.content
                })
            )

            return result

        except Exception as err:
            span.set_attributes({
                'querier.query.status': 'failed',
                'querier.error.message': str(err),
                'querier.error.type': type(err).__name__
            })


    @traced()
    def fetch(self, instructions: str, instructions_template: str, span=None):
        if instructions_template:
            span.set_attributes({
                'querier.instructions.template.path': self.instructions_template_path,
                'querier.instructions.template.key': instructions_template
            })

            if not instructions_template in self.instructions:
                log.warning(f'{instructions_template} is not a valid instructions template key')
                span.set_attributes({
                    'querier.instructions.template.key.exists': False
                })

            else:
                templated_instructions = render_template(
                    content=self.instructions[instructions_template],
                    vars={
                        'current_time': time_now()
                    }
                )
                span.set_attributes({
                    'querier.instructions.template.key.exists': True,
                    'querier.query.instructions': templated_instructions
                })
                return templated_instructions

        if instructions:
            span.set_attributes({
                'querier.query.instructions': instructions
            })
            return instructions

        return ''


    @traced()
    def render(self, prompt: str, model: str, instructions: str, span=None):
        payload = {
            'prompt': prompt,
            'model': model or settings.default_model,
            'instructions': instructions or ''
        }

        span.set_attributes({
            'querier.query.prompt': prompt,
            'querier.query.model': payload['model'],
            'querier.query.instructions': instructions
        })

        return payload


    @traced()
    def send(self, prompt: str, model: str=None, instructions: str='', span=None):
        try:
            response = self.client.ask(
                prompt=prompt,
                model=model,
                instructions=instructions
            )

            span.set_attributes(
                reword({
                    'querier.query.prompt': prompt,
                    'querier.query.model': model,
                    'querier.query.instructions': instructions,
                    'querier.query.response': response.content
                })
            )

            return response

        except Exception as err:
            span.set_attributes(
                reword({
                    'querier.error.message': f'Error occurred while sending query: {err}',
                    'querier.error.type': type(err).__name__,
                    'querier.query.prompt': prompt,
                    'querier.query.model': model,
                    'querier.query.instructions': instructions
                })
            )
            return None
        

    @traced()
    def process(self, response: str, span=None):
        return Processor.process(response)


querier = Querier(ollama_client)
