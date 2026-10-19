// Thin wrapper over the MAX Bridge (window.WebApp from max-web-app.js).
// Every call degrades gracefully when the page runs outside MAX.

const webApp = () => window.WebApp;

export function initData() {
    return webApp()?.initData || '';
}

export function startParam() {
    return webApp()?.initDataUnsafe?.start_param || '';
}

export function ready() {
    webApp()?.ready?.();
}

export function close() {
    webApp()?.close?.();
}

// Opens max.ru links (channels, chats, bots) inside MAX and everything else in
// the browser. Must be called synchronously from a user gesture.
export function openUrl(url) {
    const app = webApp();
    let host = '';
    try {
        host = new URL(url).hostname;
    } catch {
        return;
    }
    if (host === 'max.ru' && app?.openMaxLink) {
        app.openMaxLink(url);
    } else if (app?.openLink) {
        app.openLink(url);
    } else {
        window.open(url, '_blank', 'noopener');
    }
}

export function hapticTap() {
    webApp()?.HapticFeedback?.impactOccurred?.('light');
}

export const backButton = {
    show() {
        webApp()?.BackButton?.show?.();
    },
    hide() {
        webApp()?.BackButton?.hide?.();
    },
    onClick(handler) {
        webApp()?.BackButton?.onClick?.(handler);
    },
};
