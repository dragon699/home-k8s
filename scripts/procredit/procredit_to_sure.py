import argparse, csv, json, os, re, sys, urllib.error, urllib.parse, urllib.request, zipfile
import xml.etree.ElementTree as ET
from collections import Counter, defaultdict
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path


HERE = Path(__file__).resolve().parent
NS = {'m': 'http://schemas.openxmlformats.org/spreadsheetml/2006/main'}
BGN_PER_EUR = Decimal('1.95583')
SOURCE = 'procredit'
FIELD_LABELS = r'(НАРЕДИТЕЛ|ПОЛУЧАТЕЛ|СМЕТКА|BIC|ОСНОВАНИЕ|ДОП\.\s*ПОЯСНЕНИЯ)'
CARD_RE = re.compile(
    r'(ПОС плащане|ATM теглене|Card transaction|Original credit\b.*?voucher)\s+([\d.]+)\s+(?:([A-Z]{3})\s+)?(?:(\S*\d\S*)\s+)?(.*?)\s+(\d{4}\*{3,4}\d{3,4})',
    re.IGNORECASE,
)
EMPLOYERS = {'УАЙЗЪР': 'Wiser Technology', 'ПРАЙМ ХОЛДИНГ': 'Prime Holding', 'DEEL': 'Deel'}


def load_config():
    return json.loads((HERE / 'config.json').read_text(encoding='utf-8'))


def load_payees():
    path = HERE / 'payees.json'
    if path.exists():
        return json.loads(path.read_text(encoding='utf-8'))
    return {'exact': {}, 'prefix': {}}


def read_xlsx(path):
    z = zipfile.ZipFile(path)
    shared = []
    if 'xl/sharedStrings.xml' in z.namelist():
        for si in ET.fromstring(z.read('xl/sharedStrings.xml')).findall('m:si', NS):
            shared.append(''.join(t.text or '' for t in si.iter('{%s}t' % NS['m'])))
    sheet = sorted(n for n in z.namelist() if n.startswith('xl/worksheets/sheet'))[0]
    rows = []
    for r in ET.fromstring(z.read(sheet)).find('m:sheetData', NS):
        cells = {}
        for c in r.findall('m:c', NS):
            col = ''.join(ch for ch in c.get('r') if ch.isalpha())
            v = c.find('m:v', NS)
            if c.get('t') == 's' and v is not None:
                cells[col] = shared[int(v.text)]
            elif c.get('t') == 'inlineStr':
                cells[col] = ''.join(t.text or '' for t in c.iter('{%s}t' % NS['m']))
            else:
                cells[col] = v.text if v is not None else ''
        rows.append(cells)
    header_idx = next(i for i, r in enumerate(rows) if 'IBAN' in r.values())
    header, counts = {}, Counter()
    for col, name in rows[header_idx].items():
        name = name.strip()
        counts[name] += 1
        header[col] = name if counts[name] == 1 else f'{name} {counts[name]}'
    out = []
    for r in rows[header_idx + 1:]:
        rec = {header[c]: (v or '').strip() for c, v in r.items() if c in header}
        if re.match(r'BG\d{2}', rec.get('IBAN', '')):
            out.append(rec)
    return out, list(header.values())


def clean_text(s):
    s = (s or '').replace('\xa0', ' ')
    s = re.sub(r'\|(?=' + FIELD_LABELS + ')', '\x00', s)
    s = s.replace('|', '').replace('\x00', ' | ')
    s = s.replace(';', ' | ')
    return re.sub(r'\s+', ' ', s).strip(' |')


def card_merchant(text):
    s = re.sub(r'(\d)\s+(\d{1,3}\*{3,4}\d{3,4})', r'\1\2', clean_text(text))
    m = CARD_RE.search(s)
    return m.group(5).strip() if m else None


def merchant_key(text):
    merchant = card_merchant(text)
    return re.sub(r'[^A-Z0-9]', '', merchant.upper()) or None if merchant else None


