const params = new URLSearchParams(location.search);
const category = params.get('category') || '';
const day = params.get('day') || '';
const month = params.get('month') || '';
const days = Number(params.get('days')) || 0;
const localDay = (d) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
const from = days > 0 ? localDay(new Date(Date.now() - (days - 1) * 86400000)) : params.get('from') || '';
const to = params.get('to') || '';
const isDay = (v) => /^\d{4}-\d{2}-\d{2}$/.test(v);
const filterConditions = [];
let path = '/accounts';
if (category === 'uncategorized') {
  path = '/accounts/uncategorized';
} else if (/^[0-9a-f-]{36}$/.test(category)) {
  filterConditions.push({ field: 'category', op: 'is', value: category, type: 'id' });
}
if (isDay(day)) {
  filterConditions.push({ field: 'date', op: 'is', value: day, type: 'date' });
} else if (isDay(from) || isDay(to)) {
  if (isDay(from)) filterConditions.push({ field: 'date', op: 'gte', value: from, type: 'date' });
  if (isDay(to)) filterConditions.push({ field: 'date', op: 'lte', value: to, type: 'date' });
} else if (/^\d{4}-\d{2}$/.test(month)) {
  filterConditions.push({ field: 'date', op: 'is', value: month, options: { month: true }, type: 'date' });
}
history.replaceState({ usr: { filterConditions, categoryId: category || undefined }, key: 'grafana', idx: 0 }, '', path);
location.reload();
