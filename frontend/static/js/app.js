/* ============================================================
   EmlakPro — app.js  v4
   ============================================================ */

const state = {
  cfg: null,
  listings: [],
  requests: [],
  customers: [],
  editListingId: null,
  editRequestId: null,
  editDetailId:  null,
  editCustomerId: null,
  viewCustomerId: null,
  passiveTargetId: null,
  passiveStatus: 'satildi',
  coverPath: '', coverURL: '',
  galleryPaths: [], galleryExisting: [], removedImageIds: [],
  dashCharts: {},
};

/* ── Başlatma ──────────────────────────────────────────────── */
async function init() {
  const valid = await API.validateSession();
  if (!valid) { showLogin(); return; }
  try {
    await loadConfig();
    showApp();
    // Önce ilanları yükle — talepler eşleşme için state.listings'e ihtiyaç duyar
    await loadListings();
    await loadRequests();
  } catch(e) { showLogin(); }
}

window.addEventListener('session-expired', () => {
  showToast('Oturumunuz sona erdi.', 'error');
  setTimeout(showLogin, 1500);
});

async function loadConfig() {
  state.cfg = await API.getConfig();
  buildFilters();
  if (API.isAdmin()) document.querySelectorAll('.admin-only').forEach(el => el.style.display = '');
  const user = API.getUser();
  const av = document.getElementById('user-avatar');
  if (av && user) { av.textContent = (user.full_name||user.username||'U')[0].toUpperCase(); av.title = user.full_name||user.username; }
}

function showLogin() { document.getElementById('login-screen').style.display='flex'; document.getElementById('app').style.display='none'; }
function showApp()   { document.getElementById('login-screen').style.display='none'; document.getElementById('app').style.display='flex'; }

/* ── Filtreler ─────────────────────────────────────────────── */
function buildFilters() {
  const cfg = state.cfg;
  fillSelect('filter-property',  cfg.property_types, 'Tüm Mülkler');
  fillSelect('filter-tip',       cfg.listing_types,  'Satılık/Kiralık');
  fillSelect('filter-ilce',      cfg.districts,      'Tüm İlçeler');
  fillSelect('talep-tip-filter', cfg.listing_types,  'Satılık / Kiralık');
  fillSelect('talep-ilce-filter',cfg.districts,      'Tüm İlçeler');
  updatePropertyFilter('');
}

function fillSelect(id, options, placeholder) {
  const el = document.getElementById(id);
  if (!el) return;
  el.innerHTML = `<option value="">${placeholder}</option>` +
    (options||[]).map(o=>`<option>${o}</option>`).join('');
}

function updatePropertyFilter(propType) {
  const odaWrap = document.getElementById('filter-oda-wrap');
  const m2Wrap  = document.getElementById('filter-m2-wrap');
  if (!odaWrap||!m2Wrap) return;
  odaWrap.style.display = 'none';
  m2Wrap.style.display  = 'none';
  if (propType === 'Daire') {
    odaWrap.style.display = '';
    fillSelect('filter-oda', state.cfg?.field_sources?.room_options||[], 'Tüm Odalar');
  } else if (propType === 'Arsa') {
    m2Wrap.style.display = '';
  }
}

/* ── Fiyat formatlama ──────────────────────────────────────── */
function fiyatFormat(n) {
  n = parseInt(n)||0;
  if (!n) return '—';
  return n.toLocaleString('tr-TR') + ' ₺';
}
function formatPriceInput(el) {
  const raw = el.value.replace(/\./g,'').replace(/[^0-9]/g,'');
  if (!raw) { el.value = ''; el.dataset.raw = ''; return; }
  el.dataset.raw = raw;
  el.value = parseInt(raw).toLocaleString('tr-TR');
}
function getRawPrice(id) {
  const el = document.getElementById(id);
  if (!el) return '';
  return el.dataset.raw || el.value.replace(/\./g,'').replace(/[^0-9]/g,'');
}
function setPriceInput(id, val) {
  const el = document.getElementById(id);
  if (!el) return;
  if (!val) { el.value=''; el.dataset.raw=''; return; }
  el.dataset.raw = val;
  el.value = parseInt(val).toLocaleString('tr-TR');
}
function formatDisplayPrice(val) {
  if (!val) return '';
  const n = parseInt(val);
  return isNaN(n) ? '' : n.toLocaleString('tr-TR');
}

function buildCustomerSelectBlock(selectedId) {
  return `<div class="form-group">
    <label>Müşteri <span class="muted">(opsiyonel)</span></label>
    <select id="f-customer_id">
      <option value="">Müşteri seçin...</option>
    </select>
    <small class="muted" style="margin-top:4px;display:block">Müşteri bilgileri gizli tutulur</small>
  </div>`;
}
async function fillCustomerDropdown(selectedId) {
  try {
    const customers = state.customers.length ? state.customers : await API.getCustomers() || [];
    const sel = document.getElementById('f-customer_id');
    if (!sel) return;
    sel.innerHTML = '<option value="">Müşteri seçin...</option>';
    customers.forEach(c => {
      const opt = document.createElement('option');
      opt.value = c.id;
      opt.textContent = c.name + ' · ' + (c.phone||'');
      if (c.id === selectedId) opt.selected = true;
      sel.appendChild(opt);
    });
  } catch(e) { console.error('Müşteri yüklenemedi', e); }
}

function showToast(msg, type='info') {
  const t = document.getElementById('toast');
  t.textContent = msg;
  t.className = 'toast show toast-'+type;
  setTimeout(()=>t.classList.remove('show'), 3000);
}

/* ── NAVİGASYON ────────────────────────────────────────────── */
function navigateTo(page) {
  document.querySelectorAll('.nav-btn').forEach(b=>b.classList.remove('active'));
  document.querySelectorAll('.bottom-nav-btn').forEach(b=>b.classList.remove('active'));
  document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
  document.querySelector(`.nav-btn[data-page="${page}"]`)?.classList.add('active');
  document.querySelector(`.bottom-nav-btn[data-page="${page}"]`)?.classList.add('active');
  document.getElementById('page-'+page)?.classList.add('active');
  if (page==='admin')      loadAdminPanel();
  if (page==='musteriler') loadCustomers();
  if (page==='dashboard')  loadDashboard();
  if (page==='gorevler')   loadTasks();
  if (page==='pipeline')   loadPipeline();
}

document.querySelectorAll('.nav-btn').forEach(btn => {
  btn.addEventListener('click', function() { navigateTo(this.dataset.page); });
});

document.querySelectorAll('.bottom-nav-btn').forEach(btn => {
  btn.addEventListener('click', function() { navigateTo(this.dataset.page); });
});
document.querySelectorAll('.admin-tab').forEach(btn => {
  btn.addEventListener('click', function() {
    document.querySelectorAll('.admin-tab').forEach(b=>b.classList.remove('active'));
    this.classList.add('active');
    document.querySelectorAll('.admin-panel').forEach(p=>p.classList.remove('active'));
    document.getElementById(this.dataset.tab).classList.add('active');
    if (this.dataset.tab==='admin-users')    loadAdminUsers();
    if (this.dataset.tab==='admin-listings') loadAdminListings();
    if (this.dataset.tab==='admin-requests') loadAdminRequests();
    if (this.dataset.tab==='admin-settings') loadAdminSettings();
    if (this.dataset.tab==='admin-fields')   loadAdminFields();
  });
});

/* ═══════════════════════════════════════════════════════
   İLAN PLACEHOLDER
════════════════════════════════════════════════════════ */
const PROP_PLACEHOLDER = {
  'Daire':  { grad: 'linear-gradient(135deg,#1565C0 0%,#42a5f5 100%)', icon: '🏢', label: 'Daire' },
  'Villa':  { grad: 'linear-gradient(135deg,#1b5e20 0%,#66bb6a 100%)', icon: '🏡', label: 'Villa' },
  'Arsa':   { grad: 'linear-gradient(135deg,#e65100 0%,#ffb74d 100%)', icon: '🌿', label: 'Arsa'  },
  'Ticari': { grad: 'linear-gradient(135deg,#4a148c 0%,#ce93d8 100%)', icon: '🏬', label: 'Ticari'},
  'default':{ grad: 'linear-gradient(135deg,#37474f 0%,#90a4ae 100%)', icon: '🏠', label: ''      },
};

function cardPlaceholder(propType) {
  const p = PROP_PLACEHOLDER[propType] || PROP_PLACEHOLDER['default'];
  return `<div class="card-img-gradient" style="background:${p.grad}">
    <span class="card-img-icon">${p.icon}</span>
    ${p.label ? `<span class="card-img-label">${p.label}</span>` : ''}
  </div>`;
}

/* ── Durum etiketi ─────────────────────────────────────────── */
const STATUS_LABEL = { aktif:'Aktif', satildi:'Satıldı', kiralandi:'Kiralandı', bekliyor:'Bekliyor' };
const STATUS_COLOR = { aktif:'tag-green', satildi:'tag-red', kiralandi:'tag-blue', bekliyor:'tag-amber' };

/* ═══════════════════════════════════════════════════════
   İLANLAR
════════════════════════════════════════════════════════ */
async function loadListings() {
  const params = {};
  const q    = document.getElementById('search-input')?.value;
  const pt   = document.getElementById('filter-property')?.value;
  const lt   = document.getElementById('filter-tip')?.value;
  const d    = document.getElementById('filter-ilce')?.value;
  const oda  = document.getElementById('filter-oda')?.value;
  const mine = document.getElementById('filter-mine')?.checked;
  if (q)    params.q             = q;
  if (pt)   params.property_type = pt;
  if (lt)   params.listing_type  = lt;
  if (d)    params.district      = d;
  if (oda)  params.rooms         = oda;
  if (mine) params.only_mine     = '1';
  try {
    state.listings = await API.getListings(params)||[];
    renderListings();
    // Talepler zaten yüklüyse eşleşmeleri güncelle
    if (state.requests.length) renderRequests();
  } catch(e) { showToast('İlanlar yüklenemedi: '+e.message,'error'); }
}

function renderListings() {
  const grid  = document.getElementById('ilan-grid');
  const empty = document.getElementById('ilan-empty');
  const userID = API.getUserID();
  if (!state.listings.length) { grid.innerHTML=''; empty.style.display='block'; return; }
  empty.style.display='none';

  grid.innerHTML = state.listings.map(il => {
    const cfg      = state.cfg;
    const propType = il.fields?.property_type||'Daire';
    const cardKeys = cfg?.listing_fields?.card_fields?.[propType]||[];
    const tagsHTML = cardKeys.slice(0,4).map(k => {
      const v = il.fields?.[k]; return v ? `<span class="meta-tag">${v}</span>` : '';
    }).join('');

    const isOwner  = il.user_id===userID||API.isAdmin();
    const isPassive= !il.is_active;
    const isUnlisted = il.is_active && !il.is_listed;

    // Durum badge
    let badge = '';
    if (isPassive && isOwner) {
      const stKey = il.status || 'bekliyor';
      badge = `<span class="badge badge-passive">${STATUS_LABEL[stKey]||'Pasif'}</span>`;
    } else if (!isPassive) {
      badge = `<span class="badge badge-${il.fields?.listing_type==='Satılık'?'sale':'rent'}">${il.fields?.listing_type||''}</span>`;
    }

    // Listeleme uyarısı
    const unlistedBadge = isUnlisted && isOwner
      ? `<span class="badge badge-unlisted" title="Ana sayfada gösterilmiyor">👁 Gizli</span>` : '';

    const noTag = il.listing_no ? `<span class="listing-no">#${il.listing_no}</span>` : '';
    const imgHTML = il.cover_image
      ? `<img src="${il.cover_image}" alt="${il.fields?.title||''}" loading="lazy">`
      : cardPlaceholder(propType);

    const ownerActions = isOwner ? `
      <div class="card-actions">
        <button class="icon-btn icon-btn-edit" onclick="openEditListing(${il.id},event)" title="Düzenle">✏️</button>
        <button class="icon-btn icon-btn-listed ${il.is_listed?'':'icon-btn-unlisted'}"
          onclick="doToggleListed(${il.id},event)"
          title="${il.is_listed?'Ana sayfadan gizle':'Ana sayfada göster'}">${il.is_listed?'👁':'👁‍🗨'}</button>
        <button class="icon-btn icon-btn-toggle" onclick="doToggleListing(${il.id},event)"
          title="${il.is_active?'Pasif Yap':'Aktif Yap'}">${il.is_active?'⏸':'▶️'}</button>
      </div>` : '';

    const priceMin = il.fields?.price_min;
    const priceMax = il.fields?.price_max;
    const priceDisplay = priceMin||priceMax
      ? (priceMin&&priceMax ? `${fiyatFormat(priceMin)} – ${fiyatFormat(priceMax)}` : fiyatFormat(priceMin||priceMax))
      : fiyatFormat(il.fields?.price);

    return `<div class="card${isPassive?' card-passive':''}${isUnlisted?' card-unlisted':''}" data-id="${il.id}">
      <div class="card-img">
        ${imgHTML}${badge}${unlistedBadge}${noTag}${ownerActions}
      </div>
      <div class="card-body">
        <div class="card-title-row">
          <div class="card-title">${il.fields?.title||'—'}</div>
          <div class="card-type-badges">
            ${il.fields?.property_type ? `<span class="card-ptype">${il.fields.property_type}</span>` : ''}
            ${il.fields?.listing_type ? `<span class="card-ltype card-ltype-${il.fields.listing_type==='Satılık'?'sale':'rent'}">${il.fields.listing_type}</span>` : ''}
          </div>
        </div>
        <div class="card-meta">${tagsHTML}</div>
        <div class="card-footer">
          <div class="price">${priceDisplay}</div>
          <div class="card-agent">👤 ${il.owner_name||''}</div>
        </div>
      </div>
    </div>`;
  }).join('');

  document.querySelectorAll('.card').forEach(card => {
    card.addEventListener('click', e => {
      if (e.target.closest('.card-actions')) return;
      const id = parseInt(card.dataset.id);
      const il = state.listings.find(l=>l.id===id);
      if (il&&(il.user_id===API.getUserID()||API.isAdmin())) openDetailModal(id);
    });
  });
}

['search-input','filter-property','filter-tip','filter-ilce','filter-oda','filter-m2'].forEach(id=>{
  document.getElementById(id)?.addEventListener('input',  loadListings);
  document.getElementById(id)?.addEventListener('change', loadListings);
});
document.getElementById('filter-mine')?.addEventListener('change', loadListings);
document.getElementById('filter-property')?.addEventListener('change', function() {
  updatePropertyFilter(this.value); loadListings();
});

/* ── İlan Toggle (is_active) ───────────────────────────────── */
async function doToggleListing(id, e) {
  e.stopPropagation();
  const il = state.listings.find(l=>l.id===id);
  if (!il) return;

  if (il.is_active) {
    // Aktif → Pasif: durum sorusu sor
    openPassiveModal(id, il.fields?.title||'');
  } else {
    // Pasif → Aktif: direkt
    try {
      await API.toggleListing(id, {});
      await loadListings();
      showToast('İlan aktif edildi.');
    } catch(err) { showToast(err.message,'error'); }
  }
}

/* ── İlan Toggle (is_listed) ───────────────────────────────── */
async function doToggleListed(id, e) {
  e.stopPropagation();
  try {
    await API.toggleListingListed(id);
    await loadListings();
    const il = state.listings.find(l=>l.id===id);
    showToast(il?.is_listed ? 'İlan ana sayfada görünüyor.' : 'İlan ana sayfadan gizlendi.');
  } catch(err) { showToast(err.message,'error'); }
}

/* ── Pasife Alma Modal ─────────────────────────────────────── */
function openPassiveModal(id, title) {
  state.passiveTargetId = id;
  state.passiveStatus   = 'satildi';
  document.getElementById('passive-listing-name').textContent = title;
  document.getElementById('passive-price').value = '';
  document.getElementById('passive-price').dataset.raw = '';
  // Reset butonlar
  document.querySelectorAll('.status-opt').forEach(b => b.classList.remove('status-opt-sel'));
  document.querySelector('.status-opt[data-val="satildi"]')?.classList.add('status-opt-sel');
  document.getElementById('passive-overlay').classList.add('open');
}
document.querySelectorAll('.status-opt').forEach(btn => {
  btn.addEventListener('click', function() {
    document.querySelectorAll('.status-opt').forEach(b=>b.classList.remove('status-opt-sel'));
    this.classList.add('status-opt-sel');
    state.passiveStatus = this.dataset.val;
  });
});
document.getElementById('close-passive-modal').addEventListener('click', () =>
  document.getElementById('passive-overlay').classList.remove('open'));
document.getElementById('passive-iptal-btn').addEventListener('click', () =>
  document.getElementById('passive-overlay').classList.remove('open'));
document.getElementById('passive-kaydet-btn').addEventListener('click', async () => {
  const id = state.passiveTargetId;
  if (!id) return;
  const rawPrice = getRawPrice('passive-price');
  const body = {
    status: state.passiveStatus,
    closing_price: rawPrice ? parseInt(rawPrice) : null,
  };
  try {
    await API.toggleListing(id, body);
    document.getElementById('passive-overlay').classList.remove('open');
    await loadListings();
    showToast('İlan pasife alındı.');
  } catch(err) { showToast(err.message,'error'); }
});

