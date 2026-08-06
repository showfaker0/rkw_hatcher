-- 产线生产队（一公一母一队）；全局最多 5 队同时生产
CREATE TABLE IF NOT EXISTS breeding_line_runs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  line_id BIGINT UNSIGNED NOT NULL,
  stud_pet_id BIGINT UNSIGNED NOT NULL,
  dam_pet_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_blr_line (line_id),
  KEY idx_blr_stud (stud_pet_id),
  KEY idx_blr_dam (dam_pet_id),
  CONSTRAINT fk_blr_line FOREIGN KEY (line_id) REFERENCES breeding_lines (id) ON DELETE CASCADE,
  CONSTRAINT fk_blr_stud FOREIGN KEY (stud_pet_id) REFERENCES my_pets (id) ON DELETE CASCADE,
  CONSTRAINT fk_blr_dam FOREIGN KEY (dam_pet_id) REFERENCES my_pets (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='产线生产队';
