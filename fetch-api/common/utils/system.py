import os, json, yaml, jinja2



def read_file(path: str, type='json'):
    if not os.path.exists(path):
        return None

    with open(path, 'r') as file:
        if type == 'json':
            return json.load(file)

        elif type in ('yaml', 'yml'):
            return yaml.safe_load(file)

        else:
            return file.read()


def render_template(content, vars={}):
    template = jinja2.Template(
        content,
        undefined=jinja2.StrictUndefined
    )

    return template.render(vars)
