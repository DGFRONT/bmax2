class ChannelRouterApp {
    constructor() {
        this.searchTimeout = null;
        this.isInitialized = false;
        this.appData = null;
        this.token = null;
    }

    async init() {
        try {
            console.log('Initializing app...');
            await MaxBridge.init();
            Router.init();

            await this.authenticate();
            await this.loadData();

            Router.setAppData(this.appData);

            this.renderPopularChannels();
            this.renderCategories();
            this.setupEventListeners();

            const startParam = MaxBridge.getStartParam();
            if (startParam) this.handleStartParam(startParam);

            this.isInitialized = true;
            console.log('App initialized successfully');
        } catch (error) {
            console.error('Failed to initialize app:', error);
            alert('Ошибка загрузки данных. Проверьте соединение.');
        }
    }

    async authenticate() {
        const initData = window.WebApp?.initData;
        
        // Для локальной разработки (без MAX)
        if (!initData) {
            console.warn('No initData, using dev mode');
            this.token = 'dev-token';
            return;
        }

        const response = await fetch(`${MaxBridge.API_URL}/api/auth`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ initData })
        });

        if (!response.ok) throw new Error('Ошибка валидации на сервере');
        
        const data = await response.json();
        this.token = data.token;
        sessionStorage.setItem('max_app_token', this.token);
    }

    async loadData() {
        const response = await fetch(`${MaxBridge.API_URL}/api/channels`, {
            headers: { 'Authorization': `Bearer ${this.token}` }
        });

        if (!response.ok) throw new Error('Ошибка загрузки данных каналов');
        this.appData = await response.json();
    }

    setupEventListeners() {
        const searchInput = document.getElementById('searchInput');
        searchInput.addEventListener('input', (e) => {
            clearTimeout(this.searchTimeout);
            this.searchTimeout = setTimeout(() => {
                this.handleSearch(e.target.value);
            }, 300);
        });

        if (window.WebApp) {
            window.WebApp.onViewportChanged(() => {
                this.handleViewportChange();
            });
        }

        this.setupSwipeGestures();
    }

    renderPopularChannels() {
        const container = document.getElementById('popularChannels');
        const governor = this.appData.governor;
        const governorChip = this.createChannelChip(governor);
        container.appendChild(governorChip);

        const sortedPopular = [...this.appData.popularChannels]
            .sort((a, b) => b.subscribers - a.subscribers)
            .slice(0, 4);

        sortedPopular.forEach(channel => {
            const chip = this.createChannelChip(channel);
            container.appendChild(chip);
        });
    }

    createChannelChip(channel) {
        const chip = document.createElement('div');
        chip.className = 'channel-chip';
        const isGovernor = channel.id === 'governor';
        
        chip.innerHTML = `
            <div class="channel-avatar-wrapper">
                ${isGovernor && channel.avatar ? 
                    `<img src="${channel.avatar}" alt="${channel.name}" class="channel-avatar" onerror="this.style.display='none'; this.parentElement.innerHTML='<div style=\'width:100%;height:100%;display:flex;align-items:center;justify-content:center;font-size:24px;\'></div>'">` :
                    `<div style="width:100%;height:100%;display:flex;align-items:center;justify-content:center;font-size:32px;">${channel.icon || '🏢'}</div>`
                }
            </div>
            <span class="channel-name">${channel.name}</span>
        `;

        chip.addEventListener('click', () => {
            if (channel.deepLink) {
                MaxBridge.openLink(channel.deepLink);
            }
        });

        return chip;
    }

    renderCategories() {
        const container = document.getElementById('categoriesGrid');
        
        container.innerHTML = this.appData.categories.map(category => `
            <div class="category-card ${category.large ? 'large' : ''}" data-category-id="${category.id}">
                <div class="category-icon">${category.icon}</div>
                <span class="category-title">${category.title}</span>
            </div>
        `).join('');

        container.querySelectorAll('.category-card').forEach(card => {
            card.addEventListener('click', () => {
                const categoryId = card.dataset.categoryId;
                Router.openCategory(categoryId);
            });
        });
    }

    handleSearch(query) {
        const trimmedQuery = query.trim();
        const existingResults = document.querySelector('.search-results');
        if (existingResults) existingResults.remove();

        if (!trimmedQuery) return;

        const results = this.search(trimmedQuery);
        if (results.length === 0) return;

        const resultsContainer = document.createElement('div');
        resultsContainer.className = 'search-results';
        
        resultsContainer.innerHTML = results.slice(0, 10).map(org => `
            <div class="search-result-item" data-org-id="${org.id}">
                <div class="search-result-title">${this.highlightMatch(org.name, trimmedQuery)}</div>
                <div class="search-result-type">${org.category || 'Организация'}</div>
            </div>
        `).join('');

        const searchSection = document.querySelector('.search-section');
        searchSection.appendChild(resultsContainer);

        resultsContainer.querySelectorAll('.search-result-item').forEach(item => {
            item.addEventListener('click', () => {
                const orgId = item.dataset.orgId;
                const org = results.find(o => o.id === orgId);
                if (org && org.deepLink) {
                    MaxBridge.openLink(org.deepLink);
                    document.getElementById('searchInput').value = '';
                    resultsContainer.remove();
                }
            });
        });
    }

    search(query) {
        const lowerQuery = query.toLowerCase();
        const all = Router.getAllOrganizations();
        return all.filter(org => 
            org.name.toLowerCase().includes(lowerQuery) ||
            (org.category && org.category.toLowerCase().includes(lowerQuery))
        );
    }

    highlightMatch(text, query) {
        const regex = new RegExp(`(${query})`, 'gi');
        return text.replace(regex, '<mark>$1</mark>');
    }

    handleStartParam(param) {
        console.log('Start param:', param);
        if (param.startsWith('channel_')) {
            const channelId = param.replace('channel_', '');
            setTimeout(() => this.openChannelById(channelId), 500);
        } else if (param.startsWith('category_')) {
            const categoryId = param.replace('category_', '');
            setTimeout(() => Router.openCategory(categoryId), 500);
        }
    }

    openChannelById(channelId) {
        const allOrgs = Router.getAllOrganizations();
        const org = allOrgs.find(o => o.id === channelId);
        if (org && org.deepLink) {
            MaxBridge.openLink(org.deepLink);
        }
    }

    handleViewportChange() {
        const viewport = MaxBridge.getViewportSize();
        console.log('Viewport changed:', viewport);
    }

    setupSwipeGestures() {
        let touchStartX = 0;
        let touchEndX = 0;

        document.addEventListener('touchstart', (e) => {
            touchStartX = e.changedTouches[0].screenX;
        }, false);

        document.addEventListener('touchend', (e) => {
            touchEndX = e.changedTouches[0].screenX;
            this.handleSwipe(touchStartX, touchEndX);
        }, false);
    }

    handleSwipe(startX, endX) {
        const swipeThreshold = 50;
        const diff = startX - endX;
        if (diff > swipeThreshold && Router.currentScreen !== 'app') {
            Router.back();
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const app = new ChannelRouterApp();
    app.init();
});

window.Router = Router;