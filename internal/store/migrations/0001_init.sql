CREATE TABLE categories (
    id         TEXT PRIMARY KEY,
    title      TEXT    NOT NULL,
    icon       TEXT    NOT NULL DEFAULT '',
    wide       INTEGER NOT NULL DEFAULT 0 CHECK (wide IN (0, 1)),
    -- Organization the tile opens directly instead of showing a list.
    link       TEXT REFERENCES organizations (id) ON DELETE SET NULL,
    -- JSON array of ordered subheadings for the organization list.
    group_titles TEXT  NOT NULL DEFAULT '[]' CHECK (json_valid(group_titles)),
    position   INTEGER NOT NULL,
    updated_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE media (
    name         TEXT PRIMARY KEY,
    content_type TEXT NOT NULL,
    data         BLOB NOT NULL,
    sha256       TEXT NOT NULL,
    updated_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE organizations (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    url        TEXT NOT NULL UNIQUE,
    icon       TEXT NOT NULL DEFAULT '',
    avatar     TEXT REFERENCES media (name) ON UPDATE CASCADE ON DELETE SET NULL,
    group_name TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- An organization may belong to several categories; position 0 is primary.
CREATE TABLE organization_categories (
    organization_id TEXT    NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    category_id     TEXT    NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    position        INTEGER NOT NULL,
    PRIMARY KEY (organization_id, category_id)
);
CREATE INDEX organization_categories_category ON organization_categories (category_id);

-- Channels shown in the top "official channels" row, in display order.
CREATE TABLE featured (
    organization_id TEXT PRIMARY KEY REFERENCES organizations (id) ON DELETE CASCADE,
    position        INTEGER NOT NULL
);

-- MAX users who opened the mini app (from validated initData).
CREATE TABLE app_users (
    id               INTEGER PRIMARY KEY,
    first_name       TEXT    NOT NULL DEFAULT '',
    last_name        TEXT    NOT NULL DEFAULT '',
    username         TEXT    NOT NULL DEFAULT '',
    language_code    TEXT    NOT NULL DEFAULT '',
    launches         INTEGER NOT NULL DEFAULT 1,
    last_start_param TEXT    NOT NULL DEFAULT '',
    first_seen_at    TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    last_seen_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
CREATE INDEX app_users_last_seen ON app_users (last_seen_at);
