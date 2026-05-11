/* ============================================================
   EmlakPro — app.js  v3
   ============================================================ */

const state = {
  cfg: null,
  listings: [],
  requests: [],
  editListingId: null,
  editRequestId: null,
  editDetailId:  null,
  coverPath: '', coverURL: '',
  galleryPaths: [], galleryExisting: [], removedImageIds: [],
};

/* ── Başlatma ──────────────────────────────────────────────── */
async function init() {
  const valid = await API.validateSession();
  if (!valid) { showLogin(); return; }
  try {
    await loadConfig();
    showApp();
    await Promise.all([loadListings(), loadRequests()]);
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

// Input'a yazarken binlik ayraç
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

function showToast(msg, type='info') {
  const t = document.getElementById('toast');
  t.textContent = msg;
  t.className = 'toast show toast-'+type;
  setTimeout(()=>t.classList.remove('show'), 3000);
}

/* ── NAVİGASYON ────────────────────────────────────────────── */
document.querySelectorAll('.nav-btn').forEach(btn => {
  btn.addEventListener('click', function() {
    document.querySelectorAll('.nav-btn').forEach(b=>b.classList.remove('active'));
    this.classList.add('active');
    document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
    document.getElementById('page-'+this.dataset.page).classList.add('active');
    if (this.dataset.page==='admin') loadAdminPanel();
  });
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
  });
});

/* ═══════════════════════════════════════════════════════
   İLAN PLACEHOLDER — mülk tipine göre gradient
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
    const isPassive = !il.is_active;
    const badge = isPassive&&isOwner
      ? `<span class="badge badge-passive">Pasif</span>`
      : !isPassive
        ? `<span class="badge badge-${il.fields?.listing_type==='Satılık'?'sale':'rent'}">${il.fields?.listing_type||''}</span>`
        : '';

    const noTag = il.listing_no ? `<span class="listing-no">#${il.listing_no}</span>` : '';
    const imgHTML = il.cover_image
      ? `<img src="${il.cover_image}" alt="${il.fields?.title||''}" loading="lazy">`
      : cardPlaceholder(propType);

    const ownerActions = isOwner ? `
      <div class="card-actions">
        <button class="icon-btn icon-btn-edit" onclick="openEditListing(${il.id},event)" title="Düzenle">✏️</button>
        <button class="icon-btn icon-btn-toggle" onclick="doToggleListing(${il.id},event)"
          title="${il.is_active?'Pasif Yap':'Aktif Yap'}">${il.is_active?'⏸':'▶️'}</button>
      </div>` : '';

    // Fiyat — tam format
    const priceMin = il.fields?.price_min;
    const priceMax = il.fields?.price_max;
    const priceDisplay = priceMin||priceMax
      ? (priceMin&&priceMax ? `${fiyatFormat(priceMin)} – ${fiyatFormat(priceMax)}` : fiyatFormat(priceMin||priceMax))
      : fiyatFormat(il.fields?.price);

    return `<div class="card${isPassive?' card-passive':''}" data-id="${il.id}">
      <div class="card-img">
        ${imgHTML}${badge}${noTag}${ownerActions}
      </div>
      <div class="card-body">
        <div class="card-title">${il.fields?.title||'—'}</div>
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

async function doToggleListing(id,e) {
  e.stopPropagation();
  try { await API.toggleListing(id); await loadListings(); showToast('Durum güncellendi.'); }
  catch(err) { showToast(err.message,'error'); }
}

/* ─── Detay Modal ─────────────────────────────────────────── */
async function openDetailModal(id) {
  try {
    const il = await API.getListing(id);
    state.editDetailId = id;
    document.getElementById('detail-title').textContent = il.fields?.title||'İlan Detayı';
    const cfg     = state.cfg;
    const propType= il.fields?.property_type||'Daire';
    const sumKeys = cfg?.listing_fields?.summary_fields||[];

    // Fiyat gösterimi
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

    const priceRow = `<tr><td class="detail-label">Fiyat</td><td><b>${priceVal}</b></td></tr>`;
    const noHTML   = il.listing_no ? `<div class="detail-no">İlan No: <b>#${il.listing_no}</b></div>` : '';
    const coverHTML= il.cover_image ? `<div class="detail-cover"><img src="${il.cover_image}" alt="" loading="lazy"></div>` : '';
    const gallery  = il.images?.length
      ? `<div class="detail-gallery">${il.images.map(img=>
          `<img src="${img.path}" alt="" loading="lazy" onclick="openLightbox('${img.path}')">`
        ).join('')}</div>` : '';

    document.getElementById('detail-content').innerHTML = `
      ${coverHTML}${noHTML}
      <table class="detail-table">${priceRow}${rows}</table>
      ${gallery}
      ${il.fields?.description?`<div class="detail-desc"><b>Açıklama:</b><p>${il.fields.description}</p></div>`:''}
    `;
    document.getElementById('detail-overlay').classList.add('open');
  } catch(e) { showToast(e.message,'error'); }
}

