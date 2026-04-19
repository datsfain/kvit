// DATA is injected by Go as: const DATA = {...};

// Global Chart.js defaults
Chart.defaults.color = '#6e7681';
Chart.defaults.borderColor = '#1e2330';
Chart.defaults.font.family = "'Outfit', sans-serif";

// ── Helpers ──

function esc(s) {
  const d = document.createElement('div');
  d.textContent = s;
  return d.innerHTML;
}

function round2(n) { return Math.round(n * 100) / 100; }

function getCat(product) { return catMap[product] || 'uncategorized'; }

function fmtDate(iso) {
  if (!iso) return '';
  const [y, m, d] = iso.split('-');
  const months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
  return parseInt(d) + ' ' + months[parseInt(m) - 1] + ' ' + y;
}

function getProductPurchases(data, product) {
  return data.filter(e => e.product === product).sort((a, b) => b.date.localeCompare(a.date));
}

function getTripItems(date, store, excludeProduct) {
  return DATA.expenses.filter(e => e.date === date && e.store === store && e.product !== excludeProduct);
}

// ISO date format ensures string comparison works for filtering
function today() { return new Date().toISOString().slice(0, 10); }
function monthStart(d) { return d.slice(0, 7) + '-01'; }
function addMonths(dateStr, n) {
  const d = new Date(dateStr + 'T00:00:00');
  d.setMonth(d.getMonth() + n);
  return d.toISOString().slice(0, 10);
}
function dayBefore(dateStr) {
  const d = new Date(dateStr + 'T00:00:00');
  d.setDate(d.getDate() - 1);
  return d.toISOString().slice(0, 10);
}

function groupBy(data, keyFn) {
  const map = {};
  data.forEach(e => { const k = keyFn(e); map[k] = (map[k] || 0) + e.price; });
  return map;
}

function groupBy2(data, outerKeyFn, innerKeyFn) {
  const map = {};
  const innerKeys = new Set();
  data.forEach(e => {
    const ok = outerKeyFn(e), ik = innerKeyFn(e);
    innerKeys.add(ik);
    if (!map[ok]) map[ok] = {};
    map[ok][ik] = (map[ok][ik] || 0) + e.price;
  });
  return { buckets: map, keys: innerKeys };
}

const dkkTooltip = {
  callbacks: {
    label: ctx => {
      const val = ctx.parsed.y ?? ctx.parsed;
      const prefix = ctx.dataset?.label ? ctx.dataset.label + ': ' : (ctx.label ? ctx.label + ': ' : '');
      return prefix + val.toFixed(2) + ' ' + DATA.currency;
    }
  }
};

function stackedBarConfig(extra = {}) {
  return {
    ...extra,
    plugins: {
      legend: { labels: { color: '#8b949e', padding: 12 } },
      tooltip: dkkTooltip,
      ...(extra.plugins || {})
    },
    scales: {
      x: { stacked: true, ticks: { color: '#8b949e' }, grid: { display: false } },
      y: { stacked: true, ticks: { color: '#8b949e', callback: v => v + ' ' + DATA.currency }, grid: { color: '#21262d' } }
    }
  };
}

// ── Data setup ──

const catMap = {};
DATA.definitions.forEach(d => catMap[d.product] = d.category);

// ── Colors ──

const PALETTE = [
  '#e8b931','#4c9aff','#3fb950','#e05545','#d2a8ff',
  '#f78166','#79c0ff','#7ee787','#d4956a','#bc8cff',
  '#f0c75e','#a5d6ff','#c4a35a','#56d364','#ffa657',
  '#ff7b72','#8b949e','#a371f7','#f69d50','#6cb6ff'
];

