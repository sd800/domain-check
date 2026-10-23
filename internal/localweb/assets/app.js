'use strict';
(() => {
  const $ = id => document.getElementById(id);
  const root = document.documentElement;
  const copy = {
    en: { input: 'Domain name or URL', placeholder: 'Enter a domain or paste a URL', lookup: 'Look up', loading: 'Looking up', empty: 'Registration details, straight from the source.', system: 'System', light: 'Light', dark: 'Dark', theme: 'Theme', language: 'Switch to Simplified Chinese', details: 'Registration details', domain: 'Domain Name', idn: 'IDN', address: 'Address', city: 'City', state: 'State', postal_code: 'Postal Code', country: 'Country', registration_url: 'Registration URL', admin: 'Admin', organization: 'Organization', role: 'Role', phones: 'Phone', admin_emails: 'Email', noTLDRecord: 'No IANA TLD record', registration: 'Registration Time', expiration: 'Expiration Time', changed: 'Last Changed Time', registrar: 'Registrar', statuses: 'Domain Status', registrant: 'Registrant', registrant_organization: 'Organization', street: 'Street', state_province: 'State/Province', registrant_country: 'Country', contact_phones: 'Phone', contact_emails: 'Email', emails: 'Registrant Email', nameservers: 'Name Server', dnssec: 'DNSSEC', result: 'Result', failed: 'The registry query could not be completed. Please try again.', invalid: 'Enter a valid domain name or URL.', busy: 'A lookup is already running. Please try again shortly.', limit: 'Please wait a moment before making another lookup.', connecting: 'Connecting to the local service…', unavailable: 'The local service is unavailable. Enable it with domain setup.', stale: 'Using previously validated IANA data.', unsaved: 'IANA data could not be saved; the query continued online.', fallbackFailed: 'Registry RDAP and WHOIS queries could not be completed.', eligibilityRestricted: 'Registration eligibility restricted', usageRestricted: 'Domain usage restricted', notProvided: 'Registry did not provide domain-specific registration data', noRecord: 'No registry record', notRegistered: 'Not registered', reserved: 'Reserved', notAllowed: 'Registration not allowed', restricted: 'Registration restricted (online registration unavailable)', notAvailable: 'Not available for registration', denied: 'The registry denied access.', rate: 'The registry rate limit was reached. Please try again later.', unsupported: 'No public registry service is available for this suffix.', malformed: 'The registry returned an unrecognized response.' },
    zh: { input: '域名或网址', placeholder: '输入域名，或粘贴完整网址', lookup: '查询', loading: '查询中', empty: '注册信息，直接来自注册局。', system: '跟随系统', light: '浅色', dark: '深色', theme: '主题', language: 'Switch to English', details: '域名注册信息', domain: '域名', idn: '国际化域名', address: '地址', city: '城市', state: '州/地区', postal_code: '邮编', country: '国家', registration_url: '注册网址', admin: '管理联系人', organization: '组织', role: '角色', phones: '电话', admin_emails: '邮箱', noTLDRecord: 'IANA 无此顶级域名记录', registration: '注册时间', expiration: '过期时间', changed: '更新时间', registrar: '注册商', statuses: '状态', registrant: '注册人', registrant_organization: '所属组织', street: '地址', state_province: '省/州', registrant_country: '国家/地区', contact_phones: '电话', contact_emails: '邮箱', emails: '联系邮箱', nameservers: '域名服务器', dnssec: 'DNSSEC', result: '查询结果', failed: '暂时无法完成注册局查询，请稍后重试。', invalid: '请输入有效的域名或网址。', busy: '已有查询正在进行，请稍后重试。', limit: '查询较频繁，请稍后重试。', connecting: '正在连接本机服务…', unavailable: '本机服务不可用，请通过 domain setup 开启。', stale: '正在使用已验证的 IANA 缓存。', unsaved: '未能保存 IANA 数据，查询已通过在线数据完成。', fallbackFailed: '注册局 RDAP 和 WHOIS 查询均未能完成。', eligibilityRestricted: '注册资格受限', usageRestricted: '域名使用受限', notProvided: '注册局未提供此域名的独立注册信息', noRecord: '注册局无记录', notRegistered: '未注册', reserved: '已保留', notAllowed: '不允许注册', restricted: '注册受限（无法在线注册）', notAvailable: '不可注册', denied: '注册局拒绝访问。', rate: '已达到注册局查询频率限制，请稍后重试。', unsupported: '该域名后缀没有公开的注册局查询服务。', malformed: '注册局返回了无法识别的响应。' }
  };
  let language = root.lang.startsWith('zh') ? 'zh' : 'en';
  let theme = root.dataset.theme || 'auto';
  let record = null, warnings = [], failure = '', pending = false, controller = null, requestID = 0;
  let renderedRecord = null, renderedLanguage = '';

  const persist = (key, value) => { try { localStorage.setItem(key, value); } catch {} };
  const t = () => copy[language];
  let session;
  function getSession(refresh = false) {
    if (!session || refresh) {
      session = fetch('/api/session', { cache: 'no-store', credentials: 'omit', redirect: 'error' }).then(response => { if (!response.ok) throw new Error('session'); return response.json(); });
      // Keep rejection observed while the initial view is idle.
      session.catch(() => {});
    }
    return session;
  }
  getSession();
  const errorText = code => {
    if (code === 'invalid-domain') return t().invalid;
    if (code === 'local-busy') return t().busy;
    if (code === 'local-rate-limit') return t().limit;
    if (code === 'access-denied') return t().denied;
    if (code === 'rate-limited') return t().rate;
    if (['no-registry-service', 'unsupported-suffix'].includes(code)) return t().unsupported;
    if (code === 'fallback-failed') return t().fallbackFailed;
    if (code === 'invalid-response') return t().malformed;
    if (code === 'unavailable') return t().unavailable;
    return t().failed;
  };
  function render() {
    const text = t();
    root.lang = language === 'zh' ? 'zh-CN' : 'en';
    $('input-label').textContent = text.input;
    $('domain').placeholder = text.placeholder;
    $('submit-label').textContent = pending ? text.loading : text.lookup;
    $('empty-label').textContent = text.empty;
    $('language').setAttribute('aria-label', text.language);
    $('results').setAttribute('aria-label', text.details);
    $('theme-label').textContent = text[theme === 'auto' ? 'system' : theme];
    $('theme').setAttribute('aria-label', `${text.theme}: ${$('theme-label').textContent}`);
    $('lookup').classList.toggle('is-pending', pending);
    $('results').setAttribute('aria-busy', String(pending));
    $('submit').setAttribute('aria-label', pending ? text.loading : text.lookup);
    $('empty').hidden = !!record;
    $('record').hidden = !record;
    if (record && (record !== renderedRecord || language !== renderedLanguage)) {
      const fragment = document.createDocumentFragment();
      const registrant = record.contact_details
        ? ['registrant', 'registrant_organization', 'address', 'street', 'city', 'state_province', 'postal_code', 'registrant_country', 'contact_phones', 'contact_emails']
        : ['registrant', 'address', 'city', 'state', 'postal_code', 'country', 'emails'];
      const details = record.tld
        ? [['registration', 'changed', 'statuses'], ['registrant', 'address', 'registration_url'], ['admin', 'organization', 'role', 'phones', 'admin_emails'], ['nameservers', 'dnssec']]
        : [['registration', 'expiration', 'changed'], ['registrar', 'statuses'], registrant, ['nameservers', 'dnssec']];
      const groups = [['domain', 'idn'], ...(record.result ? [['result']] : details)];
      const results = { 'eligibility-restricted': text.eligibilityRestricted, 'usage-restricted': text.usageRestricted, 'not-provided': text.notProvided, 'no-record': record.tld ? text.noTLDRecord : text.noRecord, 'not-registered': text.notRegistered, reserved: text.reserved, 'not-allowed': text.notAllowed, 'registration-restricted': text.restricted, 'not-available': text.notAvailable };
      for (const keys of groups) {
        const dl = document.createElement('dl'); dl.className = 'record-group';
        const keepRegistrant = keys.includes('registrant') && keys.some(key => key !== 'registrant' && (Array.isArray(record[key]) ? record[key].some(Boolean) : record[key]));
        for (const key of keys) {
          const raw = key === 'result' ? results[record.result] || text.noRecord : language === 'zh' && record.dates_zh && key in record.dates_zh ? record.dates_zh[key] : record[key];
          for (const value of Array.isArray(raw) ? raw : [raw]) {
            if (!value && !(key === 'registrant' && keepRegistrant)) continue;
            const field = document.createElement('div'), dt = document.createElement('dt'), dd = document.createElement('dd');
            field.className = 'field'; dt.textContent = text[key]; dd.textContent = value || '';
            if (key === 'idn') dd.className = 'idn';
            if (key === 'address' || key === 'street') dd.className = 'address';
            if (key === 'registrant') dd.className = 'registrant';
            field.append(dt, dd); dl.append(field);
          }
        }
        if (dl.children.length) fragment.append(dl);
      }
      $('record').replaceChildren(fragment);
      renderedRecord = record; renderedLanguage = language;
    }
    $('notice').classList.toggle('error', !!failure);
    $('notice').textContent = failure ? errorText(failure) : warnings.length ? (warnings.some(item => item.startsWith('stale-')) ? text.stale : text.unsaved) : '';
  }
  function updateTheme(next) {
    theme = next; root.dataset.theme = next; persist('domain-check.theme', next);
    const dark = next === 'dark' || next === 'auto' && matchMedia('(prefers-color-scheme: dark)').matches;
    document.querySelector('meta[name="theme-color"]').content = dark ? '#121b2a' : '#f6f8fc';
    render();
  }
  $('theme').addEventListener('click', () => updateTheme(['auto', 'light', 'dark'][(['auto', 'light', 'dark'].indexOf(theme) + 1) % 3]));
  matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => { if (theme === 'auto') updateTheme('auto'); });
  $('language').addEventListener('click', () => { language = language === 'en' ? 'zh' : 'en'; persist('domain-check.language', language); render(); });
  async function lookup(input) {
    input = input.trim();
    if (!input) {
      ++requestID;
      if (controller) controller.abort();
      controller = null; record = null; renderedRecord = null;
      warnings = []; failure = ''; pending = false;
      $('domain').value = ''; $('record').replaceChildren();
      history.replaceState(null, '', '/'); render();
      return;
    }
    const id = ++requestID;
    if (controller) controller.abort();
    controller = new AbortController();
    const signal = controller.signal;
    pending = true; failure = ''; warnings = []; render();
    try {
      let current;
      try { current = await getSession(); } catch { current = await getSession(true); }
      if (id !== requestID) return;
      const send = token => fetch('/api/lookup', { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-Domain-Check-Token': token }, body: JSON.stringify({ input }), signal, cache: 'no-store', credentials: 'omit', redirect: 'error' });
      let response = await send(current.token);
      // A local service restart rotates the token; renew it without reloading.
      if (response.status === 403) {
        current = await getSession(true);
        if (id !== requestID) return;
        response = await send(current.token);
      }
      const data = await response.json();
      if (id !== requestID) return;
      if (!response.ok || data.error) { failure = data.error?.code || 'query-failed'; }
      else { record = data.record; warnings = data.warnings || []; $('domain').value = record.domain; history.replaceState(null, '', '/'); }
    } catch (error) { if (id === requestID && error.name !== 'AbortError') failure = 'unavailable'; }
    finally { if (id === requestID) { pending = false; render(); } }
  }
  $('lookup').addEventListener('submit', event => { event.preventDefault(); lookup($('domain').value); });
  updateTheme(theme);
  let initial = '';
  try { initial = decodeURIComponent(location.pathname.slice(1)); } catch { failure = 'invalid-domain'; render(); }
  if (initial) { $('domain').value = initial; lookup(initial); }
})();
