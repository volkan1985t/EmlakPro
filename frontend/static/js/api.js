/* ============================================================
   EmlakPro — api.js  v3
   ============================================================ */

const API = (() => {
  let _accessToken  = localStorage.getItem('access_token')  || '';
  let _refreshToken = localStorage.getItem('refresh_token') || '';
  let _user         = JSON.parse(localStorage.getItem('user') || 'null');
  let _refreshing   = null;

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
  function isLoggedIn(){ return !!_refreshToken; }

  async function doRefresh() {
    if (_refreshing) return _refreshing;
    _refreshing = (async () => {
      const res = await fetch('/api/auth/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: _refreshToken })
      });
      const data = await res.json();
      if (data.success) { setSession(data.data); return true; }
      clearSession(); return false;
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
    if (res.status === 401 && _refreshToken) {
      const ok = await doRefresh();
      if (ok) { res = await fetch('/api' + path, makeOpts()); }
      else { window.dispatchEvent(new Event('session-expired')); throw new Error('Oturum suresi doldu.'); }
    }
    const data = await res.json();
    if (!data.success) throw new Error(data.error || 'Bir hata olustu');
    return data.data;
  }

  // -- Auth --
  async function login(username, password) {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
    const data = await res.json();
    if (!data.success) throw new Error(data.error || 'Giris basarisiz');
    setSession(data.data); return data.data;
  }
  async function logout() {
    try { await request('POST', '/auth/logout', { refresh_token: _refreshToken }); } catch (_) {}
    clearSession();
  }
  async function validateSession() {
    if (!_refreshToken) return false;
    if (_accessToken) {
      try { const user = await request('GET', '/auth/me'); _user = user; localStorage.setItem('user', JSON.stringify(_user)); return true; } catch (_) {}
    }
    return doRefresh();
  }

  // -- Config --
  async function getConfig() { return request('GET', '/config'); }

  // -- Kullanici listesi (atama icin) --
  async function getUsers() { return request('GET', '/users'); }

  // -- Ilanlar --
  async function getListings(params = {}) {
    const q = new URLSearchParams(params).toString();
    return request('GET', '/listings' + (q ? '?' + q : ''));
  }
  async function getListing(id)           { return request('GET', '/listings/' + id); }
  async function getListingByToken(token) { return request('GET', '/listings/share/' + token); }
  async function createListing(data)      { return request('POST', '/listings', data); }
  async function updateListing(id, data)  { return request('PUT', '/listings/' + id, data); }
  async function toggleListing(id, data)  { return request('PATCH', '/listings/' + id + '/toggle', data || null); }
  async function toggleListingListed(id)  { return request('PATCH', '/listings/' + id + '/listed'); }
  async function updatePipeline(id, stage) { return request('PATCH', '/listings/' + id + '/pipeline', { stage }); }
  async function getListingHistory(id)    { return request('GET', '/listings/' + id + '/history'); }
  async function getListingActivities(id) { return request('GET', '/listings/' + id + '/activities'); }
  async function deleteListingImage(listingId, imgId) {
    return request('DELETE', '/listings/' + listingId + '/images/' + imgId);
  }

  // -- Upload --
  async function uploadCover(file, propType='', listingNo=0) {
    const fd = new FormData();
    fd.append('cover', file);
    fd.append('prop_type', propType);
    fd.append('listing_no', String(listingNo));
    return request('POST', '/upload/cover', fd, true);
  }
  async function uploadGallery(file, propType='', listingNo=0) {
    const fd = new FormData();
    fd.append('image', file);
    fd.append('prop_type', propType);
    fd.append('listing_no', String(listingNo));
    return request('POST', '/upload/gallery', fd, true);
  }

  // -- Talepler --
  async function getRequests(params = {}) {
    const q = new URLSearchParams(params).toString();
    return request('GET', '/requests' + (q ? '?' + q : ''));
  }
  async function createRequest(data)     { return request('POST', '/requests', data); }
  async function updateRequest(id, data) { return request('PUT', '/requests/' + id, data); }
  async function toggleRequest(id)       { return request('PATCH', '/requests/' + id + '/toggle'); }
  async function toggleRequestNotify(id) { return request('PATCH', '/requests/' + id + '/notify'); }

  // -- Musteriler (CRM) --
  async function getCustomers(params = {}) {
    const q = new URLSearchParams(params).toString();
    return request('GET', '/customers' + (q ? '?' + q : ''));
  }
  async function createCustomer(data)     { return request('POST', '/customers', data); }
  async function updateCustomer(id, data) { return request('PUT', '/customers/' + id, data); }
  async function toggleCustomer(id)       { return request('PATCH', '/customers/' + id + '/toggle'); }
  async function deleteCustomer(id)       { return request('DELETE', '/customers/' + id); }
  async function getCustomerListings(id)  { return request('GET', '/customers/' + id + '/listings'); }
  async function linkListing(customerId, listingId, note) {
    return request('POST', '/customers/' + customerId + '/listings', { listing_id: listingId, note: note||'' });
  }
  async function unlinkListing(customerId, listingId) {
    return request('DELETE', '/customers/' + customerId + '/listings/' + listingId);
  }

  // -- Dashboard --
  async function getDashboard() { return request('GET', '/dashboard'); }
  async function getRecentActivities() { return request('GET', '/activities'); }

  // -- Gorevler (Tasks) --
  async function getTasks(params = {}) {
    const q = new URLSearchParams(params).toString();
    return request('GET', '/tasks' + (q ? '?' + q : ''));
  }
  async function getTask(id)            { return request('GET',    '/tasks/' + id); }
  async function createTask(data)       { return request('POST',   '/tasks', data); }
  async function updateTask(id, data)   { return request('PUT',    '/tasks/' + id, data); }
  async function updateTaskStatus(id, status) {
    return request('PATCH', '/tasks/' + id + '/status', { status });
  }
  async function deleteTask(id)         { return request('DELETE', '/tasks/' + id); }
  async function addTaskComment(id, body)  { return request('POST', '/tasks/' + id + '/comments', { body }); }
  async function deleteTaskComment(taskId, cid) {
    return request('DELETE', '/tasks/' + taskId + '/comments/' + cid);
  }
  async function uploadTaskImage(taskId, file) {
    const fd = new FormData(); fd.append('image', file);
    return request('POST', '/tasks/' + taskId + '/images', fd, true);
  }
  async function deleteTaskImage(taskId, imgId) {
    return request('DELETE', '/tasks/' + taskId + '/images/' + imgId);
  }

  // -- Admin --
  async function adminGetUsers()        { return request('GET',    '/admin/users'); }
  async function adminCreateUser(data)  { return request('POST',   '/admin/users', data); }
  async function adminToggleUser(id)    { return request('PATCH',  '/admin/users/' + id + '/toggle'); }
  async function adminDeleteUser(id)    { return request('DELETE', '/admin/users/' + id); }
  async function adminGetListings()     { return request('GET',    '/admin/listings'); }
  async function adminDeleteListing(id) { return request('DELETE', '/admin/listings/' + id); }
  async function adminGetRequests()     { return request('GET',    '/admin/requests'); }
  async function adminDeleteRequest(id) { return request('DELETE', '/admin/requests/' + id); }
  async function adminSetChatID(id, chatID) {
    return request('PATCH', '/admin/users/' + id + '/chatid', { telegram_chat_id: chatID });
  }
  // Admin Settings -- sabahki versiyonda eklenen
  async function getAdminSettings()     { return request('GET',  '/admin/settings'); }
  async function updateAdminSettings(d) { return request('PUT',  '/admin/settings', d); }

  return {
    login, logout, getUser, getUserID, isAdmin, isLoggedIn, validateSession,
    getConfig, getUsers,
    getListings, getListing, getListingByToken,
    updatePipeline,
    createListing, updateListing, toggleListing, toggleListingListed,
    getListingHistory, getListingActivities, deleteListingImage,
    uploadCover, uploadGallery,
    getRequests, createRequest, updateRequest, toggleRequest, toggleRequestNotify,
    getCustomers, createCustomer, updateCustomer, toggleCustomer, deleteCustomer,
    getCustomerListings, linkListing, unlinkListing,
    getDashboard, getRecentActivities,
    adminGetUsers, adminCreateUser, adminToggleUser, adminDeleteUser, adminSetChatID,
    adminGetListings, adminDeleteListing,
    adminGetRequests, adminDeleteRequest,
    getAdminSettings, updateAdminSettings,
    getTasks, getTask, createTask, updateTask, updateTaskStatus, deleteTask,
    addTaskComment, deleteTaskComment, uploadTaskImage, deleteTaskImage,
  };
})();
