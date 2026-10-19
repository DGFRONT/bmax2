const Router = {
    currentScreen: 'welcome',
    history: [],
    appData: null,

    init() {
        this.setupEventListeners();
    },

    setupEventListeners() {
        document.getElementById('startBtn')?.addEventListener('click', () => {
            this.navigateTo('app');
        });

        document.getElementById('allOrgBtn')?.addEventListener('click', () => {
            this.navigateTo('allOrg');
        });

        document.getElementById('categoryBackBtn')?.addEventListener('click', () => {
            this.back();
        });

        document.getElementById('allOrgBackBtn')?.addEventListener('click', () => {
            this.back();
        });
    },

    setAppData(data) {
        this.appData = data;
    },

    navigateTo(screen, params = {}) {
        const previousScreen = this.currentScreen;
        
        document.getElementById('welcomeScreen')?.classList.add('hidden');
        document.getElementById('app')?.classList.add('hidden');
        document.getElementById('categoryScreen')?.classList.add('hidden');
        document.getElementById('allOrgScreen')?.classList.add('hidden');

        switch(screen) {
            case 'welcome':
                document.getElementById('welcomeScreen').classList.remove('hidden');
                MaxBridge.hideBackButton();
                break;
            case 'app':
                document.getElementById('app').classList.remove('hidden');
                MaxBridge.hideBackButton();
                break;
            case 'category':
                document.getElementById('categoryScreen').classList.remove('hidden');
                if (params.categoryId) this.renderCategoryScreen(params.categoryId);
                MaxBridge.showBackButton();
                break;
            case 'allOrg':
                document.getElementById('allOrgScreen').classList.remove('hidden');
                this.renderAllOrgScreen();
                MaxBridge.showBackButton();
                break;
        }

        if (previousScreen !== screen) {
            this.history.push({ screen, params });
        }

        this.currentScreen = screen;
        this.adjustViewport();
    },

    back() {
        if (this.history.length > 1) {
            this.history.pop();
            const previous = this.history[this.history.length - 1];
            this.navigateTo(previous.screen, previous.params);
        } else {
            MaxBridge.close();
        }
    },

    openCategory(categoryId) {
        this.navigateTo('category', { categoryId });
    },

    renderCategoryScreen(categoryId) {
        const category = this.appData.categories.find(c => c.id === categoryId);
        const organizations = this.appData.organizations[categoryId] || [];

        if (category) {
            document.getElementById('categoryTitle').textContent = category.title;
        }

        const container = document.getElementById('organizationsList');
        
        if (organizations.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="empty-state-icon">📭</div>
                    <div class="empty-state-text">В этой категории пока нет организаций</div>
                </div>
            `;
            return;
        }

        container.innerHTML = organizations.map(org => `
            <div class="org-item" data-org-id="${org.id}">
                <div class="org-icon">${org.icon || '🏢'}</div>
                <div class="org-info">
                    <div class="org-name">${org.name}</div>
                    <div class="org-category">${org.category || ''}</div>
                </div>
            </div>
        `).join('');

        container.querySelectorAll('.org-item').forEach(item => {
            item.addEventListener('click', () => {
                const orgId = item.dataset.orgId;
                const org = organizations.find(o => o.id === orgId);
                if (org && org.deepLink) {
                    MaxBridge.openLink(org.deepLink);
                }
            });
        });
    },

    renderAllOrgScreen() {
        const allOrgs = this.getAllOrganizations();
        const grouped = {};
        
        allOrgs.forEach(org => {
            const firstLetter = org.name.charAt(0).toUpperCase();
            if (!grouped[firstLetter]) grouped[firstLetter] = [];
            grouped[firstLetter].push(org);
        });

        const alphabet = 'АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ'.split('');
        const alphabetNav = document.getElementById('alphabetNav');
        
        alphabetNav.innerHTML = alphabet.map(letter => {
            const hasOrgs = grouped[letter] && grouped[letter].length > 0;
            return `
                <div class="alphabet-item ${hasOrgs ? '' : 'empty'}" 
                     data-letter="${letter}"
                     ${hasOrgs ? `onclick="Router.scrollToLetter('${letter}')"` : ''}>
                    ${letter}
                </div>
            `;
        }).join('');

        const container = document.getElementById('allOrgList');
        container.innerHTML = alphabet
            .filter(letter => grouped[letter])
            .map(letter => `
                <div class="letter-group" id="letter-${letter}">
                    <div class="letter-header">${letter}</div>
                    ${grouped[letter].map(org => `
                        <div class="org-item" data-org-id="${org.id}">
                            <div class="org-icon">${org.icon || '🏢'}</div>
                            <div class="org-info">
                                <div class="org-name">${org.name}</div>
                                <div class="org-category">${org.category || ''}</div>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `).join('');

        container.querySelectorAll('.org-item').forEach(item => {
            item.addEventListener('click', () => {
                const orgId = item.dataset.orgId;
                const org = allOrgs.find(o => o.id === orgId);
                if (org && org.deepLink) {
                    MaxBridge.openLink(org.deepLink);
                }
            });
        });
    },

    getAllOrganizations() {
        const all = [];
        for (const category in this.appData.organizations) {
            all.push(...this.appData.organizations[category]);
        }
        all.push(this.appData.governor);
        all.push(...this.appData.popularChannels);
        return all.sort((a, b) => a.name.localeCompare(b.name));
    },

    scrollToLetter(letter) {
        const element = document.getElementById(`letter-${letter}`);
        if (element) {
            element.scrollIntoView({ behavior: 'smooth', block: 'start' });
        }
    },

    adjustViewport() {
        const viewport = MaxBridge.getViewportSize();
        document.documentElement.style.setProperty('--vh', `${viewport.height}px`);
    }
};