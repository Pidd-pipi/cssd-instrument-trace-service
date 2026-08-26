/**
 * 灭菌管理页（/sterilization）：批次登记 + 参数录入 + 合格判定 + 批次去向。
 * 共用 StageBadge 与 usePacks Hook。
 */
window.CSSD = window.CSSD || {};
CSSD.pages = CSSD.pages || {};

CSSD.pages.sterilization = function (container) {
  container.innerHTML = '<div class="loading">加载灭菌数据...</div>';

  const washedHook = CSSD.usePacks({ stage: 'washed', limit: 500 });
  let batches = [];
  let sterilizers = [];
  let selectedPacks = new Set();

  async function load() {
    try {
      const [b, s] = await Promise.all([
        CSSD.api.listSterilizations(100),
        CSSD.api.listSterilizers()
      ]);
      batches = b || [];
      sterilizers = s || [];
    } catch (e) {
      CSSD.toast.error('加载数据失败: ' + e.message);
    }
    render();
  }

  function render() {
    const washed = washedHook.data || [];
    container.innerHTML = `
      <div class="section-title"><h2>灭菌管理</h2></div>

      <div class="grid cols-2">
        <div class="card">
          <h3>🔥 创建灭菌批次</h3>
          <div class="form-row"><label>灭菌器 *</label>
            <select id="sb-sterilizer">
              ${sterilizers.map((s) => `<option value="${CSSD.esc(s.id)}">${CSSD.esc(s.name)}（${CSSD.esc(CSSD.constants.SterilizerStatus[s.status] ? CSSD.constants.SterilizerStatus[s.status].label : s.status)}）</option>`).join('')}
            </select>
          </div>
          <div class="form-grid">
            <div class="form-row"><label>灭菌温度 ℃ *</label><input type="number" step="0.1" id="sb-temp" value="134"></div>
            <div class="form-row"><label>灭菌时长 分钟 *</label><input type="number" id="sb-duration" value="6"></div>
            <div class="form-row"><label>灭菌压力 kPa *</label><input type="number" step="0.1" id="sb-pressure" value="210"></div>
            <div class="form-row"><label>操作人</label><input id="sb-operator" placeholder="如：李护士"></div>
          </div>
          <div class="alert warn small">参数判定下限：温度 ≥ ${CSSD.constants.limits.minTempC}℃、时长 ≥ ${CSSD.constants.limits.minDurationMin} 分钟、压力 ≥ ${CSSD.constants.limits.minPressureKPa}kPa；任一项不达标整批拦截。</div>
          <div class="form-row">
            <label>待灭菌器械包（已清洗，共 ${washed.length} 个）</label>
            <div id="sb-packlist" style="max-height:220px;overflow:auto;border:1px solid var(--border);border-radius:8px;padding:8px">${renderPackChecks(washed)}</div>
          </div>
          <button class="btn primary" id="sb-create">创建批次并装载灭菌</button>
        </div>

        <div class="card">
          <h3>灭菌器设备</h3>
          <table><thead><tr><th>名称</th><th>型号</th><th>状态</th></tr></thead><tbody>
            ${sterilizers.map((s) => `
              <tr><td>${CSSD.esc(s.name)}</td><td class="small muted">${CSSD.esc(s.model || '-')}</td>
              <td><span class="badge" style="background:${CSSD.constants.SterilizerStatus[s.status] ? CSSD.constants.SterilizerStatus[s.status].color : '#6b7785'}">${CSSD.esc(CSSD.constants.SterilizerStatus[s.status] ? CSSD.constants.SterilizerStatus[s.status].label : s.status)}</span></td></tr>`).join('')}
          </tbody></table>
        </div>
      </div>

      <div class="card">
        <div class="section-title"><h3>灭菌批次列表</h3><span class="muted small">共 ${batches.length} 条</span></div>
        ${batches.length === 0 ? '<div class="empty">暂无灭菌批次</div>' : `
        <table><thead><tr>
          <th>批次号</th><th>灭菌器</th><th>温度/时长/压力</th><th>状态</th><th>结果</th><th>器械包数</th><th>创建时间</th><th>操作</th>
        </tr></thead><tbody>
        ${batches.map((b) => `
          <tr>
            <td class="mono">${CSSD.esc(b.batchNo)}</td>
            <td>${CSSD.esc(b.sterilizerName)}</td>
            <td class="small mono">${CSSD.esc(b.tempC)}℃ / ${CSSD.esc(b.durationMin)}min / ${CSSD.esc(b.pressureKPa)}kPa</td>
            <td>${CSSD.BatchStatusBadge(b.status)}</td>
            <td>${CSSD.ResultBadge(b.result)}</td>
            <td>${b.packIds ? b.packIds.length : 0}</td>
            <td class="small muted">${CSSD.esc(CSSD.fmtTime(b.createdAt))}</td>
            <td style="white-space:nowrap">
              ${b.status === 'pending' ? `<button class="btn small primary" data-act="complete" data-id="${CSSD.esc(b.id)}">完成判定</button>` : ''}
              <button class="btn small" data-act="packs" data-id="${CSSD.esc(b.id)}">器械包去向</button>
            </td>
          </tr>`).join('')}
        </tbody></table>`}
      </div>
    `;

    container.querySelector('#sb-packlist').addEventListener('change', (e) => {
      if (e.target.type === 'checkbox') {
        if (e.target.checked) selectedPacks.add(e.target.value);
        else selectedPacks.delete(e.target.value);
      }
    });
    container.querySelector('#sb-create').addEventListener('click', createBatch);
    container.querySelectorAll('button[data-act="complete"]').forEach((btn) => {
      btn.addEventListener('click', () => completeBatch(btn.dataset.id));
    });
    container.querySelectorAll('button[data-act="packs"]').forEach((btn) => {
      btn.addEventListener('click', () => showBatchPacks(btn.dataset.id));
    });
  }

  function renderPackChecks(packs) {
    if (!packs.length) return '<div class="muted small">暂无「已清洗」器械包，请先在器械包管理推进环节</div>';
    return packs.map((p) => `
      <label style="display:flex;gap:8px;align-items:center;padding:4px 0;font-weight:400">
        <input type="checkbox" value="${CSSD.esc(p.id)}"> <span class="mono">${CSSD.esc(p.barcode)}</span> ${CSSD.esc(p.name)}
      </label>`).join('');
  }

  async function createBatch() {
    const sterilizerId = container.querySelector('#sb-sterilizer').value;
    const tempC = parseFloat(container.querySelector('#sb-temp').value);
    const durationMin = parseInt(container.querySelector('#sb-duration').value, 10);
    const pressureKPa = parseFloat(container.querySelector('#sb-pressure').value);
    const operator = container.querySelector('#sb-operator').value.trim() || '系统';
    if (selectedPacks.size === 0) {
      CSSD.toast.error('请至少勾选一个待灭菌器械包');
      return;
    }
    try {
      const batch = await CSSD.api.createSterilization({
        sterilizerId: sterilizerId,
        operator: operator,
        tempC: tempC,
        durationMin: durationMin,
        pressureKPa: pressureKPa,
        packIds: Array.from(selectedPacks)
      });
      CSSD.toast.success('灭菌批次创建成功：' + batch.batchNo);
      selectedPacks.clear();
      washedHook.refresh();
      load();
    } catch (e) {
      CSSD.toast.error(e.message);
    }
  }

  async function completeBatch(id) {
    try {
      const batch = await CSSD.api.completeSterilization(id, { operator: '灭菌员' });
      const result = batch.result === 'pass' ? '✅ 参数合格，器械包已灭菌' : '⛔ 参数不合格，批次已拦截';
      CSSD.toast.success('批次 ' + batch.batchNo + ' 判定完成：' + result);
      load();
    } catch (e) {
      CSSD.toast.error(e.message);
    }
  }

  async function showBatchPacks(id) {
    const content = document.createElement('div');
    content.innerHTML = '<div class="loading">加载批次器械包...</div>';
    const m = CSSD.modal.open(CSSD.modal.build('批次器械包去向', content));
    try {
      const view = await CSSD.api.batchPacks(id);
      const b = view.batch;
      content.innerHTML = `
        <div class="kv">
          <div class="k">批次号</div><div class="mono">${CSSD.esc(b.batchNo)}</div>
          <div class="k">灭菌器</div><div>${CSSD.esc(b.sterilizerName)}</div>
          <div class="k">参数</div><div class="mono">${CSSD.esc(b.tempC)}℃ / ${CSSD.esc(b.durationMin)}min / ${CSSD.esc(b.pressureKPa)}kPa</div>
          <div class="k">结果</div><div>${CSSD.ResultBadge(b.result)} ${b.failReasons && b.failReasons.length ? '<span class="small" style="color:var(--danger)">' + CSSD.esc(b.failReasons.join('；')) + '</span>' : ''}</div>
        </div>
        <table style="margin-top:12px"><thead><tr><th>条码</th><th>名称</th><th>当前环节</th><th>最新去向</th></tr></thead><tbody>
        ${(view.packs || []).map((item) => `
          <tr>
            <td class="mono">${CSSD.esc(item.pack.barcode)}</td>
            <td>${CSSD.esc(item.pack.name)}</td>
            <td>${CSSD.StageBadge(item.pack.stage)}</td>
            <td class="small">${item.latestIssue ? CSSD.esc(item.latestIssue.department + ' ' + item.latestIssue.operatingRoom + '（' + CSSD.esc(CSSD.constants.IssueStatus[item.latestIssue.status] ? CSSD.constants.IssueStatus[item.latestIssue.status].label : item.latestIssue.status) + '）') : '未发放'}</td>
          </tr>`).join('')}
        </tbody></table>`;
    } catch (e) {
      content.innerHTML = '<div class="alert error">' + CSSD.esc(e.message) + '</div>';
    }
  }

  load();
};
