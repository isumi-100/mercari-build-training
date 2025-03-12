CREATE TABLE categories (
id INTEGER PRIMARY KEY AUTOINCREMENT,
name TEXT NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS "items" (
id INTEGER PRIMARY KEY AUTOINCREMENT,
name TEXT NOT NULL,
category_id INTEGER NOT NULL,
image TEXT NOT NULL,
FOREIGN KEY (category_id) REFERENCES categories(id)
);
