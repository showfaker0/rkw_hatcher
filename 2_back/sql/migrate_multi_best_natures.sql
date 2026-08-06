-- 若已按旧版 schema 建过库，在 Navicat 对 rkw_hatcher_server 执行本脚本升级
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS species_best_natures (
  species_id BIGINT UNSIGNED NOT NULL,
  nature_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (species_id, nature_id),
  KEY idx_sbn_nature (nature_id),
  CONSTRAINT fk_sbn_species FOREIGN KEY (species_id) REFERENCES species (id) ON DELETE CASCADE,
  CONSTRAINT fk_sbn_nature FOREIGN KEY (nature_id) REFERENCES natures (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='图鉴-最佳PVP性格（可多个）';

-- 迁移旧单字段数据（有 best_pvp_nature_id 列时）
INSERT IGNORE INTO species_best_natures (species_id, nature_id)
SELECT id, best_pvp_nature_id FROM species
WHERE best_pvp_nature_id IS NOT NULL;

ALTER TABLE species DROP FOREIGN KEY fk_species_nature;
ALTER TABLE species DROP COLUMN best_pvp_nature_id;

SET FOREIGN_KEY_CHECKS = 1;
