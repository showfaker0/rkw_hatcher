-- SQLite schema for rkw_hatcher
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS egg_groups (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS natures (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  boost TEXT NOT NULL,
  penalty TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS medals (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  type TEXT NOT NULL,
  name TEXT NOT NULL,
  UNIQUE (type, name)
);

CREATE TABLE IF NOT EXISTS species (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  "no" INTEGER,
  name TEXT NOT NULL UNIQUE,
  icon_url TEXT,
  evo_chain TEXT NOT NULL,
  notes TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime'))
);
CREATE INDEX IF NOT EXISTS idx_species_no ON species ("no");

CREATE TABLE IF NOT EXISTS species_best_natures (
  species_id INTEGER NOT NULL,
  nature_id INTEGER NOT NULL,
  PRIMARY KEY (species_id, nature_id),
  FOREIGN KEY (species_id) REFERENCES species (id) ON DELETE CASCADE,
  FOREIGN KEY (nature_id) REFERENCES natures (id)
);

CREATE TABLE IF NOT EXISTS species_egg_groups (
  species_id INTEGER NOT NULL,
  egg_group_id INTEGER NOT NULL,
  PRIMARY KEY (species_id, egg_group_id),
  FOREIGN KEY (species_id) REFERENCES species (id) ON DELETE CASCADE,
  FOREIGN KEY (egg_group_id) REFERENCES egg_groups (id)
);

CREATE TABLE IF NOT EXISTS my_pets (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  species_id INTEGER NOT NULL,
  gender TEXT NOT NULL CHECK (gender IN ('公','母')),
  nature_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT '空闲中',
  created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
  FOREIGN KEY (species_id) REFERENCES species (id),
  FOREIGN KEY (nature_id) REFERENCES natures (id)
);
CREATE INDEX IF NOT EXISTS idx_pets_species ON my_pets (species_id);
CREATE INDEX IF NOT EXISTS idx_pets_nature ON my_pets (nature_id);
CREATE INDEX IF NOT EXISTS idx_pets_status ON my_pets (status);

CREATE TABLE IF NOT EXISTS my_pet_medals (
  my_pet_id INTEGER NOT NULL,
  medal_type TEXT NOT NULL,
  medal_id INTEGER NOT NULL,
  PRIMARY KEY (my_pet_id, medal_type),
  FOREIGN KEY (my_pet_id) REFERENCES my_pets (id) ON DELETE CASCADE,
  FOREIGN KEY (medal_id) REFERENCES medals (id)
);

CREATE TABLE IF NOT EXISTS breeding_lines (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  target_species_id INTEGER NOT NULL,
  expected_nature_id INTEGER NOT NULL,
  status TEXT NOT NULL DEFAULT '进行中',
  steps_note TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
  FOREIGN KEY (target_species_id) REFERENCES species (id),
  FOREIGN KEY (expected_nature_id) REFERENCES natures (id)
);

CREATE TABLE IF NOT EXISTS breeding_line_medals (
  line_id INTEGER NOT NULL,
  medal_type TEXT NOT NULL,
  medal_id INTEGER NOT NULL,
  PRIMARY KEY (line_id, medal_type),
  FOREIGN KEY (line_id) REFERENCES breeding_lines (id) ON DELETE CASCADE,
  FOREIGN KEY (medal_id) REFERENCES medals (id)
);

CREATE TABLE IF NOT EXISTS breeding_line_schemes (
  line_id INTEGER NOT NULL,
  scheme_type INTEGER NOT NULL CHECK (scheme_type BETWEEN 1 AND 4),
  PRIMARY KEY (line_id, scheme_type),
  FOREIGN KEY (line_id) REFERENCES breeding_lines (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS breeding_line_scheme_pets (
  line_id INTEGER NOT NULL,
  scheme_type INTEGER NOT NULL,
  pet_id INTEGER NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('stud','dam')),
  PRIMARY KEY (line_id, scheme_type, pet_id, role),
  FOREIGN KEY (line_id) REFERENCES breeding_lines (id) ON DELETE CASCADE,
  FOREIGN KEY (pet_id) REFERENCES my_pets (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS breeding_line_runs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  line_id INTEGER NOT NULL,
  stud_pet_id INTEGER NOT NULL,
  dam_pet_id INTEGER NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now','localtime')),
  FOREIGN KEY (line_id) REFERENCES breeding_lines (id) ON DELETE CASCADE,
  FOREIGN KEY (stud_pet_id) REFERENCES my_pets (id) ON DELETE CASCADE,
  FOREIGN KEY (dam_pet_id) REFERENCES my_pets (id) ON DELETE CASCADE
);

CREATE TRIGGER IF NOT EXISTS trg_species_updated_at
AFTER UPDATE ON species
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE species SET updated_at = datetime('now','localtime') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trg_my_pets_updated_at
AFTER UPDATE ON my_pets
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE my_pets SET updated_at = datetime('now','localtime') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trg_breeding_lines_updated_at
AFTER UPDATE ON breeding_lines
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
  UPDATE breeding_lines SET updated_at = datetime('now','localtime') WHERE id = NEW.id;
END;
