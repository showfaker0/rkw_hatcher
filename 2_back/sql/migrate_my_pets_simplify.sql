-- 我的精灵：去掉昵称 / 异色 / 备注
ALTER TABLE my_pets
  DROP COLUMN nickname,
  DROP COLUMN is_shiny,
  DROP COLUMN notes;
