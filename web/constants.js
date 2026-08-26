/**
 * 前后端共享枚举/常量定义（与后端 domain/constants.go、domain/pack_stage.go 保持一致）。
 * - PackStage 环节枚举：to_collect/collected/washing/washed/sterilizing/sterilized/issued/in_use/expired
 * - SterilizeResult 灭菌结果：pass/fail
 * - PackType 包类型：surgical/dressing/instrument/implant
 */
window.CSSD = window.CSSD || {};

CSSD.constants = {
  PackStage: {
    to_collect:   { label: '待回收', color: '#8e99a6', icon: '📦' },
    collected:    { label: '已回收', color: '#5b8def', icon: '🧺' },
    washing:      { label: '清洗中', color: '#e08a00', icon: '🫧' },
    washed:       { label: '已清洗', color: '#2fa8c4', icon: '✨' },
    sterilizing:  { label: '灭菌中', color: '#9b59b6', icon: '🔥' },
    sterilized:   { label: '已灭菌', color: '#188038', icon: '✅' },
    issued:       { label: '已发放', color: '#e67e22', icon: '📤' },
    in_use:       { label: '使用中', color: '#c0392b', icon: '🏥' },
    expired:      { label: '已过期', color: '#6b7785', icon: '⏰' }
  },
  /** 状态机允许的手动流转（与后端 manualCycleTransitions 一致）。 */
  manualCycle: {
    to_collect: 'collected',
    collected: 'washing',
    washing: 'washed',
    issued: 'in_use',
    expired: 'washing'
  },
  SterilizeResult: {
    pass: { label: '合格', color: '#188038' },
    fail: { label: '不合格', color: '#d93025' }
  },
  BatchStatus: {
    pending: { label: '待判定', color: '#e08a00' },
    completed: { label: '已完成', color: '#188038' }
  },
  PackType: {
    surgical: '手术器械包',
    dressing: '敷料包',
    instrument: '器械包',
    implant: '植入物包'
  },
  IssueStatus: {
    issued: { label: '已发放', color: '#e67e22' },
    returned: { label: '已回收', color: '#188038' },
    lost: { label: '丢失待查', color: '#d93025' }
  },
  SterilizerStatus: {
    available: { label: '可用', color: '#188038' },
    maintenance: { label: '维护中', color: '#d93025' }
  },
  /** 灭菌参数合格判定下限（与后端 config/rules.go 默认值一致）。 */
  limits: { minTempC: 134, minDurationMin: 4, minPressureKPa: 205 }
};
