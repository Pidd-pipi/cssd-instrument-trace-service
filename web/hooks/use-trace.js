/**
 * useTrace(id)：循环追溯 Hook。
 * 被追溯页与器械包详情共用；id 变化时自动重新拉取。
 */
window.CSSD = window.CSSD || {};

CSSD.useTrace = function (id) {
  const hook = {
    id: id,
    data: null,
    loading: false,
    error: null,
    async refresh() {
      hook.loading = true;
      try { if (hook.onChange) hook.onChange(hook); } catch (e) { /* ignore */ }
      try {
        hook.data = await CSSD.api.packTrace(hook.id);
        hook.error = null;
      } catch (e) {
        hook.error = e.message;
        hook.data = null;
      } finally {
        hook.loading = false;
        try { if (hook.onChange) hook.onChange(hook); } catch (e) { /* ignore */ }
      }
    }
  };
  hook.onChange = null;
  hook.refresh();
  return hook;
};
