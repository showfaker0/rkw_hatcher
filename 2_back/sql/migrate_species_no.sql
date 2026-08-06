-- 图鉴编号：游戏 handbook 编号，用于展示与排序；保留自增 id 作为主键/外键
ALTER TABLE species
  ADD COLUMN `no` INT UNSIGNED NULL COMMENT '图鉴编号（BWIKI handbook）' AFTER id;

-- 已有数据若无编号可暂时为空；同步写入后补齐
ALTER TABLE species
  ADD UNIQUE KEY uk_species_no (`no`);
