<script>
  import { onMount } from 'svelte'
  import { api } from './api.js'

  let services = []
  let entries = []
  let totalHits = 0
  let loading = false
  let error = ''

  let filter = {
    service: '', level: '', q: '',
    from: '', to: '',
    limit: 100, offset: 0,
  }

  const LEVELS = ['TRACE','DEBUG','INFO','WARN','ERROR','FATAL']
  const LEVEL_COLORS = {
    TRACE: { bg: '#1a1a2e', fg: '#6b7280' },
    DEBUG: { bg: '#1a1a2e', fg: '#8b949e' },
    INFO:  { bg: '#0c2d48', fg: '#58a6ff' },
    WARN:  { bg: '#2d2305', fg: '#f0b429' },
    ERROR: { bg: '#3b1219', fg: '#f87171' },
    FATAL: { bg: '#450a0a', fg: '#fca5a5' },
  }

  const TIME_RANGES = [
    { label: '15m', minutes: 15 },
    { label: '1h',  minutes: 60 },
    { label: '6h',  minutes: 360 },
    { label: '24h', minutes: 1440 },
    { label: '7d',  minutes: 10080 },
  ]
  let activeRange = ''

  onMount(async () => {
    try {
      const r = await api.services()
      services = r.services || []
    } catch {}
    await search()
  })

  function setTimeRange(range) {
    activeRange = range.label
    const now = new Date()
    const from = new Date(now.getTime() - range.minutes * 60000)
    filter.from = toLocalISO(from)
    filter.to = ''
    search()
  }

  function toLocalISO(date) {
    const pad = n => String(n).padStart(2, '0')
    return `${date.getFullYear()}-${pad(date.getMonth()+1)}-${pad(date.getDate())}T${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
  }

  function clearTimeRange() {
    activeRange = ''
    filter.from = ''
    filter.to = ''
    search()
  }

  async function search(resetOffset = true) {
    if (resetOffset) filter.offset = 0
    loading = true; error = ''
    try {
      const r = await api.logs({ ...filter })
      entries = r.entries || []
      totalHits = r.total_hits || 0
    } catch (e) {
      error = e.message
    } finally {
      loading = false
    }
  }

  function nextPage() { filter.offset += filter.limit; search(false) }
  function prevPage() { filter.offset = Math.max(0, filter.offset - filter.limit); search(false) }

  function fmtTime(ts) {
    return new Date(ts).toISOString().replace('T',' ').replace('Z','').slice(0,23)
  }

  let expanded = new Set()
  function toggle(id) {
    if (expanded.has(id)) expanded.delete(id)
    else expanded.add(id)
    expanded = expanded
  }


  $: currentPage = Math.floor(filter.offset / filter.limit) + 1
  $: totalPages = Math.ceil(totalHits / filter.limit) || 1

  let dariaToast = false
  let dariaTimer
  $: if (filter.q === 'daria') {
    clearTimeout(dariaTimer)
    dariaToast = true
    dariaTimer = setTimeout(() => { dariaToast = false }, 3500)
  }
</script>

<div class="logs">
  <!-- Filters bar -->
  <div class="toolbar">
    <div class="filter-group">
      <div class="filter-item">
        <label for="log-service">Service</label>
        <select id="log-service" bind:value={filter.service} on:change={() => search()}>
          <option value="">All services</option>
          {#each services as s}<option value={s}>{s}</option>{/each}
        </select>
      </div>

      <div class="filter-item">
        <label for="log-level">Level</label>
        <select id="log-level" bind:value={filter.level} on:change={() => search()}>
          <option value="">All levels</option>
          {#each LEVELS as l}<option value={l}>{l}</option>{/each}
        </select>
      </div>

      <div class="filter-item search-item">
        <label for="log-search">Search</label>
        <div class="search-wrap">
          <svg class="search-icon" viewBox="0 0 16 16" fill="none"><circle cx="6.5" cy="6.5" r="4.5" stroke="currentColor" stroke-width="1.2"/><path d="M10 10l4 4" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>
          <input id="log-search" type="text" bind:value={filter.q}
            placeholder="Full-text search…"
            on:keydown={e => e.key === 'Enter' && search()} />
        </div>
      </div>

      <button class="btn-primary" on:click={() => search()}>
        Search
      </button>
    </div>

    <div class="time-bar">
      <div class="time-ranges">
        {#each TIME_RANGES as range}
          <button class="range-btn" class:active={activeRange === range.label}
            on:click={() => setTimeRange(range)}>{range.label}</button>
        {/each}
        {#if activeRange || filter.from || filter.to}
          <button class="range-btn clear" aria-label="Clear time range" on:click={clearTimeRange}>
            <svg viewBox="0 0 12 12" fill="none"><path d="M3 3l6 6M9 3l-6 6" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>
          </button>
        {/if}
      </div>

      <div class="time-inputs">
        <input type="datetime-local" bind:value={filter.from} step="1"
          on:change={() => { activeRange = ''; search() }} />
        <span class="time-sep">—</span>
        <input type="datetime-local" bind:value={filter.to} step="1"
          on:change={() => { activeRange = ''; search() }} />
      </div>

      <span class="hits">
        <strong>{totalHits.toLocaleString()}</strong> hits
      </span>
    </div>
  </div>

  {#if error}<div class="error-bar"><svg viewBox="0 0 16 16" fill="none"><circle cx="8" cy="8" r="6" stroke="currentColor" stroke-width="1.2"/><path d="M8 5v3M8 10v.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>{error}</div>{/if}

  <!-- Table -->
  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th style="width:170px">Timestamp</th>
          <th style="width:72px">Level</th>
          <th style="width:130px">Service</th>
          <th>Message</th>
          <th style="width:28px"></th>
        </tr>
      </thead>
      <tbody>
        {#if loading}
          <tr><td colspan="5" class="state-cell">
            <div class="loading-indicator">
              <div class="spinner"></div>
              Loading…
            </div>
          </td></tr>
        {:else if entries.length === 0}
          <tr><td colspan="5" class="state-cell">
            <div class="empty-state">
              <svg viewBox="0 0 24 24" fill="none"><path d="M9 12h6M12 9v6" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><rect x="3" y="3" width="18" height="18" rx="4" stroke="currentColor" stroke-width="1.5"/></svg>
              No results found
            </div>
          </td></tr>
        {:else}
          {#each entries as e, i (e.ID)}
            <tr class="row" class:even={i % 2 === 0} on:click={() => toggle(e.ID)}>
              <td class="ts">{fmtTime(e.Timestamp)}</td>
              <td>
                <span class="level-badge" style="background:{(LEVEL_COLORS[e.Level] || LEVEL_COLORS.DEBUG).bg};color:{(LEVEL_COLORS[e.Level] || LEVEL_COLORS.DEBUG).fg}">{e.Level}</span>
              </td>
              <td class="service-cell">{e.Service}</td>
              <td class="body-cell">
                <span class="body-text">{e.Body}</span>
                {#if e.TraceID && e.TraceID !== '00000000000000000000000000000000'}
                  <a class="trace-badge" href="#/traces/{e.TraceID}" on:click|stopPropagation>
                    <svg viewBox="0 0 12 12" fill="none"><path d="M2 4h3M5 7h4M3 10h6" stroke="currentColor" stroke-width="1" stroke-linecap="round"/></svg>
                    {e.TraceID.slice(0,8)}
                  </a>
                {/if}
              </td>
              <td class="expand-cell">{expanded.has(e.ID) ? '▾' : '▸'}</td>
            </tr>
            {#if expanded.has(e.ID)}
              <tr class="detail-row">
                <td colspan="5">
                  <div class="detail-inner">
                    {#if e.TraceID && e.TraceID !== '00000000000000000000000000000000'}
                      <div class="detail-item">
                        <span class="detail-key">trace_id</span>
                        <a class="detail-link" href="#/traces/{e.TraceID}">{e.TraceID}</a>
                      </div>
                    {/if}
                    {#if e.SpanID}
                      <div class="detail-item">
                        <span class="detail-key">span_id</span>
                        <span class="detail-val mono">{e.SpanID}</span>
                      </div>
                    {/if}
                    {#if e.Host}
                      <div class="detail-item">
                        <span class="detail-key">host</span>
                        <span class="detail-val">{e.Host}</span>
                      </div>
                    {/if}
                    {#each (e.Attrs || []) as attr}
                      <div class="detail-item">
                        <span class="detail-key">{attr.Key}</span>
                        <span class="detail-val">{attr.Value}</span>
                      </div>
                    {/each}
                  </div>
                </td>
              </tr>
            {/if}
          {/each}
        {/if}
      </tbody>
    </table>
  </div>

  <!-- Pagination -->
  <div class="pagination">
    <button on:click={prevPage} disabled={filter.offset === 0}>
      <svg viewBox="0 0 12 12" fill="none"><path d="M7 3L4 6l3 3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>
      Prev
    </button>
    <span class="page-info">Page {currentPage} of {totalPages}</span>
    <button on:click={nextPage} disabled={entries.length < filter.limit}>
      Next
      <svg viewBox="0 0 12 12" fill="none"><path d="M5 3l3 3-3 3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>
    </button>
  </div>

  {#if dariaToast}
    <div class="daria-toast">тема фильтров ещё не раскрыта</div>
  {/if}
</div>

<style>
  .logs { display: flex; flex-direction: column; height: 100%; overflow: hidden; position: relative; }

  /* --- Toolbar --- */
  .toolbar {
    display: flex; flex-direction: column; gap: 0;
    background: #0d1117; border-bottom: 1px solid #1b2230; flex-shrink: 0;
  }

  .filter-group {
    display: flex; align-items: flex-end; gap: 10px;
    padding: 12px 16px 10px;
    flex-wrap: wrap;
  }
  .filter-item { display: flex; flex-direction: column; gap: 3px; }
  .filter-item label {
    font-size: 10px; font-weight: 600; text-transform: uppercase;
    letter-spacing: 0.06em; color: #6b7994;
  }
  .filter-item select, .time-inputs input {
    background: #161b22; border: 1px solid #21262d; color: #c9d1d9;
    border-radius: 6px; padding: 6px 10px; font-size: 12px; font-family: inherit;
    transition: border-color 0.15s;
  }
  .filter-item select:focus, .time-inputs input:focus {
    outline: none; border-color: #f59e0b44;
  }

  .search-item { flex: 1; min-width: 200px; }
  .search-wrap {
    position: relative; display: flex; align-items: center;
  }
  .search-icon {
    position: absolute; left: 10px; width: 13px; height: 13px;
    color: #6b7994; pointer-events: none;
  }
  .search-wrap input {
    width: 100%;
    background: #161b22; border: 1px solid #21262d; color: #c9d1d9;
    border-radius: 6px; padding: 6px 10px 6px 30px; font-size: 12px; font-family: inherit;
    transition: border-color 0.15s;
  }
  .search-wrap input:focus { outline: none; border-color: #f59e0b44; }
  .search-wrap input::placeholder { color: #4a5568; }


  .btn-primary {
    background: #f59e0b; border: none; color: #0a0e14;
    border-radius: 6px; padding: 6px 18px; cursor: pointer;
    font-family: inherit; font-size: 12px; font-weight: 600;
    transition: background 0.15s;
    align-self: flex-end;
  }
  .btn-primary:hover { background: #fbbf24; }

  .time-bar {
    display: flex; align-items: center; gap: 12px;
    padding: 6px 16px 10px; flex-wrap: wrap;
  }
  .time-ranges { display: flex; gap: 2px; }
  .range-btn {
    background: transparent; border: 1px solid #21262d; color: #8b949e;
    border-radius: 5px; padding: 3px 10px; cursor: pointer;
    font-family: inherit; font-size: 11px; font-weight: 500;
    transition: all 0.15s;
  }
  .range-btn:hover { background: #161b22; color: #c9d1d9; border-color: #30363d; }
  .range-btn.active { background: #f59e0b22; color: #f59e0b; border-color: #f59e0b44; }
  .range-btn.clear { padding: 3px 6px; }
  .range-btn.clear svg { width: 10px; height: 10px; }

  .time-inputs { display: flex; align-items: center; gap: 6px; }
  .time-inputs input { font-size: 11px; padding: 3px 8px; }
  .time-sep { color: #4a5568; font-size: 11px; }

  .hits { color: #6b7994; font-size: 12px; margin-left: auto; }
  .hits strong { color: #c9d1d9; }

  /* --- Error --- */
  .error-bar {
    display: flex; align-items: center; gap: 8px;
    padding: 8px 16px; color: #f87171; background: #1c0d0d;
    border-bottom: 1px solid #3b1219; font-size: 12px; flex-shrink: 0;
  }
  .error-bar svg { width: 14px; height: 14px; flex-shrink: 0; }

  /* --- Table --- */
  .table-wrap { flex: 1; overflow: auto; }
  table { width: 100%; border-collapse: collapse; }

  thead th {
    position: sticky; top: 0; z-index: 1;
    background: #0d1117; padding: 8px 12px;
    text-align: left; color: #6b7994; font-weight: 600;
    border-bottom: 1px solid #1b2230;
    font-size: 11px; text-transform: uppercase; letter-spacing: 0.05em;
  }

  tbody tr.row {
    cursor: pointer;
    border-bottom: 1px solid #0f1318;
    transition: background 0.1s;
  }
  tbody tr.row.even { background: rgba(255,255,255,0.01); }
  tbody tr.row:hover { background: #161b22; }
  tbody td { padding: 7px 12px; vertical-align: top; }

  .ts {
    font-family: 'SF Mono', 'Fira Code', 'JetBrains Mono', monospace;
    font-size: 11px; color: #6b7994; white-space: nowrap;
  }

  .level-badge {
    display: inline-block; font-size: 10px; font-weight: 700;
    padding: 2px 8px; border-radius: 4px;
    letter-spacing: 0.04em; text-align: center;
    min-width: 48px;
  }

  .service-cell { color: #8b949e; font-size: 12px; }

  .body-cell { display: flex; align-items: flex-start; gap: 8px; }
  .body-text { flex: 1; word-break: break-word; line-height: 1.5; }

  .trace-badge {
    display: inline-flex; align-items: center; gap: 4px;
    padding: 1px 8px; border-radius: 4px;
    background: #0c2d48; color: #58a6ff;
    font-family: 'SF Mono', 'Fira Code', monospace; font-size: 10px;
    text-decoration: none; flex-shrink: 0; white-space: nowrap;
    transition: background 0.15s;
  }
  .trace-badge svg { width: 10px; height: 10px; }
  .trace-badge:hover { background: #163d5c; }

  .expand-cell { color: #4a5568; text-align: center; user-select: none; font-size: 10px; }

  /* --- Detail --- */
  .detail-row td { padding: 0; }
  .detail-inner {
    padding: 10px 16px 12px 48px;
    background: #0b0f14; border-bottom: 1px solid #1b2230;
  }
  .detail-item {
    display: flex; gap: 16px; padding: 3px 0;
    font-size: 12px; line-height: 1.4;
  }
  .detail-key {
    color: #6b7994; min-width: 90px; flex-shrink: 0;
    font-family: 'SF Mono', 'Fira Code', monospace; font-size: 11px;
  }
  .detail-val { color: #c9d1d9; word-break: break-all; }
  .detail-link {
    color: #58a6ff; text-decoration: none;
    font-family: 'SF Mono', 'Fira Code', monospace; font-size: 11px;
  }
  .detail-link:hover { text-decoration: underline; }
  .mono { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 11px; }

  /* --- States --- */
  .state-cell { text-align: center; padding: 48px; }
  .loading-indicator {
    display: flex; align-items: center; justify-content: center; gap: 10px;
    color: #6b7994; font-size: 13px;
  }
  .spinner {
    width: 16px; height: 16px;
    border: 2px solid #21262d; border-top-color: #f59e0b;
    border-radius: 50%; animation: spin 0.6s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .empty-state {
    display: flex; flex-direction: column; align-items: center; gap: 10px;
    color: #4a5568; font-size: 13px;
  }
  .empty-state svg { width: 32px; height: 32px; }

  /* --- Pagination --- */
  .pagination {
    display: flex; align-items: center; gap: 12px; justify-content: center;
    padding: 10px 16px; border-top: 1px solid #1b2230; flex-shrink: 0;
    background: #0d1117;
  }
  .pagination button {
    display: flex; align-items: center; gap: 4px;
    background: #161b22; border: 1px solid #21262d; color: #c9d1d9;
    border-radius: 6px; padding: 5px 12px; cursor: pointer;
    font-family: inherit; font-size: 12px;
    transition: all 0.15s;
  }
  .pagination button svg { width: 10px; height: 10px; }
  .pagination button:disabled { opacity: 0.3; cursor: default; }
  .pagination button:not(:disabled):hover { background: #21262d; border-color: #30363d; }
  .page-info { color: #6b7994; font-size: 12px; }

  .daria-toast {
    position: absolute; bottom: 56px; left: 50%; transform: translateX(-50%);
    background: #1b2230; border: 1px solid #f59e0b44; color: #f59e0b;
    padding: 10px 20px; border-radius: 8px; font-size: 13px;
    white-space: nowrap; pointer-events: none;
    animation: toast-in 0.2s ease;
    z-index: 100;
  }
  @keyframes toast-in {
    from { opacity: 0; transform: translateX(-50%) translateY(6px); }
    to   { opacity: 1; transform: translateX(-50%) translateY(0); }
  }
</style>
