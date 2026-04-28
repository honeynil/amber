<script>
  import { onMount } from 'svelte'
  import { api } from './api.js'
  import SpanRow from './SpanRow.svelte'

  export let id = ''

  let trace = null
  let loading = false
  let error = ''
  let traceStart = 0
  let traceDuration = 1

  $: if (id) load()

  async function load() {
    if (!id) return
    loading = true; error = ''; trace = null
    try {
      trace = await api.trace(id)
      if (trace.tree?.length) {
        traceStart = findMin(trace.tree)
        traceDuration = Math.max(findMax(trace.tree) - traceStart, 1)
      }
    } catch (e) { error = e.message }
    finally { loading = false }
  }

  function findMin(nodes) {
    let min = Infinity
    for (const n of nodes) {
      const t = new Date(n.span.StartTime).getTime()
      if (t < min) min = t
      if (n.children?.length) { const c = findMin(n.children); if (c < min) min = c }
    }
    return min
  }
  function findMax(nodes) {
    let max = -Infinity
    for (const n of nodes) {
      const t = new Date(n.span.EndTime).getTime()
      if (t > max) max = t
      if (n.children?.length) { const c = findMax(n.children); if (c > max) max = c }
    }
    return max
  }

  function fmtDuration(ms) {
    if (ms < 1) return `${(ms * 1000).toFixed(0)}µs`
    if (ms < 1000) return `${ms.toFixed(1)}ms`
    return `${(ms / 1000).toFixed(2)}s`
  }

  function fmtTime(ts) {
    return new Date(ts).toISOString().replace('T',' ').replace('Z','').slice(0,23)
  }

  function countErrors(nodes) {
    let count = 0
    for (const n of nodes) {
      if (n.span.Status === 2 || n.span.Status === 'ERROR') count++
      if (n.children?.length) count += countErrors(n.children)
    }
    return count
  }

  $: rootService = trace?.tree?.[0]?.span?.Service || ''
  $: rootOp = trace?.tree?.[0]?.span?.Operation || ''
  $: errorCount = trace?.tree ? countErrors(trace.tree) : 0
</script>

