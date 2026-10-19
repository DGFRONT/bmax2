// Client for the mini app back end. Requests carry the signed MAX initData,
// which the server validates on every call.

const INIT_DATA_HEADER = 'X-Max-Init-Data';

export class ApiError extends Error {
    constructor(status, message) {
        super(message);
        this.status = status;
    }
}

async function request(path, initData, options = {}) {
    const response = await fetch(path, {
        ...options,
        headers: { ...options.headers, [INIT_DATA_HEADER]: initData },
    });
    if (!response.ok) {
        let message = response.statusText;
        try {
            message = (await response.json()).error || message;
        } catch {
            // Non-JSON error body; keep the status text.
        }
        throw new ApiError(response.status, message);
    }
    return response.json();
}

export function fetchCatalog(initData) {
    return request('/api/v1/catalog', initData);
}

export function startSession(initData) {
    return request('/api/v1/session', initData, { method: 'POST' });
}
