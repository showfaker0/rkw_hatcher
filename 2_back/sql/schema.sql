-- 库：rkw_hatcher_server
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE IF NOT EXISTS egg_groups (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_egg_groups_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='蛋组字典（仅库内维护）';

CREATE TABLE IF NOT EXISTS natures (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(32) NOT NULL,
  boost VARCHAR(32) NOT NULL COMMENT '增益',
  penalty VARCHAR(32) NOT NULL COMMENT '减益',
  PRIMARY KEY (id),
  UNIQUE KEY uk_natures_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='性格字典（仅库内维护）';

CREATE TABLE IF NOT EXISTS medals (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  type VARCHAR(32) NOT NULL COMMENT '奖章类型',
  name VARCHAR(64) NOT NULL COMMENT '奖章名称',
  PRIMARY KEY (id),
  UNIQUE KEY uk_medals_type_name (type, name),
  KEY idx_medals_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='奖章字典（仅库内维护）';

CREATE TABLE IF NOT EXISTS species (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `no` INT UNSIGNED NULL COMMENT '图鉴编号（BWIKI handbook）',
  name VARCHAR(64) NOT NULL COMMENT '最终形态名',
  icon_url VARCHAR(512) NULL COMMENT '精灵图标 URL',
  evo_chain JSON NOT NULL COMMENT '进化链字符串数组',
  notes TEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uk_species_name (name),
  KEY idx_species_no (`no`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='图鉴（最终形态）';

CREATE TABLE IF NOT EXISTS species_best_natures (
  species_id BIGINT UNSIGNED NOT NULL,
  nature_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (species_id, nature_id),
  KEY idx_sbn_nature (nature_id),
  CONSTRAINT fk_sbn_species FOREIGN KEY (species_id) REFERENCES species (id) ON DELETE CASCADE,
  CONSTRAINT fk_sbn_nature FOREIGN KEY (nature_id) REFERENCES natures (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='图鉴-最佳PVP性格（可多个）';

CREATE TABLE IF NOT EXISTS species_egg_groups (
  species_id BIGINT UNSIGNED NOT NULL,
  egg_group_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (species_id, egg_group_id),
  KEY idx_seg_egg (egg_group_id),
  CONSTRAINT fk_seg_species FOREIGN KEY (species_id) REFERENCES species (id) ON DELETE CASCADE,
  CONSTRAINT fk_seg_egg FOREIGN KEY (egg_group_id) REFERENCES egg_groups (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='图鉴-蛋组多对多';

CREATE TABLE IF NOT EXISTS my_pets (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  species_id BIGINT UNSIGNED NOT NULL,
  gender ENUM('公','母') NOT NULL,
  nature_id BIGINT UNSIGNED NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT '空闲中',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_pets_species (species_id),
  KEY idx_pets_nature (nature_id),
  KEY idx_pets_nature_gender (nature_id, gender),
  KEY idx_pets_status (status),
  CONSTRAINT fk_pets_species FOREIGN KEY (species_id) REFERENCES species (id),
  CONSTRAINT fk_pets_nature FOREIGN KEY (nature_id) REFERENCES natures (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='我的宠物';

CREATE TABLE IF NOT EXISTS my_pet_medals (
  my_pet_id BIGINT UNSIGNED NOT NULL,
  medal_type VARCHAR(32) NOT NULL,
  medal_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (my_pet_id, medal_type),
  KEY idx_mpm_medal (medal_id),
  CONSTRAINT fk_mpm_pet FOREIGN KEY (my_pet_id) REFERENCES my_pets (id) ON DELETE CASCADE,
  CONSTRAINT fk_mpm_medal FOREIGN KEY (medal_id) REFERENCES medals (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='宠物奖章选择（每类型至多一个）';

CREATE TABLE IF NOT EXISTS breeding_lines (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(128) NOT NULL,
  target_species_id BIGINT UNSIGNED NOT NULL,
  expected_nature_id BIGINT UNSIGNED NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT '进行中',
  steps_note TEXT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_bl_target (target_species_id),
  KEY idx_bl_nature (expected_nature_id),
  CONSTRAINT fk_bl_target FOREIGN KEY (target_species_id) REFERENCES species (id),
  CONSTRAINT fk_bl_nature FOREIGN KEY (expected_nature_id) REFERENCES natures (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='配种线';

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
  scheme_type TINYINT UNSIGNED NOT NULL COMMENT '1强 2弱公错 3弱母错 4极低效双错',
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

SET FOREIGN_KEY_CHECKS = 1;

INSERT INTO egg_groups (id, name) VALUES
(1,'无法孵蛋'),(2,'巨灵'),(3,'两栖'),(4,'昆虫'),(5,'天空'),
(6,'动物'),(7,'妖精'),(8,'植物'),(9,'拟人'),(10,'软体'),
(11,'大地'),(12,'魔力'),(13,'海洋'),(14,'巨龙'),(15,'机械')
ON DUPLICATE KEY UPDATE name = VALUES(name);

INSERT INTO natures (id, name, boost, penalty) VALUES
(1,'沉默','生命','物攻'),
(2,'平和','生命','魔攻'),
(3,'忧郁','生命','物防'),
(4,'粗心','生命','魔防'),
(5,'踏实','生命','速度'),
(6,'逞强','物攻','生命'),
(7,'固执','物攻','魔攻'),
(8,'大胆','物攻','物防'),
(9,'调皮','物攻','魔防'),
(10,'勇敢','物攻','速度'),
(11,'理性','魔攻','生命'),
(12,'聪明','魔攻','物攻'),
(13,'专注','魔攻','物防'),
(14,'偏执','魔攻','魔防'),
(15,'冷静','魔攻','速度'),
(16,'坦率','物防','生命'),
(17,'稳重','物防','物攻'),
(18,'天真','物防','魔攻'),
(19,'懒散','物防','魔防'),
(20,'悠闲','物防','速度'),
(21,'焦虑','魔防','生命'),
(22,'警惕','魔防','物攻'),
(23,'害羞','魔防','魔攻'),
(24,'温顺','魔防','物防'),
(25,'慎重','魔防','速度'),
(26,'热情','速度','生命'),
(27,'胆小','速度','物攻'),
(28,'开朗','速度','魔攻'),
(29,'急躁','速度','物防'),
(30,'莽撞','速度','魔防')
ON DUPLICATE KEY UPDATE boost = VALUES(boost), penalty = VALUES(penalty);

INSERT INTO medals (type, name) VALUES
('体型', '大块头'),
('体型', '小不点'),
('声音', '婉转音'),
('声音', '粗嗓门')
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 图鉴数据请通过前端「更新数据」从 BWIKI 同步，不再预置占位种
