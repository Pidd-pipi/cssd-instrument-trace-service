/**
 * usePacks(filter)：器械包列表 Hook。
 * 被器械包列表页与工作台共用；filter 为返回过滤条件的函数，onChange 在数据变化时回调。
 */
window.CSSD = window.CSSD || {};

CSSD.usePacks = function (filterFn, onChange) {
  const hook = {
    data: [],
    loading: false,
    error: null,
    async refresh() {
      hook.loading = true;
      try { if (onChange) onChange(hook); } catch (e) { /* 渲染回调异常不影响数据加载 */ }
      try {
        const filter = typeof filterFn === 'function' ? filterFn() : filterFn;
        hook.data = await CSSD.api.listPacks(filter || {});
        hook.error = null;
      } catch (e) {
        hook.error = e.message;
      } finally {
        hook.loading = false;
        try { if (onChange) onChange(hook); } catch (e) { /* ignore */ }
      }
    }
  };
  hook.refresh();
  return hook;
};
