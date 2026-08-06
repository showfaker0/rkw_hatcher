-- 产线：目标奖章 + 分组方案快照；去掉单种公/母字段
-- 若外键不存在，DROP FOREIGN KEY 可能失败，可跳过直接 DROP COLUMN
SET FOREIGN_KEY_CHECKS = 0;

ALTER TABLE breeding_lines
  DROP COLUMN stud_pet_id,
  DROP COLUMN dam_pet_id;

CREATE TABLE IF NOT EXISTS breeding_line_medals (
  line_id BIGINT UNSIGNED NOT NULL,
  medal_type VARCHAR(32) NOT NULL,
  medal_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (line_id, medal_type),
  KEY idx_blm_medal (medal_id),
  CONSTRAINT fk_blm_line FOREIGN KEY (line_id) REFERENCES breeding_lines (id) ON DELETE CASCADE,
  CONSTRAINT fk_blm_medal FOREIGN KEY (medal_id) REFERENCES medals (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='产线目标奖章';

CREATE TABLE IF NOT EXISTS breeding_line_schemes (
  line_id BIGINT UNSIGNED NOT NULL,
  scheme_type TINYINT UNSIGNED NOT NULL COMMENT '1强 2弱公错 3弱母错',
  PRIMARY KEY (line_id, scheme_type),
  CONSTRAINT fk_bls_line FOREIGN KEY (line_id) REFERENCES breeding_lines (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='产线推荐方案类型';

CREATE TABLE IF NOT EXISTS breeding_line_scheme_pets (
  line_id BIGINT UNSIGNED NOT NULL,
  scheme_type TINYINT UNSIGNED NOT NULL,
  pet_id BIGINT UNSIGNED NOT NULL,
  role ENUM('stud','dam') NOT NULL,
  PRIMARY KEY (line_id, scheme_type, pet_id, role),
  KEY idx_blsp_pet (pet_id),
  CONSTRAINT fk_blsp_line FOREIGN KEY (line_id) REFERENCES breeding_lines (id) ON DELETE CASCADE,
  CONSTRAINT fk_blsp_pet FOREIGN KEY (pet_id) REFERENCES my_pets (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='产线方案内精灵快照';

SET FOREIGN_KEY_CHECKS = 1;
