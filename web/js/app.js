import * as max from './max.js';
import { ApiError, fetchCatalog, startSession } from './api.js';

const ALPHABET = 'АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЭЮЯ';
const SEARCH_LIMIT = 20;
const DEFAULT_ICON = '🏢';

const $ = (id) => document.getElementById(id);

// el builds a DOM element; text is always assigned via textContent.
function el(tag, attrs = {}, ...children) {
    const node = document.createElement(tag);
    for (const [key, value] of Object.entries(attrs)) {
        if (key === 'class') node.className = value;
        else if (key === 'text') node.textContent = value;
        else if (key.startsWith('on')) node.addEventListener(key.slice(2), value);
        else if (key === 'role' || key.startsWith('data-')) node.setAttribute(key, value);
        else node[key] = value;
    }
    node.append(...children.filter(Boolean));
    return node;
}

const normalize = (s) => s.toLowerCase().replaceAll('ё', 'е');

// Leading quotes and symbols do not count for the alphabet index.
const firstLetter = (name) => (name.match(/[\p{L}\p{N}]/u)?.[0] || '#').toUpperCase();

const state = {
    catalog: null,
    orgs: new Map(),
    categories: new Map(),
    stack: [],
};

// ---- Rendering helpers ----

function avatarOrIcon(org, imgClass, iconClass) {
    const icon = el('div', { class: iconClass, text: org.icon || DEFAULT_ICON });
    if (!org.avatar) return icon;
    const img = el('img', {
        class: imgClass,
        src: org.avatar,
        alt: '',
        loading: 'lazy',
        decoding: 'async',
        onerror: () => img.replaceWith(icon),
    });
    return img;
}

function openOrg(org) {
    max.hapticTap();
    max.openUrl(org.url);
}

function orgItem(org) {
    return el('div', {
        class: 'org-item',
        role: 'button',
        tabIndex: 0,
        'data-org-id': org.id,
        onclick: () => openOrg(org),
        onkeydown: (e) => e.key === 'Enter' && openOrg(org),
    },
        el('div', { class: 'org-icon' }, avatarOrIcon(org, 'org-avatar', 'org-emoji')),
        el('div', { class: 'org-info' },
            el('div', { class: 'org-name', text: org.name }),
            org.subtitle && el('div', { class: 'org-category', text: org.subtitle }),
        ),
    );
}

function emptyState(text) {
    return el('div', { class: 'empty-state' },
        el('div', { class: 'empty-state-icon', text: '📭' }),
        el('div', { class: 'empty-state-text', text }),
    );
}

function renderFeatured() {
    const items = state.catalog.featured.map((id) => state.orgs.get(id)).filter(Boolean);
    $('featuredList').replaceChildren(...items.map((org) => el('div', {
        class: 'channel-chip',
        role: 'button',
        tabIndex: 0,
        onclick: () => openOrg(org),
        onkeydown: (e) => e.key === 'Enter' && openOrg(org),
    },
        el('div', { class: 'channel-avatar-wrapper' },
            avatarOrIcon(org, 'channel-avatar', 'channel-emoji')),
        el('span', { class: 'channel-name', text: org.name }),
    )));
}

function renderCategories() {
    $('categoriesGrid').replaceChildren(...state.catalog.categories.map((cat) => el('div', {
        class: cat.wide ? 'category-card large' : 'category-card',
        role: 'button',
        tabIndex: 0,
        onclick: () => navigate('category', { id: cat.id }),
        onkeydown: (e) => e.key === 'Enter' && navigate('category', { id: cat.id }),
    },
        el('div', { class: 'category-icon', text: cat.icon || '📁' }),
        el('span', { class: 'category-title', text: cat.title }),
    )));
}

function renderCategoryScreen({ id, highlight }) {
    const cat = state.categories.get(id);
    $('categoryTitle').textContent = cat?.title || '';
    const orgs = (cat?.organizations || []).map((oid) => state.orgs.get(oid)).filter(Boolean);
    const list = $('organizationsList');
    list.replaceChildren(...(orgs.length
        ? orgs.map(orgItem)
        : [emptyState('В этой категории пока нет организаций')]));
    list.parentElement.scrollTop = 0;

    if (highlight) {
        const item = list.querySelector(`[data-org-id="${CSS.escape(highlight)}"]`);
        if (item) {
            item.classList.add('highlight');
            item.scrollIntoView({ block: 'center' });
        }
    }
}

