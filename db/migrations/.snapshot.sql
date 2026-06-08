CREATE TABLE IF NOT EXISTS tasks (
  id   INTEGER PRIMARY KEY,
  name text    NOT NULL,
  description  text, 
  done BOOLEAN NOT NULL
);