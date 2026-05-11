/* ============================================================
   EmlakPro — api.js  v2
   ============================================================ */

const API = (() => {
  let _accessToken  = localStorage.getItem('access_token')  || '';
  let _refreshToken = localStorage.getItem('refresh_token') || '';
  let _user         = JSON.parse(localStorage.getItem('user') || 'null');
  let _refreshing   = null; // devam eden refresh promise

  function setSession(data) {
    _accessToken  = data.access_token;
    _refreshToken = data.refresh_token;
    _user         = data.user;
    localStorage.setItem('access_token',  _accessToken);
    localStorage.setItem('refresh_token', _refreshToken);
    localStorage.setItem('user', JSON.stringify(_user));
  }

  function clearSession() {
    _accessToken = _refreshToken = '';
    _user = null;
    localStorage.removeItem('access_token');
    localStorage.removeItem('refresh_token');
    localStorage.removeItem('user');
  }

  function getUser()   { return _user; }
  function isAdmin()   { return _user?.role === 'admin'; }
  function getUserID() { return _user?.id || null; }

  // Refresh token varsa oturum açık sayılır
  function isLoggedIn() { return !!_refreshToken; }

  // Refresh işlemini tek seferlik yapar (paralel çağrıları bekletir)
  async function doRefresh() {
    if (_refreshing) return _refreshing;
    _refreshing = (async () => {
      const res = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: _refreshToken })
      });
      const data = await res.json();
      if (data.success) {
        setSession(data.data);
        return true;
      }
      clearSession();
      return false;
    })();
    try { return await _refreshing; }
    finally { _refreshing = null; }
  }

  async function request(method, path, body = null, isUpload = false) {
    const makeHeaders = () => {
      const h = {};
      if (_accessToken) h['Authorization'] = 'Bearer ' + _accessToken;
      if (!isUpload && body) h['Content-Type'] = 'application/json';
      return h;
    };

    const makeOpts = () => ({
      method,
      headers: makeHeaders(),
      body: body ? (isUpload ? body : JSON.stringify(body)) : undefined
    });

    let res = await fetch('/api' + path, makeOpts());

    // 401 → refresh dene
    if (res.status === 401 && _refreshToken) {
      const ok = await doRefresh();
      if (ok) {
        res = await fetch('/api' + path, makeOpts());
      } else {
        window.dispatchEvent(new Event('session-expired'));
        throw new Error('Oturum süresi doldu, lütfen tekrar giriş yapın.');
      }
    }

    const data = await res.json();
    if (!data.success) throw new Error(data.error || 'Bir hata oluştu');
    return data.data;
  }

  // ── Auth ──────────────────────────────────────────────────
  async function login(username, password) {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    const data = await res.json();
    if (!data.success) throw new Error(data.error || 'Giriş başarısız');
    setSession(data.data);
    return data.data;
  }

  async function logout() {
    try { await request('POST', '/auth/logout', { refresh_token: _refreshToken }); } catch (_) {}
    clearSession();
  }

  // Sayfa yüklenince token geçerliliğini doğrula
  async function validateSession() {
    if (!_refreshToken) return false;
    if (_accessToken) {
      try {
        const user = await request('GET', '/auth/me');
        _user = user;
        localStorage.setItem('user', JSON.stringify(_user));
        return true;
      } catch (_) {}
    }
    // Access token yoksa veya geçersizse refresh dene
    return doRefresh();
  }

  // ── Config ────────────────────────────────────────────────
  async function getConfig() { return request('GET', '/config'); }

  // ── İlanlar ───────────────────────────────────────────────
  async function getListings(params = {}) {
    const q = new URLSearchParams(params).toString();
    return request('GET', '/listings' + (q ? '?' + q : ''));
  }
  async function getListing(id)          { return request('GET', '/listings/' + id); }
  async function getListingByToken(token){ return request('GET', '/listings/share/' + token); }
  async function createListing(data)     { return request('POST', '/listings', data); }
  async function updateListing(id, data) { return request('PUT', '/listings/' + id, data); }
  async function toggleListing(id)       { return request('PATCH', '/listings/' + id + '/toggle'); }
  async function deleteListingImage(listingId, imgId) {
    return request('DELETE', '/listings/' + listingId + '/images/' + imgId);
  }

  // ── Upload ────────────────────────────────────────────────
  async function uploadCover(file) {
    const fd = new FormData(); fd.append('cover', file);
    return request('POST', '/upload/cover', fd, true);
  }
  async function uploadGallery(file) {
    const fd = new FormData(); fd.append('image', file);
    return request('POST', '/upload/gallery', fd, true);
  }

  // ── Talepler ──────────────────────────────────────────────
  async function getRequests(params = {}) {
    const q = new URLSearchParams(params).toString();
    return request('GET', '/requests' + (q ? '?' + q : ''));
  }
  async function createRequest(data)     { return request('POST', '/requests', data); }
  async function updateRequest(id, data) { return request('PUT', '/requests/' + id, data); }
  async function toggleRequest(id)       { return request('PATCH', '/requests/' + id + '/toggle'); }
  async function toggleRequestNotify(id) { return request('PATCH', '/requests/' + id + '/notify'); }

  // ── Admin ─────────────────────────────────────────────────
  async function adminGetUsers()          { return request('GET',    '/admin/users'); }
  async function adminCreateUser(data)    { return request('POST',   '/admin/users', data); }
  async function adminToggleUser(id)      { return request('PATCH',  '/admin/users/' + id + '/toggle'); }
  async function adminDeleteUser(id)      { return request('DELETE', '/admin/users/' + id); }
  async function adminGetListings()       { return request('GET',    '/admin/listings'); }
  async function adminDeleteListing(id)   { return request('DELETE', '/admin/listings/' + id); }
  async function adminGetRequests()       { return request('GET',    '/admin/requests'); }
  async function adminDeleteRequest(id)   { return request('DELETE', '/admin/requests/' + id); }

  return {
    login, logout, getUser, getUserID, isAdmin, isLoggedIn, validateSession,
    getConfig,
    getListings, getListing, getListingByToken,
    createListing, updateListing, toggleListing, deleteListingImage,
    uploadCover, uploadGallery,
    getRequests, createRequest, updateRequest, toggleRequest, toggleRequestNotify,
    adminGetUsers, adminCreateUser, adminToggleUser, adminDeleteUser,
    adminGetListings, adminDeleteListing,
    adminGetRequests, adminDeleteRequest,
  };
})();