function hashStr(s) {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = ((h << 5) - h + s.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}

const colorCache = {};
function getColorForCat(name) {
  if (!colorCache[name]) {
    colorCache[name] = (DATA.colors && DATA.colors[name]) || PALETTE[hashStr(name) % PALETTE.length];
  }
  return colorCache[name];
}

// ── State ──

const state = {
  charts: {},
  pieMode: 'category',
  pieLimit: 10,
  tableMode: 'category',
  tableSortCol: 'total',
  tableSortAsc: false,
  expandedCats: new Set(),
  stackedMode: 'weekly',
  filtered: [],
  excludedProducts: new Set(),
  excludedCategories: new Set(),
};

// ── Presets ──

const PRESETS = [
  { label: 'This month', fn: () => { const t = today(); return [monthStart(t), t]; }},
  { label: 'Last month', fn: () => { const ms = monthStart(today()); return [addMonths(ms, -1), dayBefore(ms)]; }},
  { label: 'Last 2 months', fn: () => { const t = today(); return [addMonths(monthStart(t), -1), t]; }},
  { label: 'Last 3 months', fn: () => { const t = today(); return [addMonths(monthStart(t), -2), t]; }},
  { label: 'Last 6 months', fn: () => { const t = today(); return [addMonths(monthStart(t), -5), t]; }},
  { label: 'This year', fn: () => { const t = today(); return [t.slice(0,4) + '-01-01', t]; }},
  { label: 'All time', fn: () => { const dates = DATA.expenses.map(e => e.date).sort(); return [dates[0], dates[dates.length-1]]; }},
];

function initPresets() {
  const container = document.getElementById('presets');
  PRESETS.forEach(p => {
    const btn = document.createElement('button');
    btn.className = 'preset-btn';
    btn.textContent = p.label;
    btn.onclick = () => {
      document.querySelectorAll('.preset-btn').forEach(b => b.classList.remove('active'));
      btn.classList.add('active');
      const [from, to] = p.fn();
      document.getElementById('dateFrom').value = from;
      document.getElementById('dateTo').value = to;
      updateAll();
    };
    container.appendChild(btn);
  });
}

// ── Exclusions ──

function addExclusion(type, value) {
  if (!value) return;
  (type === 'product' ? state.excludedProducts : state.excludedCategories).add(value);
  renderExclusions();
  updateAll();
}

function removeExclusion(type, value) {
  (type === 'product' ? state.excludedProducts : state.excludedCategories).delete(value);
  renderExclusions();
  updateAll();
}

function renderExclusions() {
  const bar = document.getElementById('exclusionsBar');
  const chips = document.getElementById('exclusionsChips');
  const prodSel = document.getElementById('excludeProductSelect');
  const catSel = document.getElementById('excludeCategorySelect');

  const hasAny = state.excludedProducts.size > 0 || state.excludedCategories.size > 0;
  bar.classList.toggle('has-exclusions', hasAny);

  let html = '';
  [...state.excludedCategories].sort().forEach(c => {
    html += '<span class="exclusion-chip chip-category" data-type="category" data-value="' + esc(c) + '">' +
      '<span class="chip-icon">folder</span>' + esc(c) +
      '<button class="chip-remove" onclick="removeExclusion(\'category\', \'' + esc(c) + '\')">×</button></span>';
  });
  [...state.excludedProducts].sort().forEach(p => {
    html += '<span class="exclusion-chip chip-product" data-type="product" data-value="' + esc(p) + '">' +
      '<span class="chip-icon">item</span>' + esc(p) +
      '<button class="chip-remove" onclick="removeExclusion(\'product\', \'' + esc(p) + '\')">×</button></span>';
  });
  chips.innerHTML = html;

  // Repopulate selects excluding already-excluded items
  const allProducts = [...new Set(DATA.expenses.map(e => e.product))].sort();
  const allCats = [...new Set(DATA.definitions.map(d => d.category))].sort();

  prodSel.innerHTML = '<option value="">+ Exclude product</option>' +
    allProducts.filter(p => !state.excludedProducts.has(p))
      .map(p => '<option value="' + esc(p) + '">' + esc(p) + '</option>').join('');
  catSel.innerHTML = '<option value="">+ Exclude category</option>' +
    allCats.filter(c => !state.excludedCategories.has(c))
      .map(c => '<option value="' + esc(c) + '">' + esc(c) + '</option>').join('');
}

// ── Toggle helpers ──

function setToggleActive(groupAttr, value) {
  const kebab = groupAttr.replace(/([A-Z])/g, '-$1').toLowerCase();
  document.querySelectorAll('[data-' + kebab + ']').forEach(b => {
    b.classList.toggle('active', b.dataset[groupAttr] === value);
  });
}

// ── Filtering ──

function getFiltered() {
  const from = document.getElementById('dateFrom').value;
  const to = document.getElementById('dateTo').value;
  const store = document.getElementById('storeFilter').value;
  const cat = document.getElementById('categoryFilter').value;
  const needsCat = cat || state.excludedCategories.size > 0;
  return DATA.expenses.filter(e => {
    if (from && e.date < from) return false;
    if (to && e.date > to) return false;
    if (store && e.store !== store) return false;
    if (state.excludedProducts.has(e.product)) return false;
    if (!needsCat) return true;
    const productCat = getCat(e.product);
    if (state.excludedCategories.has(productCat)) return false;
    if (cat && productCat !== cat) return false;
    return true;
  });
}

function updateAll() {
  state.filtered = getFiltered();
  updateStats(state.filtered);
  updatePieChart(state.filtered);
  updateStoreBar(state.filtered);
  updateDailyLine(state.filtered);
  updateWeeklyStacked(state.filtered);
  updateProductTable(state.filtered);
}

// ── Stats ──

function updateStats(data) {
  const total = data.reduce((s, e) => s + e.price, 0);
  const days = new Set(data.map(e => e.date)).size;
  const avgDay = days > 0 ? total / days : 0;

  const dayTotals = groupBy(data, e => e.date);
  let maxDay = '', maxDayTotal = 0;
  Object.entries(dayTotals).forEach(([d, t]) => { if (t > maxDayTotal) { maxDay = d; maxDayTotal = t; }});

  const prodTotals = groupBy(data, e => e.product);
  let maxProd = '', maxProdTotal = 0;
  Object.entries(prodTotals).forEach(([p, t]) => { if (t > maxProdTotal) { maxProd = p; maxProdTotal = t; }});

  document.getElementById('stats').innerHTML =
    statCard('Total Spent', total.toFixed(2), DATA.currency, '', true) +
    statCard('Avg / Day', avgDay.toFixed(2), DATA.currency, days + ' days') +
    statCard('Most Expensive Day', maxDayTotal.toFixed(2), DATA.currency, fmtDate(maxDay)) +
    statCard('Top Product', maxProdTotal.toFixed(2), DATA.currency, maxProd);
}

function statCard(label, value, unit, detail, hero) {
  return '<div class="stat-card' + (hero ? ' hero' : '') + '"><div class="label">' + esc(label) + '</div><div class="value">' +
    esc(value) + ' <span class="unit">' + esc(unit) + '</span></div>' +
    (detail ? '<div class="detail">' + esc(detail) + '</div>' : '') + '</div>';
}

// ── Pie chart ──

function setPieLimit(n) {
  state.pieLimit = n;
  setToggleActive('pieLimit', String(n));
  updatePieChart(state.filtered);
}

function setPieMode(mode) {
  state.pieMode = mode;
  setToggleActive('pieMode', mode);
  document.getElementById('pieTitle').textContent = mode === 'category' ? 'Spending by Category' : 'Spending by Product';
  document.getElementById('pieLimitGroup').style.display = mode === 'product' ? '' : 'none';
  updatePieChart(state.filtered);
}

function updatePieChart(data) {
  const grouped = groupBy(data, e => state.pieMode === 'category' ? getCat(e.product) : e.product);
  const entries = Object.entries(grouped).sort((a, b) => b[1] - a[1]);

  let labels, values;
  const limit = state.pieLimit || 10;
  if (state.pieMode === 'product' && entries.length > limit) {
    const top = entries.slice(0, limit);
    const otherTotal = entries.slice(limit).reduce((s, e) => s + e[1], 0);
    labels = top.map(e => e[0]).concat(['other']);
    values = top.map(e => round2(e[1])).concat([round2(otherTotal)]);
  } else {
    labels = entries.map(e => e[0]);
    values = entries.map(e => round2(e[1]));
  }

  const colors = state.pieMode === 'category'
    ? labels.map(l => getColorForCat(l))
    : labels.map((_, i) => PALETTE[i % PALETTE.length]);

  if (state.charts.pie) state.charts.pie.destroy();
  state.charts.pie = new Chart(document.getElementById('categoryPie'), {
    type: 'doughnut',
    data: {
      labels,
      datasets: [{ data: values, backgroundColor: colors, borderWidth: 0 }]
    },
    options: {
      plugins: {
        legend: { position: 'right', labels: { color: '#8b949e', padding: 8, font: { size: 11 } } },
        tooltip: dkkTooltip
      }
    }
  });
}

// ── Store bar ──

function updateStoreBar(data) {
  const grouped = groupBy(data, e => e.store);
  const labels = Object.keys(grouped).sort((a, b) => grouped[b] - grouped[a]);
  const values = labels.map(l => round2(grouped[l]));

  if (state.charts.storeBar) state.charts.storeBar.destroy();
  state.charts.storeBar = new Chart(document.getElementById('storeBar'), {
    type: 'bar',
    data: { labels, datasets: [{ data: values, backgroundColor: PALETTE.slice(0, labels.length), borderRadius: 4 }] },
    options: {
      plugins: { legend: { display: false }, tooltip: dkkTooltip },
      scales: {
        x: { ticks: { color: '#8b949e' }, grid: { display: false } },
        y: { ticks: { color: '#8b949e', callback: v => v + ' ' + DATA.currency }, grid: { color: '#21262d' } }
      }
    }
  });
}

// ── Daily line ──

function updateDailyLine(data) {
  const grouped = groupBy(data, e => e.date);
  const dates = Object.keys(grouped).sort();
  const values = dates.map(d => round2(grouped[d]));

  if (state.charts.dailyLine) state.charts.dailyLine.destroy();
  state.charts.dailyLine = new Chart(document.getElementById('dailyLine'), {
    type: 'line',
    data: {
      labels: dates.map(fmtDate),
      datasets: [{
        data: values, borderColor: '#58a6ff', backgroundColor: 'rgba(88,166,255,0.1)',
        fill: true, tension: 0.3, pointRadius: 4, pointHoverRadius: 6
      }]
    },
    options: {
      plugins: { legend: { display: false }, tooltip: dkkTooltip },
      scales: {
        x: { ticks: { color: '#8b949e', maxTicksLimit: 15 }, grid: { color: '#21262d' } },
        y: { ticks: { color: '#8b949e', callback: v => v + ' ' + DATA.currency }, grid: { color: '#21262d' } }
      }
    }
  });
}

// ── Stacked chart (weekly/monthly) ──

function setStackedMode(mode) {
  state.stackedMode = mode;
  setToggleActive('stackedMode', mode);
  document.getElementById('stackedTitle').textContent =
    mode === 'weekly' ? 'Weekly Spending by Category' : 'Monthly Spending by Category';
  closeDrilldown();
  updateWeeklyStacked(state.filtered);
}

function getBucketKey(dateStr) {
  if (state.stackedMode === 'monthly') return dateStr.slice(0, 7);
  const d = new Date(dateStr);
  const ws = new Date(d); ws.setDate(d.getDate() - d.getDay() + 1);
  return ws.toISOString().slice(0, 10);
}

function getBucketLabel(key) {
  if (state.stackedMode === 'monthly') {
    const [y, m] = key.split('-');
    const months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
    return months[parseInt(m) - 1] + ' ' + y;
  }
  return 'Week of ' + fmtDate(key);
}

function updateWeeklyStacked(data) {
  const { buckets, keys: allCats } = groupBy2(data, e => getBucketKey(e.date), e => getCat(e.product));
  const bucketLabels = Object.keys(buckets).sort();
  const categories = [...allCats].sort();

  const datasets = categories.map(cat => ({
    label: cat,
    data: bucketLabels.map(b => round2(buckets[b]?.[cat] || 0)),
    backgroundColor: getColorForCat(cat),
    borderRadius: 2
  }));

  if (state.charts.weeklyStacked) state.charts.weeklyStacked.destroy();
  state.charts.weeklyStacked = new Chart(document.getElementById('weeklyStacked'), {
    type: 'bar',
    data: { labels: bucketLabels.map(getBucketLabel), datasets },
    options: stackedBarConfig({
      onClick: (evt, elements) => {
        if (elements.length > 0) showDrilldown(categories[elements[0].datasetIndex], data);
      },
      plugins: {
        legend: {
          labels: { color: '#8b949e', padding: 12 },
          onClick: (evt, item) => showDrilldown(item.text, data)
        },
        tooltip: dkkTooltip
      }
    })
  });
}

// ── Drilldown ──

function showDrilldown(category, data) {
  document.getElementById('drilldownCard').style.display = 'block';
  document.getElementById('drilldownTitle').textContent = category + ' — Products';
  document.getElementById('stackedHint').textContent = 'Showing drill-down for: ' + category;

  const catData = data.filter(e => getCat(e.product) === category);
  const { buckets, keys: allProducts } = groupBy2(catData, e => getBucketKey(e.date), e => e.product);
  const bucketLabels = Object.keys(buckets).sort();
  const products = [...allProducts].sort();

  const datasets = products.map((prod, i) => ({
    label: prod,
    data: bucketLabels.map(b => round2(buckets[b]?.[prod] || 0)),
    backgroundColor: PALETTE[i % PALETTE.length],
    borderRadius: 2
  }));

  if (state.charts.drilldown) state.charts.drilldown.destroy();
  state.charts.drilldown = new Chart(document.getElementById('drilldownChart'), {
    type: 'bar',
    data: { labels: bucketLabels.map(getBucketLabel), datasets },
    options: stackedBarConfig()
  });

  document.getElementById('drilldownCard').scrollIntoView({ behavior: 'smooth' });
}

function closeDrilldown() {
  document.getElementById('drilldownCard').style.display = 'none';
  document.getElementById('stackedHint').textContent = 'Click a category in the legend to drill down into products';
  if (state.charts.drilldown) { state.charts.drilldown.destroy(); state.charts.drilldown = null; }
}

// ── Table ──

function setTableMode(mode) {
  state.tableMode = mode;
  setToggleActive('tableMode', mode);
  state.expandedCats.clear();
  updateProductTable(state.filtered);
}

function updateProductTable(data) {
  const total = data.reduce((s, e) => s + e.price, 0);
  const search = document.getElementById('tableSearch').value.toLowerCase();
  const products = groupBy(data, e => e.product);
  const tbody = document.querySelector('#productTable tbody');

  if (state.tableMode === 'flat') {
    let entries = Object.entries(products).map(([name, t]) => ({
      name, category: getCat(name), total: t, pct: total > 0 ? t / total * 100 : 0
    }));

    if (search) entries = entries.filter(e => e.name.includes(search) || e.category.includes(search));

    entries.sort((a, b) => {
      const va = a[state.tableSortCol] !== undefined ? a[state.tableSortCol] : a.name;
      const vb = b[state.tableSortCol] !== undefined ? b[state.tableSortCol] : b.name;
      if (state.tableSortCol === 'name' || state.tableSortCol === 'category') {
        return state.tableSortAsc ? String(va).localeCompare(String(vb)) : String(vb).localeCompare(String(va));
      }
      return state.tableSortAsc ? va - vb : vb - va;
    });

    tbody.innerHTML = entries.map(e =>
      '<tr><td><a class="product-link" data-product="' + esc(e.name) + '">' + esc(e.name) + '</a></td><td>' + esc(e.category) + '</td><td class="amount">' +
      e.total.toFixed(2) + '</td><td class="pct"><span class="pct-bar" style="width:' +
      Math.max(e.pct * 0.8, 2) + 'px"></span>' + e.pct.toFixed(1) + '%</td></tr>'
    ).join('');
  } else {
    const catGroups = {};
    Object.entries(products).forEach(([name, t]) => {
      const cat = getCat(name);
      if (!catGroups[cat]) catGroups[cat] = { total: 0, products: [] };
      catGroups[cat].total += t;
      catGroups[cat].products.push({ name, total: t, pct: total > 0 ? t / total * 100 : 0 });
    });

    const catEntries = Object.entries(catGroups).sort((a, b) => b[1].total - a[1].total);
    catEntries.forEach(([_, g]) => g.products.sort((a, b) => b.total - a.total));

    let html = '';
    catEntries.forEach(([cat, g]) => {
      const matchesCat = !search || cat.includes(search);
      const matchingProducts = g.products.filter(p => !search || p.name.includes(search) || cat.includes(search));
      if (!matchesCat && matchingProducts.length === 0) return;

      const expanded = state.expandedCats.has(cat);
      const icon = expanded ? '&#9660;' : '&#9654;';
      const catPct = total > 0 ? (g.total / total * 100).toFixed(1) : '0';

      html += '<tr class="cat-row" data-cat="' + esc(cat) + '"><td><span class="expand-icon">' + icon + '</span>' +
        esc(cat) + '</td><td></td><td class="amount">' + g.total.toFixed(2) +
        '</td><td class="pct"><span class="pct-bar" style="width:' +
        Math.max(parseFloat(catPct) * 0.8, 2) + 'px"></span>' + catPct + '%</td></tr>';

      const prods = search ? matchingProducts : g.products;
      prods.forEach(p => {
        html += '<tr class="product-row ' + (expanded ? '' : 'hidden') + '" data-cat="' + esc(cat) +
          '"><td><a class="product-link" data-product="' + esc(p.name) + '">' + esc(p.name) + '</a></td><td>' + esc(cat) + '</td><td class="amount">' +
          p.total.toFixed(2) + '</td><td class="pct"><span class="pct-bar" style="width:' +
          Math.max(p.pct * 0.8, 2) + 'px"></span>' + p.pct.toFixed(1) + '%</td></tr>';
      });
    });
    tbody.innerHTML = html;

    document.querySelectorAll('.cat-row').forEach(row => {
      row.onclick = () => {
        const cat = row.dataset.cat;
        if (state.expandedCats.has(cat)) state.expandedCats.delete(cat); else state.expandedCats.add(cat);
        updateProductTable(state.filtered);
      };
    });
  }

  document.querySelectorAll('.product-link').forEach(link => {
    link.onclick = (e) => { e.stopPropagation(); showProductDetail(link.dataset.product); };
  });

  document.querySelectorAll('#productTable th').forEach(th => {
    const col = th.dataset.sort;
    const arrow = th.querySelector('.sort-arrow');
    arrow.textContent = col === state.tableSortCol ? (state.tableSortAsc ? ' ▲' : ' ▼') : '';
  });
}

// ── Product detail ──

function showProductDetail(product) {
  const purchases = getProductPurchases(state.filtered, product);
  const total = purchases.reduce((s, e) => s + e.price, 0);

  document.getElementById('productDetailTitle').textContent = product;
  document.getElementById('productDetailSummary').textContent =
    total.toFixed(2) + ' ' + DATA.currency + ' · ' + purchases.length + ' purchases';

  const tbody = document.querySelector('#productDetailTable tbody');
  tbody.innerHTML = purchases.map(p =>
    '<tr><td>' + fmtDate(p.date) + '</td><td>' + esc(p.store) + '</td><td class="amount">' +
    p.price.toFixed(2) + '</td><td><a class="trip-link" data-date="' + esc(p.date) +
    '" data-store="' + esc(p.store) + '" data-product="' + esc(product) + '">see trip</a></td></tr>'
  ).join('');

  tbody.querySelectorAll('.trip-link').forEach(link => {
    link.onclick = () => toggleTrip(link);
  });

  const panel = document.getElementById('productDetail');
  panel.style.display = 'block';
  panel.scrollIntoView({ behavior: 'smooth' });
}

function toggleTrip(link) {
  const row = link.closest('tr');
  const next = row.nextElementSibling;
  if (next && next.classList.contains('trip-detail-row')) {
    next.remove();
    link.textContent = 'see trip';
    return;
  }

  const others = getTripItems(link.dataset.date, link.dataset.store, link.dataset.product);
  if (others.length === 0) {
    link.textContent = 'only item';
    return;
  }

  const tripTotal = others.reduce((s, e) => s + e.price, 0);
  const lines = others.map(e =>
    '<div class="receipt-line"><span>' + esc(e.product) + '</span><span>' + e.price.toFixed(2) + '</span></div>'
  ).join('');
  const tripRow = document.createElement('tr');
  tripRow.className = 'trip-detail-row';
  tripRow.innerHTML = '<td colspan="4"><div class="receipt">' +
    '<div class="receipt-header">' + esc(link.dataset.store) + ' · ' + fmtDate(link.dataset.date) + '</div>' +
    lines +
    '<div class="receipt-total"><span>Total</span><span>' + tripTotal.toFixed(2) + ' ' + DATA.currency + '</span></div>' +
    '</div></td>';
  row.after(tripRow);
  link.textContent = 'hide trip';
}

function closeProductDetail() {
  document.getElementById('productDetail').style.display = 'none';
}
window.closeProductDetail = closeProductDetail;
window.setPieLimit = setPieLimit;
window.addExclusion = addExclusion;
window.removeExclusion = removeExclusion;

// ── Event listeners ──

document.querySelectorAll('#productTable th').forEach(th => {
  th.onclick = () => {
    const col = th.dataset.sort;
    if (!col) return;
    if (state.tableSortCol === col) { state.tableSortAsc = !state.tableSortAsc; } else { state.tableSortCol = col; state.tableSortAsc = false; }
    updateProductTable(state.filtered);
  };
});

document.getElementById('tableSearch').addEventListener('input', () => updateProductTable(state.filtered));

document.getElementById('dateFrom').addEventListener('change', () => { document.querySelectorAll('.preset-btn').forEach(b => b.classList.remove('active')); updateAll(); });
document.getElementById('dateTo').addEventListener('change', () => { document.querySelectorAll('.preset-btn').forEach(b => b.classList.remove('active')); updateAll(); });
document.getElementById('storeFilter').addEventListener('change', updateAll);
document.getElementById('categoryFilter').addEventListener('change', updateAll);

// ── Color configurator ──

function initColorConfig() {
  const categories = [...new Set(DATA.definitions.map(d => d.category))].sort();
  const container = document.getElementById('colorConfig');
  if (!container || categories.length === 0) return;

  categories.forEach(cat => {
    const row = document.createElement('div');
    row.className = 'color-row';

    const swatch = document.createElement('input');
    swatch.type = 'color';
    swatch.value = getColorForCat(cat);
    swatch.className = 'color-picker';
    swatch.dataset.cat = cat;

    const label = document.createElement('span');
    label.className = 'color-label';
    label.textContent = cat;

    row.appendChild(swatch);
    row.appendChild(label);
    container.appendChild(row);
  });
}

function applyAndSave() {
  // Update color cache from pickers
  document.querySelectorAll('.color-picker').forEach(swatch => {
    colorCache[swatch.dataset.cat] = swatch.value;
  });
  updateAll();

  // Build and download CSV
  const categories = [...new Set(DATA.definitions.map(d => d.category))].sort();
  const csv = 'category,color\n' + categories.map(cat => cat + ',' + getColorForCat(cat)).join('\n') + '\n';
  const blob = new Blob([csv], { type: 'text/csv' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = 'colors.csv';
  a.click();

  const btn = document.getElementById('applyColorsBtn');
  btn.textContent = 'Saved!';
  setTimeout(() => { btn.textContent = 'Save colors.csv'; }, 2000);
}

// Expose to onclick handlers in HTML
window.setPieMode = setPieMode;
window.setStackedMode = setStackedMode;
window.setTableMode = setTableMode;
window.closeDrilldown = closeDrilldown;
window.applyAndSave = applyAndSave;

// ── Init ──

function initFilters() {
  const stores = [...new Set(DATA.expenses.map(e => e.store))].sort();
  const cats = [...new Set(DATA.definitions.map(d => d.category))].sort();

  const sf = document.getElementById('storeFilter');
  stores.forEach(s => { const o = document.createElement('option'); o.value = s; o.text = s; sf.add(o); });

  const cf = document.getElementById('categoryFilter');
  cats.forEach(c => { const o = document.createElement('option'); o.value = c; o.text = c; cf.add(o); });

  const dates = DATA.expenses.map(e => e.date).sort();
  document.getElementById('dateFrom').value = dates[0];
  document.getElementById('dateTo').value = dates[dates.length - 1];
}

initPresets();
initFilters();
initColorConfig();
renderExclusions();
document.getElementById('excludeProductSelect').addEventListener('change', e => {
  addExclusion('product', e.target.value);
  e.target.value = '';
});
document.getElementById('excludeCategorySelect').addEventListener('change', e => {
  addExclusion('category', e.target.value);
  e.target.value = '';
});
updateAll();
