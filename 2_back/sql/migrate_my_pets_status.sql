-- 我的宠物：增加忙碌状态，默认空闲中
-- 执行时请使用：mysql ... --default-character-set=utf8mb4
ALTER TABLE my_pets
  ADD COLUMN status VARCHAR(16) NOT NULL DEFAULT '空闲中' AFTER nature_id;