function openLightbox(src) {
  document.getElementById('lightbox-img').src = src;
  document.getElementById('lightbox').classList.add('open');
}
document.getElementById('lightbox')?.addEventListener('click',()=>
  document.getElementById('lightbox').classList.remove('open'));
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
document.getElementById('detail-edit-btn').addEventListener('click',()=>{
  document.getElementById('detail-overlay').classList.remove('open');
  openEditListing(state.editDetailId);
});

/* ─── İlan Form ───────────────────────────────────────────── */
document.getElementById('yeni-ilan-btn').addEventListener('click',()=>openIlanModal());
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
}

function renderIlanFormFields(allFields, ilan, isAdmin, propType) {
  const cfg = state.cfg;
  const propSpecificKeys = propType ? (cfg?.listing_fields?.card_fields?.[propType]||[]) : null;
  const alwaysShow = ['title','listing_type','property_type','district','neighborhood',
                      'price','price_min','price_max','area_m2','description','notes','address'];

  const html = allFields
    .filter(f => !f.admin_only||isAdmin)
    .filter(f => f.key!=='price') // price yerine price_min/max kullanıyoruz
    .map(f => {
      const isAlways = alwaysShow.includes(f.key);
      const inProp   = propSpecificKeys ? propSpecificKeys.includes(f.key) : true;
      const hidden   = propType && !isAlways && !inProp;
      const val      = ilan?.fields?.[f.key]||'';
      let input = '';
      if (f.type==='select') {
        const opts = cfg.field_sources?.[f.source]||[];
        input = `<select id="f-${f.key}" ${f.required?'required':''}>
          <option value="">Seçin...</option>
          ${opts.map(o=>`<option ${o===val?'selected':''}>${o}</option>`).join('')}
        </select>`;
      } else if (f.type==='textarea') {
        input = `<textarea id="f-${f.key}" rows="3">${val}</textarea>`;
      } else {
        // Fiyat inputlarına binlik format
        const isPrice = f.key==='price_min'||f.key==='price_max';
        input = `<input id="f-${f.key}" type="text" inputmode="${f.type==='number'?'numeric':''}"
          value="${isPrice?formatDisplayPrice(val):val}"
          ${isPrice?`data-raw="${val}"`:''}
          placeholder="${f.label}"
          ${f.required?'required':''}
          ${isPrice?'oninput="formatPriceInput(this)"':''}>`;
      }
      return `<div class="form-group" id="fg-${f.key}" ${hidden?'style="display:none"':''}>
        <label>${f.label}${f.required?' <span class="req">*</span>':''}</label>
        ${input}
      </div>`;
    }).join('');

  // Fiyat aralığı bloğu
  const priceMinVal = ilan?.fields?.price_min||'';
  const priceMaxVal = ilan?.fields?.price_max||'';
  const priceBlock = `
    <div class="form-group">
      <label>Fiyat Aralığı (₺)</label>
      <div class="price-range-row">
        <input id="f-price_min" type="text" inputmode="numeric"
          value="${formatDisplayPrice(priceMinVal)}" data-raw="${priceMinVal}"
          placeholder="En az" oninput="formatPriceInput(this)">
        <span class="price-range-sep">—</span>
        <input id="f-price_max" type="text" inputmode="numeric"
          value="${formatDisplayPrice(priceMaxVal)}" data-raw="${priceMaxVal}"
          placeholder="En fazla" oninput="formatPriceInput(this)">
      </div>
    </div>`;

  // Fiyat bloğunu title'dan sonra ekle
  const titleIdx = allFields.findIndex(f=>f.key==='title');
  const parts = html.split('</div>\n');
  document.getElementById('ilan-form-body').innerHTML = html + priceBlock;

  document.getElementById('f-property_type')?.addEventListener('change', function() {
    updateIlanFormForPropType(this.value, allFields, isAdmin);
  });
}