/* ─── Detay Modal ─────────────────────────────────────────── */
async function openDetailModal(id) {
  try {
    const il = await API.getListing(id);
    state.editDetailId = id;
    document.getElementById('detail-title').textContent = il.fields?.title||'İlan Detayı';
    const cfg     = state.cfg;
    const propType= il.fields?.property_type||'Daire';
    const sumKeys = cfg?.listing_fields?.summary_fields||[];

    const priceMin = il.fields?.price_min;
    const priceMax = il.fields?.price_max;
    const priceVal = priceMin||priceMax
      ? (priceMin&&priceMax ? `${fiyatFormat(priceMin)} – ${fiyatFormat(priceMax)}` : fiyatFormat(priceMin||priceMax))
      : fiyatFormat(il.fields?.price);

    const rows = sumKeys.map(k => {
      if (k==='price'&&(priceMin||priceMax)) return '';
      const fd = cfg?.listing_fields?.all_fields?.find(f=>f.key===k);
      const v  = il.fields?.[k];
      if (!v) return '';
      return `<tr><td class="detail-label">${fd?.label||k}</td><td>${k==='price'?fiyatFormat(v):v}</td></tr>`;
    }).join('');

    // Durum satırı
    const stKey = il.status||'aktif';
    const statusRow = `<tr><td class="detail-label">Durum</td><td>
      <span class="tag ${STATUS_COLOR[stKey]||'tag-green'}">${STATUS_LABEL[stKey]||stKey}</span>
      ${il.closing_price ? ` · <b>${fiyatFormat(il.closing_price)}</b>` : ''}
    </td></tr>`;
    // Listeleme satırı
    const listedRow = `<tr><td class="detail-label">Vitrin</td><td>
      <span class="tag ${il.is_listed?'tag-green':'tag-amber'}">${il.is_listed?'Listelendi':'Gizli'}</span>
    </td></tr>`;
    const _allImgs = [];
    if (il.cover_image) _allImgs.push(il.cover_image);
    (il.images||[]).forEach(img => _allImgs.push(img.path));
    state.lightboxImages = _allImgs;
    const priceRow = `<tr><td class="detail-label">Fiyat</td><td><b>${priceVal}</b></td></tr>`;
    const noHTML = il.listing_no ? `<div class="detail-no">İlan No: <b>#${il.listing_no}</b></div>` : '';
    const coverHTML= il.cover_image ? `<div class="detail-cover"><img src="${il.cover_image}" alt="" loading="lazy" style="cursor:zoom-in" onclick="openLightboxIdx(0)"></div>` : '';
    const gallery  = il.images?.length
      ? `<div class="detail-gallery">${il.images.map((img,i)=>{
          const idx = il.cover_image ? i+1 : i;
          return `<img src="${img.path}" alt="" loading="lazy" style="cursor:zoom-in" onclick="openLightboxIdx(${idx})">`;
        }).join('')}</div>` : '';

    document.getElementById('detail-content').innerHTML = `
      ${coverHTML}${noHTML}
      <table class="detail-table">${priceRow}${statusRow}${listedRow}${rows}</table>
      ${gallery}
      ${il.fields?.description?`<div class="detail-desc"><b>Açıklama:</b><p>${il.fields.description}</p></div>`:''}
      <div class="detail-tabs">
        <button class="detail-tab active" onclick="switchDetailTab('aktivite',` + il.id + `,this)">📋 Aktiviteler</button>
        <button class="detail-tab" onclick="switchDetailTab('kanal',` + il.id + `,this)">📢 Kanallar</button>
      </div>
      <div id="detail-tab-content"></div>
    `;
    loadDetailActivities(il.id);
    // Malik butonu
    const malikBtn = document.getElementById('detail-malik-btn');
    if (malikBtn) {
      if (il.customer_id && (API.isAdmin() || il.user_id===API.getUserID())) {
        malikBtn.style.display = '';
        malikBtn.onclick = () => goToMalik(il.customer_id);
      } else {
        malikBtn.style.display = 'none';
      }
    }
    document.getElementById('detail-overlay').classList.add('open');
  } catch(e) { showToast(e.message,'error'); }
}

async function toggleCrmAccordion(id, e) {
  if (e && e.target && e.target.closest && e.target.closest('.icon-btn')) return;
  const acc  = document.getElementById('crm-acc-'+id);
  const chev = document.getElementById('chev-'+id);
  if (!acc) return;
  const isOpen = acc.style.display !== 'none';
  if (isOpen) { acc.style.display='none'; if(chev) chev.textContent='▸'; return; }
  acc.style.display = 'block';
  if (chev) chev.textContent = '▾';
  try {
    const listings = await API.getCustomerListings(id)||[];
    const ilanListHTML = listings.length
      ? listings.map(il=>`
          <div class="crm-acc-ilan">
            ${il.cover_image?`<img src="${il.cover_image}" class="crm-acc-thumb" alt="">`:'<div class="crm-acc-thumb crm-acc-nophoto">🏠</div>'}
            <div class="crm-acc-info">
              <div class="crm-acc-title">${il.fields?.title||'—'} <span class="mini-no">#${il.listing_no||''}</span></div>
              <div class="crm-acc-meta">${il.fields?.district||''} · ${fiyatFormat(il.fields?.price_max||il.fields?.price)}</div>
            </div>
            <button class="btn btn-sm btn-danger" onclick="doCrmUnlink(${id},${il.id},event)">Kaldır</button>
          </div>`).join('')
      : '<div class="crm-acc-empty">Bağlı ilan yok.</div>';
    acc.innerHTML = ilanListHTML;
  } catch(e) { acc.innerHTML='<div class="crm-acc-empty">Yüklenemedi.</div>'; }
}

async function doCrmUnlink(customerId, listingId, e) {
  e.stopPropagation();
  if (!confirm('Bağlantıyı kaldırmak istiyor musunuz?')) return;
  try {
    await API.unlinkListing(customerId, listingId);
    showToast('Bağlantı kaldırıldı.');
    const acc = document.getElementById('crm-acc-'+customerId);
    if (!acc || acc.style.display==='none') return;
    const listings = await API.getCustomerListings(customerId)||[];
    if (!listings.length) { acc.innerHTML='<div class="crm-acc-empty">Bağlı ilan yok.</div>'; return; }
    acc.innerHTML = listings.map(il=>`
      <div class="crm-acc-ilan">
        ${il.cover_image?`<img src="${il.cover_image}" class="crm-acc-thumb" alt="">`:'<div class="crm-acc-thumb crm-acc-nophoto">🏠</div>'}
        <div class="crm-acc-info">
          <div class="crm-acc-title">${il.fields?.title||'—'} <span class="mini-no">#${il.listing_no||''}</span></div>
          <div class="crm-acc-meta">${il.fields?.district||''} · ${fiyatFormat(il.fields?.price_max||il.fields?.price)}</div>
        </div>
        <button class="btn btn-sm btn-danger" onclick="doCrmUnlink(${customerId},${il.id},event)">Kaldır</button>
      </div>`).join('');
  } catch(err) { showToast(err.message,'error'); }
}


function openCrmLinkListing(customerId, e) {
  if (e) e.stopPropagation();
  state.viewCustomerId = customerId;
  state._linkFromListingId = null;
  const sel = document.getElementById('link-listing-select');
  sel.innerHTML = '<option value="">Seçin...</option>' +
    state.listings
      .filter(il => il.is_active && !il.customer_id && (API.isAdmin() || il.user_id === API.getUserID()))
      .map(il=>`<option value="${il.id}">#${il.listing_no} ${il.fields?.title||''}</option>`).join('');
  document.getElementById('link-listing-note').value = '';
  document.getElementById('link-listing-overlay').classList.add('open');
}


async function loadDetailActivities(id) {
  const cont = document.getElementById('detail-tab-content');
  if (!cont) return;
  cont.innerHTML = '<div class="muted" style="padding:8px">Yükleniyor...</div>';
  try {
    const acts = await API.getListingActivities(id) || [];
    if (!acts.length) {
      cont.innerHTML = '<div class="muted" style="padding:12px;text-align:center;font-size:13px">Aktivite yok.</div>';
      return;
    }
    const ACT_ICON = {
      created:'✨', updated:'✏️', pipeline:'🔄', channel:'📢',
      activated:'▶️', deactivated:'⏸', listed:'👁', unlisted:'👁'
    };
    cont.innerHTML = '<div class="activity-timeline">' +
      acts.map(a => {
        const icon = ACT_ICON[a.type] || '•';
        const date = new Date(a.created_at).toLocaleDateString('tr-TR',{day:'2-digit',month:'short',hour:'2-digit',minute:'2-digit'});
        return '<div class="act-item">' +
          '<div class="act-icon">' + icon + '</div>' +
          '<div class="act-body">' +
            '<div class="act-note">' + escHtml(a.note||'') + '</div>' +
            '<div class="act-meta">' + escHtml(a.user_name||'') + ' · ' + date + '</div>' +
          '</div>' +
        '</div>';
      }).join('') +
    '</div>';
  } catch(e) {
    cont.innerHTML = '<div class="muted" style="padding:8px">Yüklenemedi.</div>';
  }
}

function switchDetailTab(tab, id, btn) {
  document.querySelectorAll('.detail-tab').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  if (tab === 'aktivite') {
    loadDetailActivities(id);
  } else if (tab === 'kanal') {
    const cont = document.getElementById('detail-tab-content');
    if (!cont) return;
    const il = state.listings.find(l => l.id === id);
    const channels = state.cfg?.listing_channels || [];
    const selected = (il?.fields?.channels || '').split(',');
    cont.innerHTML = '<div class="channels-grid" style="margin-top:8px">' +
      channels.map(ch => {
        const isSel = selected.includes(ch.key);
        return '<div class="channel-card' + (isSel?' selected':'') + '">' +
          '<span class="channel-card-icon">' + (ch.icon||'🌐') + '</span>' +
          '<div class="channel-card-name">' + escHtml(ch.label) + '</div>' +
        '</div>';
      }).join('') +
    '</div>';
  }
}

function goToMalik(customerId) {
  document.getElementById('detail-overlay').classList.remove('open');
  document.querySelectorAll('.nav-btn').forEach(b=>b.classList.remove('active'));
  document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
  document.querySelector('.nav-btn[data-page="musteriler"]')?.classList.add('active');
  document.getElementById('page-musteriler')?.classList.add('active');
  loadCustomers().then(()=>{
    setTimeout(()=>{
      const acc = document.getElementById('crm-acc-'+customerId);
      if (acc) {
        acc.closest('.crm-item')?.scrollIntoView({behavior:'smooth',block:'center'});
        if (acc.style.display==='none') toggleCrmAccordion(customerId,{target:{closest:()=>null}});
      }
    }, 300);
  });
}
function openLightboxIdx(idx) {
  if (!state.lightboxImages?.length) return;
  state.lbImages = state.lightboxImages;
  state.lbIdx = idx;
  document.getElementById('lightbox-img').src = state.lbImages[state.lbIdx];
  document.getElementById('lightbox').classList.add('open');
}

function openLightbox(src, images) {
  state.lbImages = images && images.length ? images : [src];
  state.lbIdx = state.lbImages.indexOf(src);
  if (state.lbIdx < 0) state.lbIdx = 0;
  document.getElementById('lightbox-img').src = state.lbImages[state.lbIdx];
  document.getElementById('lightbox').classList.add('open');
}

function lbGo(dir) {
  if (!state.lbImages?.length) return;
  state.lbIdx = (state.lbIdx + dir + state.lbImages.length) % state.lbImages.length;
  document.getElementById('lightbox-img').src = state.lbImages[state.lbIdx];
}

function lbClose() {
  document.getElementById('lightbox').classList.remove('open');
}

// Lightbox event'leri
// Lightbox — tek delegation
document.body.addEventListener('click', e => {
  const lb = document.getElementById('lightbox');
  if (!lb?.classList.contains('open')) return;
  if (e.target.id === 'lb-prev')       { e.stopPropagation(); lbGo(-1); return; }
  if (e.target.id === 'lb-next')       { e.stopPropagation(); lbGo(+1); return; }
  if (e.target.id === 'lb-close')      { e.stopPropagation(); lbClose(); return; }
  if (e.target.id === 'lightbox-img')  { e.stopPropagation(); lbGo(+1); return; }
  if (e.target.id === 'lightbox')      { lbClose(); return; }
});

document.addEventListener('keydown', e => {
  const lb = document.getElementById('lightbox');
  if (!lb?.classList.contains('open')) return;
  if (e.key === 'ArrowRight') lbGo(+1);
  if (e.key === 'ArrowLeft')  lbGo(-1);
  if (e.key === 'Escape')     lbClose();
});
document.getElementById('close-detail').addEventListener('click',()=>
  document.getElementById('detail-overlay').classList.remove('open'));
document.getElementById('detail-overlay').addEventListener('click',e=>{
  if (e.target===document.getElementById('detail-overlay'))
    document.getElementById('detail-overlay').classList.remove('open');
});
document.getElementById('detail-share-btn').addEventListener('click', async ()=>{
  if (!state.editDetailId) return;
  const il = state.listings.find(l=>l.id===state.editDetailId);
  if (!il) return;
  await navigator.clipboard.writeText(`${location.origin}/api/listings/share/${il.share_token}`).catch(()=>{});
  showToast('📋 Link kopyalandı!');
});

document.getElementById('detail-listed-btn')?.addEventListener('click', async ()=>{
  if (!state.editDetailId) return;
  try {
    await API.toggleListingListed(state.editDetailId);
    await loadListings();
    const il = state.listings.find(l=>l.id===state.editDetailId);
    const btn = document.getElementById('detail-listed-btn');
    if (btn) btn.textContent = il?.is_listed ? '👁 Gizle' : '👁 Göster';
    showToast(il?.is_listed ? 'İlan gösteriliyor.' : 'İlan gizlendi.');
  } catch(e) { showToast(e.message,'error'); }
});

document.getElementById('detail-toggle-btn')?.addEventListener('click', async ()=>{
  if (!state.editDetailId) return;
  const il = state.listings.find(l=>l.id===state.editDetailId);
  if (!il) return;
  document.getElementById('detail-overlay').classList.remove('open');
  if (il.is_active) {
    openPassiveModal(state.editDetailId, il.fields?.title||'');
  } else {
    try {
      await API.toggleListing(state.editDetailId, {});
      await loadListings();
      showToast('İlan aktif edildi.');
    } catch(e) { showToast(e.message,'error'); }
  }
});

document.getElementById('detail-edit-btn').addEventListener('click',()=>{
  document.getElementById('detail-overlay').classList.remove('open');
  openEditListing(state.editDetailId);
});
document.getElementById('detail-history-btn').addEventListener('click', async ()=>{
  if (!state.editDetailId) return;
  await openHistoryModal(state.editDetailId);
});

/* ─── Tarihçe Modal ───────────────────────────────────────── */
async function openHistoryModal(listingId) {
  document.getElementById('history-overlay').classList.add('open');
  document.getElementById('history-content').innerHTML = '<div class="muted" style="padding:16px">Yükleniyor...</div>';
  try {
    const history = await API.getListingHistory(listingId);
    if (!history.length) {
      document.getElementById('history-content').innerHTML = '<div class="muted" style="padding:16px;text-align:center">Tarihçe yok.</div>';
      return;
    }
    const ACTION_LABEL = {
      created:'Oluşturuldu', updated:'Güncellendi',
      activated:'Aktif Edildi', deactivated:'Pasife Alındı',
      listed:'Listeye Alındı', unlisted:'Listeden Çıkarıldı'
    };
    const ACTION_ICON = {
      created:'✨', updated:'✏️', activated:'▶️', deactivated:'⏸',
      listed:'👁', unlisted:'👁‍🗨'
    };
    document.getElementById('history-content').innerHTML =
      `<div class="history-list">${history.map(h=>`
        <div class="history-item">
          <div class="history-icon">${ACTION_ICON[h.action]||'•'}</div>
          <div class="history-info">
            <div class="history-action">${ACTION_LABEL[h.action]||h.action}</div>
            <div class="history-meta">
              ${h.user_name} · ${new Date(h.created_at).toLocaleDateString('tr-TR',{day:'2-digit',month:'long',year:'numeric',hour:'2-digit',minute:'2-digit'})}
              ${h.status&&h.status!=='aktif'?`<span class="tag tag-sm ${STATUS_COLOR[h.status]||''}" style="margin-left:6px">${STATUS_LABEL[h.status]||h.status}</span>`:''}
              ${h.closing_price?`<span class="history-price">${fiyatFormat(h.closing_price)}</span>`:''}
            </div>
          </div>
        </div>`).join('')}</div>`;
  } catch(e) {
    document.getElementById('history-content').innerHTML = `<div class="alert alert-error">${e.message}</div>`;
  }
}
document.getElementById('close-history-modal').addEventListener('click',()=>
  document.getElementById('history-overlay').classList.remove('open'));

/* ─── İlan Form ───────────────────────────────────────────── */
document.getElementById('yeni-ilan-btn').addEventListener('click',()=>openIlanModal());
document.getElementById('fab-ilan-btn')?.addEventListener('click',()=>openIlanModal());
document.getElementById('close-modal').addEventListener('click',closeIlanModal);
document.getElementById('iptal-btn').addEventListener('click',closeIlanModal);
document.getElementById('ilan-overlay').addEventListener('click',e=>{
  if (e.target===document.getElementById('ilan-overlay')) closeIlanModal();
});

function openIlanModal(ilan=null) {
  state.editListingId   = ilan?.id||null;
  state.coverPath       = ilan?.cover_image||'';
  state.coverURL        = ilan?.cover_image||'';
  state.galleryPaths    = [];
  state.galleryExisting = ilan?.images||[];
  state.removedImageIds = [];
  document.getElementById('modal-title').textContent = ilan?'İlanı Düzenle':'Yeni İlan Ekle';
  buildIlanForm(ilan);
  renderCoverPreview();
  renderGalleryPreview();
  document.getElementById('ilan-overlay').classList.add('open');
}

async function openEditListing(id,e) {
  if (e) e.stopPropagation();
  try { const il=await API.getListing(id); openIlanModal(il); }
  catch(err) { showToast(err.message,'error'); }
}

function closeIlanModal() { document.getElementById('ilan-overlay').classList.remove('open'); }

function buildIlanForm(ilan) {
  const cfg      = state.cfg;
  const isAdmin  = API.isAdmin();
  const allFields= cfg?.listing_fields?.all_fields||[];
  const propType = ilan?.fields?.property_type||'';
  renderIlanFormFields(allFields, ilan, isAdmin, propType);
  renderIlanChannels(ilan);
  renderIlanAutoTasks(ilan);
}

function renderIlanChannels(ilan) {
  const grid = document.getElementById('ilan-channels-grid');
  if (!grid) return;
  const channels = state.cfg?.listing_channels || [];
  const selected = ilan?.fields?.channels ? ilan.fields.channels.split(',') : 
    channels.filter(c=>c.active).map(c=>c.key);
  grid.innerHTML = channels.map(ch => {
    const isSel = selected.includes(ch.key);
    return '<div class="channel-card' + (isSel?' selected':'') + '" data-ch="'+ch.key+'">' +
      '<span class="channel-card-icon">' + (ch.icon||'🌐') + '</span>' +
      '<div class="channel-card-name">' + escHtml(ch.label) + '</div>' +
    '</div>';
  }).join('');
  // Event delegation
  grid.querySelectorAll('.channel-card').forEach(el => {
    el.addEventListener('click', () => el.classList.toggle('selected'));
  });
}

function renderIlanAutoTasks(ilan) {
  const list = document.getElementById('ilan-autotasks-list');
  if (!list) return;
  const tasks = state.cfg?.auto_task_templates || [];
  list.innerHTML = tasks.map(t => {
    const checked = t.active ? 'checked' : '';
    return '<label class="check-item">' +
      '<input type="checkbox" ' + checked + ' data-task="'+t.key+'" data-label="'+escHtml(t.label)+'">' +
      escHtml(t.label) +
    '</label>';
  }).join('');
}

