const MaxBridge = {
    initData: null,
    user: null,
    isInitialized: false,
    API_URL: 'https://max.rsocit.ru', // ЗАМЕНИТЕ на ваш домен

    async init() {
        return new Promise((resolve) => {
            try {
                if (!window.WebApp) {
                    console.warn('WebApp not loaded, running in dev mode');
                    this.isInitialized = true;
                    resolve(this);
                    return;
                }

                if (window.WebApp.ready) window.WebApp.ready();
                
                if (window.WebApp.expand && typeof window.WebApp.expand === 'function') {
                    try { window.WebApp.expand(); } catch (e) {}
                }

                this.initData = window.WebApp.initDataUnsafe || null;
                this.user = this.initData?.user || null;

                console.log('MAX Bridge initialized:', { user: this.user });
                this.setupTheme();
                this.isInitialized = true;
                resolve(this);
            } catch (error) {
                console.error('Failed to initialize MAX Bridge:', error);
                this.isInitialized = true;
                resolve(this);
            }
        });
    },

    setupTheme() {
        const theme = window.WebApp?.colorScheme || 'dark';
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('max_theme', theme);
    },

    openLink(url) {
        console.log('Opening link:', url);
        if (window.WebApp?.openMaxLink && typeof window.WebApp.openMaxLink === 'function') {
            window.WebApp.openMaxLink(url);
        } else {
            console.log('WebApp.openMaxLink not available, opening in browser');
            window.open(url, '_blank');
        }
    },

    showBackButton() {
        if (window.WebApp?.BackButton && typeof window.WebApp.BackButton.show === 'function') {
            window.WebApp.BackButton.show();
        }
    },

    hideBackButton() {
        if (window.WebApp?.BackButton && typeof window.WebApp.BackButton.hide === 'function') {
            window.WebApp.BackButton.hide();
        }
    },

    getStartParam() {
        return this.initData?.start_param || null;
    },

    close() {
        if (window.WebApp?.close) window.WebApp.close();
    },

    getViewportSize() {
        if (window.WebApp?.getViewportSize && typeof window.WebApp.getViewportSize === 'function') {
            return window.WebApp.getViewportSize();
        }
        return { width: window.innerWidth, height: window.innerHeight };
    }
};