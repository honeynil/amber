<script>
  import { onMount } from 'svelte'
  import { api } from './api.js'

  let services = []
  let traces = []
  let total = 0
  let loading = false
  let error = ''

  let filter = { service: '', from: '', to: '', limit: 20, offset: 0 }

  const TIME_RANGES = [
    { label: '15m', minutes: 15 },
    { label: '1h',  minutes: 60 },
    { label: '6h',  minutes: 360 },
    { label: '24h', minutes: 1440 },
    { label: '7d',  minutes: 10080 },
  ]
  let activeRange = ''

  onMount(async () => {
    try { const r = await api.services(); services = r.services || [] } catch {}
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
      const r = await api.traces({ ...filter })
      traces = r.traces || []
      total = r.total || 0
    } catch (e) { error = e.message }
    finally { loading = false }
  }

  function nextPage() { filter.offset += filter.limit; search(false) }
  function prevPage() { filter.offset = Math.max(0, filter.offset - filter.limit); search(false) }

  function fmtTime(ts) {
    return new Date(ts).toISOString().replace('T',' ').replace('Z','').slice(0,19)
  }
  function fmtDuration(ms) {
    if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`
    if (ms < 1000) return `${ms.toFixed(1)}ms`
    return `${(ms / 1000).toFixed(2)}s`
  }

  $: maxDuration = Math.max(...traces.map(t => t.duration_ms), 1)
  $: currentPage = Math.floor(filter.offset / filter.limit) + 1
  $: totalPages = Math.ceil(total / filter.limit) || 1
</script>

<div class="traces">
  <div class="toolbar">
    <div class="filter-group">
      <div class="filter-item">
        <label for="trace-service">Service</label>
        <select id="trace-service" bind:value={filter.service} on:change={() => search()}>
          <option value="">All services</option>
          {#each services as s}<option value={s}>{s}</option>{/each}
        </select>
      </div>

      <button class="btn-primary" on:click={() => search()}>Search</button>
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

      <span class="hits"><strong>{total.toLocaleString()}</strong> traces</span>
    </div>
  </div>

  {#if error}
    <div class="error-bar">
      <svg viewBox="0 0 16 16" fill="none"><circle cx="8" cy="8" r="6" stroke="currentColor" stroke-width="1.2"/><path d="M8 5v3M8 10v.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>
      {error}
    </div>
  {/if}

  <div class="table-wrap">
    <table>
      <thead>
        <tr>
          <th style="width:130px">Service</th>
          <th>Operation</th>
          <th style="width:170px">Start time</th>
          <th style="width:200px">Duration</th>
          <th style="width:70px">Spans</th>
          <th style="width:80px">Status</th>
        </tr>
      </thead>
      <tbody>
        {#if loading}
          <tr><td colspan="6" class="state-cell">
            <div class="loading-indicator"><div class="spinner"></div>Loading…</div>
          </td></tr>
        {:else if traces.length === 0}
          <tr><td colspan="6" class="state-cell">
            <div class="empty-state">
              <svg viewBox="0 0 24 24" fill="none"><path d="M3 7h4M7 12h8M5 17h12" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/><circle cx="5" cy="12" r="1.5" fill="currentColor"/></svg>
              No traces found
            </div>
          </td></tr>
        {:else}
          {#each traces as t, i}
            <tr class="row" class:even={i % 2 === 0} on:click={() => location.hash = `#/traces/${t.trace_id}`}>
              <td class="service-cell">{t.service}</td>
              <td class="op-cell">{t.operation}</td>
              <td class="ts">{fmtTime(t.start_time)}</td>
              <td class="duration-cell">
                <div class="dur-bar-wrap">
                  <div class="dur-bar" class:error={t.has_errors}
                    style="width:{Math.max((t.duration_ms / maxDuration) * 100, 2).toFixed(1)}%"></div>
                </div>
                <span class="dur-text">{fmtDuration(t.duration_ms)}</span>
              </td>
              <td class="spans-cell">{t.span_count}</td>
              <td>
                <span class="status-badge" class:status-error={t.has_errors}>
                  {#if t.has_errors}
                    <svg viewBox="0 0 12 12" fill="none"><circle cx="6" cy="6" r="4" stroke="currentColor" stroke-width="1.2"/><path d="M6 4v2.5M6 8v.5" stroke="currentColor" stroke-width="1" stroke-linecap="round"/></svg>
                  {:else}
                    <svg viewBox="0 0 12 12" fill="none"><path d="M3.5 6l2 2 3-3.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/></svg>
                  {/if}
                  {t.has_errors ? 'ERROR' : 'OK'}
                </span>
              </td>
            </tr>
          {/each}
        {/if}
      </tbody>
    </table>
  </div>

  <div class="pagination">
    <button on:click={prevPage} disabled={filter.offset === 0}>
      <svg viewBox="0 0 12 12" fill="none"><path d="M7 3L4 6l3 3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>
      Prev
    </button>
    <span class="page-info">Page {currentPage} of {totalPages}</span>
    <button on:click={nextPage} disabled={traces.length < filter.limit}>
      Next
      <svg viewBox="0 0 12 12" fill="none"><path d="M5 3l3 3-3 3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>
    </button>
  </div>
</div>

<style>
  .traces { display: flex; flex-direction: column; height: 100%; overflow: hidden; }

  /* --- Toolbar --- */
  .toolbar {
    display: flex; flex-direction: column;
    background: #0d1117; border-bottom: 1px solid #1b2230; flex-shrink: 0;
  }
  .filter-group {
    display: flex; align-items: flex-end; gap: 10px;
    padding: 12px 16px 10px; flex-wrap: wrap;
  }
  .filter-item { display: flex; flex-direction: column; gap: 3px; }
  .filter-item label {
    font-size: 10px; font-weight: 600; text-transform: uppercase;
    letter-spacing: 0.06em; color: #6b7994;
  }
  .filter-item select {
    background: #161b22; border: 1px solid #21262d; color: #c9d1d9;
    border-radius: 6px; padding: 6px 10px; font-size: 12px; font-family: inherit;
    transition: border-color 0.15s;
  }
  .filter-item select:focus { outline: none; border-color: #f59e0b44; }

  .btn-primary {
    background: #f59e0b; border: none; color: #0a0e14;
    border-radius: 6px; padding: 6px 18px; cursor: pointer;
    font-family: inherit; font-size: 12px; font-weight: 600;
    transition: background 0.15s; align-self: flex-end;
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
  .time-inputs input {
    background: #161b22; border: 1px solid #21262d; color: #c9d1d9;
    border-radius: 6px; padding: 3px 8px; font-size: 11px; font-family: inherit;
    transition: border-color 0.15s;
  }
  .time-inputs input:focus { outline: none; border-color: #f59e0b44; }
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
    cursor: pointer; border-bottom: 1px solid #0f1318;
    transition: background 0.1s;
  }
  tbody tr.row.even { background: rgba(255,255,255,0.01); }
  tbody tr.row:hover { background: #161b22; }
  tbody td { padding: 8px 12px; }

  .service-cell { color: #c9d1d9; font-weight: 500; }
  .op-cell { color: #a78bfa; }
  .ts {
    font-family: 'SF Mono', 'Fira Code', 'JetBrains Mono', monospace;
    font-size: 11px; color: #6b7994; white-space: nowrap;
  }

  .duration-cell {
    display: flex; align-items: center; gap: 8px;
  }
  .dur-bar-wrap {
    flex: 1; height: 6px; background: #161b22; border-radius: 3px; overflow: hidden;
  }
  .dur-bar {
    height: 100%; border-radius: 3px;
    background: #3b82f6; transition: width 0.2s ease;
  }
  .dur-bar.error { background: #ef4444; }
  .dur-text {
    font-family: 'SF Mono', 'Fira Code', monospace;
    font-size: 11px; color: #8b949e; width: 60px; text-align: right; flex-shrink: 0;
  }

  .spans-cell {
    text-align: center; color: #8b949e;
    font-family: 'SF Mono', monospace; font-size: 12px;
  }

  .status-badge {
    display: inline-flex; align-items: center; gap: 4px;
    font-size: 10px; font-weight: 700; padding: 3px 8px; border-radius: 4px;
    background: #0b2e1a; color: #4ade80;
    letter-spacing: 0.03em;
  }
  .status-badge svg { width: 11px; height: 11px; }
  .status-badge.status-error { background: #3b1219; color: #f87171; }

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
    font-family: inherit; font-size: 12px; transition: all 0.15s;
  }
  .pagination button svg { width: 10px; height: 10px; }
  .pagination button:disabled { opacity: 0.3; cursor: default; }
  .pagination button:not(:disabled):hover { background: #21262d; border-color: #30363d; }
  .page-info { color: #6b7994; font-size: 12px; }
</style>
