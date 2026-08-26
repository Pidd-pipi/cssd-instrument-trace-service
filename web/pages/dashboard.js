/**
 * 工作台总览页（/）：各环节在途器械包计数 + 灭菌失败拦截 + 丢失待查。
 * 共用 StageBadge 与 usePacks Hook。
 */
window.CSSD = window.CSSD || {};
CSSD.pages = CSSD.pages || {};

CSSD.pages.dashboard = function (container) {
  container.innerHTML = '<div class="loading">加载工作台数据...</div>';

  let lost = [];
  let dash = null;
  // 注意：usePacks 会同步触发一次 render（加载态），因此 dash/lost 必须先声明。
  const hook = CSSD.usePacks({ stage: 'washed', limit: 100 }, render);

  async function loadMeta() {
    try {
      dash = await CSSD.api.dashboard();
      lost = await CSSD.api.lostList();
    } catch (e) {
      dash = null;
      lost = [];
    }
    render();
  }

  function render() {
    const stages = Object.keys(CSSD.constants.PackStage);
    const byStage = (dash && dash.byStage) || {};
    const failedPacks = (hook.data || []).filter((p) => p.lastBatchResult === 'fail');

    const cards = stages.map((st) => {
      const info = CSSD.constants.PackStage[st];
      return '<div class="card stat-card">'
        + '<div class="num">' + (byStage[st] || 0) + '</div>'
        + '<div class="lbl">' + CSSD.StageBadge(st) + '</div>'
        + '</div>';
    }).join('');

    container.innerHTML = `
      <div class="section-title">
        <h2>工作台总览</h2>
        <span class="muted small">数据实时来自后端接口</span>
      </div>
      <div class="grid cols-3">
        <div class="card stat-card"><div class="num">${(dash && dash.totalPacks) || 0}</div><div class="lbl">器械包总数</div></div>
        <div class="card stat-card"><div class="num" style="color:var(--success)">${(dash && dash.sterilizedAvailable) || 0}</div><div class="lbl">可发放（已灭菌）</div></div>
        <div class="card stat-card"><div class="num" style="color:${failedPacks.length > 0 ? 'var(--danger)' : 'var(--text)'}">${(dash && dash.failedIntercepted) || 0}</div><div class="lbl">灭菌失败拦截</div></div>
      </div>
      <div class="grid cols-4">${cards}</div>

      <div class="grid cols-2">
        <div class="card">
          <h3>⛔ 灭菌失败拦截清单</h3>
          ${failedPacks.length === 0 ? '<div class="empty">暂无拦截器械包</div>' : `
          <table><thead><tr><th>条码</th><th>名称</th><th>环节</th><th>失败原因</th></tr></thead><tbody>
          ${failedPacks.map((p) => `
            <tr>
              <td class="mono">${CSSD.esc(p.barcode)}</td>
              <td>${CSSD.esc(p.name)}</td>
              <td>${CSSD.StageBadge(p.stage)}</td>
              <td class="small muted">${CSSD.esc(p.lastFailedReason || '-')}</td>
            </tr>`).join('')}
          </tbody></table>`}
        </div>

        <div class="card">
          <h3>🔍 丢失待查（发放超时未回收）</h3>
          ${lost.length === 0 ? '<div class="empty">暂无丢失待查器械包</div>' : `
          <table><thead><tr><th>条码</th><th>科室</th><th>手术间</th><th>超时(小时)</th></tr></thead><tbody>
          ${lost.map((e) => `
            <tr>
              <td class="mono">${CSSD.esc(e.pack.barcode)}</td>
              <td>${CSSD.esc(e.issue.department)}</td>
              <td>${CSSD.esc(e.issue.operatingRoom)}</td>
              <td class="small" style="color:var(--danger)">${Math.round(e.overdueHours)}h</td>
            </tr>`).join('')}
          </tbody></table>`}
        </div>
      </div>

      <div class="card">
        <h3>🕐 最近登记器械包</h3>
        ${hook.loading ? '<div class="empty">加载中...</div>' : (hook.error ? `<div class="alert error">${CSSD.esc(hook.error)}</div>` : renderRecentPacks((dash && dash.recentPacks) || []))}
      </div>
    `;
  }

  function renderRecentPacks(packs) {
    if (!packs.length) return '<div class="empty">暂无器械包</div>';
    return `<table><thead><tr><th>条码</th><th>名称</th><th>类型</th><th>环节</th><th>有效期至</th><th>更新时间</th></tr></thead><tbody>
      ${packs.map((p) => `
        <tr>
          <td class="mono">${CSSD.esc(p.barcode)}</td>
          <td>${CSSD.esc(p.name)}</td>
          <td>${CSSD.esc((CSSD.constants.PackType[p.packType] || p.packType))}</td>
          <td>${CSSD.StageBadge(p.stage)}</td>
          <td class="small">${CSSD.esc(CSSD.fmtTime(p.expiryAt))}</td>
          <td class="small muted">${CSSD.esc(CSSD.fmtTime(p.updatedAt))}</td>
        </tr>`).join('')}
    </tbody></table>`;
  }

  loadMeta();
};