def counterparty_key(name):
    return re.sub(r'[^0-9A-ZА-Я]', '', (name or '').upper())


def merchant_prefix(text):
    merchant = card_merchant(text)
    words = re.sub(r'[^A-Z0-9 ]', ' ', (merchant or '').upper()).split()
    return words[0] if words and len(words[0]) >= 4 else None


def parse_rows(path, config):
    raw, headers = read_xlsx(path)
    accounts = config['accounts']
    records, skipped = [], Counter()
    for rec in raw:
        iban = rec['IBAN']
        details = rec.get('ОСНОВАНИЕ', '')
        ref = rec.get('Референция', '')
        upper = details.upper()
        if iban not in accounts:
            skipped['unknown account ' + iban] += 1
            continue
        if not ref:
            skipped['pending or cancelled authorisation (no reference)'] += 1
            continue
        if 'ОВЕРДРАФТ' in upper.split(';')[0] or 'ОВЪРДРАФТ' in upper.split(';')[0]:
            skipped['internal overdraft booking'] += 1
            continue
        d, mth, y = rec['Вальор'].split('.')
        cur = rec.get('Валута') or 'EUR'
        amt = Decimal(rec.get('Сума') or '0')
        second_cur = rec.get('Валута 2', '')
        second_amt = rec.get('Сума 2', '')
        if cur == 'BGN':
            eur = (amt / BGN_PER_EUR).quantize(Decimal('0.01'), ROUND_HALF_UP)
            original = f'{amt:.2f} BGN'
        else:
            eur = amt.quantize(Decimal('0.01'))
            original = f'{Decimal(second_amt):.2f} {second_cur}' if second_cur and second_amt else ''
        outflow = rec.get('Операция (Дт, Кт)', '').startswith('Д')
        records.append({
            'iban': iban,
            'account': accounts[iban],
            'ref': ref,
            'date': f'{y}-{mth}-{d}',
            'datetime': rec.get('Дата и час', ''),
            'amount': eur if outflow else -eur,
            'original': original,
            'counterparty': rec.get('Име на контрагента', ''),
            'counterparty_iban': rec.get('Сметка на контрагента', ''),
            'counterparty_bank': rec.get('Банка на контрагента', ''),
            'swift': rec.get('SWIFT', ''),
            'details': clean_text(details),
            'raw_details': details,
        })
    seen = defaultdict(list)
    for r in records:
        seen[(r['iban'], r['ref'])].append(r)
    for group in seen.values():
        group.sort(key=lambda r: (r['amount'] < 0, abs(r['amount'])))
        for i, r in enumerate(group):
            r['external_id'] = f"procredit-{r['ref']}" + (f'-{i + 1}' if i else '')
    return records, skipped, headers


def title(s):
    return ' '.join(w if w.isupper() and len(w) <= 3 else w.capitalize() for w in s.split())


