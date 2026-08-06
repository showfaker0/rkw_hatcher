-- 地区/季节形态可共享同一图鉴编号，去掉 no 唯一约束，保留普通索引便于排序
ALTER TABLE species DROP INDEX uk_species_no;
ALTER TABLE species ADD KEY idx_species_no (`no`);
