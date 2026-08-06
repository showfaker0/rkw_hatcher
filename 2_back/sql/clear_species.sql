-- 清空图鉴及相关业务数据；不动 egg_groups / natures / medals
SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

DELETE FROM my_pet_medals;
DELETE FROM breeding_lines;
DELETE FROM my_pets;
DELETE FROM species_best_natures;
DELETE FROM species_egg_groups;
DELETE FROM species;
ALTER TABLE species AUTO_INCREMENT = 1;

SET FOREIGN_KEY_CHECKS = 1;