def classify(r, payees, config):
    text = r['details'].upper()
    own = set(config['accounts'])
    if 'REVOLUT' in text and ('ПОС ПЛАЩАНЕ' in text or 'CARD TRANSACTION' in text):
        return 'Revolut top-up', 'Internal transfers'
    if 'ТАКСА ATM' in text:
        return 'ProCredit Bank', 'Bank'
    if 'ATM ТЕГЛЕНЕ' in text:
        return 'ATM', 'Cash withdrawals'
    if 'CASHBACK' in text:
        return 'ProCredit cashback', 'Income'
    if 'ЗАПЛАТ' in text:
        payer = re.search(r'НАРЕДИТЕЛ:\s*([^|]+)', r['details'])
        payer = payer.group(1).strip() if payer else r['counterparty']
        for key, name in EMPLOYERS.items():
            if key in payer.upper():
                return name, 'Salary'
        return title(payer) or 'Salary', 'Salary'
    if re.search(r'ГЛАВНИЦА|ЛИХВА|КРЕДИТ|ПРОСР', text) and not r['counterparty']:
        return 'ProCredit Bank', 'Loans'
    if 'ТАКСА' in text or 'МЕС.ТАКСА' in text:
        return 'ProCredit Bank', 'Bank'
    if r['counterparty_iban'] in own:
        return config['accounts'][r['counterparty_iban']], None
    if 'СТАНЧЕВ' in r['counterparty'].upper() and r['counterparty_iban'][4:8] == 'STSA':
        return 'DSK laptop loan', 'Loans'
    key = merchant_key(r['raw_details'])
    if key:
        if key in payees['exact']:
            return payees['exact'][key], None
        first = merchant_prefix(r['raw_details'])
        if first in payees['prefix']:
            return payees['prefix'][first], None
        return title(card_merchant(r['raw_details'])), None
    if r['counterparty']:
        return payees.get('counterparty', {}).get(counterparty_key(r['counterparty'])) or title(r['counterparty']), None
    return (r['details'].split(' | ')[0] or 'ProCredit')[:60], None


def build_notes(r):
    parts = [r['details']]
    cp = ' '.join(x for x in (r['counterparty'], r['counterparty_iban'], r['counterparty_bank'], r['swift']) if x)
    if cp:
        parts.append(f'Counterparty: {cp}')
    if r['datetime']:
        parts.append(f"Time: {r['datetime']}")
    if r['original']:
        parts.append(f"Original: {r['original']}")
    parts.append(f"Ref: {r['ref']}")
    return ' | '.join(p for p in parts if p)


class Sure:
    def __init__(self, url, key):
        self.url = url.rstrip('/')
        self.key = key

    def call(self, method, path, body=None, query=None):
        full = f'{self.url}/api/v1{path}' + (f'?{urllib.parse.urlencode(query, doseq=True)}' if query else '')
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(full, data=data, method=method, headers={
            'X-Api-Key': self.key, 'Content-Type': 'application/json', 'Accept': 'application/json'})
        try:
            with urllib.request.urlopen(req, timeout=60) as resp:
                return resp.status, json.loads(resp.read() or b'null')
        except urllib.error.HTTPError as e:
            return e.code, json.loads(e.read() or b'null')

    def paged(self, path, key, query=None):
        page, out = 1, []
        while True:
            status, data = self.call('GET', path, query={**(query or {}), 'page': page, 'per_page': 100})
            if status != 200:
                sys.exit(f'GET {path} failed: {status} {data}')
            items = data.get(key, [])
            out += items
            if len(items) < 100:
                return out
            page += 1


def learn_payees(sure, config):
    accounts = {a['name']: a['id'] for a in sure.paged('/accounts', 'accounts')}
    votes, firsts = defaultdict(Counter), defaultdict(set)
    for name in set(config['accounts'].values()):
        for t in sure.paged('/transactions', 'transactions', {'account_id': accounts[name]}):
            notes = t.get('notes') or ''
            key = merchant_key(notes)
            if key and t.get('name') and t['name'] not in ('Revolut top-up', 'ATM', 'ProCredit Bank'):
                votes[key][t['name']] += 1
                first = merchant_prefix(notes)
                if first:
                    firsts[first].add(t['name'])
    exact = {k: c.most_common(1)[0][0] for k, c in votes.items()}
    prefix = {f: next(iter(ns)) for f, ns in firsts.items() if len(ns) == 1}
    payees = load_payees()
    payees.update({'exact': dict(sorted(exact.items())), 'prefix': dict(sorted(prefix.items()))})
    (HERE / 'payees.json').write_text(json.dumps(payees, ensure_ascii=False, indent=1), encoding='utf-8')
    print(f'payees.json: {len(exact)} exact names, {len(prefix)} prefixes')