<div class="trace-view">
  {#if loading}
    <div class="state-cell">
      <div class="loading-indicator"><div class="spinner"></div>Loading trace…</div>
    </div>
  {:else if error}
    <div class="error-bar">
      <svg viewBox="0 0 16 16" fill="none"><circle cx="8" cy="8" r="6" stroke="currentColor" stroke-width="1.2"/><path d="M8 5v3M8 10v.5" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>
      {error}
    </div>
  {:else if trace}
    <div class="header">
      <div class="header-top">
        <button class="back-btn" on:click={() => history.back()}>
          <svg viewBox="0 0 12 12" fill="none"><path d="M7 3L4 6l3 3" stroke="currentColor" stroke-width="1.3" stroke-linecap="round" stroke-linejoin="round"/></svg>
          Back
        </button>

        <div class="header-title">
          {#if rootService}
            <span class="h-service">{rootService}</span>
            <span class="h-sep">/</span>
          {/if}
          <span class="h-op">{rootOp}</span>
        </div>
      </div>

      <div class="header-meta">
        <div class="meta-item">
          <span class="meta-label">Trace ID</span>
          <span class="meta-value mono">{trace.trace_id}</span>
        </div>
        <div class="meta-item">
          <span class="meta-label">Duration</span>
          <span class="meta-value">{fmtDuration(traceDuration)}</span>
        </div>
        <div class="meta-item">
          <span class="meta-label">Spans</span>
          <span class="meta-value">{trace.span_count}</span>
        </div>
        <div class="meta-item">
          <span class="meta-label">Logs</span>
          <span class="meta-value">{trace.log_count}</span>
        </div>
        {#if errorCount > 0}
          <div class="meta-item error">
            <span class="meta-label">Errors</span>
            <span class="meta-value">{errorCount}</span>
          </div>
        {/if}
        <div class="meta-item">
          <span class="meta-label">Started</span>
          <span class="meta-value mono">{fmtTime(new Date(traceStart).toISOString())}</span>
        </div>
      </div>
    </div>

    <!-- Timeline ruler -->
    <div class="ruler">
      <div class="ruler-name">Service / Operation</div>
      <div class="ruler-timeline">
        <span class="ruler-mark">0ms</span>
        <span class="ruler-mark">{fmtDuration(traceDuration * 0.25)}</span>
        <span class="ruler-mark">{fmtDuration(traceDuration * 0.5)}</span>
        <span class="ruler-mark">{fmtDuration(traceDuration * 0.75)}</span>
        <span class="ruler-mark">{fmtDuration(traceDuration)}</span>
      </div>
    </div>

    <div class="timeline-wrap">
      <div class="spans">
        {#each trace.tree || [] as node}
          <SpanRow {node} depth={0} {traceStart} {traceDuration} />
        {/each}
      </div>
    </div>
  {/if}
</div>

<style>
  .trace-view { display: flex; flex-direction: column; height: 100%; overflow: hidden; }

  /* --- Header --- */
  .header {
    flex-shrink: 0; background: #0d1117; border-bottom: 1px solid #1b2230;
  }
  .header-top {
    display: flex; align-items: center; gap: 16px;
    padding: 12px 16px 8px;
  }
  .back-btn {
    display: flex; align-items: center; gap: 4px;
    background: #161b22; border: 1px solid #21262d; color: #c9d1d9;
    border-radius: 6px; padding: 5px 12px; cursor: pointer;
    font-size: 12px; font-family: inherit; transition: all 0.15s;
  }
  .back-btn svg { width: 10px; height: 10px; }
  .back-btn:hover { background: #21262d; border-color: #30363d; }

  .header-title {
    display: flex; align-items: center; gap: 6px;
    font-size: 15px;
  }
  .h-service { color: #8b949e; font-weight: 500; }
  .h-sep { color: #4a5568; }
  .h-op { color: #a78bfa; font-weight: 600; }

  .header-meta {
    display: flex; align-items: center; gap: 20px;
    padding: 0 16px 12px; flex-wrap: wrap;
  }
  .meta-item {
    display: flex; align-items: center; gap: 6px;
  }
  .meta-label {
    font-size: 10px; font-weight: 600; text-transform: uppercase;
    letter-spacing: 0.06em; color: #4a5568;
  }
  .meta-value { font-size: 12px; color: #c9d1d9; }
  .meta-item.error .meta-value { color: #f87171; font-weight: 600; }
  .meta-item.error .meta-label { color: #f8717188; }
  .mono { font-family: 'SF Mono', 'Fira Code', 'JetBrains Mono', monospace; font-size: 11px; }

  /* --- Ruler --- */
  .ruler {
    display: flex; align-items: center;
    background: #0d1117; border-bottom: 1px solid #1b2230;
    flex-shrink: 0; height: 28px;
  }
  .ruler-name {
    width: 320px; flex-shrink: 0;
    padding: 0 12px;
    font-size: 10px; font-weight: 600; text-transform: uppercase;
    letter-spacing: 0.06em; color: #4a5568;
  }
  .ruler-timeline {
    flex: 1; display: flex; justify-content: space-between;
    padding: 0 12px;
  }
  .ruler-mark {
    font-size: 9px; color: #4a5568;
    font-family: 'SF Mono', 'Fira Code', monospace;
  }

  /* --- Timeline --- */
  .timeline-wrap { flex: 1; overflow: auto; }
  .spans { min-width: 900px; }

  /* --- States --- */
  .state-cell {
    display: flex; align-items: center; justify-content: center;
    height: 100%;
  }
  .loading-indicator {
    display: flex; align-items: center; gap: 10px;
    color: #6b7994; font-size: 13px;
  }
  .spinner {
    width: 16px; height: 16px;
    border: 2px solid #21262d; border-top-color: #f59e0b;
    border-radius: 50%; animation: spin 0.6s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  .error-bar {
    display: flex; align-items: center; gap: 8px;
    padding: 12px 16px; color: #f87171; background: #1c0d0d;
    font-size: 12px;
  }
  .error-bar svg { width: 14px; height: 14px; flex-shrink: 0; }
</style>
