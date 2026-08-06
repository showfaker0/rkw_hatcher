-- 重整字典表 ID 为从 1 连续自增，并按名称修复外键
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ===== egg_groups =====
DROP TABLE IF EXISTS egg_groups_new;
CREATE TABLE egg_groups_new (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_egg_groups_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO egg_groups_new (id, name) VALUES
(1,'无法孵蛋'),(2,'巨灵'),(3,'两栖'),(4,'昆虫'),(5,'天空'),
(6,'动物'),(7,'妖精'),(8,'植物'),(9,'拟人'),(10,'软体'),
(11,'大地'),(12,'魔力'),(13,'海洋'),(14,'巨龙'),(15,'机械');

UPDATE species_egg_groups seg
JOIN egg_groups oldg ON oldg.id = seg.egg_group_id
JOIN egg_groups_new newg ON newg.name = oldg.name
SET seg.egg_group_id = newg.id;

DROP TABLE egg_groups;
RENAME TABLE egg_groups_new TO egg_groups;
ALTER TABLE egg_groups AUTO_INCREMENT = 16;

-- ===== natures =====
DROP TABLE IF EXISTS natures_new;
CREATE TABLE natures_new (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name VARCHAR(32) NOT NULL,
  boost VARCHAR(32) NOT NULL,
  penalty VARCHAR(32) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_natures_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO natures_new (id, name, boost, penalty) VALUES
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
(30,'莽撞','速度','魔防');

UPDATE species_best_natures sbn
JOIN natures oldn ON oldn.id = sbn.nature_id
JOIN natures_new newn ON newn.name = oldn.name
SET sbn.nature_id = newn.id;

UPDATE my_pets p
JOIN natures oldn ON oldn.id = p.nature_id
JOIN natures_new newn ON newn.name = oldn.name
SET p.nature_id = newn.id;

UPDATE breeding_lines b
JOIN natures oldn ON oldn.id = b.expected_nature_id
JOIN natures_new newn ON newn.name = oldn.name
SET b.expected_nature_id = newn.id;

DROP TABLE natures;
RENAME TABLE natures_new TO natures;
ALTER TABLE natures AUTO_INCREMENT = 31;

-- ===== medals（顺手整理为 1..N）=====
DROP TABLE IF EXISTS medals_new;
CREATE TABLE medals_new (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  type VARCHAR(32) NOT NULL,
  name VARCHAR(64) NOT NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_medals_type_name (type, name),
  KEY idx_medals_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO medals_new (id, type, name) VALUES
(1,'体型','大块头'),
(2,'体型','小不点'),
(3,'声音','婉转音'),
(4,'声音','粗嗓门');

UPDATE my_pet_medals mpm
JOIN medals oldm ON oldm.id = mpm.medal_id
JOIN medals_new newm ON newm.type = oldm.type AND newm.name = oldm.name
SET mpm.medal_id = newm.id;

DROP TABLE medals;
RENAME TABLE medals_new TO medals;
ALTER TABLE medals AUTO_INCREMENT = 5;

-- ===== species =====
DROP TABLE IF EXISTS species_id_map;
CREATE TABLE species_id_map (
  old_id BIGINT UNSIGNED PRIMARY KEY,
  new_id BIGINT UNSIGNED NOT NULL
);

SET @r = 0;
INSERT INTO species_id_map (old_id, new_id)
SELECT id, (@r:=@r+1) FROM species ORDER BY id;

DROP TABLE IF EXISTS species_new;
CREATE TABLE species_new LIKE species;
INSERT INTO species_new (id, name, evo_chain, notes, created_at, updated_at)
SELECT m.new_id, s.name, s.evo_chain, s.notes, s.created_at, s.updated_at
FROM species s JOIN species_id_map m ON m.old_id = s.id;

UPDATE species_egg_groups seg JOIN species_id_map m ON m.old_id = seg.species_id SET seg.species_id = m.new_id;
UPDATE species_best_natures sbn JOIN species_id_map m ON m.old_id = sbn.species_id SET sbn.species_id = m.new_id;
UPDATE my_pets p JOIN species_id_map m ON m.old_id = p.species_id SET p.species_id = m.new_id;
UPDATE breeding_lines b JOIN species_id_map m ON m.old_id = b.target_species_id SET b.target_species_id = m.new_id;

DROP TABLE species;
RENAME TABLE species_new TO species;
SELECT IFNULL(MAX(id),0)+1 INTO @sai FROM species;
SET @sql = CONCAT('ALTER TABLE species AUTO_INCREMENT = ', @sai);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- ===== my_pets =====
DROP TABLE IF EXISTS pets_id_map;
CREATE TABLE pets_id_map (
  old_id BIGINT UNSIGNED PRIMARY KEY,
  new_id BIGINT UNSIGNED NOT NULL
);
SET @r = 0;
INSERT INTO pets_id_map (old_id, new_id)
SELECT id, (@r:=@r+1) FROM my_pets ORDER BY id;

DROP TABLE IF EXISTS my_pets_new;
CREATE TABLE my_pets_new LIKE my_pets;
INSERT INTO my_pets_new (id, species_id, nickname, gender, nature_id, is_shiny, notes, created_at, updated_at)
SELECT m.new_id, p.species_id, p.nickname, p.gender, p.nature_id, p.is_shiny, p.notes, p.created_at, p.updated_at
FROM my_pets p JOIN pets_id_map m ON m.old_id = p.id;

UPDATE my_pet_medals mpm JOIN pets_id_map m ON m.old_id = mpm.my_pet_id SET mpm.my_pet_id = m.new_id;
UPDATE breeding_lines b JOIN pets_id_map m ON m.old_id = b.stud_pet_id SET b.stud_pet_id = m.new_id;
UPDATE breeding_lines b JOIN pets_id_map m ON m.old_id = b.dam_pet_id SET b.dam_pet_id = m.new_id;

DROP TABLE my_pets;
RENAME TABLE my_pets_new TO my_pets;
SELECT IFNULL(MAX(id),0)+1 INTO @pai FROM my_pets;
SET @sql = CONCAT('ALTER TABLE my_pets AUTO_INCREMENT = ', @pai);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- ===== breeding_lines =====
DROP TABLE IF EXISTS lines_id_map;
CREATE TABLE lines_id_map (
  old_id BIGINT UNSIGNED PRIMARY KEY,
  new_id BIGINT UNSIGNED NOT NULL
);
SET @r = 0;
INSERT INTO lines_id_map (old_id, new_id)
SELECT id, (@r:=@r+1) FROM breeding_lines ORDER BY id;

DROP TABLE IF EXISTS breeding_lines_new;
CREATE TABLE breeding_lines_new LIKE breeding_lines;
INSERT INTO breeding_lines_new (id, name, target_species_id, stud_pet_id, dam_pet_id, expected_nature_id, status, steps_note, created_at, updated_at)
SELECT m.new_id, b.name, b.target_species_id, b.stud_pet_id, b.dam_pet_id, b.expected_nature_id, b.status, b.steps_note, b.created_at, b.updated_at
FROM breeding_lines b JOIN lines_id_map m ON m.old_id = b.id;

DROP TABLE breeding_lines;
RENAME TABLE breeding_lines_new TO breeding_lines;
SELECT IFNULL(MAX(id),0)+1 INTO @lai FROM breeding_lines;
SET @sql = CONCAT('ALTER TABLE breeding_lines AUTO_INCREMENT = ', @lai);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

DROP TABLE IF EXISTS species_id_map;
DROP TABLE IF EXISTS pets_id_map;
DROP TABLE IF EXISTS lines_id_map;

SET FOREIGN_KEY_CHECKS = 1;

-- 校验
SELECT 'egg_groups' t, COUNT(*) c, MIN(id) mn, MAX(id) mx FROM egg_groups
UNION ALL SELECT 'natures', COUNT(*), MIN(id), MAX(id) FROM natures
UNION ALL SELECT 'medals', COUNT(*), MIN(id), MAX(id) FROM medals
UNION ALL SELECT 'species', COUNT(*), MIN(id), MAX(id) FROM species;