def push(records, sure, payees, config, enrich):
    accounts = {a['name']: a['id'] for a in sure.paged('/accounts', 'accounts')}
    categories = {c['name']: c['id'] for c in sure.paged('/categories', 'categories')}
    stats = Counter()
    for r in records:
        name, category = classify(r, payees, config)
        notes = build_notes(r)
        body = {'transaction': {
            'account_id': accounts[r['account']], 'date': r['date'], 'name': name,
            'amount': str(abs(r['amount'])), 'nature': 'expense' if r['amount'] > 0 else 'income',
            'currency': 'EUR', 'notes': notes, 'external_id': r['external_id'], 'source': SOURCE}}
        if category and category in categories:
            body['transaction']['category_id'] = categories[category]
        status, data = sure.call('POST', '/transactions', body)
        if status == 201:
            stats['created'] += 1
        elif status == 200:
            stats['already in Sure'] += 1
            if enrich and (data.get('notes') or '') != notes:
                s, _ = sure.call('PATCH', f"/transactions/{data['id']}", {'transaction': {'notes': notes}})
                stats['enriched' if s == 200 else f'enrich failed {s}'] += 1
        else:
            stats[f'failed {status}'] += 1
            print(f"FAILED {r['external_id']} {r['date']} {r['amount']}: {data}", file=sys.stderr)
    return stats


def write_csv(records, payees, config, path):
    with open(path, 'w', encoding='utf-8', newline='') as f:
        w = csv.writer(f)
        w.writerow(['date', 'account', 'name', 'amount', 'category', 'notes'])
        for r in records:
            name, category = classify(r, payees, config)
            w.writerow([r['date'], r['account'], name, f"{-r['amount']:.2f}", category or '', build_notes(r)])


def main():
    p = argparse.ArgumentParser(description='Load a ProBanking "Движения по сметки" XLSX export into Sure.')
    p.add_argument('xlsx', nargs='?', help='ProBanking XLSX export (all accounts)')
    p.add_argument('--url', default=os.getenv('SURE_URL', 'https://money.iaminyourpc.xyz'))
    p.add_argument('--api-key', default=os.getenv('SURE_API_KEY'))
    p.add_argument('--since', help='only rows with value date on or after YYYY-MM-DD')
    p.add_argument('--dry-run', action='store_true', help='parse and classify only, print a summary')
    p.add_argument('--csv', help='write a CSV for the Sure UI import instead of using the API')
    p.add_argument('--no-enrich', action='store_true', help='do not update notes of transactions already in Sure')
    p.add_argument('--learn-payees', action='store_true', help='rebuild payees.json from the names already in Sure')
    a = p.parse_args()
    config = load_config()

    if a.learn_payees:
        if not a.api_key:
            sys.exit('set SURE_API_KEY or --api-key')
        learn_payees(Sure(a.url, a.api_key), config)
        return
    if not a.xlsx:
        p.error('xlsx file is required')

    records, skipped, _ = parse_rows(a.xlsx, config)
    if a.since:
        records = [r for r in records if r['date'] >= a.since]
    payees = load_payees()
    by_acc = Counter(r['account'] for r in records)
    print(f'{len(records)} booked transactions: ' + ', '.join(f'{k} {v}' for k, v in by_acc.items()))
    for reason, n in skipped.items():
        print(f'  skipped {n}: {reason}')

    if a.dry_run:
        for r in records:
            name, category = classify(r, payees, config)
            print(f"{r['date']} {r['account']:<17} {-r['amount']:>10.2f}  {name:<32} {category or '-':<20} {r['external_id']}")
        return
    if a.csv:
        write_csv(records, payees, config, a.csv)
        print(f'wrote {a.csv}')
        return
    if not a.api_key:
        sys.exit('set SURE_API_KEY or --api-key (Sure → Settings → API keys, read/write)')
    stats = push(records, Sure(a.url, a.api_key), payees, config, not a.no_enrich)
    print(', '.join(f'{k}: {v}' for k, v in stats.items()))


if __name__ == '__main__':
    main()