function formatDisplayPrice(val) {
  if (!val) return '';
  const n = parseInt(val);
  return isNaN(n) ? '' : n.toLocaleString('tr-TR');
}

function updateIlanFormForPropType(propType, allFields, isAdmin) {
  const cfg = state.cfg;
  const propSpecificKeys = propType ? (cfg?.listing_fields?.card_fields?.[propType]||[]) : null;
  const alwaysShow = ['title','listing_type','property_type','district','neighborhood',
                      'price','price_min','price_max','area_m2','description','notes','address'];
  allFields.filter(f=>!f.admin_only||isAdmin).forEach(f => {
    const isAlways = alwaysShow.includes(f.key);
    const inProp   = propSpecificKeys ? propSpecificKeys.includes(f.key) : true;
    const fg = document.getElementById('fg-'+f.key);
    if (!fg) return;
    fg.style.display = (!propType||isAlways||inProp) ? '' : 'none';
  });
}

document.getElementById('kaydet-btn').addEventListener('click', async ()=>{
  const fields = {};
  (state.cfg?.listing_fields?.all_fields||[]).forEach(f => {
    if (f.key==='price') return;
    const el = document.getElementById('f-'+f.key);
    if (el) fields[f.key] = el.value;
  });
  fields.price_min = getRawPrice('f-price_min');
  fields.price_max = getRawPrice('f-price_max');
  // price alanını da doldur (geriye dönük uyumluluk)
  fields.price = fields.price_max || fields.price_min;

  if (!fields.title) { showToast('Başlık zorunludur','error'); return; }
  if (!fields.price_min&&!fields.price_max) { showToast('En az bir fiyat giriniz','error'); return; }

  try {
    const payload = {
      fields, cover_image: state.coverPath,
      images: state.galleryPaths.map(g=>g.path),
      remove_images: state.removedImageIds,
    };
    if (state.editListingId) {
      await API.updateListing(state.editListingId, payload);
      showToast('✅ İlan güncellendi!');
    } else {
      await API.createListing(payload);
      showToast('🎉 İlan eklendi!');
    }
    closeIlanModal();
    await loadListings();
  } catch(e) { showToast(e.message,'error'); }
});

