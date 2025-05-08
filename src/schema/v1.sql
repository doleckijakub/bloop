CREATE TABLE domains (
    domain_id SERIAL PRIMARY KEY,
    domain TEXT UNIQUE NOT NULL,
    is_https BOOLEAN NOT NULL
);

CREATE TABLE pages (
    page_id SERIAL PRIMARY KEY,
    domain_id INTEGER NOT NULL REFERENCES domains(domain_id) ON DELETE CASCADE,
    url_path TEXT NOT NULL,
    scraped_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE stems (
    word TEXT PRIMARY KEY,
    stem TEXT NOT NULL
);

CREATE TABLE terms (
    term_id SERIAL PRIMARY KEY,
    term TEXT UNIQUE NOT NULL,
    document_frequency INTEGER NOT NULL,
    idf REAL NOT NULL -- cached log(N / DF(t))
);

CREATE TABLE page_terms (
    page_id INTEGER NOT NULL REFERENCES pages(page_id) ON DELETE CASCADE,
    term_id INTEGER NOT NULL REFERENCES terms(term_id) ON DELETE CASCADE,
    PRIMARY KEY (page_id, term_id),
    term_frequency INTEGER NOT NULL,
    tf_idf REAL -- cached TF-IDF(t, d)
);

CREATE INDEX idx_terms_term ON terms(term);
CREATE INDEX idx_page_terms_term_id ON page_terms(term_id);
CREATE INDEX idx_page_terms_page_id ON page_terms(page_id);