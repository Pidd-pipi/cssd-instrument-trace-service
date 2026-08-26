/**
 * StageBadge：环节徽标组件。
 * 被工作台总览与器械包列表共用（也可用于灭菌批次去向等）。
 */
window.CSSD = window.CSSD || {};

CSSD.StageBadge = function (stage) {
  const info = CSSD.constants.PackStage[stage] || { label: stage || '未知', color: '#6b7785' };
  return '<span class="badge" style="background:' + info.color + '" title="' + CSSD.esc(info.label) + '">'
    + CSSD.esc(info.icon ? info.icon + ' ' : '') + CSSD.esc(info.label) + '</span>';
};

/** SterilizeResultBadge：灭菌结果徽标。 */
CSSD.ResultBadge = function (result) {
  if (!result) return '<span class="pill">未判定</span>';
  const info = CSSD.constants.SterilizeResult[result] || { label: result, color: '#6b7785' };
  return '<span class="badge" style="background:' + info.color + '">' + CSSD.esc(info.label) + '</span>';
};

/** BatchStatusBadge：批次状态徽标。 */
CSSD.BatchStatusBadge = function (status) {
  const info = CSSD.constants.BatchStatus[status] || { label: status || '未知', color: '#6b7785' };
  return '<span class="badge" style="background:' + info.color + '">' + CSSD.esc(info.label) + '</span>';
};

/** IssueStatusBadge：发放记录状态徽标。 */
CSSD.IssueStatusBadge = function (status) {
  const info = CSSD.constants.IssueStatus[status] || { label: status || '未知', color: '#6b7785' };
  return '<span class="badge" style="background:' + info.color + '">' + CSSD.esc(info.label) + '</span>';
};
