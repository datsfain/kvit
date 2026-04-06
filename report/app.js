// DATA is injected by Go as: const DATA = {...};

const catMap = {};
DATA.definitions.forEach(d => catMap[d.product] = d.category);

const COLORS = [
  '#58a6ff','#f78166','#3fb950','#d2a8ff','#f0883e',
  '#a5d6ff','#7ee787','#ffa657','#ff7b72','#79c0ff',
  '#d29922','#56d364','#bc8cff','#e3b341'
];

let charts = {};
let pieMode = 'category';
let tableMode = 'category';
let tableSortCol = 'total';
let tableSortAsc = false;
let expandedCats = new Set();
let currentFiltered = [];

// ── Date helpers ──

function today() { return new Date().toISOString().slice(0,10); }
function monthStart(d) { return d.slice(0,7) + '-01'; }
function addMonths(dateStr, n) {
  const d = new Date(dateStr + 'T00:00:00');
  d.setMonth(d.getMonth() + n);
  return d.toISOString().slice(0,10);
}

// ── Presets ──

const PRESETS = [
  { label: 'This month', fn: () => { const t = today(); return [monthStart(t), t]; }},
  { label: 'Last month', fn: () => { const ms = monthStart(today()); return [addMonths(ms, -1), addMonths(ms, 0, -1)]; }},
  { label: 'Last 2 months', fn: () => { const t = today(); return [addMonths(monthStart(t), -1), t]; }},
  { label: 'Last 3 months', fn: () => { const t = today(); return [addMonths(monthStart(t), -2), t]; }},
  { label: 'Last 6 months', fn: () => { const t = today(); return [addMonths(monthStart(t), -5), t]; }},
  { label: 'This year', fn: () => { const t = today(); return [t.slice(0,4) + '-01-01', t]; }},
  { label: 'All time', fn: () => { const dates = DATA.expenses.map(e=>e.date).sort(); return [dates[0], dates[dates.length-1]]; }},
];