function getSelectedChannels() {
  const selected = [];
  document.querySelectorAll('#ilan-channels-grid .channel-card.selected').forEach(el => {
    selected.push(el.dataset.ch);
  });
  return selected.join(',');
}

function getSelectedAutoTasks() {
  const tasks = [];
  document.querySelectorAll('#ilan-autotasks-list input[type="checkbox"]:checked').forEach(el => {
    tasks.push({key: el.dataset.task, label: el.dataset.label});
  });
  return tasks;
}

function renderIlanFormFields(allFields, ilan, isAdmin, propType) {
  const cfg = state.cfg;
  const propSpecificKeys = propType ? (cfg?.listing_fields?.card_fields?.[propType]||[]) : null;
  const alwaysShow = ['title','listing_type','property_type','district','neighborhood',
                      'price','price_min','price_max','area_m2','description','notes','address'];

  // Yan yana gosterilecek ciftler
  const PAIR_FIELDS = [
    ['listing_type','property_type'],
    ['district','neighborhood'],
    ['area_m2','rooms'],
    ['heating','age'],
    ['zoning','deed_status'],
    ['total_floors','aidat'],
  ];

  const listingType = ilan?.fields?.listing_type || '';
  const fields_filtered = allFields
    .filter(f => !f.admin_only||isAdmin)
    .filter(f => f.key!=='price')
    .filter(f => fieldVisible(f, 'form', listingType, propType||''));

  function buildFieldInput(f, val) {
    if (f.type==='select') {
      const opts = cfg.field_sources?.[f.source]||[];
      return `<select id="f-${f.key}" ${f.required?'required':''}>
        <option value="">Seçin...</option>
        ${opts.map(o=>`<option ${o===val?'selected':''}>${o}</option>`).join('')}
      </select>`;
    } else if (f.type==='textarea') {
      return `<textarea id="f-${f.key}" rows="3">${val}</textarea>`;
    } else {
      return `<input id="f-${f.key}" type="text" inputmode="${f.type==='number'?'numeric':''}"
        value="${val}" placeholder="${f.label}" ${f.required?'required':''}>`;
    }
  }

  function buildFieldGroup(f) {
    const val = ilan?.fields?.[f.key]||'';
    return `<div class="form-group" id="fg-${f.key}">
      <label>${f.label}${f.required?' <span class="req">*</span>':''}</label>
      ${buildFieldInput(f, val)}
    </div>`;
  }

  const rendered = new Set();
  let html = '';

  fields_filtered.forEach(f => {
    if (rendered.has(f.key)) return;
    const pair = PAIR_FIELDS.find(p => p[0]===f.key);
    if (pair) {
      const f2 = fields_filtered.find(x => x.key===pair[1]);
      if (f2) {
        html += `<div class="form-row-2">${buildFieldGroup(f)}${buildFieldGroup(f2)}</div>`;
        rendered.add(f.key); rendered.add(f2.key);
        return;
      }
    }
    html += buildFieldGroup(f);
    rendered.add(f.key);
  });

  const priceVal = ilan?.fields?.price||ilan?.fields?.price_max||'';
  const priceBlock = `
    <div class="form-group">
      <label>Fiyat (₺) <span class="req">*</span></label>
      <input id="f-price" type="text" inputmode="numeric"
        value="${formatDisplayPrice(priceVal)}" data-raw="${priceVal}"
        placeholder="Örn: 4.500.000" oninput="formatPriceInput(this)">
    </div>`;

  const customerBlock = buildCustomerSelectBlock(ilan?.customer_id);
  document.getElementById('ilan-form-body').innerHTML = html + priceBlock + customerBlock;
  fillCustomerDropdown(ilan?.customer_id || null);

  document.getElementById('f-property_type')?.addEventListener('change', function() {
    updateIlanFormForPropType(this.value, allFields, isAdmin);
  });
  document.getElementById('f-listing_type')?.addEventListener('change', function() {
    const pt = document.getElementById('f-property_type')?.value || '';
    updateIlanFormForPropType(pt, allFields, isAdmin);
  });
  // Baslangicta da uygula
  const initPt = ilan?.fields?.property_type || '';
  if (initPt) updateIlanFormForPropType(initPt, allFields, isAdmin);
}

function updateIlanFormForPropType(propType, allFields, isAdmin) {
  const listingType = document.getElementById('f-listing_type')?.value || '';
  allFields.filter(f=>!f.admin_only||isAdmin).forEach(f => {
    const fg = document.getElementById('fg-'+f.key);
    if (!fg) return;
    const visible = fieldVisible(f, 'form', listingType, propType);
    fg.style.display = visible ? '' : 'none';
    const row2 = fg.closest('.form-row-2');
    if (row2) {
      const visibles = [...row2.querySelectorAll('.form-group')].filter(g=>g.style.display!=='none');
      row2.style.display = visibles.length===0 ? 'none' : '';
      row2.style.gridTemplateColumns = visibles.length===1 ? '1fr' : '';
    }
  });
}

document.getElementById('kaydet-btn').addEventListener('click', async ()=>{
  const btn = document.getElementById('kaydet-btn');
  const originalText = btn.textContent;
  btn.disabled = true;
  btn.textContent = '⏳ Kaydediliyor...';
  const fields = {};
  (state.cfg?.listing_fields?.all_fields||[]).forEach(f => {
    if (f.key==='price') return;
    const el = document.getElementById('f-'+f.key);
    if (el) fields[f.key] = el.value;
  });
  fields.price = getRawPrice("f-price");
  fields.price_max = fields.price;
  fields.price_min = "";
  fields.channels = getSelectedChannels();
  fields.pipeline_stage = document.getElementById('f-pipeline_stage')?.value || 'bilgi_alindi';

  if (!fields.property_type) { showToast('Mülk tipi zorunludur','error'); btn.disabled=false; btn.textContent=originalText; return; }
  if (!fields.title) { showToast('Başlık zorunludur','error'); btn.disabled=false; btn.textContent=originalText; return; }
  if (!fields.price) { showToast("Fiyat zorunludur","error"); btn.disabled=false; btn.textContent=originalText; return; }

  try {
    const cid = parseInt(document.getElementById("f-customer_id")?.value)||0;
    const payload = {
      fields, cover_image: state.coverPath,
      images: state.galleryPaths.map(g=>g.path),
      remove_images: state.removedImageIds,
      customer_id: cid,
    };
    if (state.editListingId) {
      await API.updateListing(state.editListingId, payload);
      showToast('İlan güncellendi!');
    } else {
      const newListing = await API.createListing(payload);
      showToast('İlan eklendi!');
      // Otomatik gorevler olustur
      const autoTasks = getSelectedAutoTasks();
      if (autoTasks.length && newListing?.id) {
        for (const t of autoTasks) {
          try {
            await API.createTask({
              title: t.label + ' — ' + (fields.title||''),
              description: '',
              status: 'bekliyor',
              priority: 'normal',
              due_date: null,
              assignees: [API.getUserID()],
              parent_id: null,
            });
          } catch(_) {}
        }
      }
      // Pipeline guncelle
      if (newListing?.id && fields.pipeline_stage) {
        try { await API.updatePipeline(newListing.id, fields.pipeline_stage); } catch(_) {}
      }
    }
    closeIlanModal();
    await loadListings();
  } catch(e) { showToast(e.message,'error'); }
  finally { btn.disabled = false; btn.textContent = originalText; }
});

/* ─── Cover Upload ────────────────────────────────────────── */
document.getElementById('cover-zone').addEventListener('click',()=>document.getElementById('cover-input').click());
document.getElementById('cover-input').addEventListener('change', async function(){
  const file=this.files[0]; if(!file) return;
  try {
    showToast('Resim yükleniyor...','info');
    const propType = document.getElementById('f-property_type')?.value||'';
    const res = await API.uploadCover(file, propType, state.editListingId||0);
    state.coverPath=res.path; state.coverURL=res.url;
    renderCoverPreview(res.url); showToast('Vitrin resmi yüklendi.');
  } catch(e) { showToast(e.message,'error'); }
  this.value='';
});
document.getElementById('remove-cover').addEventListener('click',e=>{
  e.stopPropagation(); state.coverPath=state.coverURL=''; renderCoverPreview();
});
function renderCoverPreview(url) {
  const show=url||state.coverURL;
  document.getElementById('cover-placeholder').style.display=show?'none':'';
  document.getElementById('cover-preview').style.display=show?'block':'none';
  if(show) document.getElementById('cover-img').src=show;
}
const coverZone=document.getElementById('cover-zone');
coverZone.addEventListener('dragover',e=>{e.preventDefault();coverZone.classList.add('drag-over');});
coverZone.addEventListener('dragleave',()=>coverZone.classList.remove('drag-over'));
coverZone.addEventListener('drop',async e=>{
  e.preventDefault(); coverZone.classList.remove('drag-over');
  const file=e.dataTransfer.files[0]; if(!file?.type.startsWith('image/')) return;
  try { const res=await API.uploadCover(file); state.coverPath=res.path; state.coverURL=res.url; renderCoverPreview(res.url); }
  catch(err) { showToast(err.message,'error'); }
});

/* ─── Gallery Upload ──────────────────────────────────────── */
document.getElementById('gallery-zone').addEventListener('click',e=>{
  if(!e.target.closest('.gallery-preview')) document.getElementById('gallery-input').click();
});
document.getElementById('gallery-input').addEventListener('change', async function(){
  const files=Array.from(this.files);
  const propType2 = document.getElementById('f-property_type')?.value||'';
  const maxLeft=25-state.galleryPaths.length-state.galleryExisting.length;
  for (const file of files.slice(0,maxLeft)) {
    try { const res=await API.uploadGallery(file, propType2, state.editListingId||0); state.galleryPaths.push({path:res.path,url:res.url}); renderGalleryPreview(); }
    catch(e) { showToast(e.message,'error'); }
  }
  if(files.length>maxLeft) showToast(`En fazla ${maxLeft} resim daha eklenebilir.`,'error');
  this.value='';
});
function renderGalleryPreview() {
  const cont=document.getElementById('gallery-preview');
  cont.innerHTML=
    state.galleryExisting.map(img=>`<div class="photo-thumb"><img src="${img.path}" alt="" loading="lazy">
      <button class="remove-photo" data-type="existing" data-id="${img.id}">×</button></div>`).join('') +
    state.galleryPaths.map((g,i)=>`<div class="photo-thumb"><img src="${g.url}" alt="" loading="lazy">
      <button class="remove-photo" data-type="new" data-idx="${i}">×</button></div>`).join('');
  cont.querySelectorAll('.remove-photo').forEach(btn=>{
    btn.addEventListener('click',e=>{
      e.stopPropagation();
      if(btn.dataset.type==='existing') {
        const imgId=parseInt(btn.dataset.id);
        state.removedImageIds.push(imgId);
        state.galleryExisting=state.galleryExisting.filter(i=>i.id!==imgId);
      } else { state.galleryPaths.splice(parseInt(btn.dataset.idx),1); }
      renderGalleryPreview();
    });
  });
}

/* ═══════════════════════════════════════════════════════
   TALEPLER
════════════════════════════════════════════════════════ */
async function loadRequests() {
  try {
    await ensureUsers();
    state.requests=await API.getRequests({})||[]; renderRequests(); }
  catch(e) { showToast('Talepler yüklenemedi: '+e.message,'error'); }
}

function calcMatchScore(talep, ilan) {
  if(!ilan.is_active) return 0;
  // Mülk tipi ve satış tipi belirtilmişse kesin eşleşme zorunlu
  if(talep.fields?.property_type && talep.fields.property_type !== ilan.fields?.property_type) return 0;
  if(talep.fields?.listing_type  && talep.fields.listing_type  !== ilan.fields?.listing_type)  return 0;
  let score=0, total=0;
  const check=(tVal,iVal,w)=>{ total+=w; if(!tVal) score+=w; else if(tVal===iVal) score+=w; };
  check(talep.fields?.listing_type,  ilan.fields?.listing_type,  25);
  check(talep.fields?.property_type, ilan.fields?.property_type, 20);
  check(talep.fields?.district,      ilan.fields?.district,      20);
  total+=20;
  const budgetMax = parseInt(talep.fields?.budget_max||talep.fields?.budget)||0;
  const budgetMin = parseInt(talep.fields?.budget_min)||0;
  const price     = parseInt(ilan.fields?.price_max||ilan.fields?.price)||0;
  if (!budgetMax) score+=20;
  else if (price<=budgetMax) score+=20;          // bütçe içinde
  else if (price<=budgetMax*1.1) score+=10;      // %10 üstüne kadar kısmi puan
  else return 0;                                 // %10 üstünde — eşleşme yok
  // Alt sınır: bütçe_min'in %10 altına kadar kabul et
  if (budgetMin && price < budgetMin*0.9) return 0;
  total+=15;
  const tOda=talep.fields?.rooms, iOda=ilan.fields?.rooms;
  if(!tOda) score+=15; else if(tOda===iOda) score+=15;
  return Math.round((score/total)*100);
}

function scoreColor(pct) {
  if(pct>=80) return {bg:'#eaf3de',c:'#27500a'};
  if(pct>=60) return {bg:'#faeeda',c:'#633806'};
  return {bg:'#fcebeb',c:'#501313'};
}