function renderAllOrgScreen() {
    const groups = new Map();
    for (const org of state.catalog.organizations) {
        const letter = firstLetter(org.name);
        if (!groups.has(letter)) groups.set(letter, []);
        groups.get(letter).push(org);
    }

    const letters = [...ALPHABET, ...[...groups.keys()].filter((l) => !ALPHABET.includes(l))];
    $('alphabetNav').replaceChildren(...letters.map((letter) => {
        const has = groups.has(letter);
        return el('div', {
            class: has ? 'alphabet-item' : 'alphabet-item empty',
            text: letter,
            onclick: has ? () => $(`letter-${letter}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' }) : null,
        });
    }));

    $('allOrgList').replaceChildren(...letters.filter((l) => groups.has(l)).map((letter) =>
        el('div', { class: 'letter-group', id: `letter-${letter}` },
            el('div', { class: 'letter-header', text: letter }),
            ...groups.get(letter).map(orgItem),
        )));
}

// ---- Search ----

function highlightMatch(text, query) {
    const index = normalize(text).indexOf(query);
    if (index < 0) return [text];
    return [
        text.slice(0, index),
        el('mark', { text: text.slice(index, index + query.length) }),
        text.slice(index + query.length),
    ];
}

function renderSearch(raw) {
    const box = $('searchResults');
    const query = normalize(raw.trim());
    if (!query || !state.catalog) {
        box.replaceChildren();
        box.classList.add('hidden');
        return;
    }

    const results = [];
    for (const org of state.catalog.organizations) {
        if (normalize(org.name).includes(query) || normalize(org.subtitle).includes(query)) {
            results.push(org);
            if (results.length === SEARCH_LIMIT) break;
        }
    }

    box.replaceChildren(...(results.length
        ? results.map((org) => el('div', {
            class: 'search-result-item',
            role: 'button',
            tabIndex: 0,
            onclick: () => openOrg(org),
            onkeydown: (e) => e.key === 'Enter' && openOrg(org),
        },
            el('div', { class: 'search-result-title' }, ...highlightMatch(org.name, query)),
            el('div', { class: 'search-result-type', text: org.subtitle || 'Организация' }),
        ))
        : [el('div', { class: 'search-empty', text: 'Ничего не найдено' })]));
    box.classList.remove('hidden');
}

// ---- Navigation ----

const SCREENS = { welcome: 'welcomeScreen', main: 'mainScreen', category: 'categoryScreen', allOrg: 'allOrgScreen' };

function show(screen, params) {
    for (const [name, id] of Object.entries(SCREENS)) {
        $(id).classList.toggle('hidden', name !== screen);
    }
    if (screen === 'category') renderCategoryScreen(params);
    if (screen === 'allOrg') renderAllOrgScreen();

    if (state.stack.length > 1) max.backButton.show();
    else max.backButton.hide();
}

function navigate(screen, params = {}) {
    if (screen === 'main') state.stack = [];
    state.stack.push({ screen, params });
    show(screen, params);
}

function back() {
    if (state.stack.length <= 1) return;
    state.stack.pop();
    const { screen, params } = state.stack[state.stack.length - 1];
    show(screen, params);
}

// Supports deep links like https://max.ru/<bot>?startapp=category_health
// and startapp=org_<id> (channel_<id> kept for old links).
function applyStartParam(param) {
    const [, kind, id] = param.match(/^(category|org|channel)_(.+)$/) || [];
    if (kind === 'category' && state.categories.has(id)) {
        navigate('category', { id });
    } else if (kind && state.orgs.has(id)) {
        const catId = state.catalog.categories.find((c) => c.organizations.includes(id))?.id;
        if (catId) navigate('category', { id: catId, highlight: id });
    }
}

// ---- Data loading ----

function setStatus(text, retry = false) {
    $('status').classList.toggle('hidden', !text);
    $('statusText').textContent = text;
    $('retryBtn').classList.toggle('hidden', !retry);
}

async function loadCatalog(initData) {
    setStatus('Загрузка…');
    try {
        const catalog = await fetchCatalog(initData);
        state.catalog = catalog;
        state.orgs = new Map(catalog.organizations.map((o) => [o.id, o]));
        state.categories = new Map(catalog.categories.map((c) => [c.id, c]));
        renderFeatured();
        renderCategories();
        $('catalog').classList.remove('hidden');
        setStatus('');
        return true;
    } catch (err) {
        console.error('catalog load failed', err);
        const expired = err instanceof ApiError && err.status === 401;
        setStatus(expired
            ? 'Сессия устарела. Закройте и снова откройте приложение.'
            : 'Не удалось загрузить данные. Проверьте соединение.', !expired);
        return false;
    }
}

async function init() {
    max.backButton.onClick(back);
    document.querySelectorAll('[data-back]').forEach((btn) => btn.addEventListener('click', back));
    $('allOrgBtn').addEventListener('click', () => state.catalog && navigate('allOrg'));

    let searchTimer;
    $('searchInput').addEventListener('input', (e) => {
        clearTimeout(searchTimer);
        searchTimer = setTimeout(() => renderSearch(e.target.value), 150);
    });

    const initData = max.initData();
    const param = max.startParam();

    if (!initData) {
        navigate('main');
        setStatus('Откройте приложение в мессенджере MAX.');
        max.ready();
        return;
    }

    $('retryBtn').addEventListener('click', () => loadCatalog(initData));
    startSession(initData).catch((err) => console.warn('session not recorded', err));
    const loading = loadCatalog(initData);

    if (param) {
        navigate('main');
        max.ready();
        if (await loading) applyStartParam(param);
        return;
    }

    show('welcome');
    $('startBtn').addEventListener('click', () => navigate('main'), { once: true });
    max.ready();
}

init();