function initPresets() {
  const container = document.getElementById('presets');
  PRESETS.forEach((p, i) => {
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

// ── Filtering ──

function getFiltered() {
  const from = document.getElementById('dateFrom').value;
  const to = document.getElementById('dateTo').value;
  const store = document.getElementById('storeFilter').value;
  const cat = document.getElementById('categoryFilter').value;
  return DATA.expenses.filter(e => {
    if (from && e.date < from) return false;
    if (to && e.date > to) return false;
    if (store && e.store !== store) return false;
    if (cat && catMap[e.product] !== cat) return false;
    return true;
  });
}

function updateAll() {
  currentFiltered = getFiltered();
  updateStats(currentFiltered);
  updatePieChart(currentFiltered);
  updateStoreBar(currentFiltered);
  updateDailyLine(currentFiltered);
  updateWeeklyStacked(currentFiltered);
  updateProductTable(currentFiltered);
}

// ── Stats ──

function updateStats(data) {
  const total = data.reduce((s, e) => s + e.price, 0);
  const days = new Set(data.map(e => e.date)).size;
  const avgDay = days > 0 ? total / days : 0;

  const dayTotals = {};
  data.forEach(e => { dayTotals[e.date] = (dayTotals[e.date] || 0) + e.price; });
  let maxDay = '', maxDayTotal = 0;
  Object.entries(dayTotals).forEach(([d, t]) => { if (t > maxDayTotal) { maxDay = d; maxDayTotal = t; }});

  const prodTotals = {};
  data.forEach(e => { prodTotals[e.product] = (prodTotals[e.product] || 0) + e.price; });
  let maxProd = '', maxProdTotal = 0;
  Object.entries(prodTotals).forEach(([p, t]) => { if (t > maxProdTotal) { maxProd = p; maxProdTotal = t; }});

  document.getElementById('stats').innerHTML =
    statCard('Total Spent', total.toFixed(2), 'DKK', '') +
    statCard('Avg / Day', avgDay.toFixed(2), 'DKK', days + ' days') +
    statCard('Most Expensive Day', maxDayTotal.toFixed(2), 'DKK', maxDay) +
    statCard('Top Product', maxProdTotal.toFixed(2), 'DKK', maxProd);
}

function statCard(label, value, unit, detail) {
  return '<div class="stat-card"><div class="label">' + label + '</div><div class="value">' +
    value + ' <span class="unit">' + unit + '</span></div>' +
    (detail ? '<div class="detail">' + detail + '</div>' : '') + '</div>';
}

// ── Pie chart ──

function setPieMode(mode) {
  pieMode = mode;
  document.querySelectorAll('.chart-card:first-child .toggle-btn').forEach(b => b.classList.remove('active'));
  document.querySelector('.toggle-btn[onclick*="setPieMode(\'' + mode + '\')"]').classList.add('active');
  document.getElementById('pieTitle').textContent = mode === 'category' ? 'Spending by Category' : 'Spending by Product';
  updatePieChart(currentFiltered);
}
// Expose to onclick
window.setPieMode = setPieMode;

function updatePieChart(data) {
  const grouped = {};
  data.forEach(e => {
    const key = pieMode === 'category' ? (catMap[e.product] || 'uncategorized') : e.product;
    grouped[key] = (grouped[key] || 0) + e.price;
  });
  const entries = Object.entries(grouped).sort((a, b) => b[1] - a[1]);
  let labels, values;
  if (pieMode === 'product' && entries.length > 10) {
    const top = entries.slice(0, 10);
    const otherTotal = entries.slice(10).reduce((s, e) => s + e[1], 0);
    labels = top.map(e => e[0]).concat(['other']);
    values = top.map(e => Math.round(e[1] * 100) / 100).concat([Math.round(otherTotal * 100) / 100]);
  } else {
    labels = entries.map(e => e[0]);
    values = entries.map(e => Math.round(e[1] * 100) / 100);
  }

  if (charts.pie) charts.pie.destroy();
  charts.pie = new Chart(document.getElementById('categoryPie'), {
    type: 'doughnut',
    data: {
      labels: labels,
      datasets: [{ data: values, backgroundColor: COLORS.slice(0, labels.length), borderWidth: 0 }]
    },
    options: {
      plugins: {
        legend: { position: 'right', labels: { color: '#8b949e', padding: 8, font: { size: 11 } } },
        tooltip: { callbacks: { label: ctx => ctx.label + ': ' + ctx.parsed.toFixed(2) + ' DKK' } }
      }
    }
  });
}

// ── Store bar ──

function updateStoreBar(data) {
  const grouped = {};
  data.forEach(e => { grouped[e.store] = (grouped[e.store] || 0) + e.price; });
  const labels = Object.keys(grouped).sort((a, b) => grouped[b] - grouped[a]);
  const values = labels.map(l => Math.round(grouped[l] * 100) / 100);

  if (charts.storeBar) charts.storeBar.destroy();
  charts.storeBar = new Chart(document.getElementById('storeBar'), {
    type: 'bar',
    data: { labels, datasets: [{ data: values, backgroundColor: COLORS.slice(0, labels.length), borderRadius: 4 }] },
    options: {
      plugins: { legend: { display: false }, tooltip: { callbacks: { label: ctx => ctx.parsed.y.toFixed(2) + ' DKK' } } },
      scales: {
        x: { ticks: { color: '#8b949e' }, grid: { display: false } },
        y: { ticks: { color: '#8b949e', callback: v => v + ' DKK' }, grid: { color: '#21262d' } }
      }
    }
  });
}

// ── Daily line ──

function updateDailyLine(data) {
  const grouped = {};
  data.forEach(e => { grouped[e.date] = (grouped[e.date] || 0) + e.price; });
  const dates = Object.keys(grouped).sort();
  const values = dates.map(d => Math.round(grouped[d] * 100) / 100);

  if (charts.dailyLine) charts.dailyLine.destroy();
  charts.dailyLine = new Chart(document.getElementById('dailyLine'), {
    type: 'line',
    data: {
      labels: dates,
      datasets: [{
        data: values, borderColor: '#58a6ff', backgroundColor: 'rgba(88,166,255,0.1)',
        fill: true, tension: 0.3, pointRadius: 4, pointHoverRadius: 6
      }]
    },
    options: {
      plugins: { legend: { display: false }, tooltip: { callbacks: { label: ctx => ctx.parsed.y.toFixed(2) + ' DKK' } } },
      scales: {
        x: { ticks: { color: '#8b949e', maxTicksLimit: 15 }, grid: { color: '#21262d' } },
        y: { ticks: { color: '#8b949e', callback: v => v + ' DKK' }, grid: { color: '#21262d' } }
      }
    }
  });
}

// ── Stacked chart (weekly/monthly) ──

let stackedMode = 'weekly';

function setStackedMode(mode) {
  stackedMode = mode;
  document.querySelectorAll('#stackedTitle').forEach(el => {
    el.textContent = mode === 'weekly' ? 'Weekly Spending by Category' : 'Monthly Spending by Category';
  });
  const btns = document.querySelectorAll('#stackedTitle + .toggle-group .toggle-btn');
  btns.forEach(b => b.classList.remove('active'));
  document.querySelector('.toggle-btn[onclick*="setStackedMode(\'' + mode + '\')"]').classList.add('active');
  closeDrilldown();
  updateWeeklyStacked(currentFiltered);
}
window.setStackedMode = setStackedMode;

function getBucketKey(dateStr) {
  if (stackedMode === 'monthly') {
    return dateStr.slice(0, 7); // YYYY-MM
  }
  const d = new Date(dateStr);
  const ws = new Date(d); ws.setDate(d.getDate() - d.getDay() + 1);
  return ws.toISOString().slice(0, 10);
}

function getBucketLabel(key) {
  if (stackedMode === 'monthly') {
    const [y, m] = key.split('-');
    const months = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'];
    return months[parseInt(m)-1] + ' ' + y;
  }
  return 'Week of ' + key;
}

function updateWeeklyStacked(data) {
  const buckets = {};
  const allCats = new Set();
  data.forEach(e => {
    const bk = getBucketKey(e.date);
    const cat = catMap[e.product] || 'uncategorized';
    allCats.add(cat);
    if (!buckets[bk]) buckets[bk] = {};
    buckets[bk][cat] = (buckets[bk][cat] || 0) + e.price;
  });
  const bucketLabels = Object.keys(buckets).sort();
  const categories = [...allCats].sort();
  const datasets = categories.map((cat, i) => ({
    label: cat,
    data: bucketLabels.map(b => Math.round((buckets[b][cat] || 0) * 100) / 100),
    backgroundColor: COLORS[i % COLORS.length], borderRadius: 2
  }));

  if (charts.weeklyStacked) charts.weeklyStacked.destroy();
  charts.weeklyStacked = new Chart(document.getElementById('weeklyStacked'), {
    type: 'bar',
    data: { labels: bucketLabels.map(getBucketLabel), datasets },
    options: {
      onClick: (evt, elements) => {
        if (elements.length > 0) {
          const dsIndex = elements[0].datasetIndex;
          const cat = categories[dsIndex];
          showDrilldown(cat, data);
        }
      },
      plugins: {
        legend: {
          labels: { color: '#8b949e', padding: 12 },
          onClick: (evt, item, legend) => {
            const cat = item.text;
            showDrilldown(cat, data);
          }
        },
        tooltip: { callbacks: { label: ctx => ctx.dataset.label + ': ' + ctx.parsed.y.toFixed(2) + ' DKK' } }
      },
      scales: {
        x: { stacked: true, ticks: { color: '#8b949e' }, grid: { display: false } },
        y: { stacked: true, ticks: { color: '#8b949e', callback: v => v + ' DKK' }, grid: { color: '#21262d' } }
      }
    }
  });
}

// ── Drilldown (products within a category) ──

function showDrilldown(category, data) {
  const card = document.getElementById('drilldownCard');
  card.style.display = 'block';
  document.getElementById('drilldownTitle').textContent = category + ' — Products';
  document.getElementById('stackedHint').textContent = 'Showing drill-down for: ' + category;

  const catData = data.filter(e => (catMap[e.product] || 'uncategorized') === category);

  const buckets = {};
  const allProducts = new Set();
  catData.forEach(e => {
    const bk = getBucketKey(e.date);
    allProducts.add(e.product);
    if (!buckets[bk]) buckets[bk] = {};
    buckets[bk][e.product] = (buckets[bk][e.product] || 0) + e.price;
  });

  const bucketLabels = Object.keys(buckets).sort();
  const products = [...allProducts].sort();
  const datasets = products.map((prod, i) => ({
    label: prod,
    data: bucketLabels.map(b => Math.round((buckets[b][prod] || 0) * 100) / 100),
    backgroundColor: COLORS[i % COLORS.length], borderRadius: 2
  }));

  if (charts.drilldown) charts.drilldown.destroy();
  charts.drilldown = new Chart(document.getElementById('drilldownChart'), {
    type: 'bar',
    data: { labels: bucketLabels.map(getBucketLabel), datasets },
    options: {
      plugins: {
        legend: { labels: { color: '#8b949e', padding: 12 } },
        tooltip: { callbacks: { label: ctx => ctx.dataset.label + ': ' + ctx.parsed.y.toFixed(2) + ' DKK' } }
      },
      scales: {
        x: { stacked: true, ticks: { color: '#8b949e' }, grid: { display: false } },
        y: { stacked: true, ticks: { color: '#8b949e', callback: v => v + ' DKK' }, grid: { color: '#21262d' } }
      }
    }
  });

  card.scrollIntoView({ behavior: 'smooth' });
}

function closeDrilldown() {
  document.getElementById('drilldownCard').style.display = 'none';
  document.getElementById('stackedHint').textContent = 'Click a category in the legend to drill down into products';
  if (charts.drilldown) { charts.drilldown.destroy(); charts.drilldown = null; }
}
window.closeDrilldown = closeDrilldown;

// ── Table ──

function setTableMode(mode) {
  tableMode = mode;
  document.querySelectorAll('.table-controls .toggle-btn').forEach(b => b.classList.remove('active'));
  document.querySelector('.table-controls .toggle-btn[onclick*="setTableMode(\'' + mode + '\')"]').classList.add('active');
  expandedCats.clear();
  updateProductTable(currentFiltered);
}
window.setTableMode = setTableMode;

function updateProductTable(data) {
  const total = data.reduce((s, e) => s + e.price, 0);
  const search = document.getElementById('tableSearch').value.toLowerCase();

  const products = {};
  data.forEach(e => {
    if (!products[e.product]) products[e.product] = 0;
    products[e.product] += e.price;
  });

  const tbody = document.querySelector('#productTable tbody');

  if (tableMode === 'flat') {
    let entries = Object.entries(products).map(([name, t]) => ({
      name, category: catMap[name] || '-', total: t, pct: total > 0 ? t/total*100 : 0
    }));

    if (search) {
      entries = entries.filter(e => e.name.includes(search) || e.category.includes(search));
    }

    entries.sort((a, b) => {
      let va = a[tableSortCol] || a.name, vb = b[tableSortCol] || b.name;
      if (tableSortCol === 'name' || tableSortCol === 'category') {
        return tableSortAsc ? String(va).localeCompare(String(vb)) : String(vb).localeCompare(String(va));
      }
      return tableSortAsc ? va - vb : vb - va;
    });

    tbody.innerHTML = entries.map(e =>
      '<tr><td>' + e.name + '</td><td>' + e.category + '</td><td class="amount">' +
      e.total.toFixed(2) + '</td><td class="pct">' + e.pct.toFixed(1) + '%</td></tr>'
    ).join('');
  } else {
    const catGroups = {};
    Object.entries(products).forEach(([name, t]) => {
      const cat = catMap[name] || 'uncategorized';
      if (!catGroups[cat]) catGroups[cat] = { total: 0, products: [] };
      catGroups[cat].total += t;
      catGroups[cat].products.push({ name, total: t, pct: total > 0 ? t/total*100 : 0 });
    });

    let catEntries = Object.entries(catGroups).sort((a, b) => b[1].total - a[1].total);
    catEntries.forEach(([_, g]) => { g.products.sort((a, b) => b.total - a.total); });

    let html = '';
    catEntries.forEach(([cat, g]) => {
      const matchesCat = !search || cat.includes(search);
      const matchingProducts = g.products.filter(p => !search || p.name.includes(search) || cat.includes(search));
      if (!matchesCat && matchingProducts.length === 0) return;

      const expanded = expandedCats.has(cat);
      const icon = expanded ? '&#9660;' : '&#9654;';
      const catPct = total > 0 ? (g.total / total * 100).toFixed(1) : '0';

      html += '<tr class="cat-row" data-cat="' + cat + '"><td><span class="expand-icon">' + icon + '</span>' +
        cat + '</td><td></td><td class="amount">' + g.total.toFixed(2) +
        '</td><td class="pct">' + catPct + '%</td></tr>';

      const prods = search ? matchingProducts : g.products;
      prods.forEach(p => {
        html += '<tr class="product-row ' + (expanded ? '' : 'hidden') + '" data-cat="' + cat +
          '"><td>' + p.name + '</td><td>' + cat + '</td><td class="amount">' +
          p.total.toFixed(2) + '</td><td class="pct">' + p.pct.toFixed(1) + '%</td></tr>';
      });
    });
    tbody.innerHTML = html;

    document.querySelectorAll('.cat-row').forEach(row => {
      row.onclick = () => {
        const cat = row.dataset.cat;
        if (expandedCats.has(cat)) expandedCats.delete(cat); else expandedCats.add(cat);
        updateProductTable(currentFiltered);
      };
    });
  }

  document.querySelectorAll('#productTable th').forEach(th => {
    const col = th.dataset.sort;
    const arrow = th.querySelector('.sort-arrow');
    if (col === tableSortCol) {
      arrow.textContent = tableSortAsc ? ' ▲' : ' ▼';
    } else {
      arrow.textContent = '';
    }
  });
}

// ── Sort handler ──

document.querySelectorAll('#productTable th').forEach(th => {
  th.onclick = () => {
    const col = th.dataset.sort;
    if (!col) return;
    if (tableSortCol === col) { tableSortAsc = !tableSortAsc; } else { tableSortCol = col; tableSortAsc = false; }
    updateProductTable(currentFiltered);
  };
});

// ── Search handler ──

document.getElementById('tableSearch').addEventListener('input', () => updateProductTable(currentFiltered));

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

document.getElementById('dateFrom').addEventListener('change', () => { document.querySelectorAll('.preset-btn').forEach(b => b.classList.remove('active')); updateAll(); });
document.getElementById('dateTo').addEventListener('change', () => { document.querySelectorAll('.preset-btn').forEach(b => b.classList.remove('active')); updateAll(); });
document.getElementById('storeFilter').addEventListener('change', updateAll);
document.getElementById('categoryFilter').addEventListener('change', updateAll);

initPresets();
initFilters();
updateAll();
