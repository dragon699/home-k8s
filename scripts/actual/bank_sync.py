import os, json
import requests


class BankSync:
    def __init__(self):
        self.url = os.getenv('ACTUAL_MCP_URL', 'http://actual-mcp.mcp.svc:3000/mcp')
        self.timeout = int(os.getenv('ACTUAL_MCP_TIMEOUT', '300'))
        self.session = requests.Session()
        self.session.headers.update({'Content-Type': 'application/json', 'Accept': 'application/json, text/event-stream'})
        self.request_id = 0


    def log(self, msg: str, warn: bool = False, crash: bool = False):
        if crash:
            print(f'[ X ] {msg}')
            raise SystemExit(1)

        if warn:
            print(f'[ ! ] {msg}')
            return True

        print(f'[ > ] {msg}')


    def parse(self, response: requests.Response) -> dict:
        response.encoding = 'utf-8'
        if 'text/event-stream' in response.headers.get('Content-Type', ''):
            events = [line[6:] for line in response.text.splitlines() if line.startswith('data: ')]
            return json.loads(events[-1]) if events else {}

        return response.json() if response.text else {}


    def rpc(self, method: str, params: dict | None = None, notify: bool = False) -> dict:
        body = {'jsonrpc': '2.0', 'method': method}
        if params is not None:
            body['params'] = params
        if not notify:
            self.request_id += 1
            body['id'] = self.request_id

        try:
            response = self.session.post(self.url, json=body, timeout=self.timeout)
            response.raise_for_status()
        except requests.RequestException as err:
            self.log(f'mcp: {method} failed, got this -> {err}', crash=True)

        if 'mcp-session-id' in response.headers:
            self.session.headers['mcp-session-id'] = response.headers['mcp-session-id']

        if notify:
            return {}

        message = self.parse(response)
        if 'error' in message:
            self.log(f'mcp: {method} returned an error -> {message["error"]}', crash=True)

        return message.get('result', {})


    def call(self, tool: str, arguments: dict | None = None) -> str:
        result = self.rpc('tools/call', {'name': tool, 'arguments': arguments or {}})
        text = '\n'.join(item.get('text', '') for item in result.get('content', []) if item.get('type') == 'text')

        if result.get('isError'):
            self.log(f'{tool}: {text}', crash=True)

        return text


    def run(self):
        server = self.rpc('initialize', {
            'protocolVersion': '2025-03-26',
            'capabilities': {},
            'clientInfo': {'name': 'bank-sync', 'version': '1'},
        }).get('serverInfo', {})
        self.rpc('notifications/initialized', notify=True)
        self.log(f'mcp: Connected to {server.get("name")} {server.get("version")} at {self.url}')

        self.log('bank-sync: Syncing all linked accounts..')
        self.log(f'bank-sync: {self.call("run-bank-sync")}')
        self.log('Have a good day <:')


if __name__ == '__main__':
    BankSync().run()