/* ─── Cover Upload ────────────────────────────────────────── */
document.getElementById('cover-zone').addEventListener('click',()=>document.getElementById('cover-input').click());
document.getElementById('cover-input').addEventListener('change', async function(){
  const file=this.files[0]; if(!file) return;
  try {
    showToast('Resim yükleniyor...','info');
    const res = await API.uploadCover(file);
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
  const maxLeft=12-state.galleryPaths.length-state.galleryExisting.length;
  for (const file of files.slice(0,maxLeft)) {
    try { const res=await API.uploadGallery(file); state.galleryPaths.push({path:res.path,url:res.url}); renderGalleryPreview(); }
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
  try { state.requests=await API.getRequests()||[]; renderRequests(); }
  catch(e) { showToast('Talepler yüklenemedi: '+e.message,'error'); }
}

function calcMatchScore(talep, ilan) {
  if(!ilan.is_active) return 0;
  let score=0, total=0;
  const check=(tVal,iVal,w)=>{ total+=w; if(!tVal) score+=w; else if(tVal===iVal) score+=w; };
  check(talep.fields?.listing_type,  ilan.fields?.listing_type,  25);
  check(talep.fields?.property_type, ilan.fields?.property_type, 20);
  check(talep.fields?.district,      ilan.fields?.district,      20);
  // Bütçe: talep budget_max ile ilan price karşılaştır
  total+=20;
  const budgetMax = parseInt(talep.fields?.budget_max||talep.fields?.budget)||0;
  const budgetMin = parseInt(talep.fields?.budget_min)||0;
  const price     = parseInt(ilan.fields?.price_max||ilan.fields?.price)||0;
  if (!budgetMax) score+=20;
  else if (price<=budgetMax) score+=20;
  else if (price<=budgetMax*1.1) score+=10;
  // Bütçe alt sınır
  if (budgetMin && price < budgetMin) return 0; // alt sınırın altı = eşleşme yok
  // Oda
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
  const list=document.getElementById('talep-list');
  const q =document.getElementById('talep-search')?.value?.toLowerCase()||'';
  const lt=document.getElementById('talep-tip-filter')?.value||'';
  const d =document.getElementById('talep-ilce-filter')?.value||'';
  let data=state.requests.filter(t=>{
    if(lt&&t.fields?.listing_type!==lt) return false;
    if(d&&t.fields?.district!==d) return false;
    if(q&&!t.fields?.client_name?.toLowerCase().includes(q)&&!t.fields?.district?.toLowerCase().includes(q)) return false;
    return true;
  });
  if(!data.length){
    list.innerHTML='<div class="empty-state"><div class="big-icon">🎯</div><p>Talep bulunamadı.</p></div>';
    return;
  }
  const colors=['#1565C0','#6a1b9a','#1b5e20','#c62828','#e65100','#00695c'];
  list.innerHTML=data.map((t,idx)=>{
    const c=colors[idx%colors.length];
    const harf=(t.fields?.client_name||'M')[0].toUpperCase();
    const matches=state.listings
      .map(il=>({il,score:calcMatchScore(t,il)}))
      .filter(m=>m.score>0)
      .sort((a,b)=>b.score-a.score);

    // Bütçe gösterimi
    const bMin=t.fields?.budget_min, bMax=t.fields?.budget_max||t.fields?.budget;
    const budgetDisplay = bMin&&bMax
      ? `${fiyatFormat(bMin)} – ${fiyatFormat(bMax)}`
      : bMax ? `max ${fiyatFormat(bMax)}` : '';

    const tags=[
      t.fields?.listing_type  ?`<span class="tag tag-blue">${t.fields.listing_type}</span>`:'',
      t.fields?.property_type ?`<span class="tag tag-purple">${t.fields.property_type}</span>`:'',
      t.fields?.district      ?`<span class="tag tag-green">${t.fields.district}</span>`:'',
      budgetDisplay           ?`<span class="tag tag-amber">${budgetDisplay}</span>`:'',
    ].join('');

    return `<div class="talep-card${t.is_active?'':' talep-passive'}" id="talep-${t.id}">
      <div class="talep-header" onclick="toggleTalepAcc(${t.id})">
        <div class="talep-avatar" style="background:${c}22;color:${c}">${harf}</div>
        <div class="talep-info">
          <div class="talep-name">${t.fields?.client_name||'—'}<span class="phone"> · ${t.fields?.phone||''}</span></div>
          <div class="talep-desc">${t.fields?.notes||''}</div>
          <div class="talep-tags">${tags}</div>
        </div>
        <div class="talep-right">
          <div class="ok-btn" id="ok-${t.id}"><span class="chevron">▾</span></div>
          <div class="eslesme-badge${matches.length?'':' zero'}">${matches.length}</div>
          <div class="zil-btn${t.notify_me?' active':''}" onclick="doToggleNotify(${t.id},event)">
            🔔${t.notify_me?'<span class="zil-dot"></span>':''}
          </div>
          <div class="toggle-btn${t.is_active?' on':''}" onclick="doToggleRequest(${t.id},event)">
            <span class="toggle-knob"></span>
          </div>
          <button class="icon-btn icon-btn-edit" onclick="openEditRequest(${t.id},event)">✏️</button>
        </div>
      </div>
      <div class="accordion-body" id="acc-${t.id}">
        <div class="acc-title">Eşleşen İlanlar (${matches.length})</div>
        <div class="ilan-mini-list">
          ${!matches.length
            ? '<div class="empty-acc">🔍 Eşleşen ilan bulunamadı.</div>'
            : matches.map(({il,score})=>{
                const r=scoreColor(score);
                const cfg=state.cfg;
                const propType=il.fields?.property_type||'Daire';
                const cardKeys=cfg?.listing_fields?.card_fields?.[propType]||[];
                const detailTags=cardKeys.map(k=>{
                  const v=il.fields?.[k]; return v?`<span class="meta-tag">${v}</span>`:'';
                }).join('');
                const priceMin=il.fields?.price_min, priceMax=il.fields?.price_max;
                const priceDisp=priceMin||priceMax
                  ?(priceMin&&priceMax?`${fiyatFormat(priceMin)}–${fiyatFormat(priceMax)}`:fiyatFormat(priceMin||priceMax))
                  :fiyatFormat(il.fields?.price);
                const imgThumb=il.cover_image
                  ?`<img src="${il.cover_image}" alt="" class="ilan-mini-thumb" loading="lazy">`
                  :`<div class="ilan-mini-icon">${(PROP_PLACEHOLDER[propType]||PROP_PLACEHOLDER.default).icon}</div>`;
                return `<div class="ilan-mini" onclick="openDetailModal(${il.id})">
                  ${imgThumb}
                  <div class="ilan-mini-info">
                    <div class="ilan-mini-title">${il.fields?.title||'—'}
                      ${il.listing_no?`<span class="mini-no">#${il.listing_no}</span>`:''}
                    </div>
                    <div class="ilan-mini-tags">${detailTags}</div>
                  </div>
                  <div class="ilan-mini-right">
                    <span class="eslesme-pill" style="background:${r.bg};color:${r.c}">%${score}</span>
                    <span class="ilan-mini-price">${priceDisp}</span>
                  </div>
                </div>`;
              }).join('')
          }
        </div>
      </div>
    </div>`;
  }).join('');
}

function toggleTalepAcc(id) {
  document.getElementById('acc-'+id).classList.toggle('open');
  document.getElementById('ok-'+id).classList.toggle('open');
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

function buildTalepForm(talep) {
  const cfg    = state.cfg;
  // Önce kritik alanlar, en sona notlar
  const commonRaw = cfg?.request_fields?.common||[];
  const notesField = commonRaw.find(f=>f.key==='notes');
  const common = [...commonRaw.filter(f=>f.key!=='notes'&&f.key!=='budget')];

  const propType  = talep?.fields?.property_type||'';
  const extraKeys = propType ? (cfg?.request_fields?.by_property?.[propType]||[]) : [];
  const extraFields = extraKeys
    .map(k=>cfg?.listing_fields?.all_fields?.find(f=>f.key===k))
    .filter(Boolean);

  const buildInput = (f, val='') => {
    if (f.type==='select') {
      const opts=cfg.field_sources?.[f.source]||[];
      return `<select id="tf-${f.key}" ${f.required?'required':''}>
        <option value="">Seçin...</option>
        ${opts.map(o=>`<option ${o===val?'selected':''}>${o}</option>`).join('')}
      </select>`;
    }
    if (f.type==='textarea') return `<textarea id="tf-${f.key}" rows="2">${val}</textarea>`;
    return `<input id="tf-${f.key}" type="${f.type}" value="${val}" placeholder="${f.label}" ${f.required?'required':''}>`;
  };

  // Bütçe bloğu — min/max
  const bMin=talep?.fields?.budget_min||'', bMax=talep?.fields?.budget_max||talep?.fields?.budget||'';
  const budgetBlock=`
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

  const html = [...common, ...extraFields].map(f=>{
    const val=talep?.fields?.[f.key]||'';
    return `<div class="form-group">
      <label>${f.label}${f.required?' <span class="req">*</span>':''}</label>
      ${buildInput(f,val)}
    </div>`;
  }).join('');

  // Notlar en alta
  const notesHTML = notesField ? `<div class="form-group">
    <label>${notesField.label}</label>
    <textarea id="tf-notes" rows="2">${talep?.fields?.notes||''}</textarea>
  </div>` : '';

  document.getElementById('talep-form-body').innerHTML =
    html + budgetBlock + notesHTML +
    `<div class="form-group">
      <label style="display:flex;align-items:center;gap:8px;cursor:pointer">
        <input type="checkbox" id="tf-notify" ${talep?.notify_me?'checked':''}>
        Yeni eşleşmelerde bildir
      </label>
    </div>`;

  // property_type değişince — sadece ilgili alanları güncelle, mevcut değerleri koru
  document.getElementById('tf-property_type')?.addEventListener('change', function() {
    const currentVals = {};
    document.querySelectorAll('#talep-form-body [id^="tf-"]').forEach(el=>{
      currentVals[el.id.replace('tf-','')]=el.type==='checkbox'?el.checked:el.value;
    });
    const newTalep = { ...talep, fields: { ...talep?.fields, ...currentVals, property_type: this.value }, notify_me: document.getElementById('tf-notify')?.checked };
    buildTalepForm(newTalep);
    // property_type'ı restore et
    const pt = document.getElementById('tf-property_type');
    if(pt) pt.value = this.value;
  });
}

document.getElementById('talep-kaydet-btn').addEventListener('click', async ()=>{
  const cfg=state.cfg;
  const fields={};
  const commonRaw=cfg?.request_fields?.common||[];
  const propType=document.getElementById('tf-property_type')?.value||'';
  const extraKeys=propType?(cfg?.request_fields?.by_property?.[propType]||[]):[];
  const extraFields=extraKeys.map(k=>cfg?.listing_fields?.all_fields?.find(f=>f.key===k)).filter(Boolean);
  [...commonRaw.filter(f=>f.key!=='budget'), ...extraFields].forEach(f=>{
    const el=document.getElementById('tf-'+f.key); if(el) fields[f.key]=el.value;
  });
  fields.budget_min=getRawPrice('tf-budget_min');
  fields.budget_max=getRawPrice('tf-budget_max');
  fields.budget=fields.budget_max||fields.budget_min; // geriye dönük
  fields.notes=document.getElementById('tf-notes')?.value||'';

  if(!fields.client_name){showToast('Müşteri adı zorunludur','error');return;}
  if(!fields.phone){showToast('Telefon zorunludur','error');return;}
  const notify=document.getElementById('tf-notify')?.checked||false;
  try {
    if(state.editRequestId){await API.updateRequest(state.editRequestId,{fields,notify_me:notify});showToast('✅ Talep güncellendi!');}
    else{await API.createRequest({fields,notify_me:notify});showToast('🎉 Talep eklendi!');}
    closeTalepModal(); await loadRequests();
  } catch(e){showToast(e.message,'error');}
});

/* ═══════════════════════════════════════════════════════
   ADMIN
════════════════════════════════════════════════════════ */
async function loadAdminPanel(){await loadAdminUsers();}
async function loadAdminUsers(){
  try{
    const users=await API.adminGetUsers();
    document.getElementById('users-table').innerHTML=`<table class="admin-table">
      <thead><tr><th>Kullanıcı</th><th>E-posta</th><th>Rol</th><th>Durum</th><th>İşlem</th></tr></thead>
      <tbody>${(users||[]).map(u=>`<tr>
        <td><b>${u.full_name}</b><br><small class="muted">@${u.username}</small></td>
        <td>${u.email||'—'}</td>
        <td><span class="tag ${u.role==='admin'?'tag-red':'tag-blue'}">${u.role}</span></td>
        <td><span class="tag ${u.is_active?'tag-green':'tag-amber'}">${u.is_active?'Aktif':'Pasif'}</span></td>
        <td>${u.role!=='admin'?`
          <button class="btn btn-sm btn-outline" onclick="adminToggleUser(${u.id})">${u.is_active?'Pasif Yap':'Aktif Yap'}</button>
          <button class="btn btn-sm btn-danger" onclick="adminDeleteUser(${u.id})">Sil</button>
        `:'—'}</td>
      </tr>`).join('')}</tbody></table>`;
  }catch(e){showToast(e.message,'error');}
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
        <td><span class="tag ${il.is_active?'tag-green':'tag-amber'}">${il.is_active?'Aktif':'Pasif'}</span></td>
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

init();