function renderRequests() {
  const list = document.getElementById('talep-list');
  const q    = document.getElementById('talep-search')?.value?.toLowerCase()||'';
  const lt   = document.getElementById('talep-tip-filter')?.value||'';
  const d    = document.getElementById('talep-ilce-filter')?.value||'';
  const benim = document.getElementById('talep-benim-btn')?.classList.contains('active');
  const myID  = API.getUserID();

  let data = state.requests.filter(t => {
    if (lt && t.fields?.listing_type !== lt) return false;
    if (d  && t.fields?.district     !== d)  return false;
    if (benim && t.user_id !== myID)          return false;
    if (q && !t.fields?.client_name?.toLowerCase().includes(q)
          && !t.fields?.district?.toLowerCase().includes(q)) return false;
    return true;
  });

  if (!data.length) {
    list.innerHTML = '<div class="empty-state"><div class="big-icon">🎯</div><p>Talep bulunamadı.</p></div>';
    return;
  }

  const avatarColors = ['#1565C0','#6a1b9a','#1b5e20','#c62828','#e65100','#00695c'];

  list.innerHTML = data.map((t, idx) => {
    const c      = avatarColors[idx % avatarColors.length];
    const harf   = (t.fields?.client_name||'M')[0].toUpperCase();
    const isOwner = String(t.user_id) === String(myID);

    const matches = state.listings
      .map(il => ({il, score: calcMatchScore(t, il)}))
      .filter(m => m.score > 0)
      .sort((a, b) => b.score - a.score);

    const bMin = t.fields?.budget_min;
    const bMax = t.fields?.budget_max || t.fields?.budget;
    const budgetDisplay = bMin && bMax
      ? `${fiyatFormat(bMin)} – ${fiyatFormat(bMax)}`
      : bMax ? `maks ${fiyatFormat(bMax)}` : '';

    // Eşleşme badge rengi
    const matchBadgeClass = matches.length >= 3 ? 'match-high'
      : matches.length >= 1 ? 'match-mid' : 'match-zero';

    // Kriterler + açıklama
    const tags = [
      t.fields?.listing_type  ? `<span class="rq-tag t-blue">${t.fields.listing_type}</span>`  : '',
      t.fields?.property_type ? `<span class="rq-tag t-purple">${t.fields.property_type}</span>` : '',
      t.fields?.district      ? `<span class="rq-tag t-green">${t.fields.district}</span>`     : '',
      t.fields?.neighborhood  ? `<span class="rq-tag t-teal">${t.fields.neighborhood}</span>`  : '',
      t.fields?.rooms         ? `<span class="rq-tag t-gray">${t.fields.rooms}</span>`         : '',
      budgetDisplay           ? `<span class="rq-tag t-amber">${budgetDisplay}</span>`         : '',
    ].filter(Boolean).join('');

    const notesHTML = t.fields?.notes
      ? `<div class="rq-note">${escHtml(t.fields.notes)}</div>` : '';

    // Talep sahibi adı
    const ownerUser = state.allUsers?.find(u => u.id === t.user_id);
    const ownerName = ownerUser?.full_name || ownerUser?.username || '?';
    const nameHTML = isOwner
      ? `<span class="rq-owner"><i class="ti ti-user"></i> ${escHtml(ownerName)}</span>
         <span class="rq-mine">Benim müşterim</span>`
      : `<span class="rq-owner"><i class="ti ti-user"></i> ${escHtml(ownerName)}</span>`;

    // Akordiyon içi — sadece owner ise Malik butonu
    const musteriKart = (isOwner && t.customer_id) ? `
      <div class="rq-malik-row">
        <button class="rq-malik-btn" onclick="goToMalik(${t.customer_id});event.stopPropagation()">
          <i class="ti ti-user"></i> Müşteriye Git
        </button>
      </div>` : '';

    // İlanlar
    const ilanlarHTML = !matches.length
      ? '<div class="rq-empty"><i class="ti ti-search"></i> Eşleşen ilan bulunamadı</div>'
      : matches.map(({il, score}) => {
          const r = scoreColor(score);
          const propType = il.fields?.property_type||'Daire';
          const cardKeys = state.cfg?.listing_fields?.card_fields?.[propType]||[];
          const priceMin = il.fields?.price_min, priceMax = il.fields?.price_max;
          const priceDisp = priceMin||priceMax
            ? (priceMin&&priceMax ? `${fiyatFormat(priceMin)}–${fiyatFormat(priceMax)}` : fiyatFormat(priceMin||priceMax))
            : fiyatFormat(il.fields?.price);
          const thumb = il.cover_image
            ? `<img src="${il.cover_image}" class="rq-ilan-thumb" loading="lazy" alt="">`
            : `<div class="rq-ilan-icon">${(PROP_PLACEHOLDER[propType]||PROP_PLACEHOLDER.default).icon}</div>`;
          const scoreClass = score >= 80 ? 'spill-high' : score >= 60 ? 'spill-mid' : 'spill-low';
          // Sabit gösterilecek alanlar: mülk tipi, mahalle
          const metaParts = [
            il.fields?.property_type,
            il.fields?.neighborhood || il.fields?.district,
          ].filter(Boolean).map(v => `<span class="rq-meta-tag">${v}</span>`).join('');
          return `<div class="rq-ilan-row" onclick="openDetailModal(${il.id})">
            ${thumb}
            <div class="req-ilan-info">
              <div class="rq-ilan-title">${escHtml(il.fields?.title||'—')}
                ${il.listing_no ? `<span class="rq-ilan-no">#${il.listing_no}</span>` : ''}
              </div>
              <div class="rq-ilan-meta">${metaParts}</div>
            </div>
            <div class="rq-ilan-right">
              <span class="req-score-pill ${scoreClass}">%${score}</span>
              <span class="rq-price">${priceDisp}</span>
            </div>
          </div>`;
        }).join('');

    return `<div class="rq-card${t.is_active ? '' : ' rq-passive'}" id="talep-${t.id}">
      <div class="rq-hdr" onclick="toggleTalepAcc(${t.id})">
        <div class="rq-av" style="background:${c}20;color:${c}">${harf}</div>
        <div class="rq-body">
          <div class="rq-r1">
            ${nameHTML}
          </div>
          ${notesHTML}
          <div class="rq-tags">${tags}</div>
        </div>
        <div class="rq-acts">
          <span class="rq-match ${matchBadgeClass}">${matches.length} eşleşme</span>
          ${t.customer_id ? `<span class="rq-link" title="Müşteri bağlı"><i class="ti ti-link" aria-hidden="true"></i></span>` : ''}
          <button class="rq-ibt${t.notify_me?' rq-notify-on':''}" onclick="doToggleNotify(${t.id},event)" title="${t.notify_me?'Bildirimi Kapat':'Bildirim Aç'}">
            <i class="ti ${t.notify_me ? 'ti-bell-ringing' : 'ti-bell'}" aria-hidden="true"></i>
          </button>
          <button class="rq-ibt" onclick="openEditRequest(${t.id},event)" title="Düzenle">
            <i class="ti ti-edit" aria-hidden="true"></i>
          </button>
          ${(isOwner || API.isAdmin()) ? `
          <button class="rq-ibt rq-danger" onclick="doDeleteRequest(${t.id},event)" title="Sil">
            <i class="ti ti-trash" aria-hidden="true"></i>
          </button>
          <button class="rq-sbtn${t.is_active?' rq-active':' rq-passive'}" onclick="doToggleRequest(${t.id},event)">
            <i class="ti ${t.is_active?'ti-player-pause':'ti-player-play'}" aria-hidden="true"></i>
            ${t.is_active ? 'Aktif' : 'Pasif'}
          </button>` : ''}
          <i class="ti ti-chevron-down rq-chev" id="ok-${t.id}" aria-hidden="true"></i>
        </div>
      </div>
      <div class="rq-acc" id="acc-${t.id}">
        ${musteriKart}
        <div class="rq-ilanlar">
          <div class="rq-acc-title">Eşleşen ilanlar (${matches.length})</div>
          ${ilanlarHTML}
        </div>
      </div>
    </div>`;
  }).join('');
}

function talepViewToggle(view, btn) {
  document.querySelectorAll('.toolbar-toggle').forEach(b => b.classList.remove('active'));
  btn.classList.add('active');
  renderRequests();
}

function toggleTalepAcc(id) {
  const acc  = document.getElementById('acc-'+id);
  const chev = document.getElementById('ok-'+id);
  const card = document.getElementById('talep-'+id);
  acc.classList.toggle('open');
  if (chev) chev.classList.toggle('open');
  if (card) card.classList.toggle('rq-open');
}
async function doDeleteRequest(id, e) {
  e.stopPropagation();
  if (!confirm('Bu talebi silmek istediğinizden emin misiniz?')) return;
  try {
    await API.adminDeleteRequest(id);
    await loadRequests();
    showToast('Talep silindi.');
  } catch(err) { showToast(err.message, 'error'); }
}

async function doToggleNotify(id,e) {
  e.stopPropagation();
  try { await API.toggleRequestNotify(id); await loadRequests(); } catch(err) { showToast(err.message,'error'); }
}
async function doToggleRequest(id,e) {
  e.stopPropagation();
  try { await API.toggleRequest(id); await loadRequests(); showToast('Talep durumu güncellendi.'); }
  catch(err) { showToast(err.message,'error'); }
}
['talep-search','talep-tip-filter','talep-ilce-filter'].forEach(id=>{
  document.getElementById(id)?.addEventListener('input', renderRequests);
  document.getElementById(id)?.addEventListener('change',renderRequests);
});

/* ─── Talep Modal ─────────────────────────────────────────── */
document.getElementById('yeni-talep-btn').addEventListener('click',()=>openTalepModal());
document.getElementById('close-talep-modal').addEventListener('click',closeTalepModal);
document.getElementById('talep-iptal-btn').addEventListener('click',closeTalepModal);

function openTalepModal(talep=null) {
  state.editRequestId=talep?.id||null;
  document.getElementById('talep-modal-title').textContent=talep?'Talebi Düzenle':'Yeni Talep Ekle';
  buildTalepForm(talep);
  document.getElementById('talep-overlay').classList.add('open');
}
async function openEditRequest(id,e) {
  if(e) e.stopPropagation();
  const t=state.requests.find(r=>r.id===id);
  if(t) openTalepModal(t);
}
function closeTalepModal() { document.getElementById('talep-overlay').classList.remove('open'); }

/* ── Müşteri Combobox (Talep Formu) ─────────────────────────── */
let _comboboxDebounce = null;
let _comboboxSelected = null; // {id, name, phone}

function buildCustomerCombobox(name='', phone='', customerId='') {
  return `
  <input type="hidden" id="tf-customer_id" value="${customerId||''}">
  <div class="form-group customer-combobox-group">
    <label>Müşteri Adı <span class="req">*</span>
      <span class="combobox-hint" id="combobox-linked-hint" style="display:none">
        🔗 <a href="#" onclick="clearComboboxLink(event)">Bağlantıyı kaldır</a>
      </span>
    </label>
    <div class="combobox-wrap">
      <input id="tf-client_name" type="text" autocomplete="off" placeholder="Ad yazın veya müşteri arayın…"
        value="${escapeAttr(name)}" oninput="onComboboxInput(this)">
      <ul id="combobox-dropdown" class="combobox-dropdown" style="display:none"></ul>
    </div>
  </div>
  <div class="form-group">
    <label>Telefon <span class="req">*</span></label>
    <input id="tf-phone" type="text" value="${escapeAttr(phone)}" placeholder="05xx xxx xx xx">
  </div>`;
}

function escapeAttr(s) {
  return (s||'').replace(/"/g,'&quot;').replace(/'/g,'&#39;');
}

function onComboboxInput(el) {
  clearTimeout(_comboboxDebounce);
  // Eğer daha önce bir müşteri seçilmişse, link'i temizle
  if (_comboboxSelected) clearComboboxLinkState();
  const q = el.value.trim();
  if (q.length < 2) { hideComboboxDropdown(); return; }
  _comboboxDebounce = setTimeout(() => searchComboboxCustomers(q), 250);
}

async function searchComboboxCustomers(q) {
  try {
    const customers = await API.getCustomers({q}) || [];
    renderComboboxDropdown(customers);
  } catch(e) { hideComboboxDropdown(); }
}

function renderComboboxDropdown(customers) {
  const dd = document.getElementById('combobox-dropdown');
  if (!dd) return;
  if (!customers.length) { hideComboboxDropdown(); return; }
  dd.innerHTML = customers.map(c => `
    <li class="combobox-item" onclick="selectComboboxCustomer(${c.id},'${escapeAttr(c.name)}','${escapeAttr(c.phone||'')}')">
      <span class="combobox-name">${c.name}</span>
      ${c.phone ? `<span class="combobox-phone">${c.phone}</span>` : ''}
    </li>`).join('');
  dd.style.display = 'block';
}

function selectComboboxCustomer(id, name, phone) {
  _comboboxSelected = {id, name, phone};
  const nameEl  = document.getElementById('tf-client_name');
  const phoneEl = document.getElementById('tf-phone');
  const cidEl   = document.getElementById('tf-customer_id');
  const hint    = document.getElementById('combobox-linked-hint');
  if (nameEl)  nameEl.value  = name;
  if (phoneEl && phone) phoneEl.value = phone;
  if (cidEl)   cidEl.value   = id;
  if (hint)    hint.style.display = 'inline';
  hideComboboxDropdown();
}

function clearComboboxLink(e) {
  e?.preventDefault();
  clearComboboxLinkState();
}

function clearComboboxLinkState() {
  _comboboxSelected = null;
  const cidEl = document.getElementById('tf-customer_id');
  const hint  = document.getElementById('combobox-linked-hint');
  if (cidEl) cidEl.value = '';
  if (hint)  hint.style.display = 'none';
}

function hideComboboxDropdown() {
  const dd = document.getElementById('combobox-dropdown');
  if (dd) dd.style.display = 'none';
}

// Dışarı tıklayınca dropdown kapansın
document.addEventListener('click', e => {
  if (!e.target.closest('.combobox-wrap')) hideComboboxDropdown();
});

// ── Talep formu — statik alan tanımları ─────────────────────────
// Her mülk tipi için hangi alanların gösterileceği burada tanımlı.
// İleride DB'ye taşınabilir ama şimdilik sade ve net.
const TALEP_FIELDS = {
  'Daire':  ['district','neighborhood','rooms','budget'],
  'Arsa':   ['district','neighborhood','zoning','budget'],
  'Villa':  ['district','neighborhood','budget'],
  'Ticari': ['district','neighborhood','budget'],
  // Bilinmeyen mülk tipleri için varsayılan
  '_default': ['district','neighborhood','budget'],
};

// Alan tanımları — label, type, source
const TALEP_FIELD_DEFS = {
  district:     { label:'İlçe',         type:'select', source:'districts',     required:true  },
  neighborhood: { label:'Mahalle',      type:'select', source:'neighborhoods', required:false },
  rooms:        { label:'Oda Sayısı',   type:'select', source:'room_options',  required:false },
  zoning:       { label:'İmar Durumu',  type:'select', source:'zoning_options',required:false },
  budget:       { label:'Bütçe Aralığı (₺)', type:'budget', required:false },
};

function buildTalepForm(talep) {
  const cfg      = state.cfg;
  const propType = talep?.fields?.property_type || '';

  // ── 1. Mülk tipine göre alan listesi ────────────────────────────
  const fieldKeys = TALEP_FIELDS[propType] || TALEP_FIELDS['_default'];

  // ── 2. Select options helper ─────────────────────────────────────
  const buildSelect = (key, def, val) => {
    const opts = cfg?.field_sources?.[def.source] || [];
    return `<select id="tf-${key}" ${def.required ? 'required' : ''}>
      <option value="">Seçin…</option>
      ${opts.map(o => `<option ${o===val ? 'selected' : ''}>${o}</option>`).join('')}
    </select>`;
  };

  // ── 3. Bütçe bloğu ───────────────────────────────────────────────
  const bMin = talep?.fields?.budget_min || '';
  const bMax = talep?.fields?.budget_max || talep?.fields?.budget || '';
  const budgetHTML = `
    <div class="form-group">
      <label>Bütçe Aralığı (₺)</label>
      <div class="price-range-row">
        <input id="tf-budget_min" type="text" inputmode="numeric"
          value="${formatDisplayPrice(bMin)}" data-raw="${bMin}"
          placeholder="En az" oninput="formatPriceInput(this)">
        <span class="price-range-sep">—</span>
        <input id="tf-budget_max" type="text" inputmode="numeric"
          value="${formatDisplayPrice(bMax)}" data-raw="${bMax}"
          placeholder="En fazla" oninput="formatPriceInput(this)">
      </div>
    </div>`;

  // ── 4. Alanları render et ────────────────────────────────────────
  const fieldsHTML = fieldKeys.map(key => {
    if (key === 'budget') return budgetHTML;
    const def = TALEP_FIELD_DEFS[key];
    if (!def) return '';
    const val = talep?.fields?.[key] || '';
    const input = def.type === 'select'
      ? buildSelect(key, def, val)
      : `<input id="tf-${key}" type="text" value="${val}" placeholder="${def.label}" ${def.required ? 'required' : ''}>`;
    return `<div class="form-group">
      <label>${def.label}${def.required ? ' <span class="req">*</span>' : ''}</label>
      ${input}
    </div>`;
  }).join('');

  // ── 5. Notlar ────────────────────────────────────────────────────
  const notesHTML = `
    <div class="form-group">
      <label>Notlar</label>
      <textarea id="tf-notes" rows="2">${talep?.fields?.notes || ''}</textarea>
    </div>`;

  // ── 6. Mülk tipi seçimi — her zaman ilk alan ────────────────────
  const propTypes = cfg?.property_types || ['Daire','Villa','Arsa','Ticari'];
  const propTypeHTML = `
    <div class="form-group">
      <label>Mülk Tipi <span class="req">*</span></label>
      <select id="tf-property_type" required>
        <option value="">Seçin…</option>
        ${propTypes.map(p => `<option ${p===propType ? 'selected' : ''}>${p}</option>`).join('')}
      </select>
    </div>`;

  // ── 7. Combobox (müşteri) — en üstte ────────────────────────────
  _comboboxSelected = null;
  const comboboxBlock = buildCustomerCombobox(
    talep?.fields?.client_name || '',
    talep?.fields?.phone || '',
    talep?.fields?.customer_id || ''
  );
  const hasCid = !!(talep?.fields?.customer_id);

  // ── 8. DOM'a yaz ─────────────────────────────────────────────────
  document.getElementById('talep-form-body').innerHTML =
    comboboxBlock +
    propTypeHTML +
    fieldsHTML +
    notesHTML +
    `<div class="form-group">
      <label style="display:flex;align-items:center;gap:8px;cursor:pointer">
        <input type="checkbox" id="tf-notify" ${talep?.notify_me ? 'checked' : ''}>
        Yeni eşleşmelerde bildir
      </label>
    </div>`;

  // ── 9. Mülk tipi değişince formu yeniden oluştur ─────────────────
  document.getElementById('tf-property_type')?.addEventListener('change', function() {
    const cid = document.getElementById('tf-customer_id')?.value || '';
    const currentVals = {};
    document.querySelectorAll('#talep-form-body [id^="tf-"]').forEach(el => {
      currentVals[el.id.replace('tf-', '')] = el.type === 'checkbox' ? el.checked : el.value;
    });
    buildTalepForm({
      ...talep,
      fields: { ...talep?.fields, ...currentVals, property_type: this.value, customer_id: cid },
      notify_me: document.getElementById('tf-notify')?.checked
    });
  });

  if (hasCid) {
    const hint = document.getElementById('combobox-linked-hint');
    if (hint) hint.style.display = 'inline';
  }
}

document.getElementById('talep-kaydet-btn').addEventListener('click', async ()=>{
  const fields = {};

  // Sabit alanlar
  fields.property_type = document.getElementById('tf-property_type')?.value||'';
  fields.client_name   = document.getElementById('tf-client_name')?.value?.trim()||'';
  fields.phone         = document.getElementById('tf-phone')?.value?.trim()||'';
  fields.notes         = document.getElementById('tf-notes')?.value||'';
  fields.budget_min    = getRawPrice('tf-budget_min');
  fields.budget_max    = getRawPrice('tf-budget_max');
  fields.budget        = fields.budget_max || fields.budget_min;

  // Mülk tipine göre değişen alanlar — TALEP_FIELDS'dan oku
  const fieldKeys = TALEP_FIELDS[fields.property_type] || TALEP_FIELDS['_default'];
  fieldKeys.forEach(key => {
    if (key === 'budget') return; // yukarıda halledildi
    const el = document.getElementById('tf-' + key);
    if (el) fields[key] = el.value;
  });

  const customerId = document.getElementById('tf-customer_id')?.value||'';
  if (customerId) fields.customer_id = customerId;

  if(!fields.client_name){showToast('Müşteri adı zorunludur','error');return;}
  if(!fields.phone){showToast('Telefon zorunludur','error');return;}
  const notify=document.getElementById('tf-notify')?.checked||false;
  try {
    // Müşteri combobox'tan seçilmemişse (el ile yazıldıysa) → otomatik müşteri oluştur/bul
    if (!fields.customer_id) {
      try {
        // Önce aynı isim+telefon var mı kontrol et
        const existing = await API.getCustomers({q: fields.client_name}) || [];
        const match = existing.find(c =>
          c.name.toLowerCase() === fields.client_name.toLowerCase() &&
          (c.phone||'').replace(/\s/g,'') === (fields.phone||'').replace(/\s/g,'')
        );
        if (match) {
          fields.customer_id = String(match.id);
        } else {
          // Yeni müşteri oluştur
          const newC = await API.createCustomer({
            name:  fields.client_name,
            phone: fields.phone,
          });
          if (newC?.id) {
            fields.customer_id = String(newC.id);
            showToast('👤 Müşteri otomatik oluşturuldu', 'info');
          }
        }
      } catch(cerr) {
        // Müşteri oluşturma başarısız olsa bile talebi kaydet
        console.warn('Otomatik müşteri oluşturma başarısız:', cerr);
      }
    }
    if(state.editRequestId){await API.updateRequest(state.editRequestId,{fields,notify_me:notify});showToast('✅ Talep güncellendi!');}
    else{await API.createRequest({fields,notify_me:notify});showToast('🎉 Talep eklendi!');}
    closeTalepModal(); await loadRequests();
    // Müşteri listesi açıksa yenile
    if (document.getElementById('page-musteriler')?.classList.contains('active')) loadCustomers();
  } catch(e){showToast(e.message,'error');}
});

/* ═══════════════════════════════════════════════════════
   MÜŞTERİLER (CRM)
════════════════════════════════════════════════════════ */
async function loadCustomers() {
  const q = document.getElementById('musteri-search')?.value || '';
  try {
    state.customers = await API.getCustomers(q ? {q} : {})||[];
    renderCustomers();
  } catch(e) { showToast('Müşteriler yüklenemedi: '+e.message,'error'); }
}

function renderCustomers() {
  const list = document.getElementById('musteri-list');
  if (!state.customers.length) {
    list.innerHTML = '<div class="empty-state"><div class="big-icon">👥</div><p>Müşteri bulunamadı.</p></div>';
    return;
  }
  const colors=['#1565C0','#6a1b9a','#1b5e20','#c62828','#e65100','#00695c'];
  list.innerHTML = state.customers.map((c, idx)=>{
    const col = colors[idx % colors.length];
    const harf = (c.name||'M')[0].toUpperCase();
    return `<div class="crm-item${c.is_active?'':' crm-passive'}">
      <div class="crm-card" onclick="toggleCrmAccordion(${c.id},event)">
        <div class="crm-avatar" style="background:${col}22;color:${col}">${harf}</div>
        <div class="crm-info">
          <div class="crm-name">${c.name}</div>
          <div class="crm-meta">
            ${c.phone?`📞 ${c.phone}`:''}
            ${c.email?` · 📧 ${c.email}`:''}
            ${c.source?` · <span class="tag tag-sm tag-blue">${c.source}</span>`:''}
          </div>
          ${c.notes?`<div class="crm-notes muted">${c.notes.slice(0,80)}${c.notes.length>80?'…':''}</div>`:''}
        </div>
        <div class="crm-actions">
          <button class="btn btn-sm btn-outline" onclick="openCrmLinkListing(${c.id},event)">+ İlan Bağla</button>
          <button class="icon-btn icon-btn-edit" onclick="openEditCustomer(${c.id},event)">✏️</button>
          <button class="icon-btn" onclick="doToggleCustomer(${c.id},event)" title="${c.is_active?'Pasife Al':'Aktif Et'}">${c.is_active?'⏸':'▶️'}</button>
          <button class="icon-btn icon-btn-delete" onclick="doDeleteCustomer(${c.id},event)">🗑️</button>
          <span class="crm-chevron" id="chev-${c.id}">▸</span>
        </div>
      </div>
      <div class="crm-accordion" id="crm-acc-${c.id}" style="display:block">
        <div class="crm-acc-loading">Yükleniyor...</div>
      </div>
    </div>`;
  }).join('');
  setTimeout(() => {
    state.customers.forEach(c => { toggleCrmAccordion(c.id, {target:{closest:()=>null}}); });
  }, 100);
}

document.getElementById('musteri-search')?.addEventListener('input', loadCustomers);
document.getElementById('yeni-musteri-btn').addEventListener('click', ()=>openMusteriModal());
document.getElementById('close-musteri-modal').addEventListener('click', closeMusteriModal);
document.getElementById('musteri-iptal-btn').addEventListener('click', closeMusteriModal);

function openMusteriModal(c=null) {
  state.editCustomerId = c?.id||null;
  document.getElementById('musteri-modal-title').textContent = c ? 'Müşteri Düzenle' : 'Yeni Müşteri';
  document.getElementById('m-name').value    = c?.name||'';
  document.getElementById('m-phone').value   = c?.phone||'';
  document.getElementById('m-email').value   = c?.email||'';
  document.getElementById('m-source').value  = c?.source||'';
  document.getElementById('m-notes').value   = c?.notes||'';
  document.getElementById('musteri-overlay').classList.add('open');
}
async function openEditCustomer(id, e) {
  if(e) e.stopPropagation();
  const c = state.customers.find(x=>x.id===id);
  if(c) openMusteriModal(c);
}
function closeMusteriModal() { document.getElementById('musteri-overlay').classList.remove('open'); }

document.getElementById('musteri-kaydet-btn').addEventListener('click', async ()=>{
  const name  = document.getElementById('m-name').value.trim();
  const phone = document.getElementById('m-phone').value.trim();
  const email = document.getElementById('m-email').value.trim();
  const source= document.getElementById('m-source').value;
  const notes = document.getElementById('m-notes').value.trim();
  if (!name) { showToast('Ad zorunludur','error'); return; }
  try {
    const data = {name, phone, email, source, notes};
    if (state.editCustomerId) {
      await API.updateCustomer(state.editCustomerId, data);
      showToast('✅ Müşteri güncellendi!');
    } else {
      await API.createCustomer(data);
      showToast('🎉 Müşteri eklendi!');
    }
    closeMusteriModal();
    await loadCustomers();
  } catch(e) { showToast(e.message,'error'); }
});

async function doToggleCustomer(id, e) {
  e.stopPropagation();
  try { await API.toggleCustomer(id); await loadCustomers(); showToast('Durum güncellendi.'); }
  catch(err) { showToast(err.message,'error'); }
}
async function doDeleteCustomer(id, e) {
  e.stopPropagation();
  if (!confirm('Müşteriyi silmek istediğinizden emin misiniz?')) return;
  try { await API.deleteCustomer(id); await loadCustomers(); showToast('Müşteri silindi.'); }
  catch(err) { showToast(err.message,'error'); }
}

/* ── Müşteri Detay ─────────────────────────────────────────── */
async function openCustomerDetail(id) {
  state.viewCustomerId = id;
  const c = state.customers.find(x=>x.id===id);
  if (!c) return;
  document.getElementById('musteri-detail-name').textContent = c.name;
  document.getElementById('musteri-detail-content').innerHTML = `
    <div class="crm-detail-row"><span class="detail-label">Telefon</span><span>${c.phone||'—'}</span></div>
    <div class="crm-detail-row"><span class="detail-label">E-posta</span><span>${c.email||'—'}</span></div>
    <div class="crm-detail-row"><span class="detail-label">Kaynak</span><span>${c.source||'—'}</span></div>
    <div class="crm-detail-row"><span class="detail-label">Danışman</span><span>${c.owner_name||'—'}</span></div>
    ${c.notes?`<div class="crm-detail-row"><span class="detail-label">Notlar</span><span>${c.notes}</span></div>`:''}
  `;
  document.getElementById('musteri-detail-overlay').classList.add('open');
  await loadCustomerListings(id);
}
document.getElementById('close-musteri-detail').addEventListener('click',()=>
  document.getElementById('musteri-detail-overlay').classList.remove('open'));
document.getElementById('musteri-detail-edit-btn').addEventListener('click',()=>{
  document.getElementById('musteri-detail-overlay').classList.remove('open');
  const c = state.customers.find(x=>x.id===state.viewCustomerId);
  if(c) openMusteriModal(c);
});

async function loadCustomerListings(customerId) {
  try {
    const listings = await API.getCustomerListings(customerId)||[];
    const cont = document.getElementById('musteri-listings-list');
    if (!listings.length) {
      cont.innerHTML = '<div class="muted" style="padding:8px 0;font-size:13px">Henüz bağlı ilan yok.</div>';
      return;
    }
    cont.innerHTML = listings.map(il=>`
      <div class="ilan-mini" style="cursor:default">
        ${il.cover_image?`<img src="${il.cover_image}" alt="" class="ilan-mini-thumb">`:`<div class="ilan-mini-icon">${(PROP_PLACEHOLDER[il.fields?.property_type]||PROP_PLACEHOLDER.default).icon}</div>`}
        <div class="ilan-mini-info">
          <div class="ilan-mini-title">${il.fields?.title||'—'} ${il.listing_no?`<span class="mini-no">#${il.listing_no}</span>`:''}</div>
          <div class="ilan-mini-tags"><span class="meta-tag">${il.fields?.district||''}</span></div>
        </div>
        <div class="ilan-mini-right">
          <span class="ilan-mini-price">${fiyatFormat(il.fields?.price_max||il.fields?.price)}</span>
          <button class="btn btn-sm btn-danger" onclick="doUnlinkListing(${customerId},${il.id})">Kaldır</button>
        </div>
      </div>`).join('');
  } catch(e) { showToast(e.message,'error'); }
}

async function doUnlinkListing(customerId, listingId) {
  try {
    await API.unlinkListing(customerId, listingId);
    await loadCustomerListings(customerId);
    showToast('Bağlantı kaldırıldı.');
  } catch(e) { showToast(e.message,'error'); }
}

/* ── İlan Bağla ────────────────────────────────────────────── */
document.getElementById('link-listing-btn').addEventListener('click', ()=>{
  const sel = document.getElementById('link-listing-select');
  sel.innerHTML = '<option value="">Seçin...</option>' +
    state.listings
      .filter(il => il.is_active && !il.customer_id && (API.isAdmin() || il.user_id === API.getUserID()))
      .map(il=>`<option value="${il.id}">#${il.listing_no} ${il.fields?.title||''}</option>`).join('');
  document.getElementById('link-listing-note').value = '';
  document.getElementById('link-listing-overlay').classList.add('open');
});
document.getElementById('close-link-listing').addEventListener('click',()=>
  document.getElementById('link-listing-overlay').classList.remove('open'));
document.getElementById('link-listing-iptal').addEventListener('click',()=>
  document.getElementById('link-listing-overlay').classList.remove('open'));
document.getElementById('link-listing-kaydet').addEventListener('click', async ()=>{
  const listingId = parseInt(document.getElementById('link-listing-select').value);
  const note      = document.getElementById('link-listing-note').value;
  if (!listingId) { showToast('İlan seçin','error'); return; }
  try {
    await API.linkListing(state.viewCustomerId, listingId, note);
    document.getElementById('link-listing-overlay').classList.remove('open');
    await loadCustomerListings(state.viewCustomerId);
    showToast('İlan bağlandı!');
  } catch(e) { showToast(e.message,'error'); }
});

/* ═══════════════════════════════════════════════════════
   DASHBOARD
════════════════════════════════════════════════════════ */
let _dashCharts = {};

async function loadDashboard() {
  document.getElementById('dash-content').innerHTML =
    '<div class="empty-state"><div class="big-icon">📊</div><p>Yükleniyor...</p></div>';
  try {
    const d = await API.getDashboard();
    renderDashboard(d);
  } catch(e) {
    document.getElementById('dash-content').innerHTML =
      `<div class="alert alert-error">Dashboard yüklenemedi: ${e.message}</div>`;
  }
}

function renderDashboard(d) {
  // Mevcut grafikleri yok et
  Object.values(_dashCharts).forEach(c => c.destroy());
  _dashCharts = {};

  const isAdmin = API.isAdmin();
  document.getElementById('dash-content').innerHTML = `
    <!-- Özet Kartlar -->
    <div class="dash-cards">
      <div class="dash-card dash-card-blue">
        <div class="dash-card-num">${d.total_listings}</div>
        <div class="dash-card-lbl">Toplam İlan</div>
      </div>
      <div class="dash-card dash-card-green">
        <div class="dash-card-num">${d.active_listings}</div>
        <div class="dash-card-lbl">Aktif İlan</div>
      </div>
      <div class="dash-card dash-card-amber">
        <div class="dash-card-num">${d.passive_listings}</div>
        <div class="dash-card-lbl">Pasif İlan</div>
      </div>
      <div class="dash-card dash-card-purple">
        <div class="dash-card-num">${d.listed_listings}</div>
        <div class="dash-card-lbl">Vitrin'de</div>
      </div>
    </div>

    <!-- Grafikler -->
    <div class="dash-charts">
      <div class="dash-chart-box">
        <div class="dash-chart-title">Aylık İlan Akışı (12 Ay)</div>
        <canvas id="chart-monthly" height="220"></canvas>
      </div>
      <div class="dash-chart-box">
        <div class="dash-chart-title">Durum Dağılımı</div>
        <canvas id="chart-status" height="220"></canvas>
      </div>
      <div class="dash-chart-box">
        <div class="dash-chart-title">Tip Dağılımı</div>
        <canvas id="chart-type" height="220"></canvas>
      </div>
      <div class="dash-chart-box">
        <div class="dash-chart-title">Son Aktiviteler</div>
        <div id="dash-activities-inline" style="max-height:440px;overflow-y:auto"></div>
      </div>
      <div class="dash-chart-box">
        <div class="dash-chart-title">İlçe Dağılımı (Top 10)</div>
        <canvas id="chart-district" height="220"></canvas>
      </div>
      ${isAdmin ? `<div class="dash-chart-box dash-chart-wide">
        <div class="dash-chart-title">Danışman Performansı</div>
        <canvas id="chart-agents" height="160"></canvas>
      </div>` : ''}
    </div>

    `;

  // Tüm 12 ay etiketleri
  const months = buildLast12Months();

  // 1. Aylık grafik
  const addedMap  = Object.fromEntries((d.monthly_added||[]).map(m=>[m.month,m.count]));
  const closedMap = Object.fromEntries((d.monthly_closed||[]).map(m=>[m.month,m.count]));
  _dashCharts.monthly = new Chart(document.getElementById('chart-monthly'), {
    type: 'bar',
    data: {
      labels: months.map(m=>m.label),
      datasets: [
        { label:'Eklenen', data: months.map(m=>addedMap[m.key]||0), backgroundColor:'#1565C0cc', borderRadius:4 },
        { label:'Kapanan', data: months.map(m=>closedMap[m.key]||0), backgroundColor:'#FFD700cc', borderRadius:4 },
      ]
    },
    options: { responsive:true, plugins:{legend:{position:'bottom'}}, scales:{y:{beginAtZero:true,ticks:{stepSize:1}}} }
  });

  // 2. Durum donut
  const stLabels = { aktif:'Aktif', satildi:'Satıldı', kiralandi:'Kiralandı', bekliyor:'Bekliyor' };
  const stColors = { aktif:'#1b5e20', satildi:'#c62828', kiralandi:'#1565C0', bekliyor:'#e65100' };
  const stKeys = Object.keys(d.by_status||{});
  _dashCharts.status = new Chart(document.getElementById('chart-status'), {
    type: 'doughnut',
    data: {
      labels: stKeys.map(k=>stLabels[k]||k),
      datasets:[{ data: stKeys.map(k=>d.by_status[k]), backgroundColor: stKeys.map(k=>stColors[k]||'#90a4ae') }]
    },
    options:{ responsive:true, plugins:{legend:{position:'bottom'}} }
  });

  // 3. Tip dağılımı (pasta)
  const typeKeys = Object.keys(d.by_type||{}).filter(k=>k&&k!=='—');
  _dashCharts.type = new Chart(document.getElementById('chart-type'), {
    type: 'pie',
    data: {
      labels: typeKeys,
      datasets:[{ data: typeKeys.map(k=>d.by_type[k]), backgroundColor:['#1565C0','#FFD700','#1b5e20','#c62828'] }]
    },
    options:{ responsive:true, plugins:{legend:{position:'bottom'}} }
  });

  // 4. İlçe bar
  const districts = (d.by_district||[]).filter(x=>x.district&&x.district!=='—');
  _dashCharts.district = new Chart(document.getElementById('chart-district'), {
    type: 'bar',
    data: {
      labels: districts.map(x=>x.district),
      datasets:[{ label:'İlan', data: districts.map(x=>x.count), backgroundColor:'#1565C0aa', borderRadius:4 }]
    },
    options:{ responsive:true, indexAxis:'y', plugins:{legend:{display:false}}, scales:{x:{beginAtZero:true,ticks:{stepSize:1}}} }
  });

  // Son aktiviteler
  loadRecentActivities();

  // 5. Danışman bar (admin only)
  if (isAdmin && d.top_agents?.length) {
    _dashCharts.agents = new Chart(document.getElementById('chart-agents'), {
      type: 'bar',
      data: {
        labels: d.top_agents.map(a=>a.name),
        datasets:[{ label:'İlan', data: d.top_agents.map(a=>a.count), backgroundColor:'#FFD700cc', borderRadius:4 }]
      },
      options:{ responsive:true, plugins:{legend:{display:false}}, scales:{y:{beginAtZero:true,ticks:{stepSize:1}}} }
    });
  }
}


async function loadRecentActivities() {
  const cont = document.getElementById('dash-activities') || document.getElementById('dash-activities-inline');
  if (!cont) return;
  try {
    const acts = await API.getRecentActivities() || [];
    if (!acts.length) {
      cont.innerHTML = '<div class="muted" style="padding:12px;text-align:center;font-size:13px">Aktivite yok.</div>';
      return;
    }
    const ACT_ICON = {
      created:'✨', updated:'✏️', pipeline:'🔄',
      activated:'▶️', deactivated:'⏸', listed:'👁'
    };
    const ACT_COLOR = {
      created:'#1b5e20', updated:'#1565C0', pipeline:'#e65100',
      activated:'#1565C0', deactivated:'#c62828', listed:'#6a1b9a', unlisted:'#78909c'
    };
    cont.innerHTML = acts.map(a => {
      const color = ACT_COLOR[a.type] || '#78909c';
      const date = new Date(a.created_at).toLocaleDateString('tr-TR',{day:'2-digit',month:'short',hour:'2-digit',minute:'2-digit'});
      const namePart = a.user_name ? '<b>'+escHtml(a.user_name)+'</b> ' : '';
      return '<div class="act-item">' +
        '<div class="act-dot-colored" style="background:'+color+'"></div>' +
        '<div class="act-body">' +
          '<div class="act-note" style="font-size:13px">'+namePart+escHtml(a.note||'')+(a.listing_title?(' <span style="font-weight:600;color:var(--navy)">'+escHtml(a.listing_title)+'</span>'):'')+'</div>' +
          '<div class="act-meta">'+date+'</div>' +
        '</div>' +
      '</div>';
    }).join('');
  } catch(e) { console.error(e); }
}

function buildLast12Months() {
  const months = [];
  const now = new Date();
  for (let i = 11; i >= 0; i--) {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
    const key = `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}`;
    const label = d.toLocaleDateString('tr-TR', {month:'short', year:'2-digit'});
    months.push({key, label});
  }
  return months;
}

/* ═══════════════════════════════════════════════════════
   GÖREVLER (Tasks)
════════════════════════════════════════════════════════ */
state.tasks       = [];
state.allUsers    = [];
state.taskView    = 'kanban'; // 'kanban' | 'list'
state.editTaskId  = null;
state.viewTaskId  = null;
state.taskParentId = null;

const PRIORITY_LABEL = { dusuk:'Düşük', normal:'Normal', yuksek:'Yüksek', acil:'Acil' };
const PRIORITY_CLASS = { dusuk:'prio-dusuk', normal:'prio-normal', yuksek:'prio-yuksek', acil:'prio-acil' };
const STATUS_TASK_LABEL = { bekliyor:'Başladı', devam_ediyor:'Devam Ediyor', tamamlandi:'Tamamlandı', iptal:'İptal' };
const STATUS_TASK_CLASS = { bekliyor:'status-bekliyor', devam_ediyor:'status-devam', tamamlandi:'status-tamamlandi', iptal:'status-iptal' };

async function loadTasks() {
  const params = {};
  const sf = document.getElementById('task-status-filter')?.value;
  const pf = document.getElementById('task-priority-filter')?.value;
  if (sf) params.status   = sf;
  if (pf) params.priority = pf;
  try {
    state.tasks = await API.getTasks(params) || [];
    renderTasks();
  } catch(e) { showToast('Görevler yüklenemedi: '+e.message,'error'); }
}

async function ensureUsers() {
  if (!state.allUsers.length) {
    state.allUsers = await API.getUsers() || [];
  }
}

function renderTasks() {
  const tasks = state.tasks;
  const empty  = document.getElementById('task-empty');
  if (!tasks.length) {
    empty.style.display = 'block';
    clearKanban(); clearTaskTable(); return;
  }
  empty.style.display = 'none';
  if (state.taskView === 'kanban') renderKanban(tasks);
  else renderTaskTable(tasks);
}

function clearKanban() {
  ['bekliyor','devam_ediyor','tamamlandi','iptal'].forEach(s => {
    const el = document.getElementById('cards-'+s);
    if (el) el.innerHTML = '';
    const cnt = document.getElementById('cnt-'+s);
    if (cnt) cnt.textContent = '0';
  });
}

function clearTaskTable() {
  const tb = document.getElementById('task-table-body');
  if (tb) tb.innerHTML = '';
}

function renderKanban(tasks) {
  clearKanban();
  const groups = { bekliyor:[], devam_ediyor:[], tamamlandi:[], iptal:[] };
  tasks.forEach(t => { if (groups[t.status]) groups[t.status].push(t); });

  Object.keys(groups).forEach(status => {
    const cont = document.getElementById('cards-'+status);
    const cnt  = document.getElementById('cnt-'+status);
    if (!cont) return;
    cnt.textContent = groups[status].length;
    cont.innerHTML  = groups[status].map(t => taskCardHTML(t)).join('');
    cont.querySelectorAll('.task-card').forEach(card => {
      card.addEventListener('click', () => openGorevDetail(parseInt(card.dataset.id)));
      card.addEventListener('dragstart', onDragStart);
      card.addEventListener('dragend',   onDragEnd);
    });
  });

  document.querySelectorAll('.kanban-cards').forEach(col => {
    col.addEventListener('dragover',  onDragOver);
    col.addEventListener('dragleave', onDragLeave);
    col.addEventListener('drop',      onDrop);
  });
}

function taskCardHTML(t) {
  const due  = t.due_date ? dueChipHTML(t.due_date) : '';
  const assigneeChips = (t.assignees||[]).slice(0,4).map(a =>
    `<div class="assignee-mini" title="${escHtml(a.full_name)}">${a.full_name[0].toUpperCase()}</div>`
  ).join('');
  const subCount = (t.subtasks||[]).length;
  return `<div class="task-card" data-id="${t.id}" draggable="true">
    <div class="task-card-title">${escHtml(t.title)}</div>
    <div class="task-card-meta">
      <span class="priority-badge ${PRIORITY_CLASS[t.priority]||'prio-normal'}">${PRIORITY_LABEL[t.priority]||t.priority}</span>
      ${due}
      ${subCount ? `<span class="sub-chip">↳ ${subCount}</span>` : ''}
      <div class="task-card-assignees">${assigneeChips}</div>
    </div>
  </div>`;
}

function dueChipHTML(dueDateStr) {
  if (!dueDateStr) return '';
  const due  = new Date(dueDateStr);
  const now  = new Date();
  const diff = Math.ceil((due - now) / 86400000);
  let cls = 'due-ok', label = due.toLocaleDateString('tr-TR', {day:'2-digit',month:'short'});
  if (diff < 0)  { cls = 'due-late';    label = '⚠ '+label; }
  else if (diff <= 3) { cls = 'due-warning'; }
  return `<span class="due-chip ${cls}">${label}</span>`;
}

function renderTaskTable(tasks) {
  clearTaskTable();
  const tbody = document.getElementById('task-table-body');
  if (!tbody) return;
  tbody.innerHTML = tasks.map(t => {
    const assignees = (t.assignees||[]).map(a=>a.full_name).join(', ')||'—';
    const due = t.due_date ? new Date(t.due_date).toLocaleDateString('tr-TR') : '—';
    const subCount = (t.subtasks||[]).length;
    return `<tr data-id="${t.id}" onclick="openGorevDetail(${t.id})">
      <td><b>${escHtml(t.title)}</b></td>
      <td><span class="status-badge ${STATUS_TASK_CLASS[t.status]||''}">${STATUS_TASK_LABEL[t.status]||t.status}</span></td>
      <td><span class="priority-badge ${PRIORITY_CLASS[t.priority]||'prio-normal'}">${PRIORITY_LABEL[t.priority]||t.priority}</span></td>
      <td>${escHtml(assignees)}</td>
      <td>${due}</td>
      <td>${subCount ? `<span class="sub-chip">${subCount}</span>` : '—'}</td>
    </tr>`;
  }).join('');
}

/* ── Kanban drag & drop ───────────────────────────────── */
let _dragId = null;
function onDragStart(e) {
  _dragId = parseInt(e.currentTarget.dataset.id);
  e.currentTarget.classList.add('dragging');
  e.dataTransfer.effectAllowed = 'move';
}
function onDragEnd(e) { e.currentTarget.classList.remove('dragging'); }
function onDragOver(e) {
  e.preventDefault();
  e.dataTransfer.dropEffect = 'move';
  e.currentTarget.closest('.kanban-col').classList.add('drag-over');
}
function onDragLeave(e) {
  const col = e.currentTarget.closest('.kanban-col');
  if (!col) return;
  // Sürüklenen eleman kolonun içine gidiyorsa kaldırma
  if (col.contains(e.relatedTarget)) return;
  col.classList.remove('drag-over');
}
async function onDrop(e) {
  e.preventDefault();
  const col = e.currentTarget.closest('.kanban-col');
  col.classList.remove('drag-over');
  const newStatus = col.dataset.status;
  if (!_dragId || !newStatus) return;
  const task = state.tasks.find(t => t.id === _dragId);
  if (!task || task.status === newStatus) return;
  try {
    await API.updateTaskStatus(_dragId, newStatus);
    await loadTasks();
  } catch(e) { showToast(e.message,'error'); }
}

/* ── View toggle ─────────────────────────────────────── */
document.getElementById('task-kanban-btn')?.addEventListener('click', ()=>{
  state.taskView = 'kanban';
  document.getElementById('task-kanban').style.display = '';
  document.getElementById('task-list-view').style.display = 'none';
  document.getElementById('task-kanban-btn').classList.add('active');
  document.getElementById('task-list-btn').classList.remove('active');
  renderTasks();
});
document.getElementById('task-list-btn')?.addEventListener('click', ()=>{
  state.taskView = 'list';
  document.getElementById('task-kanban').style.display = 'none';
  document.getElementById('task-list-view').style.display = '';
  document.getElementById('task-list-btn').classList.add('active');
  document.getElementById('task-kanban-btn').classList.remove('active');
  renderTasks();
});

/* ── Filters ─────────────────────────────────────────── */
document.getElementById('task-status-filter')?.addEventListener('change', loadTasks);
document.getElementById('task-priority-filter')?.addEventListener('change', loadTasks);

/* ── New Task button ─────────────────────────────────── */
document.getElementById('yeni-gorev-btn')?.addEventListener('click', async ()=>{
  state.taskParentId = null;
  await openGorevModal(null);
});

/* ── Task modal ──────────────────────────────────────── */
async function openGorevModal(task, parentId) {
  state.editTaskId   = task ? task.id : null;
  state.taskParentId = parentId || null;
  await ensureUsers();

  document.getElementById('gorev-modal-title').textContent = task
    ? (parentId ? 'Alt Görev Düzenle' : 'Görev Düzenle')
    : (parentId ? 'Alt Görev Ekle' : 'Yeni Görev');

  document.getElementById('g-title').value    = task ? task.title : '';
  document.getElementById('g-desc').value     = task ? task.description : '';
  document.getElementById('g-status').value   = task ? task.status : 'bekliyor';
  document.getElementById('g-priority').value = task ? task.priority : 'normal';
  document.getElementById('g-due').value      = task && task.due_date
    ? task.due_date.substring(0, 10) : '';

  const selected = new Set((task?.assignees||[]).map(a => a.id));
  document.getElementById('g-assignees').innerHTML = state.allUsers.map(u => `
    <div class="assignee-cb-item${selected.has(u.id)?' selected':''}" data-uid="${u.id}" onclick="toggleAssigneeCb(this)">
      <div class="assignee-chip-avatar">${u.full_name[0].toUpperCase()}</div>
      ${escHtml(u.full_name)}
    </div>`).join('');

  document.getElementById('gorev-overlay').classList.add('open');
  document.getElementById('g-title').focus();
}

function toggleAssigneeCb(el) { el.classList.toggle('selected'); }

function closeGorevModal() { document.getElementById('gorev-overlay').classList.remove('open'); }
document.getElementById('close-gorev-modal')?.addEventListener('click', closeGorevModal);
document.getElementById('gorev-iptal-btn')?.addEventListener('click', closeGorevModal);

document.getElementById('gorev-kaydet-btn')?.addEventListener('click', async ()=>{
  const title    = document.getElementById('g-title').value.trim();
  const desc     = document.getElementById('g-desc').value.trim();
  const status   = document.getElementById('g-status').value;
  const priority = document.getElementById('g-priority').value;
  const due      = document.getElementById('g-due').value || null;
  if (!title) { showToast('Başlık zorunludur','error'); return; }

  const assignees = [...document.querySelectorAll('#g-assignees .selected')]
    .map(el => parseInt(el.dataset.uid));

  const payload = {
    title, description: desc, status, priority,
    due_date: due ? new Date(due).toISOString() : null,
    assignees,
    parent_id: state.taskParentId || null,
  };

  try {
    if (state.editTaskId) {
      await API.updateTask(state.editTaskId, payload);
      showToast('✅ Görev güncellendi!');
    } else {
      await API.createTask(payload);
      showToast('🎉 Görev eklendi!');
    }
    closeGorevModal();
    await loadTasks();
    if (state.viewTaskId) await refreshGorevDetail(state.viewTaskId);
  } catch(e) { showToast(e.message,'error'); }
});

/* ── Task detail modal ───────────────────────────────── */
async function openGorevDetail(id) {
  state.viewTaskId = id;
  try {
    const t = await API.getTask(id);
    renderGorevDetail(t);
    document.getElementById('gorev-detail-overlay').classList.add('open');
  } catch(e) { showToast(e.message,'error'); }
}

async function refreshGorevDetail(id) {
  try {
    const t = await API.getTask(id);
    renderGorevDetail(t);
  } catch(_) {}
}

function renderGorevDetail(t) {
  document.getElementById('gd-title').textContent = t.title;
  document.getElementById('gd-priority-badge').className = `priority-badge ${PRIORITY_CLASS[t.priority]||'prio-normal'}`;
  document.getElementById('gd-priority-badge').textContent = PRIORITY_LABEL[t.priority]||t.priority;

  document.getElementById('gd-status').className   = `status-badge ${STATUS_TASK_CLASS[t.status]||''}`;
  document.getElementById('gd-status').textContent = STATUS_TASK_LABEL[t.status]||t.status;

  document.getElementById('gd-due').textContent = t.due_date
    ? new Date(t.due_date).toLocaleDateString('tr-TR') : '—';
  document.getElementById('gd-creator').textContent = t.creator_name || '—';

  const descBlock = document.getElementById('gd-desc-block');
  const descEl    = document.getElementById('gd-desc');
  if (t.description) {
    descEl.textContent  = t.description;
    descBlock.style.display = '';
  } else {
    descBlock.style.display = 'none';
  }

  // Assignees
  document.getElementById('gd-assignees').innerHTML = (t.assignees||[]).length
    ? (t.assignees||[]).map(a => `
        <div class="assignee-chip">
          <div class="assignee-chip-avatar">${a.full_name[0].toUpperCase()}</div>
          ${escHtml(a.full_name)}
        </div>`).join('')
    : '<span class="muted" style="font-size:13px">Atanan yok</span>';

  // Subtasks
  const subtasks = t.subtasks || [];
  document.getElementById('gd-subtasks').innerHTML = subtasks.length
    ? `<div class="subtask-list">${subtasks.map(st => `
        <div class="subtask-row" onclick="openGorevDetail(${st.id})">
          <input type="checkbox" class="subtask-checkbox" ${st.status==='tamamlandi'?'checked':''} onclick="event.stopPropagation();toggleSubtaskStatus(${st.id},${st.status==='tamamlandi'})">
          <span style="${st.status==='tamamlandi'?'text-decoration:line-through;color:var(--muted)':''}">${escHtml(st.title)}</span>
          <span class="priority-badge ${PRIORITY_CLASS[st.priority]||'prio-normal'}" style="margin-left:auto;font-size:10px">${PRIORITY_LABEL[st.priority]}</span>
        </div>`).join('')}
      </div>`
    : '<div class="muted" style="font-size:13px">Alt görev yok.</div>';

  // Images
  const images = t.images || [];
  document.getElementById('gd-images').innerHTML = images.length
    ? images.map(img => `
        <div class="task-img-wrap">
          <img src="${img.path}" alt="" onclick="openLightbox(this.src)">
          <button class="task-img-del" onclick="deleteTaskImg(${t.id},${img.id})" title="Sil">×</button>
        </div>`).join('')
    : '<div class="muted" style="font-size:13px">Resim yok.</div>';

  // Comments
  const comments = t.comments || [];
  const meID = API.getUserID();
  document.getElementById('gd-comments').innerHTML = comments.length
    ? comments.map(c => `
        <div class="task-comment">
          <div class="task-comment-header">
            <span class="task-comment-author">${escHtml(c.user_name)}</span>
            <span style="display:flex;align-items:center;gap:4px">
              <span class="task-comment-time">${timeAgo(c.created_at)}</span>
              ${(c.user_id===meID||API.isAdmin()) ? `<button class="task-comment-del" onclick="deleteTaskComment(${t.id},${c.id})">🗑</button>` : ''}
            </span>
          </div>
          <div class="task-comment-body">${escHtml(c.body)}</div>
        </div>`).join('')
    : '<div class="muted" style="font-size:13px;padding:8px 0">Henüz yorum yok.</div>';

  document.getElementById('gd-comment-text').value = '';
}

async function toggleSubtaskStatus(id, isDone) {
  const newStatus = isDone ? 'bekliyor' : 'tamamlandi';
  try {
    await API.updateTaskStatus(id, newStatus);
    await refreshGorevDetail(state.viewTaskId);
    await loadTasks();
  } catch(e) { showToast(e.message,'error'); }
}

async function deleteTaskImg(taskId, imgId) {
  if (!confirm('Resmi silmek istediğinize emin misiniz?')) return;
  try {
    await API.deleteTaskImage(taskId, imgId);
    await refreshGorevDetail(taskId);
  } catch(e) { showToast(e.message,'error'); }
}

async function deleteTaskComment(taskId, cid) {
  if (!confirm('Yorumu silmek istediğinize emin misiniz?')) return;
  try {
    await API.deleteTaskComment(taskId, cid);
    await refreshGorevDetail(taskId);
  } catch(e) { showToast(e.message,'error'); }
}

document.getElementById('close-gorev-detail')?.addEventListener('click', ()=>{
  document.getElementById('gorev-detail-overlay').classList.remove('open');
  state.viewTaskId = null;
});

document.getElementById('gd-edit-btn')?.addEventListener('click', async ()=>{
  const t = state.tasks.find(t=>t.id===state.viewTaskId)
    || await API.getTask(state.viewTaskId);
  document.getElementById('gorev-detail-overlay').classList.remove('open');
  await openGorevModal(t);
});

document.getElementById('gd-add-subtask-btn')?.addEventListener('click', async ()=>{
  document.getElementById('gorev-detail-overlay').classList.remove('open');
  await openGorevModal(null, state.viewTaskId);
});

// Image upload from detail
document.getElementById('gd-img-input')?.addEventListener('change', async (e)=>{
  const file = e.target.files[0]; if (!file) return;
  try {
    await API.uploadTaskImage(state.viewTaskId, file);
    await refreshGorevDetail(state.viewTaskId);
    showToast('Resim eklendi!');
  } catch(e) { showToast(e.message,'error'); }
  e.target.value = '';
});

// Comment submit
document.getElementById('gd-comment-btn')?.addEventListener('click', async ()=>{
  const body = document.getElementById('gd-comment-text').value.trim();
  if (!body) return;
  try {
    await API.addTaskComment(state.viewTaskId, body);
    await refreshGorevDetail(state.viewTaskId);
  } catch(e) { showToast(e.message,'error'); }
});
document.getElementById('gd-comment-text')?.addEventListener('keydown', e=>{
  if (e.key==='Enter' && e.ctrlKey) document.getElementById('gd-comment-btn').click();
});

/* ── Helpers ─────────────────────────────────────────── */
function escHtml(s) {
  if (!s) return '';
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

function timeAgo(dateStr) {
  if (!dateStr) return '';
  const d   = new Date(dateStr);
  const sec = Math.floor((Date.now() - d) / 1000);
  if (sec < 60) return 'az önce';
  if (sec < 3600) return Math.floor(sec/60)+'dk önce';
  if (sec < 86400) return Math.floor(sec/3600)+'sa önce';
  return d.toLocaleDateString('tr-TR');
}

/* ═══════════════════════════════════════════════════════
   ADMIN
════════════════════════════════════════════════════════ */
async function loadAdminPanel(){ await loadAdminUsers(); await loadAdminSettings(); }
async function loadAdminUsers(){
  try{
    const users=await API.adminGetUsers();
    document.getElementById('users-table').innerHTML=`<table class="admin-table">
      <thead><tr><th>Kullanıcı</th><th>E-posta</th><th>Rol</th><th>Durum</th><th>Telegram Chat ID</th><th>İşlem</th></tr></thead>
      <tbody>${(users||[]).map(u=>`<tr>
        <td><b>${u.full_name}</b><br><small class="muted">@${u.username}</small></td>
        <td>${u.email||'—'}</td>
        <td><span class="tag ${u.role==='admin'?'tag-red':'tag-blue'}">${u.role}</span></td>
        <td><span class="tag ${u.is_active?'tag-green':'tag-amber'}">${u.is_active?'Aktif':'Pasif'}</span></td>
        <td>
          <div style="display:flex;gap:6px;align-items:center">
            <input type="text" id="tg-input-${u.id}"
              value="${u.telegram_chat_id||''}"
              placeholder="Chat ID"
              style="width:130px;padding:6px 10px;border:1.5px solid var(--border);border-radius:6px;font-size:12px;font-family:monospace;background:var(--bg)">
            <button class="btn btn-sm btn-blue" onclick="adminSaveTelegram(${u.id})">Kaydet</button>
          </div>
          ${u.telegram_username?'<small class="muted" style="margin-top:3px;display:block">@'+u.telegram_username+'</small>':''}
        </td>
        <td>${u.role!=='admin'?`
          <button class="btn btn-sm btn-outline" onclick="adminToggleUser(${u.id})">${u.is_active?'Pasif Yap':'Aktif Yap'}</button>
          <button class="btn btn-sm btn-danger" onclick="adminDeleteUser(${u.id})">Sil</button>
        `:'—'}</td>
      </tr>`).join('')}</tbody></table>`;
  }catch(e){showToast(e.message,'error');}
}
async function adminSaveTelegram(id){
  const val=document.getElementById('tg-input-'+id)?.value||'';
  try{await API.adminSetChatID(id,val);showToast('Chat ID kaydedildi.');}
  catch(e){showToast(e.message,'error');}
}
async function adminToggleUser(id){try{await API.adminToggleUser(id);await loadAdminUsers();}catch(e){showToast(e.message,'error');}}
async function adminDeleteUser(id){
  if(!confirm('Bu kullanıcıyı silmek istediğinize emin misiniz?'))return;
  try{await API.adminDeleteUser(id);await loadAdminUsers();showToast('Kullanıcı silindi.');}catch(e){showToast(e.message,'error');}
}
async function loadAdminListings(){
  try{
    const listings=await API.adminGetListings();
    document.getElementById('admin-listings-list').innerHTML=`<table class="admin-table">
      <thead><tr><th>#No</th><th>Başlık</th><th>Ekleyen</th><th>Durum</th><th>İşlem</th></tr></thead>
      <tbody>${(listings||[]).map(il=>`<tr>
        <td class="muted">${il.listing_no||'—'}</td><td>${il.fields?.title||'—'}</td>
        <td>${il.owner_name||'—'}</td>
        <td>
          <span class="tag ${il.is_active?'tag-green':'tag-amber'}">${il.is_active?'Aktif':'Pasif'}</span>
          ${!il.is_listed?'<span class="tag tag-sm" style="background:#eee">Gizli</span>':''}
        </td>
        <td><button class="btn btn-sm btn-danger" onclick="adminDeleteListing(${il.id})">Sil</button></td>
      </tr>`).join('')}</tbody></table>`;
  }catch(e){showToast(e.message,'error');}
}
async function adminDeleteListing(id){
  if(!confirm('İlanı silmek istediğinize emin misiniz?'))return;
  try{await API.adminDeleteListing(id);await loadAdminListings();showToast('İlan silindi.');}catch(e){showToast(e.message,'error');}
}
async function loadAdminRequests(){
  try{
    const reqs=await API.adminGetRequests();
    document.getElementById('admin-requests-list').innerHTML=`<table class="admin-table">
      <thead><tr><th>Müşteri</th><th>Ekleyen</th><th>Durum</th><th>İşlem</th></tr></thead>
      <tbody>${(reqs||[]).map(r=>`<tr>
        <td>${r.fields?.client_name||'—'} · ${r.fields?.phone||''}</td>
        <td>${r.owner_name||'—'}</td>
        <td><span class="tag ${r.is_active?'tag-green':'tag-amber'}">${r.is_active?'Aktif':'Pasif'}</span></td>
        <td><button class="btn btn-sm btn-danger" onclick="adminDeleteRequest(${r.id})">Sil</button></td>
      </tr>`).join('')}</tbody></table>`;
  }catch(e){showToast(e.message,'error');}
}
async function adminDeleteRequest(id){
  if(!confirm('Talebi silmek istediğinize emin misiniz?'))return;
  try{await API.adminDeleteRequest(id);await loadAdminRequests();showToast('Talep silindi.');}catch(e){showToast(e.message,'error');}
}
document.getElementById('yeni-kullanici-btn')?.addEventListener('click',()=>document.getElementById('user-overlay').classList.add('open'));
document.getElementById('close-user-modal').addEventListener('click',()=>document.getElementById('user-overlay').classList.remove('open'));
document.getElementById('user-iptal-btn').addEventListener('click',()=>document.getElementById('user-overlay').classList.remove('open'));
document.getElementById('user-kaydet-btn').addEventListener('click', async()=>{
  const username=document.getElementById('u-username').value.trim();
  const fullname=document.getElementById('u-fullname').value.trim();
  const email=document.getElementById('u-email').value.trim();
  const password=document.getElementById('u-password').value;
  if(!username||!password){showToast('Kullanıcı adı ve şifre zorunludur','error');return;}
  try{
    await API.adminCreateUser({username,full_name:fullname,email,password,role:'agent'});
    document.getElementById('user-overlay').classList.remove('open');
    await loadAdminUsers(); showToast('✅ Kullanıcı eklendi!');
  }catch(e){showToast(e.message,'error');}
});




/* ═══════════════════════════════════════════════════════
   SHOW_ON HELPER
════════════════════════════════════════════════════════ */
function fieldVisible(field, context, listingType, propType) {
  const showOn = field.show_on;
  if (!showOn) return true;
  const list = showOn[context] || [];
  if (!list.length) return false;
  if (list.includes('*')) return true;

  const pt = propType || '';

  // Sadece mulk tipine gore kontrol — kiralık/satılık farketmez
  if (!pt) return list.length > 0;
  return list.includes(pt) || list.includes('*');
}

/* ═══════════════════════════════════════════════════════
   PIPELINE
════════════════════════════════════════════════════════ */
const PIPELINE_STAGES = [
  {key:'bilgi_alindi', label:'Bilgi Alındı'},
  {key:'hazirlik',     label:'Hazırlık'},
  {key:'ilana_alindi', label:'İlanda'},
  {key:'muzakere',     label:'Müzakere'},
  {key:'sozlesme',     label:'Sözleşme'},
  {key:'kapandi',      label:'Kapandı'},
];

async function loadPipeline() {
  try {
    const listings = await API.getListings();
    renderPipeline(listings||[]);
  } catch(e) { showToast('Pipeline yüklenemedi: '+e.message,'error'); }
}

function renderPipeline(listings) {
  const groups = {};
  PIPELINE_STAGES.forEach(s => { groups[s.key] = []; });
  listings.forEach(il => {
    const stage = il.pipeline_stage || 'bilgi_alindi';
    if (groups[stage]) groups[stage].push(il);
    else groups['bilgi_alindi'].push(il);
  });

  PIPELINE_STAGES.forEach(s => {
    const cont = document.getElementById('pcards-'+s.key);
    const cnt  = document.getElementById('pcnt-'+s.key);
    if (!cont) return;
    cnt.textContent = groups[s.key].length;
    cont.innerHTML = groups[s.key].map(il => {
      const price = il.fields?.price_max || il.fields?.price || '';
      const canEdit = API.isAdmin() || il.user_id===API.getUserID();
      return `<div class="pipeline-card" data-id="${il.id}" data-stage="${s.key}" draggable="${canEdit}" onclick="openDetailModal(${il.id})">
        <div class="pipeline-card-title">${escHtml(il.fields?.title||'—')}</div>
        <div class="pipeline-card-meta">${il.fields?.district||''} · ${il.fields?.property_type||''}</div>
        <div class="pipeline-card-price">${price ? fiyatFormat(price) : '—'}</div>
        <div class="pipeline-card-no">#${il.listing_no||''} · ${il.owner_name||''}</div>
        ${canEdit ? `
        <select class="pipeline-stage-select" onchange="doPipelineChange(${il.id},this.value,event)">
          ${PIPELINE_STAGES.map(st=>`<option value="${st.key}" ${st.key===s.key?'selected':''}>${st.label}</option>`).join('')}
        </select>` : ''}
      </div>`;
    }).join('') || '<div style="font-size:12px;color:var(--muted);text-align:center;padding:12px">Boş</div>';

    // Drag & drop
    cont.querySelectorAll('.pipeline-card[draggable="true"]').forEach(card => {
      card.addEventListener('dragstart', e => {
        e.dataTransfer.setData('text/plain', card.dataset.id+'|'+card.dataset.stage);
        card.classList.add('dragging');
        e.stopPropagation();
      });
      card.addEventListener('dragend', e => card.classList.remove('dragging'));
    });

    // Drop zone
    const col = cont.closest('.pipeline-col');
    col.addEventListener('dragover', e => { e.preventDefault(); col.classList.add('pipeline-drag-over'); });
    col.addEventListener('dragleave', e => { if (!col.contains(e.relatedTarget)) col.classList.remove('pipeline-drag-over'); });
    col.addEventListener('drop', async e => {
      e.preventDefault();
      col.classList.remove('pipeline-drag-over');
      const [dragId, oldStage] = (e.dataTransfer.getData('text/plain')||'').split('|');
      const newStage = col.dataset.stage;
      if (!dragId || !newStage || oldStage === newStage) return;
      try {
        await API.updatePipeline(parseInt(dragId), newStage);
        await loadPipeline();
        showToast('Aşama güncellendi.');
      } catch(err) { showToast(err.message,'error'); }
    });
  });
}

async function doPipelineChange(id, stage, e) {
  e.stopPropagation();
  try {
    await API.updatePipeline(id, stage);
    await loadPipeline();
    showToast('Aşama güncellendi.');
  } catch(err) { showToast(err.message,'error'); }
}


/* ═══════════════════════════════════════════════════════
   ALAN YÖNETİMİ
════════════════════════════════════════════════════════ */
async function loadAdminFields() {
  const cont = document.getElementById('admin-fields-content');
  if (!cont) return;
  try {
    const s = await API.getAdminSettings();
    renderAdminFields(s.all_fields || state.cfg?.listing_fields?.all_fields || []);
  } catch(e) {
    cont.innerHTML = '<div class="alert alert-error">Yüklenemedi: ' + e.message + '</div>';
  }
}

function renderAdminFields(allFields) {
  const cont = document.getElementById('admin-fields-content');
  if (!cont) return;

  const cfg = state.cfg;
  const propTypes = cfg?.property_types || ['Daire','Villa','Arsa','Ticari'];
  const contexts  = ['form','card','detail','telegram','talep'];
  const ctxLabels = {form:'Yeni İlan', card:'Kart', detail:'Detay', telegram:'Telegram', talep:'Talep'};

  // BÖLÜM 1 — Alan listesi
  let html = '<div class="fields-toolbar">' +
    '<button class="btn btn-gold" onclick="saveAdminFields()">💾 Kaydet</button>' +
    '<button class="btn btn-outline" onclick="addNewField()">+ Yeni Alan</button>' +
  '</div>';

  html += '<div class="af-section-title">Alan Listesi</div>';
  html += '<div class="af-list" id="af-list">';
  allFields.forEach((f, idx) => {
    const taskChk   = f.show_on?.task     ? 'checked' : '';
    const talepChk  = f.show_on?.talep    ? 'checked' : '';
    const telChk    = f.show_on?.telegram?.length ? 'checked' : '';
    html += '<div class="af-row" data-key="' + f.key + '" draggable="true">' +
      '<span class="af-drag">☰</span>' +
      '<span class="af-label">' + escHtml(f.label) + ' <small class="muted">(' + f.key + ')</small></span>' +
      '<span class="af-type field-type">' + f.type + '</span>' +
      '<label class="af-check-label"><input type="checkbox" class="af-meta-cb" data-key="' + f.key + '" data-meta="task" ' + taskChk + '> Görev</label>' +
      '<label class="af-check-label"><input type="checkbox" class="af-meta-cb" data-key="' + f.key + '" data-meta="talep" ' + talepChk + '> Talep</label>' +
      '<button class="btn btn-sm btn-outline" onclick="editField(\'' + f.key + '\')">✏️</button>' +
      '<button class="btn btn-sm btn-danger" onclick="removeField(\'' + f.key + '\')">🗑</button>' +
    '</div>';
  });
  html += '</div>';

  // BÖLÜM 2 — Görünüm checkboxları
  html += '<div class="af-section-title" style="margin-top:24px">Görünüm Ayarları</div>';
  html += '<div class="fields-table-wrap"><table class="fields-table"><thead><tr>';
  html += '<th rowspan="2">Alan</th>';
  contexts.forEach(ctx => {
    html += '<th colspan="' + propTypes.length + '">' + ctxLabels[ctx] + '</th>';
  });
  html += '</tr><tr>';
  contexts.forEach(() => {
    propTypes.forEach(pt => {
      html += '<th class="combo-header"><span class="combo-pt">' + pt.slice(0,3) + '</span></th>';
    });
  });
  html += '</tr></thead><tbody>';

  allFields.forEach(f => {
    html += '<tr data-key="' + f.key + '"><td><span class="field-label">' + escHtml(f.label) + '</span></td>';
    contexts.forEach(ctx => {
      const showList = f.show_on?.[ctx] || [];
      propTypes.forEach(pt => {
        const checked = showList.includes('*') || showList.includes(pt);
        html += '<td class="combo-cell"><input type="checkbox" class="field-cb"' +
          ' data-field="' + f.key + '" data-ctx="' + ctx + '" data-combo="' + pt + '"' +
          (checked ? ' checked' : '') + '></td>';
      });
    });
    html += '</tr>';
  });

  html += '</tbody></table></div>';
  cont.innerHTML = html;

  // Drag & drop sıralama
  initFieldDragSort();
}

function initFieldDragSort() {
  const list = document.getElementById('af-list');
  if (!list) return;
  let dragEl = null;
  list.querySelectorAll('.af-row').forEach(row => {
    row.addEventListener('dragstart', e => { dragEl = row; row.classList.add('dragging'); e.stopPropagation(); });
    row.addEventListener('dragend',   e => { row.classList.remove('dragging'); dragEl = null; });
    row.addEventListener('dragover',  e => {
      e.preventDefault();
      if (!dragEl || dragEl === row) return;
      const rect = row.getBoundingClientRect();
      const mid  = rect.top + rect.height/2;
      if (e.clientY < mid) list.insertBefore(dragEl, row);
      else list.insertBefore(dragEl, row.nextSibling);
    });
  });
}

function editField(key) {
  const f = state.cfg?.listing_fields?.all_fields?.find(x => x.key === key);
  if (!f) return;
  document.getElementById('nf-key').value   = f.key;
  document.getElementById('nf-label').value = f.label;
  document.getElementById('nf-type').value  = f.type;
  onNewFieldTypeChange(f.type);
  // Secenekleri doldur
  const optList = document.getElementById('nf-options-list');
  optList.innerHTML = '';
  if (f.type === 'select' && f.source) {
    const opts = state.cfg?.field_sources?.[f.source] || [];
    opts.forEach(o => {
      const span = document.createElement('span');
      span.className = 'setting-tag';
      span.innerHTML = escHtml(o) + '<button onclick="this.closest(\'.setting-tag\').remove()" style="background:none;border:none;cursor:pointer;color:var(--muted);font-size:14px;padding:0 2px">×</button>';
      optList.appendChild(span);
    });
  }
  document.getElementById('new-field-overlay').classList.add('open');
  // Kaydet butonunu guncelle modunu ac
  document.querySelector('#new-field-overlay .btn-gold').onclick = () => saveNewField(key);
}

async function saveAdminFields() {
  const cfg = state.cfg;
  const propTypes = cfg?.property_types || ['Daire','Villa','Arsa','Ticari'];
  const contexts  = ['form','card','detail','talep'];

  // Sıralama — af-list'teki sıraya göre
  const orderedKeys = [...document.querySelectorAll('#af-list .af-row')].map(r => r.dataset.key);

  // Her alan için show_on topla
  const allFields = state.cfg?.listing_fields?.all_fields || [];
  const fieldMap = {};
  allFields.forEach(f => {
    fieldMap[f.key] = {
      ...f,
      show_on: { form:[], card:[], detail:[], telegram:[], talep:[] }
    };
  });

  // Görünüm checkboxları
  document.querySelectorAll('.field-cb').forEach(cb => {
    if (!cb.checked) return;
    const key   = cb.dataset.field;
    const ctx   = cb.dataset.ctx;
    const combo = cb.dataset.combo;
    if (fieldMap[key] && !fieldMap[key].show_on[ctx].includes(combo)) {
      fieldMap[key].show_on[ctx].push(combo);
    }
  });

  // Meta checkboxlar (görev, talep, telegram)
  document.querySelectorAll('.af-meta-cb').forEach(cb => {
    const key  = cb.dataset.key;
    const meta = cb.dataset.meta;
    if (!fieldMap[key]) return;
    if (meta === 'telegram') {
      fieldMap[key].show_on.telegram = cb.checked ? propTypes : [];
    } else {
      fieldMap[key][meta] = cb.checked;
    }
  });

  // Sıralamaya göre diz
  const updatedFields = orderedKeys
    .map(k => fieldMap[k])
    .filter(Boolean);

  // Sıralamada olmayan alanları sona ekle
  allFields.forEach(f => {
    if (!orderedKeys.includes(f.key)) updatedFields.push(fieldMap[f.key]);
  });

  if (state.cfg?.listing_fields) {
    state.cfg.listing_fields.all_fields = updatedFields;
  }

  const customListsPayload = state.cfg?.custom_lists || {};

  try {
    await API.updateAdminSettings({ 
      all_fields: updatedFields,
      custom_lists: customListsPayload
    });
    showToast('Alan ayarları kaydedildi!');
    await loadConfig();
  } catch(e) { showToast(e.message, 'error'); }
}

function removeField(key) {
  if (!confirm(key + ' alanını silmek istiyor musunuz?')) return;
  if (state.cfg?.listing_fields?.all_fields) {
    state.cfg.listing_fields.all_fields = state.cfg.listing_fields.all_fields.filter(f => f.key !== key);
  }
  loadAdminFields();
}

function addNewField() {
  // Modal ac
  document.getElementById('nf-key').value = '';
  document.getElementById('nf-label').value = '';
  document.getElementById('nf-type').value = 'text';
  document.getElementById('nf-options-block').style.display = 'none';
  document.getElementById('nf-options-list').innerHTML = '';
  document.getElementById('nf-option-input').value = '';
  document.getElementById('new-field-overlay').classList.add('open');
  return;
}

function onNewFieldTypeChange(type) {
  document.getElementById('nf-options-block').style.display = type === 'select' ? '' : 'none';
}

function addNewFieldOption() {
  const input = document.getElementById('nf-option-input');
  const val = input.value.trim();
  if (!val) return;
  const list = document.getElementById('nf-options-list');
  const span = document.createElement('span');
  span.className = 'setting-tag';
  span.innerHTML = escHtml(val) + '<button onclick="this.closest(\".setting-tag\").remove()" style="background:none;border:none;cursor:pointer;color:var(--muted);font-size:14px;padding:0 2px">×</button>';
  list.appendChild(span);
  input.value = '';
  input.focus();
}

function saveNewField() {
  const key = document.getElementById('nf-key').value.trim().toLowerCase().replace(/\s+/g,'_');
  const label = document.getElementById('nf-label').value.trim();
  const type = document.getElementById('nf-type').value;

  if (!key) { showToast('Alan key zorunludur', 'error'); return; }
  if (!label) { showToast('Etiket zorunludur', 'error'); return; }
  if (!state.cfg?.listing_fields?.all_fields) return;
  if (state.cfg.listing_fields.all_fields.find(f => f.key === key)) {
    showToast('Bu key zaten mevcut!', 'error'); return;
  }

  let source = '';
  if (type === 'select') {
    // Secenekleri custom_lists'e ekle
    const options = [...document.querySelectorAll('#nf-options-list .setting-tag')]
      .map(t => t.childNodes[0].textContent.trim());
    if (options.length > 0) {
      const listKey = key + '_options';
      if (!state.cfg.custom_lists) state.cfg.custom_lists = {};
      state.cfg.custom_lists[listKey] = options;
      if (!state.cfg.field_sources) state.cfg.field_sources = {};
      state.cfg.field_sources[listKey] = options;
      source = listKey; // "cephe" -> "cephe_options"
    } else {
      // Seceneksiz select - source bos kalsin
      source = '';
    }
  }

  const newField = {
    key, label, type, required: false, searchable: false,
    source,
    show_on: {
      form: state.cfg?.listing_types?.flatMap(lt =>
        state.cfg?.property_types?.map(pt => lt+'/'+pt)
      ) || [],
      card: [], detail: [], telegram: []
    }
  };

  state.cfg.listing_fields.all_fields.push(newField);
  document.getElementById('new-field-overlay').classList.remove('open');
  loadAdminFields();
  showToast('Alan eklendi — kaydetmeyi unutmayın!');
}

function _addNewFieldOld() {
  const key = prompt('Alan key (örn: cephe):')?.trim().toLowerCase().replace(/\s+/g,'_');
  if (!key) return;
  const label = prompt('Alan etiketi (örn: Cephe):')?.trim();
  if (!label) return;
  const type = prompt('Tip (text/select/number/textarea):', 'text')?.trim() || 'text';
  let source = '';
  if (type === 'select') {
    // Mevcut kaynakları listele
    const sources = Object.keys(state.cfg?.field_sources || {});
    const sourceList = sources.join(', ');
    source = prompt('Kaynak listesi:\n' + sourceList + '\n\nKaynak adı girin:')?.trim() || '';
  }

  const newField = {
    key, label, type, required: false, searchable: false,
    source,
    show_on: { form:[], card:[], detail:[], telegram:[] }
  };

  if (!state.cfg?.listing_fields?.all_fields) return;
  if (state.cfg.listing_fields.all_fields.find(f => f.key === key)) {
    showToast('Bu key zaten mevcut!', 'error'); return;
  }
  state.cfg.listing_fields.all_fields.push(newField);
  loadAdminFields();
  showToast('Alan eklendi — kaydetmeyi unutmayın!');
}

/* ═══════════════════════════════════════════════════════
   ADMIN SETTINGS
════════════════════════════════════════════════════════ */
async function loadAdminSettings() {
  const cont = document.getElementById('admin-settings-content');
  if (!cont) return;
  try {
    const s = await API.getAdminSettings();
    s.listing_channels    = s.listing_channels    || state.cfg?.listing_channels    || [];
    s.auto_task_templates = s.auto_task_templates || state.cfg?.auto_task_templates || [];
    s.custom_lists        = s.custom_lists        || state.cfg?.custom_lists        || {};
    renderAdminSettings(s);
  } catch(e) { cont.innerHTML = `<div class="alert alert-error">Ayarlar yüklenemedi: ${e.message}</div>`; }
}

function renderAdminSettings(s) {
  const cont = document.getElementById('admin-settings-content');
  if (!cont) return;

  const channelsHTML   = buildChannelsHTML(s.listing_channels||[]);
  const autoTasksHTML  = buildAutoTasksHTML(s.auto_task_templates||[]);
  const ds = s.daily_summary || {};
  const summaryHTML = `
    <div class="setting-group">
      <div class="setting-group-header">
        <span class="setting-group-title">📅 Günlük Telegram Özeti</span>
      </div>
      <div class="setting-row">
        <label class="setting-label">
          <input type="checkbox" id="ds-enabled" ${ds.enabled ? 'checked' : ''}>
          Günlük özet aktif
        </label>
      </div>
      <div class="setting-row">
        <label class="setting-label">Gönderim saati</label>
        <select id="ds-hour" style="width:100px">
          ${[6,7,8,9,10,11,12].map(h =>
            `<option value="${h}" ${(ds.hour||9)===h?'selected':''}>${h.toString().padStart(2,'0')}:00</option>`
          ).join('')}
        </select>
      </div>
      <div class="setting-row">
        <label class="setting-label">
          <input type="checkbox" id="ds-sunday" ${ds.send_sunday ? 'checked' : ''}>
          Pazar günü de gönder
        </label>
      </div>
    </div>`;

  cont.innerHTML = `
    <div class="settings-grid">
      <div class="settings-col">
        ${channelsHTML}
      </div>
      <div class="settings-col">
        ${autoTasksHTML}
      </div>
    </div>
    ${summaryHTML}
    <div class="settings-footer">
      <button class="btn btn-gold" onclick="saveAdminSettings()">Ayarları Kaydet</button>
    </div>`;

  state._adminSettings = s;

  setTimeout(() => {
    document.getElementById('setting-channels')?.addEventListener('click', e => {
      const btn = e.target.closest('.talep-toggle-btn');
      const del = e.target.closest('[data-ch]');
      if (btn) { btn.classList.toggle('on'); btn.querySelector('.talep-toggle-text').textContent = btn.classList.contains('on') ? 'Acik' : 'Kapali'; }
      if (del) removeChannelItem(del.dataset.ch);
    });
    document.getElementById('setting-autotasks')?.addEventListener('click', e => {
      const btn = e.target.closest('.talep-toggle-btn');
      const del = e.target.closest('[data-at]');
      if (btn) { btn.classList.toggle('on'); btn.querySelector('.talep-toggle-text').textContent = btn.classList.contains('on') ? 'Acik' : 'Kapali'; }
      if (del) removeAutoTaskItem(del.dataset.at);
    });
  }, 100);
}

function addSettingItem(containerId) {
  const val = prompt('Yeni değer:');
  if (!val?.trim()) return;
  const cont = document.getElementById(containerId);
  if (!cont) return;
  const span = document.createElement('span');
  span.className = 'setting-tag';
  span.innerHTML = `${escHtml(val.trim())}<button onclick="removeSettingItem('${containerId}',this)" data-val="${escHtml(val.trim())}">&times;</button>`;
  cont.appendChild(span);
}

function removeSettingItem(containerId, btn) {
  btn.closest('.setting-tag').remove();
}

function getSettingList(key) {
  const cont = document.getElementById('setting-'+key);
  if (!cont) return null;
  return [...cont.querySelectorAll('.setting-tag')].map(t=>t.childNodes[0].textContent.trim());
}

function addCustomField(propType) {
  const key   = prompt('Alan key (örn: balcony):')?.trim().toLowerCase().replace(/\s+/g,'_');
  const label = prompt('Alan etiketi (örn: Balkon):')?.trim();
  if (!key || !label) return;
  const cont = document.getElementById('custom-fields-'+propType);
  if (!cont) return;
  const id = `byprop-${propType}-${key}`;
  const row = document.createElement('div');
  row.className = 'setting-toggle-row';
  row.innerHTML = `
    <span>${escHtml(label)} <small class="muted">(${escHtml(key)})</small></span>
    <button class="talep-toggle-btn on" id="${id}"
      onclick="this.classList.toggle('on');this.querySelector('.talep-toggle-text').textContent=this.classList.contains('on')?'Açık':'Kapalı'">
      <span class="talep-toggle-knob"></span>
      <span class="talep-toggle-text">Açık</span>
    </button>`;
  cont.appendChild(row);
  const cfgFields = state.cfg?.listing_fields?.all_fields;
  if (cfgFields && !cfgFields.find(f=>f.key===key)) {
    cfgFields.push({key, label, type:'text', required:false});
  }
  if (state._adminSettings) {
    if (!state._adminSettings.all_fields) state._adminSettings.all_fields = [];
    if (!state._adminSettings.all_fields.find(f=>f.key===key)) {
      state._adminSettings.all_fields.push({key, label, type:'text', required:false});
    }
  }
}



function buildChannelsHTML(channels) {
  let rows = channels.map(ch => {
    const onClass = ch.active ? ' on' : '';
    const onText  = ch.active ? 'Acik' : 'Kapali';
    const div = document.createElement('div');
    div.className = 'setting-toggle-row';
    div.id = 'ch-row-' + ch.key;
    div.innerHTML =
      '<span>' + escHtml(ch.icon||'') + ' ' + escHtml(ch.label) + '</span>' +
      '<div style="display:flex;gap:8px;align-items:center">' +
        '<button class="talep-toggle-btn' + onClass + '" id="ch-toggle-' + ch.key + '">' +
          '<span class="talep-toggle-knob"></span>' +
          '<span class="talep-toggle-text">' + onText + '</span>' +
        '</button>' +
        '<button class="btn btn-sm btn-danger" data-ch="' + ch.key + '">x</button>' +
      '</div>';
    return div.outerHTML;
  }).join('');
  return '<div class="setting-group">' +
    '<div class="setting-group-header">' +
      '<span class="setting-group-title">Yayin Kanallari</span>' +
      '<button class="btn btn-sm btn-outline" onclick="addChannelItem()">+ Ekle</button>' +
    '</div>' +
    '<div id="setting-channels">' + rows + '</div>' +
  '</div>';
}

function buildAutoTasksHTML(tasks) {
  let rows = tasks.map(t => {
    const onClass = t.active ? ' on' : '';
    const onText  = t.active ? 'Acik' : 'Kapali';
    const div = document.createElement('div');
    div.className = 'setting-toggle-row';
    div.id = 'at-row-' + t.key;
    div.innerHTML =
      '<span>' + escHtml(t.label) + '</span>' +
      '<div style="display:flex;gap:8px;align-items:center">' +
        '<button class="talep-toggle-btn' + onClass + '" id="at-toggle-' + t.key + '">' +
          '<span class="talep-toggle-knob"></span>' +
          '<span class="talep-toggle-text">' + onText + '</span>' +
        '</button>' +
        '<button class="btn btn-sm btn-danger" data-at="' + t.key + '">x</button>' +
      '</div>';
    return div.outerHTML;
  }).join('');
  return '<div class="setting-group">' +
    '<div class="setting-group-header">' +
      '<span class="setting-group-title">Otomatik Gorevler</span>' +
      '<button class="btn btn-sm btn-outline" onclick="addAutoTaskItem()">+ Ekle</button>' +
    '</div>' +
    '<div id="setting-autotasks">' + rows + '</div>' +
  '</div>';
}

function addChannelItem() {
  const label = prompt('Kanal adı (örn: Zillow):')?.trim();
  if (!label) return;
  const key = label.toLowerCase().replace(/\s+/g,'_');
  const icon = prompt('İkon (emoji, örn: 🌐):')?.trim() || '🌐';
  const cont = document.getElementById('setting-channels');
  if (!cont) return;
  const row = document.createElement('div');
  row.className = 'setting-toggle-row';
  row.id = 'ch-row-'+key;
  row.innerHTML = `
    <span>${icon} ${escHtml(label)}</span>
    <div style="display:flex;gap:8px;align-items:center">
      <button class="talep-toggle-btn on" id="ch-toggle-${key}"
        onclick="this.classList.toggle('on');this.querySelector('.talep-toggle-text').textContent=this.classList.contains('on')?'Açık':'Kapalı'">
        <span class="talep-toggle-knob"></span>
        <span class="talep-toggle-text">Açık</span>
      </button>
      <button class="btn btn-sm btn-danger" onclick="removeChannelItem('${key}')">×</button>
    </div>`;
  cont.appendChild(row);
  if (!state._adminSettings) state._adminSettings = {};
  if (!state._adminSettings.listing_channels) state._adminSettings.listing_channels = [];
  state._adminSettings.listing_channels.push({key, label, icon, active: true});
  if (state.cfg) {
    if (!state.cfg.listing_channels) state.cfg.listing_channels = [];
    state.cfg.listing_channels.push({key, label, icon, active: true});
  }
}

function removeChannelItem(key) {
  document.getElementById('ch-row-'+key)?.remove();
}

function addAutoTaskItem() {
  const label = prompt('Görev açıklaması (örn: Fotoğraf çekimi):')?.trim();
  if (!label) return;
  const key = label.toLowerCase().replace(/\s+/g,'_').replace(/[^a-z0-9_]/g,'');
  const cont = document.getElementById('setting-autotasks');
  if (!cont) return;
  const row = document.createElement('div');
  row.className = 'setting-toggle-row';
  row.id = 'at-row-'+key;
  row.innerHTML = `
    <span>${escHtml(label)}</span>
    <div style="display:flex;gap:8px;align-items:center">
      <button class="talep-toggle-btn on" id="at-toggle-${key}"
        onclick="this.classList.toggle('on');this.querySelector('.talep-toggle-text').textContent=this.classList.contains('on')?'Açık':'Kapalı'">
        <span class="talep-toggle-knob"></span>
        <span class="talep-toggle-text">Açık</span>
      </button>
      <button class="btn btn-sm btn-danger" onclick="removeAutoTaskItem('${key}')">×</button>
    </div>`;
  cont.appendChild(row);
  if (!state._adminSettings) state._adminSettings = {};
  if (!state._adminSettings.auto_task_templates) state._adminSettings.auto_task_templates = [];
  state._adminSettings.auto_task_templates.push({key, label, active: true, priority: 'normal'});
}

function removeAutoTaskItem(key) {
  document.getElementById('at-row-'+key)?.remove();
}

async function saveAdminSettings() {
  const payload = {
    property_types:  getSettingList('property_types'),
    listing_types:   getSettingList('listing_types'),
    districts:       getSettingList('districts'),
    neighborhoods:   getSettingList('neighborhoods'),
    room_options:    getSettingList('room_options'),
    heating_options: getSettingList('heating_options'),
    floor_options:   getSettingList('floor_options'),
    zoning_options:  getSettingList('zoning_options'),
  };

  // request_common: sabit sistem alanları + admin alan listesinde "Talep" işaretli olanlar
  // "Talep" checkbox'ı işaretli alanları af-list'teki sıraya göre al
  const baseFields = [
    {key:'client_name', label:'Müşteri Adı', type:'text', required:true},
    {key:'phone', label:'Telefon', type:'text', required:true},
  ];
  const allFieldsCurrent = state.cfg?.listing_fields?.all_fields || [];
  const fieldDefMap = {};
  allFieldsCurrent.forEach(f => { fieldDefMap[f.key] = f; });

  // af-list sırasına göre "talep" işaretli alanları topla
  const talepFields = [];
  document.querySelectorAll('#af-list .af-row').forEach(row => {
    const key = row.dataset.key;
    const cb  = row.querySelector('.af-meta-cb[data-meta="talep"]');
    if (cb && cb.checked && fieldDefMap[key]) {
      const f = fieldDefMap[key];
      talepFields.push({
        key:      f.key,
        label:    f.label,
        type:     f.type,
        required: f.required || false,
        ...(f.source ? {source: f.source} : {}),
      });
    }
  });
  payload.request_common = [...baseFields, ...talepFields];

  // request_by_property — alan yönetimi tablosundaki show_on.talep checkboxlarından oku
  const byPropNew = {};
  (payload.property_types||[]).forEach(pt => {
    const active = [];
    document.querySelectorAll(`.field-cb[data-ctx="talep"][data-combo="${CSS.escape(pt)}"]`).forEach(cb => {
      if (cb.checked) active.push(cb.dataset.field);
    });
    byPropNew[pt] = active;
  });
  payload.request_by_property = byPropNew;

  // Günlük özet ayarları
  payload.daily_summary = {
    enabled:     document.getElementById('ds-enabled')?.checked || false,
    hour:        parseInt(document.getElementById('ds-hour')?.value || '9'),
    send_sunday: document.getElementById('ds-sunday')?.checked || false,
  };
  payload.all_fields = state.cfg?.listing_fields?.all_fields || [];

  // Ozel listeler
  const customLists = {};
  document.querySelectorAll('#custom-lists-container .setting-tags').forEach(cont => {
    const key = cont.id.replace('setting-custom_','');
    const items = [...cont.querySelectorAll('.setting-tag')].map(t=>t.childNodes[0].textContent.trim());
    customLists[key] = items;
  });
  payload.custom_lists = customLists;

  // Kanallar
  const channels = [];
  document.querySelectorAll('#setting-channels .setting-toggle-row').forEach(row => {
    const key = row.id.replace('ch-row-','');
    const label = row.querySelector('span')?.textContent?.trim() || key;
    const active = document.getElementById('ch-toggle-'+key)?.classList.contains('on') || false;
    const icon = label.match(/^(\S+)/)?.[1] || '🌐';
    const cleanLabel = label.replace(icon,'').trim();
    channels.push({key, label: cleanLabel, icon, active});
  });
  payload.listing_channels = channels;

  // Otomatik gorevler
  const autoTasks = [];
  document.querySelectorAll('#setting-autotasks .setting-toggle-row').forEach(row => {
    const key = row.id.replace('at-row-','');
    const label = row.querySelector('span')?.textContent?.trim() || key;
    const active = document.getElementById('at-toggle-'+key)?.classList.contains('on') || false;
    autoTasks.push({key, label, active, priority: 'normal'});
  });
  payload.auto_task_templates = autoTasks;

  try {
    await API.updateAdminSettings(payload);
    showToast('Ayarlar kaydedildi!');
    await loadConfig();
    await loadAdminSettings();
  } catch(e) { showToast(e.message,'error'); }
}

/* ═══════════════════════════════════════════════════════
   AUTH
════════════════════════════════════════════════════════ */
document.getElementById('login-btn').addEventListener('click',doLogin);
document.getElementById('login-password').addEventListener('keydown',e=>{if(e.key==='Enter')doLogin();});
async function doLogin(){
  const username=document.getElementById('login-username').value.trim();
  const password=document.getElementById('login-password').value;
  const errEl=document.getElementById('login-error');
  errEl.style.display='none';
  if(!username||!password){errEl.textContent='Kullanıcı adı ve şifre gerekli.';errEl.style.display='block';return;}
  const btn=document.getElementById('login-btn');
  btn.textContent='Giriş yapılıyor...'; btn.disabled=true;
  try{
    await API.login(username,password);
    await loadConfig(); showApp();
    await Promise.all([loadListings(),loadRequests()]);
  }catch(e){errEl.textContent=e.message;errEl.style.display='block';}
  finally{btn.textContent='Giriş Yap';btn.disabled=false;}
}
document.getElementById('logout-btn').addEventListener('click',async()=>{await API.logout();showLogin();});


document.getElementById('nf-option-input')?.addEventListener('keydown', e => {
  if (e.key === 'Enter') { e.preventDefault(); addNewFieldOption(); }
});

init();
